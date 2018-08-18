package main

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
)

// PullImageHandler handles pulling a docker image
func PullImageHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	imageExists, err := ImageExists(params["id"])
	if err != nil {
		json.NewEncoder(w).Encode(PullResponse{ID: params["id"], Error: err.Error()})
		return
	}

	if !imageExists {
		existsInRegistry, err := ImageExistsInRegistry(params["id"])
		if err == nil && existsInRegistry {
			go func() {
				err2 := PullImage(params["id"])
				if err2 != nil {
					json.NewEncoder(w).Encode(PullResponse{ID: params["id"], Error: err2.Error(), Status: "Error"})
					return
				}
			}()
			json.NewEncoder(w).Encode(PullResponse{ID: params["id"], Status: "Downloading"})
			return
		}

		json.NewEncoder(w).Encode(PullResponse{ID: params["id"], Error: "Can't find image in DockerHub", Status: "Error"})
		return
	}

	json.NewEncoder(w).Encode(PullResponse{ID: params["id"], Status: "Downloaded"})
}

// SaveImageHandler handles saving a docker image
func SaveImageHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	imageExists, err := ImageExists(params["id"])
	if err != nil {
		json.NewEncoder(w).Encode(PullResponse{ID: params["id"], Error: err.Error()})
		return
	}

	if imageExists {
		if !fileExists(downloadsFolder+"/"+params["id"]+".tar") && fileExists(downloadsFolder+"/"+params["id"]+".tar.zip") {
			json.NewEncoder(w).Encode(SaveResponse{ID: params["id"],
				URL:    "/download/" + params["id"] + ".tar.zip",
				Size:   getFileSize(downloadsFolder + "/" + params["id"] + ".tar.zip"),
				Status: "Ready",
			})
			return
		}

		if !fileExists(downloadsFolder + "/" + params["id"] + ".tar") {
			go func() {
				SaveImage(params["id"], downloadsFolder)
				ZipFiles(downloadsFolder+"/"+params["id"]+".tar.zip", []string{"/tmp/" + params["id"] + ".tar"})
				os.Remove(downloadsFolder + "/" + params["id"] + ".tar")
			}()
		}

		json.NewEncoder(w).Encode(SaveResponse{ID: params["id"],
			URL:    "/download/" + params["id"] + ".tar.zip",
			Status: "Saving"})

	} else {
		json.NewEncoder(w).Encode(SaveResponse{ID: params["id"], Error: "Image has to be pulled first", Status: "Error"})
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
		HealthCheckResponse{
			Memory:     memory.Total,
			UsedMemory: memory.Used,
			OS:         host.OS,
			Platform:   host.Platform,
			Error:      errorMsg,
		})
}
