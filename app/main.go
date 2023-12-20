package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func main() {
	var err error;
	
	image, err := ParseImage(os.Args[2])
	PanicIf(err)
	
	command := os.Args[3]
	args := os.Args[4:len(os.Args)]

	localContainerRoot, err := os.MkdirTemp("", "mocker");
	PanicIf(err)
	defer os.RemoveAll(localContainerRoot)
	
	layers, err := FetchLayers(image);
	PanicIf(err)
	
	for _, layer := range layers {
		err := PullBlob(image, layer, localContainerRoot);
		PanicIf(err)
	}
	
	err = StartContainer(localContainerRoot);
	PanicIf(err)
	
	cmdPath, err := exec.LookPath(command);
	PanicIf(err)

	info, err := os.Stat(cmdPath)
	PanicIf(err)
	
	if info.IsDir() {
		PanicIf(fmt.Errorf("'%s' is a directory", cmdPath))
	}

	err = os.Chmod(cmdPath, 0755);
	PanicIf(err)

  cmd := exec.Command(command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{	Cloneflags: syscall.CLONE_NEWPID }
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	err = cmd.Start()
	PanicIf(err)

	err = cmd.Wait()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		PanicIf(err)
	}
}

func PanicIf(err error) {
	if err != nil { panic(err) }
}

func CopyPreservingMode(src string, dst string) (err error) {
	srcFile, err := os.Open(src);
	if err != nil { return }
	defer srcFile.Close();

	err = os.MkdirAll(filepath.Dir(dst), 0755);
	if err != nil { return }
	
	dstFile, err := os.Create(dst);
	if err != nil { return }
	defer dstFile.Close();
	
	_, err = io.Copy(dstFile, srcFile);
	if err != nil { return }
	
	srcInfo, err := srcFile.Stat();
	if err != nil { return }
	
	err = dstFile.Chmod(srcInfo.Mode());
	return
}

func StartContainer(root string) (err error) {
	err = syscall.Chroot(root);
	if err != nil { return }
	
	err = os.Chdir("/");
	if err != nil { return }

	// Make sure /dev/null exists
	err = os.MkdirAll("/dev", 0755);
	if err != nil { return }
	
	devNull, err := os.Create("/dev/null");
	if err != nil { return }
	devNull.Close();
	
	return;
}

