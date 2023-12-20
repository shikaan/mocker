package main

import (
	"log"
	"os"
)

var shouldLog bool = false;

func init() {
	shouldLog = os.Getenv("DEBUG") == "true";
}

func Debug(args ...interface{}) {
	if !shouldLog { return }

	log.Println(args...)
}

func Debugf(message string, args ...interface{}) {
	if !shouldLog { return }

	log.Printf(message, args...)
}

func Info(args ...interface{}) {
	log.Println(args...)
}

func Infof(message string, args ...interface{}) {
	log.Printf(message, args...)
}