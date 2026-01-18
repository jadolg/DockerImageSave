package main

import (
	"strings"
	"testing"
)

func TestValidateRegistry(t *testing.T) {
	tests := []struct {
		name     string
		registry string
		wantErr  bool
		errMsg   string
	}{
		// Valid registries
		{name: "valid docker hub", registry: "registry-1.docker.io", wantErr: false},
		{name: "valid gcr", registry: "gcr.io", wantErr: false},
		{name: "valid ghcr", registry: "ghcr.io", wantErr: false},
		{name: "valid quay", registry: "quay.io", wantErr: false},
		{name: "valid ecr", registry: "123456789.dkr.ecr.us-east-1.amazonaws.com", wantErr: false},
		{name: "valid custom registry", registry: "myregistry.example.com", wantErr: false},
		{name: "valid registry with port", registry: "myregistry.example.com:5000", wantErr: false},
		{name: "valid registry with subdomain", registry: "docker.my.company.com", wantErr: false},
		{name: "valid single word with tld", registry: "registry.io", wantErr: false},
		{name: "valid registry with hyphen", registry: "my-registry.example.com", wantErr: false},
		{name: "valid registry with numbers", registry: "registry1.example.com", wantErr: false},
		{name: "valid registry numbers only subdomain", registry: "123.example.com", wantErr: false},

		// Empty and length validation
		{name: "empty registry", registry: "", wantErr: true, errMsg: "registry cannot be empty"},
		{name: "registry too long", registry: strings.Repeat("a", 254), wantErr: true, errMsg: "registry hostname too long"},
		{name: "registry at max length", registry: strings.Repeat("a", 253), wantErr: false},

		// Invalid format
		{name: "invalid starts with hyphen", registry: "-registry.example.com", wantErr: true, errMsg: "invalid registry hostname"},
		{name: "invalid ends with hyphen", registry: "registry-.example.com", wantErr: true, errMsg: "invalid registry hostname"},
		{name: "invalid double dot", registry: "registry..example.com", wantErr: true, errMsg: "invalid registry hostname"},
		{name: "invalid special chars", registry: "registry@example.com", wantErr: true, errMsg: "invalid registry hostname"},
		{name: "invalid spaces", registry: "registry example.com", wantErr: true, errMsg: "invalid registry hostname"},
		{name: "invalid unicode", registry: "registry.exämple.com", wantErr: true, errMsg: "invalid registry hostname"},
		{name: "invalid path in registry", registry: "registry.example.com/path", wantErr: true, errMsg: "invalid registry hostname"},
		{name: "invalid scheme in registry", registry: "https://registry.example.com", wantErr: true, errMsg: "invalid registry hostname"},
		{name: "invalid port non-numeric", registry: "registry.example.com:abc", wantErr: true, errMsg: "invalid registry hostname"},
		{name: "invalid underscore", registry: "my_registry.example.com", wantErr: true, errMsg: "invalid registry hostname"},

		// SSRF protection - localhost variants
		{name: "ssrf localhost", registry: "localhost", wantErr: true, errMsg: "registry hostname not allowed"},
		{name: "ssrf localhost with port", registry: "localhost:5000", wantErr: true, errMsg: "registry hostname not allowed"},
		{name: "ssrf LOCALHOST uppercase", registry: "LOCALHOST", wantErr: true, errMsg: "registry hostname not allowed"},
		{name: "ssrf localhost mixed case", registry: "LocalHost", wantErr: true, errMsg: "registry hostname not allowed"},
		{name: "ssrf 127.0.0.1", registry: "127.0.0.1", wantErr: true, errMsg: "registry hostname not allowed"},
		{name: "ssrf 127.0.0.1 with port", registry: "127.0.0.1:8080", wantErr: true, errMsg: "registry hostname not allowed"},

		// SSRF protection - private networks
		{name: "ssrf 10.x.x.x", registry: "10.0.0.1", wantErr: true, errMsg: "registry hostname not allowed"},
		{name: "ssrf 10.x.x.x with port", registry: "10.255.255.255:5000", wantErr: true, errMsg: "registry hostname not allowed"},
		{name: "ssrf 172.x.x.x", registry: "172.16.0.1", wantErr: true, errMsg: "registry hostname not allowed"},
		{name: "ssrf 172.31.x.x", registry: "172.31.255.255", wantErr: true, errMsg: "registry hostname not allowed"},
		{name: "ssrf 192.168.x.x", registry: "192.168.0.1", wantErr: true, errMsg: "registry hostname not allowed"},
		{name: "ssrf 192.168.x.x with port", registry: "192.168.1.100:5000", wantErr: true, errMsg: "registry hostname not allowed"},

		// SSRF protection - special addresses
		{name: "ssrf 0.0.0.0", registry: "0.0.0.0", wantErr: true, errMsg: "registry hostname not allowed"},
		{name: "ssrf metadata endpoint", registry: "169.254.169.254", wantErr: true, errMsg: "registry hostname not allowed"},
		{name: "ssrf metadata with port", registry: "169.254.169.254:80", wantErr: true, errMsg: "registry hostname not allowed"},

		// Edge cases that should be allowed (look like internal but aren't)
		{name: "valid 10 prefix domain", registry: "10news.com", wantErr: false},
		{name: "valid 192 prefix domain", registry: "192com.example.com", wantErr: false},
		{name: "valid localhost in domain", registry: "notlocalhost.com", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRegistry(tt.registry)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateRegistry(%q) expected error containing %q, got nil", tt.registry, tt.errMsg)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateRegistry(%q) error = %q, want error containing %q", tt.registry, err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateRegistry(%q) unexpected error: %v", tt.registry, err)
				}
			}
		})
	}
}

