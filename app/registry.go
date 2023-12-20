package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const REGISTRY_URL = "https://registry-1.docker.io/v2"
const AUTH_URL = "https://auth.docker.io"
const REPOSITORY = "library"

type Image struct {
	Name string
	Tag string
}

type AuthenticationResponse struct {
	Token string `json:"token"`
	AccessToken string `json:"access_token"`
	ExpiresIn int `json:"expires_in"`
	IssuedAt string `json:"issued_at"`
}
	
type Layer struct {
	MediaType string `json:"mediaType"`
	Digest string `json:"digest"`
}

type ManifestResponse struct {
	SchemaVersion int `json:"schemaVersion"`
	MediaType string `json:"mediaType"`
	Layers []Layer `json:"layers,omitempty"`
}

// Cache the token for the duration of the process
var TOKEN string;

func authenticate(image Image) (token string, err error) {
	if TOKEN != "" {
		return TOKEN, nil;
	}

	Debugf("Retrieving Access Token for %s", REGISTRY_URL);
	
	url := fmt.Sprintf("%s/token?service=registry.docker.io&scope=repository:%s/%s:pull", AUTH_URL, REPOSITORY, image.Name);
	
	response, err := http.Get(url);
	if err != nil { return };
	
	decoder := json.NewDecoder(response.Body);
	
	var blob AuthenticationResponse;
	decoder.Decode(&blob);
	token = blob.Token;
	TOKEN = token;
	
	Debug("Authentication OK");
	return;
}

func FetchLayers(image Image) (layers []Layer, err error) {
	token, err := authenticate(image);
	if err != nil { return };
	
	Infof("%s: Pulling from '%s:%s'...", image.Tag, REPOSITORY, image.Name);
	url := fmt.Sprintf("%s/%s/%s/manifests/%s", REGISTRY_URL, REPOSITORY, image.Name, image.Tag);
	request, err := http.NewRequest("GET", url, nil);
	if err != nil { return };
	request.Header.Add("Authorization", "Bearer " + token);
	request.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json");
	
	client := &http.Client{};
	response, err := client.Do(request);
	if err != nil { return };

	decoder := json.NewDecoder(response.Body);
	
	var blob ManifestResponse;
	decoder.Decode(&blob);
	layers = blob.Layers;
	
	Debugf("Done fetching layers for '%s:%s'", image.Name, image.Tag);
	return;
}

func PullBlob(image Image, layer Layer, destination string) (err error) {
	token, err := authenticate(image);
	if err != nil { return };
 
	short := layer.Digest[7:19];
	Infof("%s: Pulling layer", short);
	
	url := fmt.Sprintf("%s/%s/%s/blobs/%s", REGISTRY_URL, REPOSITORY, image.Name, layer.Digest);
	
	var request *http.Request;
	var response *http.Response;
	
	request, err = http.NewRequest("GET", url, nil);
	if err != nil { return };
	
	request.Header.Add("Authorization", "Bearer " + token);
	
	client := &http.Client{};
	response, err = client.Do(request);
	if err != nil { return };
	defer response.Body.Close()
	
	temporaryFile, err := os.CreateTemp("", "")
	if err != nil { return }
	
	_, err = io.Copy(temporaryFile, response.Body)
	defer temporaryFile.Close()

	extractTarball(temporaryFile.Name(), destination)
	
	Infof("%s: Pull complete", short);
	return
}

func ParseImage(image string) (Image, error) {
	parts := strings.Split(image, ":")

	switch len(parts)	{
	case 1:
		return Image{parts[0], "latest"}, nil;
	case 2:
		return Image{parts[0], parts[1]}, nil;
	default:
		return Image{}, fmt.Errorf("invalid image format: %s", image);
	}
}

func extractTarball(source string, destination string) (err error) {
	cmd := exec.Command("tar", "-xvf", source, "-C", destination)
	return cmd.Run()
}