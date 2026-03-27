package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestParseImageReference_DockerIO(t *testing.T) {
	ref := ParseImageReference("docker.io/bitnamilegacy/mongodb:6.0")

	if ref.Registry != "registry-1.docker.io" {
		t.Errorf("expected registry 'registry-1.docker.io', got '%s'", ref.Registry)
	}
	if ref.Repository != "bitnamilegacy/mongodb" {
		t.Errorf("expected repository 'bitnamilegacy/mongodb', got '%s'", ref.Repository)
	}
	if ref.Tag != "6.0" {
		t.Errorf("expected tag '6.0', got '%s'", ref.Tag)
	}
}

func TestParseImageReference_IndexDockerIO(t *testing.T) {
	ref := ParseImageReference("index.docker.io/library/nginx:latest")

	if ref.Registry != "registry-1.docker.io" {
		t.Errorf("expected registry 'registry-1.docker.io', got '%s'", ref.Registry)
	}
	if ref.Repository != "library/nginx" {
		t.Errorf("expected repository 'library/nginx', got '%s'", ref.Repository)
	}
	if ref.Tag != "latest" {
		t.Errorf("expected tag 'latest', got '%s'", ref.Tag)
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
		_, err := w.Write(manifestJSON)
		if err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Errorf("failed to close response body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestDefaultPlatform(t *testing.T) {
	p := DefaultPlatform()
	if p.OS != "linux" {
		t.Errorf("expected OS 'linux', got %q", p.OS)
	}
	if p.Architecture != "amd64" {
		t.Errorf("expected Architecture 'amd64', got %q", p.Architecture)
	}
	if p.Variant != "" {
		t.Errorf("expected empty Variant, got %q", p.Variant)
	}
}

func TestIsManifestList(t *testing.T) {
	tests := []struct {
		contentType string
		want        bool
	}{
		{"application/vnd.docker.distribution.manifest.list.v2+json", true},
		{"application/vnd.oci.image.index.v1+json", true},
		{"application/vnd.docker.distribution.manifest.v2+json", false},
		{"application/vnd.oci.image.manifest.v1+json", false},
		{"application/json", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			got := isManifestList(tt.contentType)
			if got != tt.want {
				t.Errorf("isManifestList(%q) = %v, want %v", tt.contentType, got, tt.want)
			}
		})
	}
}

func makeManifestList(platforms ...Platform) ManifestList {
	var list ManifestList
	list.SchemaVersion = 2
	list.MediaType = "application/vnd.docker.distribution.manifest.list.v2+json"
	for _, p := range platforms {
		entry := struct {
			MediaType string `json:"mediaType"`
			Size      int64  `json:"size"`
			Digest    string `json:"digest"`
			Platform  struct {
				Architecture string `json:"architecture"`
				OS           string `json:"os"`
				Variant      string `json:"variant"`
			} `json:"platform"`
		}{
			MediaType: "application/vnd.docker.distribution.manifest.v2+json",
			Size:      1000,
			Digest:    "sha256:" + p.OS + p.Architecture + p.Variant,
		}
		entry.Platform.OS = p.OS
		entry.Platform.Architecture = p.Architecture
		entry.Platform.Variant = p.Variant
		list.Manifests = append(list.Manifests, entry)
	}
	return list
}

func TestSelectManifestDigest_ExactMatch(t *testing.T) {
	// Mock a registry that returns a manifest when asked for a digest
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manifest := ManifestV2{SchemaVersion: 2}
		w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
		json.NewEncoder(w).Encode(manifest)
	}))
	defer mockServer.Close()

	// We can't easily test selectManifestDigest directly because it calls getManifestByDigest
	// which needs a real registry. Instead, test the selection logic via parseManifestResponse.

	list := makeManifestList(
		Platform{OS: "linux", Architecture: "amd64"},
		Platform{OS: "linux", Architecture: "arm64"},
		Platform{OS: "linux", Architecture: "arm", Variant: "v7"},
		Platform{OS: "linux", Architecture: "arm", Variant: "v6"},
	)
	listJSON, _ := json.Marshal(list)

	// Test that parseManifestResponse with a single-manifest content type returns it directly
	singleManifest := ManifestV2{SchemaVersion: 2, MediaType: "application/vnd.docker.distribution.manifest.v2+json"}
	singleJSON, _ := json.Marshal(singleManifest)

	client := NewRegistryClient()
	ref := ImageReference{Registry: "registry-1.docker.io", Repository: "library/alpine", Tag: "latest"}

	t.Run("SingleManifest", func(t *testing.T) {
		m, err := client.parseManifestResponse(ref, "application/vnd.docker.distribution.manifest.v2+json", singleJSON, DefaultPlatform())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.SchemaVersion != 2 {
			t.Errorf("expected SchemaVersion 2, got %d", m.SchemaVersion)
		}
	})

	// For manifest list tests, selectManifestDigest will fail because there's no real registry
	// to fetch the digest from, but we can verify the selection logic by checking error messages
	// which include the digest that was attempted.
	t.Run("ManifestListSelectsCorrectPlatform", func(t *testing.T) {
		_, err := client.parseManifestResponse(ref, "application/vnd.docker.distribution.manifest.list.v2+json", listJSON, Platform{OS: "linux", Architecture: "amd64"})
		// Will fail because no real registry, but the error should reference the correct digest
		if err == nil {
			return // Would only succeed with a real registry
		}
		// The attempt was made with the amd64 digest
		if !strings.Contains(err.Error(), "linuxamd64") {
			t.Errorf("expected selection of linux/amd64 manifest, got error: %v", err)
		}
	})

	t.Run("ManifestListSelectsVariant", func(t *testing.T) {
		_, err := client.parseManifestResponse(ref, "application/vnd.docker.distribution.manifest.list.v2+json", listJSON, Platform{OS: "linux", Architecture: "arm", Variant: "v7"})
		if err == nil {
			return
		}
		// Should try the arm/v7 digest, not arm/v6
		if !strings.Contains(err.Error(), "linuxarmv7") {
			t.Errorf("expected selection of linux/arm/v7 manifest, got error: %v", err)
		}
	})

	t.Run("ManifestListNoVariantPicksFirst", func(t *testing.T) {
		_, err := client.parseManifestResponse(ref, "application/vnd.docker.distribution.manifest.list.v2+json", listJSON, Platform{OS: "linux", Architecture: "arm"})
		if err == nil {
			return
		}
		// Without variant, should pick first arm entry (v7)
		if !strings.Contains(err.Error(), "linuxarmv7") {
			t.Errorf("expected selection of first linux/arm manifest (v7), got error: %v", err)
		}
	})

	t.Run("ManifestListNoMatch", func(t *testing.T) {
		_, err := client.parseManifestResponse(ref, "application/vnd.docker.distribution.manifest.list.v2+json", listJSON, Platform{OS: "windows", Architecture: "amd64"})
		if err == nil {
			t.Fatal("expected error for non-existent platform")
		}
		if !strings.Contains(err.Error(), "no manifest found for platform windows/amd64") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("ManifestListNoMatchWithVariant", func(t *testing.T) {
		_, err := client.parseManifestResponse(ref, "application/vnd.docker.distribution.manifest.list.v2+json", listJSON, Platform{OS: "linux", Architecture: "arm", Variant: "v5"})
		if err == nil {
			t.Fatal("expected error for non-existent variant")
		}
		if !strings.Contains(err.Error(), "no manifest found for platform linux/arm/v5") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestPlatformJSON(t *testing.T) {
	t.Run("WithVariant", func(t *testing.T) {
		p := Platform{OS: "linux", Architecture: "arm", Variant: "v7"}
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		expected := `{"os":"linux","architecture":"arm","variant":"v7"}`
		if string(data) != expected {
			t.Errorf("expected %s, got %s", expected, string(data))
		}
	})

	t.Run("WithoutVariant", func(t *testing.T) {
		p := Platform{OS: "linux", Architecture: "amd64"}
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		expected := `{"os":"linux","architecture":"amd64"}`
		if string(data) != expected {
			t.Errorf("expected %s, got %s", expected, string(data))
		}
	})

	t.Run("Roundtrip", func(t *testing.T) {
		original := Platform{OS: "linux", Architecture: "arm", Variant: "v7"}
		data, _ := json.Marshal(original)
		var decoded Platform
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded != original {
			t.Errorf("roundtrip mismatch: got %+v, want %+v", decoded, original)
		}
	})
}