func TestValidateRepository(t *testing.T) {
	tests := []struct {
		name       string
		repository string
		wantErr    bool
		errMsg     string
	}{
		// Valid repositories
		{name: "valid simple name", repository: "nginx", wantErr: false},
		{name: "valid with namespace", repository: "library/nginx", wantErr: false},
		{name: "valid deep path", repository: "myorg/myteam/myapp", wantErr: false},
		{name: "valid with numbers", repository: "nginx123", wantErr: false},
		{name: "valid with hyphen", repository: "my-app", wantErr: false},
		{name: "valid with underscore", repository: "my_app", wantErr: false},
		{name: "valid with dot", repository: "my.app", wantErr: false},
		{name: "valid with mixed separators", repository: "my-app_v1.0", wantErr: false},
		{name: "valid namespace with separators", repository: "my-org/my_app", wantErr: false},
		{name: "valid single char", repository: "a", wantErr: false},
		{name: "valid single number", repository: "1", wantErr: false},

		// Empty and length validation
		{name: "empty repository", repository: "", wantErr: true, errMsg: "repository cannot be empty"},
		{name: "repository too long", repository: strings.Repeat("a", 257), wantErr: true, errMsg: "repository name too long"},
		{name: "repository at max length", repository: strings.Repeat("a", 256), wantErr: false},

		// Invalid format
		{name: "invalid uppercase", repository: "MyApp", wantErr: true, errMsg: "invalid repository name"},
		{name: "invalid starts with hyphen", repository: "-myapp", wantErr: true, errMsg: "invalid repository name"},
		{name: "invalid starts with dot", repository: ".myapp", wantErr: true, errMsg: "invalid repository name"},
		{name: "invalid starts with underscore", repository: "_myapp", wantErr: true, errMsg: "invalid repository name"},
		{name: "invalid ends with hyphen", repository: "myapp-", wantErr: true, errMsg: "invalid repository name"},
		{name: "invalid ends with dot", repository: "myapp.", wantErr: true, errMsg: "invalid repository name"},
		{name: "invalid ends with underscore", repository: "myapp_", wantErr: true, errMsg: "invalid repository name"},
		{name: "invalid double hyphen", repository: "my--app", wantErr: true, errMsg: "invalid repository name"},
		{name: "invalid double dot", repository: "my..app", wantErr: true, errMsg: "invalid repository name"},
		{name: "invalid double underscore", repository: "my__app", wantErr: true, errMsg: "invalid repository name"},
		{name: "invalid special chars", repository: "my@app", wantErr: true, errMsg: "invalid repository name"},
		{name: "invalid spaces", repository: "my app", wantErr: true, errMsg: "invalid repository name"},
		{name: "invalid leading slash", repository: "/myapp", wantErr: true, errMsg: "invalid repository name"},
		{name: "invalid trailing slash", repository: "myapp/", wantErr: true, errMsg: "invalid repository name"},
		{name: "invalid double slash", repository: "myorg//myapp", wantErr: true, errMsg: "invalid repository name"},
		{name: "invalid colon", repository: "myapp:latest", wantErr: true, errMsg: "invalid repository name"},
		{name: "invalid unicode", repository: "myäpp", wantErr: true, errMsg: "invalid repository name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRepository(tt.repository)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateRepository(%q) expected error containing %q, got nil", tt.repository, tt.errMsg)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateRepository(%q) error = %q, want error containing %q", tt.repository, err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateRepository(%q) unexpected error: %v", tt.repository, err)
				}
			}
		})
	}
}

