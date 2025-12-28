package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	dockerimagesave "github.com/jadolg/DockerImageSave"
)

// PullImageHandler handles pulling a docker image
func PullImageHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	user := dockerimagesave.Sanitize(params["user"])
	imageID := dockerimagesave.Sanitize(params["id"])
	if user != "" {
		imageID = user + "/" + imageID
	}

	state := imageStateManager.GetState(imageID)

	if state != nil && state.Status == StatusError {
		log.Printf("Image '%s' previously failed: %s", imageID, state.Error)
		_ = json.NewEncoder(w).Encode(dockerimagesave.PullResponse{ID: imageID, Error: state.Error, Status: "Error"})
		return
	}

	if state != nil && state.Status == StatusPulling {
		log.Printf("Image '%s' is currently being pulled", imageID)
		_ = json.NewEncoder(w).Encode(dockerimagesave.PullResponse{ID: imageID, Status: "Downloading"})
		return
	}

	if state != nil && (state.Status == StatusPulled || state.Status == StatusSaving ||
		state.Status == StatusCompressing || state.Status == StatusReady) {
		log.Printf("Image '%s' was already pulled.", imageID)
		_ = json.NewEncoder(w).Encode(dockerimagesave.PullResponse{ID: imageID, Status: "Downloaded"})
		return
	}

	log.Printf("Requested pulling image '%s'", imageID)

	existsInRegistry, err := dockerimagesave.ImageExistsInRegistry(imageID)
	if err != nil || !existsInRegistry {
		log.Printf("Image '%s' does not exist in registry.", imageID)
		imageStateManager.SetError(imageID, "Can't find image in DockerHub")
		errorsTotalMetric.Inc()
		_ = json.NewEncoder(w).Encode(dockerimagesave.PullResponse{ID: imageID, Error: "Can't find image in DockerHub", Status: "Error"})
		return
	}

	imageStateManager.SetStatus(imageID, StatusPulling)
	log.Printf("Image '%s' exists in registry. Pulling image.", imageID)

	go func() {
		err := dockerimagesave.PullImage(imageID)
		if err != nil {
			imageStateManager.SetError(imageID, err.Error())
			errorsTotalMetric.Inc()
			log.Printf("Error pulling image %s: %v", imageID, err)
			return
		}
		imageStateManager.SetStatus(imageID, StatusPulled)
		pullsCountMetric.Inc()
		log.Printf("Image '%s' pulled successfully", imageID)
	}()

	log.Printf("Responding image '%s' is still being downloaded.", imageID)
	_ = json.NewEncoder(w).Encode(dockerimagesave.PullResponse{ID: imageID, Status: "Downloading"})
}

// SaveImageHandler handles saving a docker image
func SaveImageHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	user := dockerimagesave.Sanitize(params["user"])
	imageID := dockerimagesave.Sanitize(params["id"])
	cleanImageID := strings.Replace(imageID, ":", "_", 1)
	imageName := dockerimagesave.RemoveDoubleDots(cleanImageID)

	if user != "" {
		imageID = user + "/" + imageID
		imageName = user + "_" + imageName
	}

	log.Printf("Requested saving image '%s'.", imageID)

	state := imageStateManager.GetState(imageID)

	if state != nil && state.Status == StatusError {
		log.Printf("Image '%s' previously failed: %s", imageID, state.Error)
		_ = json.NewEncoder(w).Encode(dockerimagesave.SaveResponse{ID: imageID, Error: state.Error, Status: "Error"})
		return
	}

	if state != nil && state.Status == StatusReady {
		log.Printf("Image '%s' is ready to be downloaded.", imageID)
		_ = json.NewEncoder(w).Encode(dockerimagesave.SaveResponse{
			ID:     imageID,
			URL:    state.URL,
			Size:   state.Size,
			Status: "Ready",
		})
		return
	}

	if state != nil && (state.Status == StatusSaving || state.Status == StatusCompressing) {
		log.Printf("Image '%s' is currently being saved/compressed.", imageID)
		_ = json.NewEncoder(w).Encode(dockerimagesave.SaveResponse{
			ID:     imageID,
			URL:    "download/" + imageName + ".tar.zip",
			Status: "Saving",
		})
		return
	}

	if state != nil && state.Status == StatusPulling {
		log.Printf("Image '%s' is still being pulled.", imageID)
		_ = json.NewEncoder(w).Encode(dockerimagesave.SaveResponse{
			ID:     imageID,
			Error:  "Image is still being pulled, please wait",
			Status: "Pulling",
		})
		return
	}

	if state == nil || state.Status != StatusPulled {
		log.Printf("Image '%s' has to be pulled before it's saved", imageID)
		_ = json.NewEncoder(w).Encode(dockerimagesave.SaveResponse{ID: imageID, Error: "Image has to be pulled first", Status: "Error"})
		return
	}

	imageStateManager.SetStatus(imageID, StatusSaving)
	log.Printf("Saving image '%s' into file %s", imageID, downloadsFolder+"/"+imageName+".tar.zip")

	go func() {
		err := dockerimagesave.SaveImage(imageID, downloadsFolder)
		if err != nil {
			imageStateManager.SetError(imageID, fmt.Sprintf("Error saving image: %v", err))
			errorsTotalMetric.Inc()
			log.Println(err)
			return
		}

		imageStateManager.SetStatus(imageID, StatusCompressing)
		log.Printf("Compressing image '%s'", imageID)

		err = dockerimagesave.ZipFiles(downloadsFolder+"/"+imageName+".tar.zip", []string{downloadsFolder + "/" + imageName + ".tar"})
		if err != nil {
			imageStateManager.SetError(imageID, fmt.Sprintf("Error compressing image: %v", err))
			errorsTotalMetric.Inc()
			log.Println(err)
			return
		}

		err = os.Remove(downloadsFolder + "/" + imageName + ".tar")
		if err != nil {
			log.Print(err)
		}
		log.Printf("Removed uncompressed image file '%s'", downloadsFolder+"/"+imageName+".tar")

		url := "download/" + imageName + ".tar.zip"
		size := dockerimagesave.GetFileSize(downloadsFolder + "/" + imageName + ".tar.zip")
		imageStateManager.SetReady(imageID, url, size)
		log.Printf("Image '%s' is ready for download", imageID)
	}()

	log.Printf("Responding image '%s' is still being saved.", imageID)
	_ = json.NewEncoder(w).Encode(dockerimagesave.SaveResponse{
		ID:     imageID,
		URL:    "download/" + imageName + ".tar.zip",
		Status: "Saving",
	})
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
