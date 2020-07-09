package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jadolg/DockerImageSave"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
)

// PullImageHandler handles pulling a docker image
func PullImageHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	user := params["user"]
	imageID := params["id"]
	if user != "" {
		imageID = user + "/" + imageID
	}
	imageExists, err := dockerimagesave.ImageExists(imageID)
	if err != nil {
		log.Printf("Error checking if image '%s' exists locally", imageID)
		json.NewEncoder(w).Encode(dockerimagesave.PullResponse{ID: imageID, Error: err.Error(), Status: "Error"})
		return
	}

	log.Printf("Requested pulling image '%s'", imageID)

	if !imageExists {
		log.Printf("Image '%s' does not exist locally", imageID)
		existsInRegistry, err := dockerimagesave.ImageExistsInRegistry(imageID)
		if err == nil && existsInRegistry {
			log.Printf("Image '%s' exists in registry. Pulling image.", imageID)
			go func() {
				err2 := dockerimagesave.PullImage(imageID)
				if err2 != nil {
					json.NewEncoder(w).Encode(dockerimagesave.PullResponse{ID: imageID, Error: err2.Error(), Status: "Error"})
					return
				}
			}()
			log.Printf("Responding image '%s' is still being downloaded.", imageID)
			json.NewEncoder(w).Encode(dockerimagesave.PullResponse{ID: imageID, Status: "Downloading"})
			return
		}
		log.Printf("Image '%s' does not exist in registry.", imageID)
		json.NewEncoder(w).Encode(dockerimagesave.PullResponse{ID: imageID, Error: "Can't find image in DockerHub", Status: "Error"})
		return
	}

	log.Printf("Image '%s' was already pulled.", imageID)
	json.NewEncoder(w).Encode(dockerimagesave.PullResponse{ID: imageID, Status: "Downloaded"})
}

// SaveImageHandler handles saving a docker image
func SaveImageHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	user := params["user"]
	imageID := params["id"]
	cleanImageID := strings.Replace(imageID, ":", "_", 1)
	imageName := cleanImageID

	if user != "" {
		imageID = user + "/" + imageID
		imageName = user + "_" + imageName
	}

	imageExists, err := dockerimagesave.ImageExists(imageID)
	if err != nil {
		json.NewEncoder(w).Encode(dockerimagesave.PullResponse{ID: imageID, Error: err.Error()})
		return
	}

	log.Printf("Requested saving image '%s'.", imageID)

	if imageExists {
		log.Printf("Image '%s' has already being pulled.", imageID)
		if !dockerimagesave.FileExists(downloadsFolder+"/"+imageName+".tar") && dockerimagesave.FileExists(downloadsFolder+"/"+imageName+".tar.zip") {
			log.Printf("Image '%s' is ready to be downloaded.", imageID)
			json.NewEncoder(w).Encode(dockerimagesave.SaveResponse{ID: imageID,
				URL:    "download/" + imageName + ".tar.zip",
				Size:   dockerimagesave.GetFileSize(downloadsFolder + "/" + imageName + ".tar.zip"),
				Status: "Ready",
			})
			return
		}

		if !dockerimagesave.FileExists(downloadsFolder + "/" + imageName + ".tar") {
			log.Printf("Saving image '%s' into file %s", imageID, downloadsFolder+"/"+imageName+".tar.zip")
			go func() {
				err := dockerimagesave.SaveImage(imageID, downloadsFolder)
				if err != nil {
					log.Println(err)
				}
				err = dockerimagesave.ZipFiles(downloadsFolder+"/"+imageName+".tar.zip", []string{downloadsFolder + "/" + imageName + ".tar"})
				if err != nil {
					log.Println(err)
				}
				os.Remove(downloadsFolder + "/" + imageName + ".tar")
				log.Printf("Removed uncompressed image file '%s'", downloadsFolder+"/"+imageName+".tar")
			}()
		}

		log.Printf("Responding image '%s' is still being saved.", imageID)
		json.NewEncoder(w).Encode(dockerimagesave.SaveResponse{ID: imageID,
			URL:    "download/" + imageName + ".tar.zip",
			Status: "Saving"})

	} else {
		log.Printf("Image '%s' has to be pulled before it's saved", imageID)
		json.NewEncoder(w).Encode(dockerimagesave.SaveResponse{ID: imageID, Error: "Image has to be pulled first", Status: "Error"})
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
