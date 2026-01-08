package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewAuthMiddleware(t *testing.T) {
	config := &AuthConfig{
		Enabled:  true,
		Username: "admin",
		Password: "secret",
	}

	auth := NewAuthMiddleware(config)
	if auth == nil {
		t.Fatal("expected non-nil AuthMiddleware")
	}
	if auth.config != config {
		t.Error("expected config to be set")
	}
}

func TestAuthMiddleware_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   *AuthConfig
		expected bool
	}{
		{
			name:     "nil config",
			config:   nil,
			expected: false,
		},
		{
			name:     "disabled",
			config:   &AuthConfig{Enabled: false},
			expected: false,
		},
		{
			name:     "enabled",
			config:   &AuthConfig{Enabled: true},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewAuthMiddleware(tt.config)
			if got := auth.IsEnabled(); got != tt.expected {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAuthMiddleware_BasicAuth(t *testing.T) {
	config := &AuthConfig{
		Enabled:  true,
		Username: "admin",
		Password: "secret123",
	}
	auth := NewAuthMiddleware(config)

	tests := []struct {
		name           string
		username       string
		password       string
		expectedStatus int
	}{
		{
			name:           "valid credentials",
			username:       "admin",
			password:       "secret123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid username",
			username:       "wronguser",
			password:       "secret123",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid password",
			username:       "admin",
			password:       "wrongpassword",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "empty credentials",
			username:       "",
			password:       "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := auth.WrapFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.username != "" || tt.password != "" {
				req.SetBasicAuth(tt.username, tt.password)
			}
			rec := httptest.NewRecorder()

			handler(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Check WWW-Authenticate header on unauthorized
			if tt.expectedStatus == http.StatusUnauthorized {
				if rec.Header().Get("WWW-Authenticate") == "" {
					t.Error("expected WWW-Authenticate header")
				}
			}
		})
	}
}

func TestAuthMiddleware_APIKey_Header(t *testing.T) {
	config := &AuthConfig{
		Enabled: true,
		APIKeys: []string{"valid-key-123", "another-valid-key"},
	}
	auth := NewAuthMiddleware(config)

	tests := []struct {
		name           string
		apiKey         string
		expectedStatus int
	}{
		{
			name:           "valid API key",
			apiKey:         "valid-key-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "another valid API key",
			apiKey:         "another-valid-key",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid API key",
			apiKey:         "invalid-key",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "empty API key",
			apiKey:         "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := auth.WrapFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}
			rec := httptest.NewRecorder()

			handler(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestAuthMiddleware_APIKey_QueryParam(t *testing.T) {
	config := &AuthConfig{
		Enabled: true,
		APIKeys: []string{"query-key-456"},
	}
	auth := NewAuthMiddleware(config)

	tests := []struct {
		name           string
		queryKey       string
		expectedStatus int
	}{
		{
			name:           "valid API key in query",
			queryKey:       "query-key-456",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid API key in query",
			queryKey:       "wrong-key",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := auth.WrapFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			url := "/test"
			if tt.queryKey != "" {
				url = "/test?api_key=" + tt.queryKey
			}
			req := httptest.NewRequest("GET", url, nil)
			rec := httptest.NewRecorder()

			handler(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestAuthMiddleware_Disabled(t *testing.T) {
	// When auth is disabled, requests should pass through
	config := &AuthConfig{
		Enabled:  false,
		Username: "admin",
		Password: "secret",
	}
	auth := NewAuthMiddleware(config)

	handler := auth.WrapFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Request without any credentials should succeed
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d when auth disabled, got %d", http.StatusOK, rec.Code)
	}
}

func TestAuthMiddleware_NilConfig(t *testing.T) {
	auth := NewAuthMiddleware(nil)

	handler := auth.WrapFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Request without any credentials should succeed when config is nil
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d when config is nil, got %d", http.StatusOK, rec.Code)
	}
}

func TestAuthMiddleware_Wrap(t *testing.T) {
	config := &AuthConfig{
		Enabled:  true,
		Username: "admin",
		Password: "secret",
	}
	auth := NewAuthMiddleware(config)

	// Test Wrap method with http.Handler
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	wrappedHandler := auth.Wrap(innerHandler)

	// Test with valid credentials
	req := httptest.NewRequest("GET", "/test", nil)
	req.SetBasicAuth("admin", "secret")
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Test without credentials
	req = httptest.NewRequest("GET", "/test", nil)
	rec = httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAuthMiddleware_CombinedAuth(t *testing.T) {
	// Test with both basic auth and API keys configured
	config := &AuthConfig{
		Enabled:  true,
		Username: "admin",
		Password: "secret",
		APIKeys:  []string{"api-key-123"},
	}
	auth := NewAuthMiddleware(config)

	handler := auth.WrapFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name           string
		setupRequest   func(*http.Request)
		expectedStatus int
	}{
		{
			name: "basic auth works",
			setupRequest: func(r *http.Request) {
				r.SetBasicAuth("admin", "secret")
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "API key header works",
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-API-Key", "api-key-123")
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "no credentials fails",
			setupRequest: func(r *http.Request) {
				// No credentials
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			tt.setupRequest(req)
			rec := httptest.NewRecorder()

			handler(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestSecureCompare(t *testing.T) {
	tests := []struct {
		a        string
		b        string
		expected bool
	}{
		{"password", "password", true},
		{"password", "Password", false},
		{"password", "passwor", false},
		{"password", "passwordx", false},
		{"", "", true},
		{"a", "", false},
		{"", "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			if got := secureCompare(tt.a, tt.b); got != tt.expected {
				t.Errorf("secureCompare(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestValidateAPIKey(t *testing.T) {
	auth := NewAuthMiddleware(&AuthConfig{
		Enabled: true,
		APIKeys: []string{"key1", "key2", "key3"},
	})

	tests := []struct {
		key      string
		expected bool
	}{
		{"key1", true},
		{"key2", true},
		{"key3", true},
		{"key4", false},
		{"", false},
		{"KEY1", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := auth.validateAPIKey(tt.key); got != tt.expected {
				t.Errorf("validateAPIKey(%q) = %v, want %v", tt.key, got, tt.expected)
			}
		})
	}
}

func TestValidateBasicAuth(t *testing.T) {
	auth := NewAuthMiddleware(&AuthConfig{
		Enabled:  true,
		Username: "admin",
		Password: "secret123",
	})

	tests := []struct {
		username string
		password string
		expected bool
	}{
		{"admin", "secret123", true},
		{"admin", "wrong", false},
		{"wrong", "secret123", false},
		{"", "secret123", false},
		{"admin", "", false},
		{"", "", false},
		{"ADMIN", "secret123", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.username+"_"+tt.password, func(t *testing.T) {
			if got := auth.validateBasicAuth(tt.username, tt.password); got != tt.expected {
				t.Errorf("validateBasicAuth(%q, %q) = %v, want %v", tt.username, tt.password, got, tt.expected)
			}
		})
	}
}

func TestValidateBasicAuth_NoUsernameConfigured(t *testing.T) {
	// When no username is configured, basic auth should fail
	auth := NewAuthMiddleware(&AuthConfig{
		Enabled:  true,
		Username: "",
		Password: "secret",
		APIKeys:  []string{"key1"}, // Only API keys configured
	})

	if auth.validateBasicAuth("anyuser", "secret") {
		t.Error("expected basic auth to fail when no username configured")
	}
}