func TestValidateTag(t *testing.T) {
	tests := []struct {
		name    string
		tag     string
		wantErr bool
		errMsg  string
	}{
		// Valid tags
		{name: "valid latest", tag: "latest", wantErr: false},
		{name: "valid semver", tag: "v1.0.0", wantErr: false},
		{name: "valid with hyphen", tag: "my-tag", wantErr: false},
		{name: "valid with underscore", tag: "my_tag", wantErr: false},
		{name: "valid with dot", tag: "1.0.0", wantErr: false},
		{name: "valid numbers only", tag: "123", wantErr: false},
		{name: "valid starts with underscore", tag: "_tag", wantErr: false},
		{name: "valid uppercase", tag: "LATEST", wantErr: false},
		{name: "valid mixed case", tag: "MyTag", wantErr: false},
		{name: "valid single char", tag: "a", wantErr: false},
		{name: "valid single number", tag: "1", wantErr: false},
		{name: "valid single underscore", tag: "_", wantErr: false},
		{name: "valid complex tag", tag: "v1.2.3-alpha.1_build.456", wantErr: false},
		{name: "valid sha-like", tag: "sha-abc123def456", wantErr: false},
		{name: "valid 128 chars", tag: strings.Repeat("a", 128), wantErr: false},

		// Empty validation
		{name: "empty tag", tag: "", wantErr: true, errMsg: "tag cannot be empty"},

		// Length validation (max 128 chars)
		{name: "tag too long 129 chars", tag: strings.Repeat("a", 129), wantErr: true, errMsg: "invalid tag"},

		// Invalid format
		{name: "invalid starts with hyphen", tag: "-tag", wantErr: true, errMsg: "invalid tag"},
		{name: "invalid starts with dot", tag: ".tag", wantErr: true, errMsg: "invalid tag"},
		{name: "invalid special chars @", tag: "tag@1", wantErr: true, errMsg: "invalid tag"},
		{name: "invalid special chars !", tag: "tag!", wantErr: true, errMsg: "invalid tag"},
		{name: "invalid special chars #", tag: "tag#1", wantErr: true, errMsg: "invalid tag"},
		{name: "invalid special chars $", tag: "tag$1", wantErr: true, errMsg: "invalid tag"},
		{name: "invalid special chars %", tag: "tag%1", wantErr: true, errMsg: "invalid tag"},
		{name: "invalid special chars ^", tag: "tag^1", wantErr: true, errMsg: "invalid tag"},
		{name: "invalid special chars &", tag: "tag&1", wantErr: true, errMsg: "invalid tag"},
		{name: "invalid special chars *", tag: "tag*1", wantErr: true, errMsg: "invalid tag"},
		{name: "invalid spaces", tag: "my tag", wantErr: true, errMsg: "invalid tag"},
		{name: "invalid slash", tag: "my/tag", wantErr: true, errMsg: "invalid tag"},
		{name: "invalid colon", tag: "my:tag", wantErr: true, errMsg: "invalid tag"},
		{name: "invalid unicode", tag: "täg", wantErr: true, errMsg: "invalid tag"},
		{name: "invalid newline", tag: "tag\n", wantErr: true, errMsg: "invalid tag"},
		{name: "invalid tab", tag: "tag\t", wantErr: true, errMsg: "invalid tag"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTag(tt.tag)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateTag(%q) expected error containing %q, got nil", tt.tag, tt.errMsg)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateTag(%q) error = %q, want error containing %q", tt.tag, err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateTag(%q) unexpected error: %v", tt.tag, err)
				}
			}
		})
	}
}

