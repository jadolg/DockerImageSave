package main

import (
	"testing"
)

func TestNewStateManager(t *testing.T) {
	sm := NewStateManager()
	if sm == nil {
		t.Fatal("NewStateManager returned nil")
	}
	if sm.states == nil {
		t.Fatal("StateManager.states is nil")
	}
}

func TestStateManager_GetState_NotFound(t *testing.T) {
	sm := NewStateManager()
	state := sm.GetState("nonexistent")
	if state != nil {
		t.Errorf("Expected nil for nonexistent image, got %v", state)
	}
}

func TestStateManager_SetStatus(t *testing.T) {
	sm := NewStateManager()
	imageID := "test/image:latest"

	sm.SetStatus(imageID, StatusPulling)

	state := sm.GetState(imageID)
	if state == nil {
		t.Fatal("Expected state, got nil")
	}
	if state.Status != StatusPulling {
		t.Errorf("Expected status %s, got %s", StatusPulling, state.Status)
	}
	if state.ImageID != imageID {
		t.Errorf("Expected imageID %s, got %s", imageID, state.ImageID)
	}

	// Test transitioning status
	sm.SetStatus(imageID, StatusPulled)
	state = sm.GetState(imageID)
	if state.Status != StatusPulled {
		t.Errorf("Expected status %s, got %s", StatusPulled, state.Status)
	}

	// Test creating new state with SetStatus
	sm2 := NewStateManager()
	sm2.SetStatus(imageID, StatusSaving)
	state2 := sm2.GetState(imageID)
	if state2.Status != StatusSaving {
		t.Errorf("Expected status %s, got %s", StatusSaving, state2.Status)
	}
}

func TestStateManager_SetReady(t *testing.T) {
	sm := NewStateManager()
	imageID := "test/image:latest"
	url := "download/test_image_latest.tar.zip"
	size := int64(1024)

	sm.SetReady(imageID, url, size)

	state := sm.GetState(imageID)
	if state.Status != StatusReady {
		t.Errorf("Expected status %s, got %s", StatusReady, state.Status)
	}
	if state.URL != url {
		t.Errorf("Expected URL %s, got %s", url, state.URL)
	}
	if state.Size != size {
		t.Errorf("Expected size %d, got %d", size, state.Size)
	}
}

func TestStateManager_SetReady_ClearsError(t *testing.T) {
	sm := NewStateManager()
	imageID := "test/image:latest"

	sm.SetError(imageID, "some error")
	sm.SetReady(imageID, "url", 100)

	state := sm.GetState(imageID)
	if state.Error != "" {
		t.Errorf("Expected error to be cleared, got %s", state.Error)
	}
}

func TestStateManager_SetError(t *testing.T) {
	sm := NewStateManager()
	imageID := "test/image:latest"
	errMsg := "pull failed: image not found"

	sm.SetError(imageID, errMsg)

	state := sm.GetState(imageID)
	if state.Status != StatusError {
		t.Errorf("Expected status %s, got %s", StatusError, state.Status)
	}
	if state.Error != errMsg {
		t.Errorf("Expected error %s, got %s", errMsg, state.Error)
	}
}

func TestStateManager_IsPulling(t *testing.T) {
	sm := NewStateManager()
	imageID := "test/image:latest"

	if sm.IsPulling(imageID) {
		t.Error("Expected IsPulling to return false for nonexistent image")
	}

	sm.SetStatus(imageID, StatusPulling)
	if !sm.IsPulling(imageID) {
		t.Error("Expected IsPulling to return true")
	}

	sm.SetStatus(imageID, StatusPulled)
	if sm.IsPulling(imageID) {
		t.Error("Expected IsPulling to return false after SetStatus to Pulled")
	}
}

func TestStateManager_IsPulled(t *testing.T) {
	sm := NewStateManager()
	imageID := "test/image:latest"

	if sm.IsPulled(imageID) {
		t.Error("Expected IsPulled to return false for nonexistent image")
	}

	sm.SetStatus(imageID, StatusPulling)
	if sm.IsPulled(imageID) {
		t.Error("Expected IsPulled to return false while pulling")
	}

	sm.SetStatus(imageID, StatusPulled)
	if !sm.IsPulled(imageID) {
		t.Error("Expected IsPulled to return true")
	}

	// IsPulled should also be true for later stages
	sm.SetStatus(imageID, StatusSaving)
	if !sm.IsPulled(imageID) {
		t.Error("Expected IsPulled to return true during saving")
	}

	sm.SetStatus(imageID, StatusCompressing)
	if !sm.IsPulled(imageID) {
		t.Error("Expected IsPulled to return true during compressing")
	}

	sm.SetReady(imageID, "url", 100)
	if !sm.IsPulled(imageID) {
		t.Error("Expected IsPulled to return true when ready")
	}
}

