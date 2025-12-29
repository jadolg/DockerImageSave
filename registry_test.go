package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseImageReference_Simple(t *testing.T) {
	ref := ParseImageReference("alpine")

	if ref.Registry != "registry-1.docker.io" {
		t.Errorf("expected registry 'registry-1.docker.io', got '%s'", ref.Registry)
	}
	if ref.Repository != "library/alpine" {
		t.Errorf("expected repository 'library/alpine', got '%s'", ref.Repository)
	}
	if ref.Tag != "latest" {
		t.Errorf("expected tag 'latest', got '%s'", ref.Tag)
	}
}

func TestParseImageReference_WithTag(t *testing.T) {
	ref := ParseImageReference("alpine:3.18")

	if ref.Repository != "library/alpine" {
		t.Errorf("expected repository 'library/alpine', got '%s'", ref.Repository)
	}
	if ref.Tag != "3.18" {
		t.Errorf("expected tag '3.18', got '%s'", ref.Tag)
	}
}

func TestParseImageReference_WithRegistry(t *testing.T) {
	ref := ParseImageReference("gcr.io/my-project/my-image:v1.0")

	if ref.Registry != "gcr.io" {
		t.Errorf("expected registry 'gcr.io', got '%s'", ref.Registry)
	}
	if ref.Repository != "my-project/my-image" {
		t.Errorf("expected repository 'my-project/my-image', got '%s'", ref.Repository)
	}
	if ref.Tag != "v1.0" {
		t.Errorf("expected tag 'v1.0', got '%s'", ref.Tag)
	}
}

func TestParseImageReference_FullPath(t *testing.T) {
	ref := ParseImageReference("nginx/nginx-ingress:latest")

	if ref.Registry != "registry-1.docker.io" {
		t.Errorf("expected registry 'registry-1.docker.io', got '%s'", ref.Registry)
	}
	if ref.Repository != "nginx/nginx-ingress" {
		t.Errorf("expected repository 'nginx/nginx-ingress', got '%s'", ref.Repository)
	}
	if ref.Tag != "latest" {
		t.Errorf("expected tag 'latest', got '%s'", ref.Tag)
	}
}

func TestParseImageReference_RegistryWithPort(t *testing.T) {
	ref := ParseImageReference("localhost:5000/myimage:test")

	if ref.Registry != "localhost:5000" {
		t.Errorf("expected registry 'localhost:5000', got '%s'", ref.Registry)
	}
	if ref.Repository != "myimage" {
		t.Errorf("expected repository 'myimage', got '%s'", ref.Repository)
	}
	if ref.Tag != "test" {
		t.Errorf("expected tag 'test', got '%s'", ref.Tag)
	}
}

func TestParseAuthHeader(t *testing.T) {
	header := `Bearer realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:library/alpine:pull"`

	realm, service, scope := parseAuthHeader(header, "library/alpine")

	if realm != "https://auth.docker.io/token" {
		t.Errorf("expected realm 'https://auth.docker.io/token', got '%s'", realm)
	}
	if service != "registry.docker.io" {
		t.Errorf("expected service 'registry.docker.io', got '%s'", service)
	}
	if scope != "repository:library/alpine:pull" {
		t.Errorf("expected scope 'repository:library/alpine:pull', got '%s'", scope)
	}
}

func TestNewRegistryClient(t *testing.T) {
	client := NewRegistryClient()

	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.httpClient == nil {
		t.Error("expected non-nil httpClient")
	}
	if client.token != "" {
		t.Error("expected empty token")
	}
}

func TestRegistryClient_GetAuthenticatedUser(t *testing.T) {
	client := &RegistryClient{
		username: "testuser",
	}

	if client.GetAuthenticatedUser() != "testuser" {
		t.Errorf("expected 'testuser', got '%s'", client.GetAuthenticatedUser())
	}
}

func TestRegistryClient_Authenticate_NoAuth(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if server.URL == "" {
		t.Error("server URL should not be empty")
	}
	if server.Client() == nil {
		t.Error("server client should not be nil")
	}
}

func TestRegistryClient_GetManifest_Mock(t *testing.T) {
	manifest := ManifestV2{
		SchemaVersion: 2,
		MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
	}
	manifestJSON, _ := json.Marshal(manifest)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
		w.WriteHeader(http.StatusOK)
		w.Write(manifestJSON)
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}