func TestValidateDigest(t *testing.T) {
	tests := []struct {
		name    string
		digest  string
		wantErr bool
		errMsg  string
	}{
		// Valid digests
		{name: "valid sha256", digest: "sha256:abc123def456", wantErr: false},
		{name: "valid sha256 full", digest: "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4", wantErr: false},
		{name: "valid sha512", digest: "sha512:abc123def456789", wantErr: false},
		{name: "valid md5", digest: "md5:abc123", wantErr: false},
		{name: "valid custom algo", digest: "blake2b:abc123def", wantErr: false},
		{name: "valid short hash", digest: "sha256:a", wantErr: false},

		// Empty validation
		{name: "empty digest", digest: "", wantErr: true, errMsg: "digest cannot be empty"},

		// Length validation
		{name: "digest too long", digest: "sha256:" + strings.Repeat("a", 250), wantErr: true, errMsg: "digest too long"},

		// Invalid format
		{name: "invalid no colon", digest: "sha256abc123", wantErr: true, errMsg: "invalid digest format"},
		{name: "invalid double colon", digest: "sha256::abc123", wantErr: true, errMsg: "invalid digest format"},
		{name: "invalid uppercase algo", digest: "SHA256:abc123", wantErr: true, errMsg: "invalid digest format"},
		{name: "invalid uppercase hash", digest: "sha256:ABC123", wantErr: true, errMsg: "invalid digest format"},
		{name: "invalid mixed case hash", digest: "sha256:AbC123", wantErr: true, errMsg: "invalid digest format"},
		{name: "invalid non-hex chars g", digest: "sha256:abcdefg", wantErr: true, errMsg: "invalid digest format"},
		{name: "invalid non-hex chars z", digest: "sha256:xyz123", wantErr: true, errMsg: "invalid digest format"},
		{name: "invalid special chars", digest: "sha256:abc@123", wantErr: true, errMsg: "invalid digest format"},
		{name: "invalid spaces", digest: "sha256:abc 123", wantErr: true, errMsg: "invalid digest format"},
		{name: "invalid empty algo", digest: ":abc123", wantErr: true, errMsg: "invalid digest format"},
		{name: "invalid empty hash", digest: "sha256:", wantErr: true, errMsg: "invalid digest format"},
		{name: "invalid just colon", digest: ":", wantErr: true, errMsg: "invalid digest format"},
		{name: "invalid algo with hyphen", digest: "sha-256:abc123", wantErr: true, errMsg: "invalid digest format"},
		{name: "invalid algo with underscore", digest: "sha_256:abc123", wantErr: true, errMsg: "invalid digest format"},
		{name: "invalid unicode in hash", digest: "sha256:äbc123", wantErr: true, errMsg: "invalid digest format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDigest(tt.digest)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateDigest(%q) expected error containing %q, got nil", tt.digest, tt.errMsg)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateDigest(%q) error = %q, want error containing %q", tt.digest, err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateDigest(%q) unexpected error: %v", tt.digest, err)
				}
			}
		})
	}
}

