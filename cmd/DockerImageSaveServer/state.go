package main

import (
	"sync"
)

// ImageStatus represents the current status of an image operation
type ImageStatus string

const (
	StatusPulling     ImageStatus = "Pulling"
	StatusPulled      ImageStatus = "Pulled"
	StatusSaving      ImageStatus = "Saving"
	StatusCompressing ImageStatus = "Compressing"
	StatusReady       ImageStatus = "Ready"
	StatusError       ImageStatus = "Error"
)

// ImageState holds the complete state of an image operation
type ImageState struct {
	ImageID string      `json:"image_id"`
	Status  ImageStatus `json:"status"`
	Error   string      `json:"error,omitempty"`
	URL     string      `json:"url,omitempty"`
	Size    int64       `json:"size,omitempty"`
}

// StateManager manages the state of all image operations
type StateManager struct {
	sync.RWMutex
	states map[string]*ImageState
}

// NewStateManager creates a new StateManager instance
func NewStateManager() *StateManager {
	return &StateManager{
		states: make(map[string]*ImageState),
	}
}

// GetState returns the current state of an image, or nil if not found
func (sm *StateManager) GetState(imageID string) *ImageState {
	sm.RLock()
	defer sm.RUnlock()
	if state, exists := sm.states[imageID]; exists {
		// Return a copy to prevent external modification
		stateCopy := *state
		return &stateCopy
	}
	return nil
}

// SetStatus sets the status of an image
func (sm *StateManager) SetStatus(imageID string, status ImageStatus) {
	sm.Lock()
	defer sm.Unlock()
	if state, exists := sm.states[imageID]; exists {
		state.Status = status
	} else {
		sm.states[imageID] = &ImageState{
			ImageID: imageID,
			Status:  status,
		}
	}
}

// SetReady marks an image as ready for download with URL and size
func (sm *StateManager) SetReady(imageID, url string, size int64) {
	sm.Lock()
	defer sm.Unlock()
	if state, exists := sm.states[imageID]; exists {
		state.Status = StatusReady
		state.URL = url
		state.Size = size
		state.Error = ""
	} else {
		sm.states[imageID] = &ImageState{
			ImageID: imageID,
			Status:  StatusReady,
			URL:     url,
			Size:    size,
		}
	}
}

// SetError marks an image operation as failed with an error message
func (sm *StateManager) SetError(imageID, errMsg string) {
	sm.Lock()
	defer sm.Unlock()
	if state, exists := sm.states[imageID]; exists {
		state.Status = StatusError
		state.Error = errMsg
	} else {
		sm.states[imageID] = &ImageState{
			ImageID: imageID,
			Status:  StatusError,
			Error:   errMsg,
		}
	}
}

// IsPulling returns true if the image is currently being pulled
func (sm *StateManager) IsPulling(imageID string) bool {
	sm.RLock()
	defer sm.RUnlock()
	if state, exists := sm.states[imageID]; exists {
		return state.Status == StatusPulling
	}
	return false
}

// IsPulled returns true if the image has been pulled (or is in later stages)
func (sm *StateManager) IsPulled(imageID string) bool {
	sm.RLock()
	defer sm.RUnlock()
	if state, exists := sm.states[imageID]; exists {
		return state.Status == StatusPulled ||
			state.Status == StatusSaving ||
			state.Status == StatusCompressing ||
			state.Status == StatusReady
	}
	return false
}

// IsSaving returns true if the image is currently being saved
func (sm *StateManager) IsSaving(imageID string) bool {
	sm.RLock()
	defer sm.RUnlock()
	if state, exists := sm.states[imageID]; exists {
		return state.Status == StatusSaving
	}
	return false
}

// IsCompressing returns true if the image is currently being compressed
func (sm *StateManager) IsCompressing(imageID string) bool {
	sm.RLock()
	defer sm.RUnlock()
	if state, exists := sm.states[imageID]; exists {
		return state.Status == StatusCompressing
	}
	return false
}

// IsReady returns true if the image is ready for download
func (sm *StateManager) IsReady(imageID string) bool {
	sm.RLock()
	defer sm.RUnlock()
	if state, exists := sm.states[imageID]; exists {
		return state.Status == StatusReady
	}
	return false
}

// HasError returns true if the image operation has failed
func (sm *StateManager) HasError(imageID string) bool {
	sm.RLock()
	defer sm.RUnlock()
	if state, exists := sm.states[imageID]; exists {
		return state.Status == StatusError
	}
	return false
}

// GetError returns the error message for a failed image operation
func (sm *StateManager) GetError(imageID string) string {
	sm.RLock()
	defer sm.RUnlock()
	if state, exists := sm.states[imageID]; exists && state.Status == StatusError {
		return state.Error
	}
	return ""
}

// IsProcessing returns true if the image is in any processing state
func (sm *StateManager) IsProcessing(imageID string) bool {
	sm.RLock()
	defer sm.RUnlock()
	if state, exists := sm.states[imageID]; exists {
		return state.Status == StatusPulling ||
			state.Status == StatusSaving ||
			state.Status == StatusCompressing
	}
	return false
}

// Delete removes an image from the state manager
func (sm *StateManager) Delete(imageID string) {
	sm.Lock()
	defer sm.Unlock()
	delete(sm.states, imageID)
}

// Global state manager instance
var imageStateManager = NewStateManager()
