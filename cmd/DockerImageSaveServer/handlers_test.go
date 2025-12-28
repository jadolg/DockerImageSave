package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	dockerimagesave "github.com/jadolg/DockerImageSave"
)

func createTestRequest(t *testing.T, method, path string, vars map[string]string) *http.Request {
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req = mux.SetURLVars(req, vars)
	return req
}

func resetStateManager() {
	imageStateManager = NewStateManager()
}

func TestPullImageHandler_ErrorState(t *testing.T) {
	resetStateManager()
	imageID := "test/image:latest"
	errorMsg := "Can't find image in DockerHub"

	imageStateManager.SetError(imageID, errorMsg)

	req := createTestRequest(t, "GET", "/pull/test/image:latest", map[string]string{
		"user": "test",
		"id":   "image:latest",
	})
	rr := httptest.NewRecorder()

	PullImageHandler(rr, req)

	var response dockerimagesave.PullResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}

	if response.Status != "Error" {
		t.Errorf("Expected status 'Error', got '%s'", response.Status)
	}
	if response.Error != errorMsg {
		t.Errorf("Expected error '%s', got '%s'", errorMsg, response.Error)
	}
	if response.ID != imageID {
		t.Errorf("Expected ID '%s', got '%s'", imageID, response.ID)
	}
}

func TestPullImageHandler_PullingState(t *testing.T) {
	resetStateManager()
	imageID := "test/image:latest"

	imageStateManager.SetStatus(imageID, StatusPulling)

	req := createTestRequest(t, "GET", "/pull/test/image:latest", map[string]string{
		"user": "test",
		"id":   "image:latest",
	})
	rr := httptest.NewRecorder()

	PullImageHandler(rr, req)

	var response dockerimagesave.PullResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}

	if response.Status != "Downloading" {
		t.Errorf("Expected status 'Downloading', got '%s'", response.Status)
	}
	if response.ID != imageID {
		t.Errorf("Expected ID '%s', got '%s'", imageID, response.ID)
	}
}

func TestPullImageHandler_PulledState(t *testing.T) {
	resetStateManager()
	imageID := "test/image:latest"

	imageStateManager.SetStatus(imageID, StatusPulled)

	req := createTestRequest(t, "GET", "/pull/test/image:latest", map[string]string{
		"user": "test",
		"id":   "image:latest",
	})
	rr := httptest.NewRecorder()

	PullImageHandler(rr, req)

	var response dockerimagesave.PullResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}

	if response.Status != "Downloaded" {
		t.Errorf("Expected status 'Downloaded', got '%s'", response.Status)
	}
}

func TestPullImageHandler_SavingState_ReturnsDownloaded(t *testing.T) {
	resetStateManager()
	imageID := "test/image:latest"

	imageStateManager.SetStatus(imageID, StatusSaving)

	req := createTestRequest(t, "GET", "/pull/test/image:latest", map[string]string{
		"user": "test",
		"id":   "image:latest",
	})
	rr := httptest.NewRecorder()

	PullImageHandler(rr, req)

	var response dockerimagesave.PullResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}

	if response.Status != "Downloaded" {
		t.Errorf("Expected status 'Downloaded', got '%s'", response.Status)
	}
}

func TestPullImageHandler_ReadyState_ReturnsDownloaded(t *testing.T) {
	resetStateManager()
	imageID := "test/image:latest"

	imageStateManager.SetReady(imageID, "download/test_image_latest.tar.zip", 1024)

	req := createTestRequest(t, "GET", "/pull/test/image:latest", map[string]string{
		"user": "test",
		"id":   "image:latest",
	})
	rr := httptest.NewRecorder()

	PullImageHandler(rr, req)

	var response dockerimagesave.PullResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}

	if response.Status != "Downloaded" {
		t.Errorf("Expected status 'Downloaded', got '%s'", response.Status)
	}
}

func TestPullImageHandler_NoUser(t *testing.T) {
	resetStateManager()
	imageID := "alpine:latest"

	imageStateManager.SetStatus(imageID, StatusPulled)

	req := createTestRequest(t, "GET", "/pull/alpine:latest", map[string]string{
		"user": "",
		"id":   "alpine:latest",
	})
	rr := httptest.NewRecorder()

	PullImageHandler(rr, req)

	var response dockerimagesave.PullResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}

	if response.Status != "Downloaded" {
		t.Errorf("Expected status 'Downloaded', got '%s'", response.Status)
	}
	if response.ID != imageID {
		t.Errorf("Expected ID '%s', got '%s'", imageID, response.ID)
	}
}

func TestSaveImageHandler_ErrorState(t *testing.T) {
	resetStateManager()
	imageID := "test/image:latest"
	errorMsg := "Error saving image: disk full"

	imageStateManager.SetError(imageID, errorMsg)

	req := createTestRequest(t, "GET", "/save/test/image:latest", map[string]string{
		"user": "test",
		"id":   "image:latest",
	})
	rr := httptest.NewRecorder()

	SaveImageHandler(rr, req)

	var response dockerimagesave.SaveResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}

	if response.Status != "Error" {
		t.Errorf("Expected status 'Error', got '%s'", response.Status)
	}
	if response.Error != errorMsg {
		t.Errorf("Expected error '%s', got '%s'", errorMsg, response.Error)
	}
}

func TestSaveImageHandler_ReadyState(t *testing.T) {
	resetStateManager()
	imageID := "test/image:latest"
	url := "download/test_image_latest.tar.zip"
	size := int64(2048)

	imageStateManager.SetReady(imageID, url, size)

	req := createTestRequest(t, "GET", "/save/test/image:latest", map[string]string{
		"user": "test",
		"id":   "image:latest",
	})
	rr := httptest.NewRecorder()

	SaveImageHandler(rr, req)

	var response dockerimagesave.SaveResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}

	if response.Status != "Ready" {
		t.Errorf("Expected status 'Ready', got '%s'", response.Status)
	}
	if response.URL != url {
		t.Errorf("Expected URL '%s', got '%s'", url, response.URL)
	}
	if response.Size != size {
		t.Errorf("Expected size %d, got %d", size, response.Size)
	}
}

