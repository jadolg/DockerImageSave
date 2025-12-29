package main

import (
	"sync"
)

// RegistryCredentials holds authentication credentials for a registry
type RegistryCredentials struct {
	Username string
	Password string
}

// CredentialStore manages credentials for multiple registries
type CredentialStore struct {
	credentials map[string]RegistryCredentials
	mu          sync.RWMutex
}

var globalCredentialStore = &CredentialStore{
	credentials: make(map[string]RegistryCredentials),
}

// SetCredentials sets credentials for a specific registry
func SetCredentials(registry string, username, password string) {
	globalCredentialStore.mu.Lock()
	defer globalCredentialStore.mu.Unlock()
	globalCredentialStore.credentials[normalizeRegistry(registry)] = RegistryCredentials{
		Username: username,
		Password: password,
	}
}

// GetCredentials retrieves credentials for a specific registry
func GetCredentials(registry string) (RegistryCredentials, bool) {
	globalCredentialStore.mu.RLock()
	defer globalCredentialStore.mu.RUnlock()
	creds, ok := globalCredentialStore.credentials[normalizeRegistry(registry)]
	return creds, ok
}

// normalizeRegistry normalizes registry names for consistent lookup
func normalizeRegistry(registry string) string {
	switch registry {
	case "docker.io", "index.docker.io", "registry-1.docker.io":
		return "registry-1.docker.io"
	}
	return registry
}