func TestStateManager_IsSaving(t *testing.T) {
	sm := NewStateManager()
	imageID := "test/image:latest"

	if sm.IsSaving(imageID) {
		t.Error("Expected IsSaving to return false for nonexistent image")
	}

	sm.SetStatus(imageID, StatusSaving)
	if !sm.IsSaving(imageID) {
		t.Error("Expected IsSaving to return true")
	}

	sm.SetStatus(imageID, StatusCompressing)
	if sm.IsSaving(imageID) {
		t.Error("Expected IsSaving to return false after SetStatus to Compressing")
	}
}

func TestStateManager_IsCompressing(t *testing.T) {
	sm := NewStateManager()
	imageID := "test/image:latest"

	if sm.IsCompressing(imageID) {
		t.Error("Expected IsCompressing to return false for nonexistent image")
	}

	sm.SetStatus(imageID, StatusCompressing)
	if !sm.IsCompressing(imageID) {
		t.Error("Expected IsCompressing to return true")
	}

	sm.SetReady(imageID, "url", 100)
	if sm.IsCompressing(imageID) {
		t.Error("Expected IsCompressing to return false after SetReady")
	}
}

func TestStateManager_IsReady(t *testing.T) {
	sm := NewStateManager()
	imageID := "test/image:latest"

	if sm.IsReady(imageID) {
		t.Error("Expected IsReady to return false for nonexistent image")
	}

	sm.SetStatus(imageID, StatusPulled)
	if sm.IsReady(imageID) {
		t.Error("Expected IsReady to return false when only pulled")
	}

	sm.SetReady(imageID, "url", 100)
	if !sm.IsReady(imageID) {
		t.Error("Expected IsReady to return true")
	}
}

func TestStateManager_HasError(t *testing.T) {
	sm := NewStateManager()
	imageID := "test/image:latest"

	if sm.HasError(imageID) {
		t.Error("Expected HasError to return false for nonexistent image")
	}

	sm.SetStatus(imageID, StatusPulling)
	if sm.HasError(imageID) {
		t.Error("Expected HasError to return false when pulling")
	}

	sm.SetError(imageID, "error")
	if !sm.HasError(imageID) {
		t.Error("Expected HasError to return true")
	}
}

func TestStateManager_GetError(t *testing.T) {
	sm := NewStateManager()
	imageID := "test/image:latest"
	errMsg := "test error message"

	if sm.GetError(imageID) != "" {
		t.Error("Expected GetError to return empty string for nonexistent image")
	}

	sm.SetStatus(imageID, StatusPulling)
	if sm.GetError(imageID) != "" {
		t.Error("Expected GetError to return empty string when not in error state")
	}

	sm.SetError(imageID, errMsg)
	if sm.GetError(imageID) != errMsg {
		t.Errorf("Expected GetError to return %s, got %s", errMsg, sm.GetError(imageID))
	}
}

func TestStateManager_IsProcessing(t *testing.T) {
	sm := NewStateManager()
	imageID := "test/image:latest"

	if sm.IsProcessing(imageID) {
		t.Error("Expected IsProcessing to return false for nonexistent image")
	}

	sm.SetStatus(imageID, StatusPulling)
	if !sm.IsProcessing(imageID) {
		t.Error("Expected IsProcessing to return true when pulling")
	}

	sm.SetStatus(imageID, StatusPulled)
	if sm.IsProcessing(imageID) {
		t.Error("Expected IsProcessing to return false when pulled")
	}

	sm.SetStatus(imageID, StatusSaving)
	if !sm.IsProcessing(imageID) {
		t.Error("Expected IsProcessing to return true when saving")
	}

	sm.SetStatus(imageID, StatusCompressing)
	if !sm.IsProcessing(imageID) {
		t.Error("Expected IsProcessing to return true when compressing")
	}

	sm.SetReady(imageID, "url", 100)
	if sm.IsProcessing(imageID) {
		t.Error("Expected IsProcessing to return false when ready")
	}

	sm.SetError(imageID, "error")
	if sm.IsProcessing(imageID) {
		t.Error("Expected IsProcessing to return false when error")
	}
}

func TestStateManager_Delete(t *testing.T) {
	sm := NewStateManager()
	imageID := "test/image:latest"

	sm.SetStatus(imageID, StatusPulling)
	if sm.GetState(imageID) == nil {
		t.Fatal("Expected state to exist before delete")
	}

	sm.Delete(imageID)
	if sm.GetState(imageID) != nil {
		t.Error("Expected state to be nil after delete")
	}
}

func TestStateManager_GetState_ReturnsCopy(t *testing.T) {
	sm := NewStateManager()
	imageID := "test/image:latest"

	sm.SetStatus(imageID, StatusPulling)
	state1 := sm.GetState(imageID)
	state1.Status = StatusError // Modify the returned copy

	state2 := sm.GetState(imageID)
	if state2.Status != StatusPulling {
		t.Error("GetState should return a copy, original state should not be modified")
	}
}