func TestSaveImageHandler_SavingState(t *testing.T) {
	resetStateManager()
	imageID := "test/image:latest"

	imageStateManager.SetStatus(imageID, StatusSaving)

	req := createTestRequest(t, "GET", "/save/test/image:latest", map[string]string{
		"user": "test",
		"id":   "image:latest",
	})
	rr := httptest.NewRecorder()

	SaveImageHandler(rr, req)

	var response dockerimagesave.SaveResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}

	if response.Status != "Saving" {
		t.Errorf("Expected status 'Saving', got '%s'", response.Status)
	}
}

func TestSaveImageHandler_CompressingState(t *testing.T) {
	resetStateManager()
	imageID := "test/image:latest"

	imageStateManager.SetStatus(imageID, StatusCompressing)

	req := createTestRequest(t, "GET", "/save/test/image:latest", map[string]string{
		"user": "test",
		"id":   "image:latest",
	})
	rr := httptest.NewRecorder()

	SaveImageHandler(rr, req)

	var response dockerimagesave.SaveResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}

	if response.Status != "Saving" {
		t.Errorf("Expected status 'Saving', got '%s'", response.Status)
	}
}

func TestSaveImageHandler_PullingState(t *testing.T) {
	resetStateManager()
	imageID := "test/image:latest"

	imageStateManager.SetStatus(imageID, StatusPulling)

	req := createTestRequest(t, "GET", "/save/test/image:latest", map[string]string{
		"user": "test",
		"id":   "image:latest",
	})
	rr := httptest.NewRecorder()

	SaveImageHandler(rr, req)

	var response dockerimagesave.SaveResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}

	if response.Status != "Pulling" {
		t.Errorf("Expected status 'Pulling', got '%s'", response.Status)
	}
	if response.Error == "" {
		t.Error("Expected error message about image still being pulled")
	}
}

func TestSaveImageHandler_NotPulled(t *testing.T) {
	resetStateManager()

	req := createTestRequest(t, "GET", "/save/test/image:latest", map[string]string{
		"user": "test",
		"id":   "image:latest",
	})
	rr := httptest.NewRecorder()

	SaveImageHandler(rr, req)

	var response dockerimagesave.SaveResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}

	if response.Status != "Error" {
		t.Errorf("Expected status 'Error', got '%s'", response.Status)
	}
	if response.Error != "Image has to be pulled first" {
		t.Errorf("Expected 'Image has to be pulled first' error, got '%s'", response.Error)
	}
}

func TestSaveImageHandler_NoUser(t *testing.T) {
	resetStateManager()
	imageID := "alpine:latest"

	imageStateManager.SetReady(imageID, "download/alpine_latest.tar.zip", 1024)

	req := createTestRequest(t, "GET", "/save/alpine:latest", map[string]string{
		"user": "",
		"id":   "alpine:latest",
	})
	rr := httptest.NewRecorder()

	SaveImageHandler(rr, req)

	var response dockerimagesave.SaveResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}

	if response.Status != "Ready" {
		t.Errorf("Expected status 'Ready', got '%s'", response.Status)
	}
	if response.ID != imageID {
		t.Errorf("Expected ID '%s', got '%s'", imageID, response.ID)
	}
}

func TestSaveImageHandler_TransitionFromPulledToSaving(t *testing.T) {
	resetStateManager()
	imageID := "test/image:latest"

	imageStateManager.SetStatus(imageID, StatusPulled)

	req := createTestRequest(t, "GET", "/save/test/image:latest", map[string]string{
		"user": "test",
		"id":   "image:latest",
	})
	rr := httptest.NewRecorder()

	SaveImageHandler(rr, req)

	var response dockerimagesave.SaveResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}

	if response.Status != "Saving" {
		t.Errorf("Expected status 'Saving', got '%s'", response.Status)
	}

	state := imageStateManager.GetState(imageID)
	if state.Status != StatusSaving {
		t.Errorf("Expected internal state to be 'Saving', got '%s'", state.Status)
	}
}

func TestSaveImageHandler_URLConstruction(t *testing.T) {
	resetStateManager()

	tests := []struct {
		name        string
		user        string
		id          string
		expectedURL string
	}{
		{
			name:        "with user",
			user:        "library",
			id:          "nginx:latest",
			expectedURL: "download/library_nginx_latest.tar.zip",
		},
		{
			name:        "without user",
			user:        "",
			id:          "alpine:3.18",
			expectedURL: "download/alpine_3.18.tar.zip",
		},
		{
			name:        "with complex user",
			user:        "myorg",
			id:          "myimage:v1.0.0",
			expectedURL: "download/myorg_myimage_v1.0.0.tar.zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetStateManager()

			imageID := tt.id
			if tt.user != "" {
				imageID = tt.user + "/" + tt.id
			}
			imageStateManager.SetStatus(imageID, StatusSaving)

			req := createTestRequest(t, "GET", "/save/"+imageID, map[string]string{
				"user": tt.user,
				"id":   tt.id,
			})
			rr := httptest.NewRecorder()

			SaveImageHandler(rr, req)

			var response dockerimagesave.SaveResponse
			err := json.NewDecoder(rr.Body).Decode(&response)
			if err != nil {
				t.Fatal(err)
			}

			if response.URL != tt.expectedURL {
				t.Errorf("Expected URL '%s', got '%s'", tt.expectedURL, response.URL)
			}
		})
	}
}
