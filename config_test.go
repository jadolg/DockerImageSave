package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, tempDir)

	configContent := `
port: 9090
cache_dir: /var/cache/images
registries:
  ghcr.io:
    username: testuser
    password: testpass
  registry.example.com:
    username: admin
    password: secret
`
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if config.Port != 9090 {
		t.Errorf("expected port 9090, got %d", config.Port)
	}

	if config.CacheDir != "/var/cache/images" {
		t.Errorf("expected cache_dir '/var/cache/images', got '%s'", config.CacheDir)
	}

	if len(config.Registries) != 2 {
		t.Errorf("expected 2 registries, got %d", len(config.Registries))
	}

	ghcrCreds, ok := config.Registries["ghcr.io"]
	if !ok {
		t.Error("expected ghcr.io registry credentials")
	} else {
		if ghcrCreds.Username != "testuser" {
			t.Errorf("expected username 'testuser', got '%s'", ghcrCreds.Username)
		}
		if ghcrCreds.Password != "testpass" {
			t.Errorf("expected password 'testpass', got '%s'", ghcrCreds.Password)
		}
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, tempDir)

	configContent := `{}`
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if config.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", config.Port)
	}
}

func TestLoadConfig_InvalidPort(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, tempDir)

	configContent := `port: 99999`
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("expected error for invalid port")
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, tempDir)

	configContent := `invalid: yaml: content: [`
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestParseByteSize(t *testing.T) {
	tests := []struct {
		input string
		want  int64
		isErr bool
	}{
		{"1024", 1024, false},
		{"512B", 512, false},
		{"1K", 1024, false},
		{"1KB", 1024, false},
		{"500M", 500 * 1024 * 1024, false},
		{"500MB", 500 * 1024 * 1024, false},
		{"2G", 2 * 1024 * 1024 * 1024, false},
		{"2GB", 2 * 1024 * 1024 * 1024, false},
		{"1T", 1024 * 1024 * 1024 * 1024, false},
		{"1TB", 1024 * 1024 * 1024 * 1024, false},
		{"1.5G", int64(1.5 * 1024 * 1024 * 1024), false},
		{"  2G  ", 2 * 1024 * 1024 * 1024, false},
		{"2g", 2 * 1024 * 1024 * 1024, false},
		{"notanumber", 0, true},
		{"xyzG", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseByteSize(tt.input)
			if tt.isErr {
				if err == nil {
					t.Errorf("parseByteSize(%q): expected error, got %d", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("parseByteSize(%q): unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("parseByteSize(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestLoadConfig_MaxImageSize(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupTempDir(t, tempDir)

	configContent := `max_image_size: 2G`
	configPath := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	want := ByteSize(2 * 1024 * 1024 * 1024)
	if config.MaxImageSize != want {
		t.Errorf("MaxImageSize = %d, want %d", config.MaxImageSize, want)
	}
}

func TestApplyCredentials(t *testing.T) {
	config := &Config{
		Port: 8080,
		Registries: map[string]RegistryConfig{
			"ghcr.io": {
				Username: "testuser",
				Password: "testpass",
			},
		},
	}

	config.ApplyCredentials()

	creds, ok := GetCredentials("ghcr.io")
	if !ok {
		t.Error("expected credentials for ghcr.io")
	}
	if creds.Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", creds.Username)
	}
	if creds.Password != "testpass" {
		t.Errorf("expected password 'testpass', got '%s'", creds.Password)
	}
}
