package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/jadolg/DockerImageSave"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
)

// PullImageHandler handles pulling a docker image
func PullImageHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	imageExists, err := dockerimagesave.ImageExists(params["id"])
	if err != nil {
		log.Printf("Error checking if image '%s' exists locally", params["id"])
		json.NewEncoder(w).Encode(dockerimagesave.PullResponse{ID: params["id"], Error: err.Error(), Status: "Error"})
		return
	}

	log.Printf("Requested pulling image '%s'", params["id"])

	if !imageExists {
		log.Printf("Image '%s' does not exist locally", params["id"])
		existsInRegistry, err := dockerimagesave.ImageExistsInRegistry(params["id"])
		if err == nil && existsInRegistry {
			log.Printf("Image '%s' exists in registry. Pulling image.", params["id"])
			go func() {
				err2 := dockerimagesave.PullImage(params["id"])
				if err2 != nil {
					json.NewEncoder(w).Encode(dockerimagesave.PullResponse{ID: params["id"], Error: err2.Error(), Status: "Error"})
					return
				}
			}()
			json.NewEncoder(w).Encode(dockerimagesave.PullResponse{ID: params["id"], Status: "Downloading"})
			return
		}
		log.Printf("Image '%s' does not exist in registry.", params["id"])
		json.NewEncoder(w).Encode(dockerimagesave.PullResponse{ID: params["id"], Error: "Can't find image in DockerHub", Status: "Error"})
		return
	}

	log.Printf("Image '%s' was already downloaded.", params["id"])
	json.NewEncoder(w).Encode(dockerimagesave.PullResponse{ID: params["id"], Status: "Downloaded"})
}

// SaveImageHandler handles saving a docker image
func SaveImageHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	imageExists, err := dockerimagesave.ImageExists(params["id"])
	if err != nil {
		json.NewEncoder(w).Encode(dockerimagesave.PullResponse{ID: params["id"], Error: err.Error()})
		return
	}

	if imageExists {
		if !dockerimagesave.FileExists(downloadsFolder+"/"+params["id"]+".tar") && dockerimagesave.FileExists(downloadsFolder+"/"+params["id"]+".tar.zip") {
			json.NewEncoder(w).Encode(dockerimagesave.SaveResponse{ID: params["id"],
				URL:    "download/" + params["id"] + ".tar.zip",
				Size:   dockerimagesave.GetFileSize(downloadsFolder + "/" + params["id"] + ".tar.zip"),
				Status: "Ready",
			})
			return
		}

		if !dockerimagesave.FileExists(downloadsFolder + "/" + params["id"] + ".tar") {
			go func() {
				dockerimagesave.SaveImage(params["id"], downloadsFolder)
				dockerimagesave.ZipFiles(downloadsFolder+"/"+params["id"]+".tar.zip", []string{"/tmp/" + params["id"] + ".tar"})
				os.Remove(downloadsFolder + "/" + params["id"] + ".tar")
			}()
		}

		json.NewEncoder(w).Encode(dockerimagesave.SaveResponse{ID: params["id"],
			URL:    "download/" + params["id"] + ".tar.zip",
			Status: "Saving"})

	} else {
		json.NewEncoder(w).Encode(dockerimagesave.SaveResponse{ID: params["id"], Error: "Image has to be pulled first", Status: "Error"})
	}
}

// HealthCheckHandler responds with data about the host
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	memory, err1 := mem.VirtualMemory()
	host, err2 := host.Info()
	errorMsg := ""
	if err1 != nil {
		errorMsg = err1.Error()
	}
	if err2 != nil {
		errorMsg = err2.Error()
	}
	json.NewEncoder(w).Encode(
		dockerimagesave.HealthCheckResponse{
			Memory:     memory.Total,
			UsedMemory: memory.Used,
			OS:         host.OS,
			Platform:   host.Platform,
			Error:      errorMsg,
		})
}
