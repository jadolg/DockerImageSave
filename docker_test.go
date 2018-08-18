package main

import (
	"os"
	"testing"
)

func TestPullImage(t *testing.T) {
	err := PullImage("busybox:1.29.2")
	if err != nil {
		t.Fail()
	}
}

func TestSaveImage(t *testing.T) {
	err := SaveImage("busybox:1.29.2", "/tmp")
	if err != nil {
		t.Fail()
	}
	if _, err := os.Stat("/tmp/busybox:1.29.2.tar"); os.IsNotExist(err) {
		t.Fail()
	}
}

func TestImageExists(t *testing.T) {
	exists, err := ImageExists("busybox:1.29.2")
	if err != nil || !exists {
		t.Fail()
	}
}

func TestImageExistsInRegistry(t *testing.T) {
	exists, err := ImageExistsInRegistry("busybox:1.29.2")
	if err != nil || !exists {
		t.Fail()
	}

	notExists, err2 := ImageExistsInRegistry("qweqwe")
	if err2 != nil || notExists {
		t.Fail()
	}
}