func TestStateManager_ConcurrentAccess(t *testing.T) {
	sm := NewStateManager()
	imageID := "test/image:latest"
	done := make(chan bool)

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			sm.SetStatus(imageID, StatusPulling)
			sm.SetStatus(imageID, StatusPulled)
			sm.SetStatus(imageID, StatusSaving)
			sm.SetStatus(imageID, StatusCompressing)
			sm.SetReady(imageID, "url", int64(i))
		}
		done <- true
	}()

	// Concurrent reads
	go func() {
		for i := 0; i < 100; i++ {
			_ = sm.GetState(imageID)
			_ = sm.IsPulling(imageID)
			_ = sm.IsPulled(imageID)
			_ = sm.IsReady(imageID)
			_ = sm.HasError(imageID)
		}
		done <- true
	}()

	<-done
	<-done
	// Test passes if no race condition panic occurs
}

func TestStateManager_FullWorkflow(t *testing.T) {
	sm := NewStateManager()
	imageID := "nginx:latest"
	url := "download/nginx_latest.tar.zip"
	size := int64(50000)

	// Initial state - nothing exists
	if sm.GetState(imageID) != nil {
		t.Error("Expected no initial state")
	}

	// Start pulling
	sm.SetStatus(imageID, StatusPulling)
	if !sm.IsPulling(imageID) {
		t.Error("Expected IsPulling to be true")
	}
	if !sm.IsProcessing(imageID) {
		t.Error("Expected IsProcessing to be true during pulling")
	}

	// Pull complete
	sm.SetStatus(imageID, StatusPulled)
	if sm.IsPulling(imageID) {
		t.Error("Expected IsPulling to be false after pull complete")
	}
	if !sm.IsPulled(imageID) {
		t.Error("Expected IsPulled to be true")
	}

	// Start saving
	sm.SetStatus(imageID, StatusSaving)
	if !sm.IsSaving(imageID) {
		t.Error("Expected IsSaving to be true")
	}
	if !sm.IsProcessing(imageID) {
		t.Error("Expected IsProcessing to be true during saving")
	}

	// Start compressing
	sm.SetStatus(imageID, StatusCompressing)
	if !sm.IsCompressing(imageID) {
		t.Error("Expected IsCompressing to be true")
	}
	if !sm.IsProcessing(imageID) {
		t.Error("Expected IsProcessing to be true during compressing")
	}

	// Ready for download
	sm.SetReady(imageID, url, size)
	if !sm.IsReady(imageID) {
		t.Error("Expected IsReady to be true")
	}
	if sm.IsProcessing(imageID) {
		t.Error("Expected IsProcessing to be false when ready")
	}

	state := sm.GetState(imageID)
	if state.URL != url {
		t.Errorf("Expected URL %s, got %s", url, state.URL)
	}
	if state.Size != size {
		t.Errorf("Expected size %d, got %d", size, state.Size)
	}
}

func TestStateManager_ErrorWorkflow(t *testing.T) {
	sm := NewStateManager()
	imageID := "invalid/image:latest"
	errMsg := "image not found in registry"

	// Start pulling
	sm.SetStatus(imageID, StatusPulling)
	if !sm.IsPulling(imageID) {
		t.Error("Expected IsPulling to be true")
	}

	// Pull fails
	sm.SetError(imageID, errMsg)
	if sm.IsPulling(imageID) {
		t.Error("Expected IsPulling to be false after error")
	}
	if !sm.HasError(imageID) {
		t.Error("Expected HasError to be true")
	}
	if sm.GetError(imageID) != errMsg {
		t.Errorf("Expected error message %s, got %s", errMsg, sm.GetError(imageID))
	}
	if sm.IsProcessing(imageID) {
		t.Error("Expected IsProcessing to be false after error")
	}
}

func TestStateManager_MultipleImages(t *testing.T) {
	sm := NewStateManager()
	image1 := "nginx:latest"
	image2 := "alpine:latest"
	image3 := "ubuntu:22.04"

	sm.SetStatus(image1, StatusPulling)
	sm.SetStatus(image2, StatusPulled)
	sm.SetError(image3, "error")

	if !sm.IsPulling(image1) {
		t.Error("Expected image1 to be pulling")
	}
	if !sm.IsPulled(image2) {
		t.Error("Expected image2 to be pulled")
	}
	if !sm.HasError(image3) {
		t.Error("Expected image3 to have error")
	}

	// Verify independence
	if sm.IsPulling(image2) {
		t.Error("Expected image2 to not be pulling")
	}
	if sm.HasError(image1) {
		t.Error("Expected image1 to not have error")
	}
}