func TestValidateImageReference(t *testing.T) {
	tests := []struct {
		name    string
		ref     ImageReference
		wantErr bool
		errMsg  string
	}{
		// Valid references
		{
			name:    "valid docker hub image",
			ref:     ImageReference{Registry: "registry-1.docker.io", Repository: "library/nginx", Tag: "latest"},
			wantErr: false,
		},
		{
			name:    "valid gcr image",
			ref:     ImageReference{Registry: "gcr.io", Repository: "myproject/myimage", Tag: "v1.0.0"},
			wantErr: false,
		},
		{
			name:    "valid ghcr image",
			ref:     ImageReference{Registry: "ghcr.io", Repository: "owner/repo", Tag: "sha-abc123"},
			wantErr: false,
		},
		{
			name:    "valid custom registry with port",
			ref:     ImageReference{Registry: "myregistry.example.com:5000", Repository: "myapp", Tag: "1.0"},
			wantErr: false,
		},

		// Invalid registry
		{
			name:    "invalid empty registry",
			ref:     ImageReference{Registry: "", Repository: "library/nginx", Tag: "latest"},
			wantErr: true,
			errMsg:  "registry cannot be empty",
		},
		{
			name:    "invalid localhost registry",
			ref:     ImageReference{Registry: "localhost", Repository: "myapp", Tag: "latest"},
			wantErr: true,
			errMsg:  "registry hostname not allowed",
		},
		{
			name:    "invalid private ip registry",
			ref:     ImageReference{Registry: "192.168.1.1", Repository: "myapp", Tag: "latest"},
			wantErr: true,
			errMsg:  "registry hostname not allowed",
		},

		// Invalid repository
		{
			name:    "invalid empty repository",
			ref:     ImageReference{Registry: "registry-1.docker.io", Repository: "", Tag: "latest"},
			wantErr: true,
			errMsg:  "repository cannot be empty",
		},
		{
			name:    "invalid uppercase repository",
			ref:     ImageReference{Registry: "registry-1.docker.io", Repository: "MyApp", Tag: "latest"},
			wantErr: true,
			errMsg:  "invalid repository name",
		},

		// Invalid tag
		{
			name:    "invalid empty tag",
			ref:     ImageReference{Registry: "registry-1.docker.io", Repository: "library/nginx", Tag: ""},
			wantErr: true,
			errMsg:  "tag cannot be empty",
		},
		{
			name:    "invalid tag with special chars",
			ref:     ImageReference{Registry: "registry-1.docker.io", Repository: "library/nginx", Tag: "tag@1"},
			wantErr: true,
			errMsg:  "invalid tag",
		},

		// Multiple invalid fields (should fail on first - registry)
		{
			name:    "multiple invalid fields",
			ref:     ImageReference{Registry: "localhost", Repository: "MyApp", Tag: "@invalid"},
			wantErr: true,
			errMsg:  "registry hostname not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateImageReference(tt.ref)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateImageReference() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateImageReference() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateImageReference() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestBuildRegistryURL(t *testing.T) {
	tests := []struct {
		name       string
		registry   string
		pathFormat string
		args       []interface{}
		wantURL    string
		wantErr    bool
		errMsg     string
	}{
		// Valid URL building
		{
			name:       "simple v2 endpoint",
			registry:   "registry-1.docker.io",
			pathFormat: "/v2/",
			args:       nil,
			wantURL:    "https://registry-1.docker.io/v2/",
			wantErr:    false,
		},
		{
			name:       "manifest endpoint",
			registry:   "registry-1.docker.io",
			pathFormat: "/v2/%s/manifests/%s",
			args:       []interface{}{"library/nginx", "latest"},
			wantURL:    "https://registry-1.docker.io/v2/library%2Fnginx/manifests/latest",
			wantErr:    false,
		},
		{
			name:       "blob endpoint",
			registry:   "gcr.io",
			pathFormat: "/v2/%s/blobs/%s",
			args:       []interface{}{"myproject/myimage", "sha256:abc123"},
			wantURL:    "https://gcr.io/v2/myproject%2Fmyimage/blobs/sha256:abc123",
			wantErr:    false,
		},
		{
			name:       "registry with port",
			registry:   "myregistry.example.com:5000",
			pathFormat: "/v2/%s/manifests/%s",
			args:       []interface{}{"myapp", "v1.0"},
			wantURL:    "https://myregistry.example.com:5000/v2/myapp/manifests/v1.0",
			wantErr:    false,
		},
		{
			name:       "path with special chars to escape",
			registry:   "registry-1.docker.io",
			pathFormat: "/v2/%s/manifests/%s",
			args:       []interface{}{"my/repo", "tag with space"},
			wantURL:    "https://registry-1.docker.io/v2/my%2Frepo/manifests/tag%20with%20space",
			wantErr:    false,
		},
		{
			name:       "non-string args",
			registry:   "registry-1.docker.io",
			pathFormat: "/v2/test/%d",
			args:       []interface{}{123},
			wantURL:    "https://registry-1.docker.io/v2/test/123",
			wantErr:    false,
		},

		// Invalid registry - should fail validation
		{
			name:       "invalid empty registry",
			registry:   "",
			pathFormat: "/v2/",
			args:       nil,
			wantErr:    true,
			errMsg:     "registry cannot be empty",
		},
		{
			name:       "invalid localhost registry",
			registry:   "localhost",
			pathFormat: "/v2/",
			args:       nil,
			wantErr:    true,
			errMsg:     "registry hostname not allowed",
		},
		{
			name:       "invalid private ip",
			registry:   "192.168.1.1",
			pathFormat: "/v2/",
			args:       nil,
			wantErr:    true,
			errMsg:     "registry hostname not allowed",
		},
		{
			name:       "invalid metadata endpoint",
			registry:   "169.254.169.254",
			pathFormat: "/latest/meta-data/",
			args:       nil,
			wantErr:    true,
			errMsg:     "registry hostname not allowed",
		},
		{
			name:       "invalid registry format",
			registry:   "https://registry.example.com",
			pathFormat: "/v2/",
			args:       nil,
			wantErr:    true,
			errMsg:     "invalid registry hostname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, err := buildRegistryURL(tt.registry, tt.pathFormat, tt.args...)
			if tt.wantErr {
				if err == nil {
					t.Errorf("buildRegistryURL() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("buildRegistryURL() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("buildRegistryURL() unexpected error: %v", err)
					return
				}
				if gotURL != tt.wantURL {
					t.Errorf("buildRegistryURL() = %q, want %q", gotURL, tt.wantURL)
				}
			}
		})
	}
}

// Test SSRF bypass attempts
func TestValidateRegistry_SSRFBypassAttempts(t *testing.T) {
	tests := []struct {
		name     string
		registry string
		wantErr  bool
	}{
		// Various SSRF bypass attempts that should be blocked
		{name: "decimal ip", registry: "2130706433", wantErr: true},                    // 127.0.0.1 as decimal
		{name: "ipv6 localhost brackets", registry: "[::1]", wantErr: true},            // IPv6 localhost
		{name: "ipv6 localhost", registry: "::1", wantErr: true},                       // IPv6 localhost without brackets
		{name: "zero padded", registry: "127.0.0.01", wantErr: true},                   // Zero padded
		{name: "url encoded", registry: "127%2e0%2e0%2e1", wantErr: true},              // URL encoded dots
		{name: "mixed notation", registry: "0x7f.0.0.1", wantErr: true},                // Hex notation
		{name: "octal notation", registry: "0177.0.0.1", wantErr: true},                // Octal notation
		{name: "double url encoded", registry: "127%252e0%252e0%252e1", wantErr: true}, // Double encoded

		// These should be allowed as they're valid external domains
		{name: "localhost subdomain", registry: "localhost.example.com", wantErr: false},
		{name: "contains 127", registry: "test127.example.com", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRegistry(tt.registry)
			if tt.wantErr && err == nil {
				t.Errorf("validateRegistry(%q) expected error for SSRF bypass attempt, got nil", tt.registry)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateRegistry(%q) unexpected error: %v", tt.registry, err)
			}
		})
	}
}

// Test helper functions
func TestIsNumeric(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{name: "empty string", s: "", want: false},
		{name: "single digit", s: "5", want: true},
		{name: "multiple digits", s: "12345", want: true},
		{name: "zero", s: "0", want: true},
		{name: "leading zeros", s: "007", want: true},
		{name: "contains letter", s: "123a", want: false},
		{name: "contains space", s: "123 ", want: false},
		{name: "contains dot", s: "12.3", want: false},
		{name: "contains hyphen", s: "12-3", want: false},
		{name: "hex prefix", s: "0x123", want: false},
		{name: "unicode digit", s: "１２３", want: false}, // Full-width digits
		{name: "negative sign", s: "-123", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNumeric(tt.s); got != tt.want {
				t.Errorf("isNumeric(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestAllNumeric(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  bool
	}{
		{name: "all numeric", parts: []string{"192", "168", "1", "1"}, want: true},
		{name: "one non-numeric", parts: []string{"192", "168", "1", "abc"}, want: false},
		{name: "empty slice", parts: []string{}, want: true},
		{name: "single numeric", parts: []string{"123"}, want: true},
		{name: "single non-numeric", parts: []string{"abc"}, want: false},
		{name: "contains empty string", parts: []string{"192", "", "1", "1"}, want: false},
		{name: "all zeros", parts: []string{"0", "0", "0", "0"}, want: true},
		{name: "mixed valid", parts: []string{"127", "0", "0", "1"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := allNumeric(tt.parts); got != tt.want {
				t.Errorf("allNumeric(%v) = %v, want %v", tt.parts, got, tt.want)
			}
		})
	}
}

func TestSanitizeImageName(t *testing.T) {
	tests := []struct {
		name      string
		imageName string
		want      string
		wantErr   bool
		errMsg    string
	}{
		// Valid image names
		{name: "simple image", imageName: "nginx", want: "nginx", wantErr: false},
		{name: "image with tag", imageName: "nginx:latest", want: "nginx:latest", wantErr: false},
		{name: "image with namespace", imageName: "library/nginx:1.0", want: "library/nginx:1.0", wantErr: false},
		{name: "full reference", imageName: "gcr.io/myproject/myimage:v1", want: "gcr.io/myproject/myimage:v1", wantErr: false},
		{name: "with whitespace", imageName: "  nginx:latest  ", want: "nginx:latest", wantErr: false},

		// Invalid - empty
		{name: "empty string", imageName: "", wantErr: true, errMsg: "image name cannot be empty"},
		{name: "only whitespace", imageName: "   ", wantErr: true, errMsg: "image name cannot be empty"},

		// Invalid - too long
		{name: "too long", imageName: strings.Repeat("a", 257), wantErr: true, errMsg: "image name too long"},

		// Invalid - path traversal
		{name: "path traversal", imageName: "../etc/passwd", wantErr: true, errMsg: "invalid characters in image name"},
		{name: "double dots", imageName: "nginx..latest", wantErr: true, errMsg: "invalid characters in image name"},

		// Invalid - bad characters
		{name: "special chars", imageName: "nginx<script>", wantErr: true, errMsg: "image name contains invalid characters"},
		{name: "starts with dot", imageName: ".nginx", wantErr: true, errMsg: "image name contains invalid characters"},

		// SSRF protection
		{name: "localhost registry", imageName: "localhost:5000/myimage:latest", wantErr: true, errMsg: "registry hostname not allowed"},
		{name: "private IP", imageName: "192.168.1.1:5000/myimage:latest", wantErr: true, errMsg: "registry hostname not allowed"},
		{name: "metadata endpoint", imageName: "169.254.169.254:80/meta:latest", wantErr: true, errMsg: "registry hostname not allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sanitizeImageName(tt.imageName)
			if tt.wantErr {
				if err == nil {
					t.Errorf("sanitizeImageName(%q) expected error containing %q, got nil", tt.imageName, tt.errMsg)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("sanitizeImageName(%q) error = %q, want error containing %q", tt.imageName, err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("sanitizeImageName(%q) unexpected error: %v", tt.imageName, err)
					return
				}
				if got != tt.want {
					t.Errorf("sanitizeImageName(%q) = %q, want %q", tt.imageName, got, tt.want)
				}
			}
		})
	}
}

func TestSanitizeFilenameComponent(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Basic cases
		{name: "simple name", input: "nginx", want: "nginx"},
		{name: "with numbers", input: "nginx123", want: "nginx123"},
		{name: "with hyphen", input: "my-app", want: "my-app"},
		{name: "with underscore", input: "my_app", want: "my_app"},
		{name: "with dot", input: "v1.0.0", want: "v1.0.0"},

		// Path separator handling
		{name: "unix path", input: "path/to/file", want: "path_to_file"},
		{name: "windows path", input: "path\\to\\file", want: "path_to_file"},
		{name: "mixed separators", input: "path/to\\file", want: "path_to_file"},
		{name: "multiple slashes", input: "a/b/c/d", want: "a_b_c_d"},

		// Path traversal prevention
		{name: "double dots", input: "..", want: "_"},
		{name: "path traversal", input: "../etc/passwd", want: "__etc_passwd"},
		{name: "embedded traversal", input: "foo/../bar", want: "foo___bar"},
		{name: "multiple traversals", input: "../../..", want: "_____"},

		// Whitespace handling
		{name: "leading whitespace", input: "  nginx", want: "nginx"},
		{name: "trailing whitespace", input: "nginx  ", want: "nginx"},
		{name: "both whitespace", input: "  nginx  ", want: "nginx"},

		// Empty handling
		{name: "empty string", input: "", want: "unknown"},
		{name: "only whitespace", input: "   ", want: "unknown"},
		{name: "only dots", input: "..", want: "_"},

		// Complex cases
		{name: "complex path", input: "library/nginx", want: "library_nginx"},
		{name: "registry style", input: "gcr.io/project/image", want: "gcr.io_project_image"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeFilenameComponent(tt.input); got != tt.want {
				t.Errorf("sanitizeFilenameComponent(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// Benchmark tests
func BenchmarkValidateRegistry(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = validateRegistry("registry-1.docker.io")
	}
}

func BenchmarkValidateRepository(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = validateRepository("library/nginx")
	}
}

func BenchmarkValidateTag(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = validateTag("v1.0.0")
	}
}

func BenchmarkValidateDigest(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = validateDigest("sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4")
	}
}

func BenchmarkBuildRegistryURL(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = buildRegistryURL("registry-1.docker.io", "/v2/%s/manifests/%s", "library/nginx", "latest")
	}
}

func BenchmarkSanitizeImageName(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = sanitizeImageName("gcr.io/myproject/myimage:v1.0.0")
	}
}

func BenchmarkSanitizeFilenameComponent(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = sanitizeFilenameComponent("library/nginx")
	}
}
