package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	dockerimagesave "github.com/jadolg/DockerImageSave"
)

// PullImageRequest pulls a docker image on server
func PullImageRequest(imageid string) (dockerimagesave.PullResponse, error) {
	resp, err := http.Get(ServiceURL + "pull/" + imageid)
	if err != nil {
		return dockerimagesave.PullResponse{}, err
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)

	var pullResponse dockerimagesave.PullResponse
	err = json.Unmarshal(b, &pullResponse)
	if err != nil {
		return dockerimagesave.PullResponse{}, err
	}

	return pullResponse, nil
}

// SaveImageRequest Saves a docker image on server
func SaveImageRequest(imageid string) (dockerimagesave.SaveResponse, error) {
	resp, err := http.Get(ServiceURL + "save/" + imageid)
	if err != nil {
		return dockerimagesave.SaveResponse{}, err
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)

	var saveResponse dockerimagesave.SaveResponse
	err = json.Unmarshal(b, &saveResponse)
	if err != nil {
		return dockerimagesave.SaveResponse{}, err
	}

	return saveResponse, nil
}
