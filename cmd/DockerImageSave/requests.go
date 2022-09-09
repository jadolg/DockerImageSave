package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	dockerimagesave "github.com/jadolg/DockerImageSave"
)

// PullImageRequest pulls a docker image on server
func PullImageRequest(imageid string) (dockerimagesave.PullResponse, error) {
	resp, err := http.Get(ServiceURL + "pull/" + imageid)
	defer dockerimagesave.CloseResponse(resp)
	if err != nil {
		return dockerimagesave.PullResponse{}, err
	}
	b, _ := io.ReadAll(resp.Body)

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
	defer dockerimagesave.CloseResponse(resp)
	if err != nil {
		return dockerimagesave.SaveResponse{}, err
	}
	b, _ := io.ReadAll(resp.Body)

	var saveResponse dockerimagesave.SaveResponse
	err = json.Unmarshal(b, &saveResponse)
	if err != nil {
		return dockerimagesave.SaveResponse{}, err
	}

	return saveResponse, nil
}

// SearchRequest is a wrapper around the docker search API
func SearchRequest(term string) (dockerimagesave.SearchResponse, error) {
	termWithSpaces := strings.ReplaceAll(term, " ", "%20")
	resp, err := http.Get(fmt.Sprintf("%s/search?term=%s", ServiceURL, termWithSpaces))
	defer dockerimagesave.CloseResponse(resp)
	if err != nil {
		return dockerimagesave.SearchResponse{}, err
	}
	b, _ := io.ReadAll(resp.Body)

	var searchResponse dockerimagesave.SearchResponse
	err = json.Unmarshal(b, &searchResponse)
	if err != nil {
		return dockerimagesave.SearchResponse{}, err
	}

	return searchResponse, nil
}
