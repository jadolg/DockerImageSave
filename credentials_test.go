package main

import (
	"sync"
	"testing"
)

func TestSetAndGetCredentials(t *testing.T) {
	globalCredentialStore.mu.Lock()
	globalCredentialStore.credentials = make(map[string]RegistryCredentials)
	globalCredentialStore.mu.Unlock()

	SetCredentials("test.registry.io", "testuser", "testpass")

	creds, ok := GetCredentials("test.registry.io")
	if !ok {
		t.Fatal("expected credentials to exist")
	}
	if creds.Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", creds.Username)
	}
	if creds.Password != "testpass" {
		t.Errorf("expected password 'testpass', got '%s'", creds.Password)
	}
}

func TestGetCredentials_NotFound(t *testing.T) {
	_, ok := GetCredentials("nonexistent.registry.io")
	if ok {
		t.Error("expected credentials to not exist")
	}
}

func TestNormalizeRegistry(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"docker.io", "registry-1.docker.io"},
		{"index.docker.io", "registry-1.docker.io"},
		{"registry-1.docker.io", "registry-1.docker.io"},
		{"gcr.io", "gcr.io"},
		{"custom.registry.com", "custom.registry.com"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeRegistry(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeRegistry(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCredentialsConcurrency(t *testing.T) {
	globalCredentialStore.mu.Lock()
	globalCredentialStore.credentials = make(map[string]RegistryCredentials)
	globalCredentialStore.mu.Unlock()

	var wg sync.WaitGroup
	registries := []string{"r1.io", "r2.io", "r3.io", "r4.io", "r5.io"}

	for _, reg := range registries {
		wg.Add(1)
		go func(registry string) {
			defer wg.Done()
			SetCredentials(registry, "user-"+registry, "pass-"+registry)
		}(reg)
	}
	wg.Wait()

	for _, reg := range registries {
		wg.Add(1)
		go func(registry string) {
			defer wg.Done()
			creds, ok := GetCredentials(registry)
			if !ok {
				t.Errorf("expected credentials for %s to exist", registry)
				return
			}
			if creds.Username != "user-"+registry {
				t.Errorf("expected username 'user-%s', got '%s'", registry, creds.Username)
			}
		}(reg)
	}
	wg.Wait()
}
