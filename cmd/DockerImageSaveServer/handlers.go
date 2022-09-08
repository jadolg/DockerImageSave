package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jadolg/DockerImageSave"
)

// PullImageHandler handles pulling a docker image
func PullImageHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	user := dockerimagesave.Sanitize(params["user"])
	imageID := dockerimagesave.Sanitize(params["id"])
	if user != "" {
		imageID = user + "/" + imageID
	}
	imageExists, err := dockerimagesave.ImageExists(imageID)
	if err != nil {
		log.Printf("Error checking if image '%s' exists locally", imageID)
		errorsTotalMetric.Inc()
		_ = json.NewEncoder(w).Encode(dockerimagesave.PullResponse{ID: imageID, Error: err.Error(), Status: "Error"})
		return
	}

	log.Printf("Requested pulling image '%s'", imageID)

	if !imageExists {
		log.Printf("Image '%s' does not exist locally", imageID)
		existsInRegistry, err := dockerimagesave.ImageExistsInRegistry(imageID)
		if err == nil && existsInRegistry {
			log.Printf("Image '%s' exists in registry. Pulling image.", imageID)
			go func() {
				// TODO: This strategy is just plain stupid. Rework into a queue.
				err2 := dockerimagesave.PullImage(imageID)
				if err2 != nil {
					errorsTotalMetric.Inc()
					log.Printf("Error pulling image %s: %v", imageID, err2)
					return
				}
				pullsCountMetric.Inc()
			}()
			log.Printf("Responding image '%s' is still being downloaded.", imageID)
			_ = json.NewEncoder(w).Encode(dockerimagesave.PullResponse{ID: imageID, Status: "Downloading"})
			return
		}
		log.Printf("Image '%s' does not exist in registry.", imageID)
		_ = json.NewEncoder(w).Encode(dockerimagesave.PullResponse{ID: imageID, Error: "Can't find image in DockerHub", Status: "Error"})
		return
	}

	log.Printf("Image '%s' was already pulled.", imageID)
	_ = json.NewEncoder(w).Encode(dockerimagesave.PullResponse{ID: imageID, Status: "Downloaded"})
}

// SaveImageHandler handles saving a docker image
func SaveImageHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	user := dockerimagesave.Sanitize(params["user"])
	user = dockerimagesave.RemoveDoubleDots(user)
	imageID := dockerimagesave.Sanitize(params["id"])
	cleanImageID := strings.Replace(imageID, ":", "_", 1)
	imageName := dockerimagesave.RemoveDoubleDots(cleanImageID)

	if user != "" {
		imageID = user + "/" + imageID
		imageName = user + "_" + imageName
	}

	imageExists, err := dockerimagesave.ImageExists(imageID)
	if err != nil {
		errorsTotalMetric.Inc()
		_ = json.NewEncoder(w).Encode(dockerimagesave.PullResponse{ID: imageID, Error: err.Error()})
		return
	}

	log.Printf("Requested saving image '%s'.", imageID)

	if imageExists {
		log.Printf("Image '%s' has already being pulled.", imageID)
		if !dockerimagesave.FileExists(downloadsFolder+"/"+imageName+".tar") && dockerimagesave.FileExists(downloadsFolder+"/"+imageName+".tar.zip") {
			log.Printf("Image '%s' is ready to be downloaded.", imageID)
			_ = json.NewEncoder(w).Encode(dockerimagesave.SaveResponse{ID: imageID,
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
					errorsTotalMetric.Inc()
					log.Println(err)
				}
				err = dockerimagesave.ZipFiles(downloadsFolder+"/"+imageName+".tar.zip", []string{downloadsFolder + "/" + imageName + ".tar"})
				if err != nil {
					errorsTotalMetric.Inc()
					log.Println(err)
				}
				err = os.Remove(downloadsFolder + "/" + imageName + ".tar")
				if err != nil {
					log.Print(err)
				}
				log.Printf("Removed uncompressed image file '%s'", downloadsFolder+"/"+imageName+".tar")
			}()
		}

		log.Printf("Responding image '%s' is still being saved.", imageID)
		_ = json.NewEncoder(w).Encode(dockerimagesave.SaveResponse{ID: imageID,
			URL:    "download/" + imageName + ".tar.zip",
			Status: "Saving"})

	} else {
		log.Printf("Image '%s' has to be pulled before it's saved", imageID)
		errorsTotalMetric.Inc()
		_ = json.NewEncoder(w).Encode(dockerimagesave.SaveResponse{ID: imageID, Error: "Image has to be pulled first", Status: "Error"})
	}
}

// HealthCheckHandler responds with data about the host
func HealthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	_, err := fmt.Fprintf(w, "OK")
	if err != nil {
		log.Print(err)
	}
}

// SearchHandler handles searching images
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	term := r.FormValue("term")
	term = dockerimagesave.Sanitize(term)
	term = strings.ReplaceAll(term, " ", "%20")
	search, err := dockerimagesave.Search(term)
	if err != nil {
		log.Printf("error searching for %s", term)
		errorsTotalMetric.Inc()
		_ = json.NewEncoder(w).Encode(dockerimagesave.SearchResponse{Term: term, Error: fmt.Sprintf("Error searching for: '%s'", term), Status: "Error"})
		return
	}
	_ = json.NewEncoder(w).Encode(dockerimagesave.SearchResponse{
		Term:         term,
		Status:       "OK",
		SearchResult: search,
	})
}
