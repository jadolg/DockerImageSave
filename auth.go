package main

import (
	"crypto/subtle"
	"net/http"
)

// AuthMiddleware provides HTTP authentication for private hosting
type AuthMiddleware struct {
	config *AuthConfig
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(config *AuthConfig) *AuthMiddleware {
	return &AuthMiddleware{config: config}
}

// IsEnabled returns true if authentication is configured and enabled
func (a *AuthMiddleware) IsEnabled() bool {
	return a.config != nil && a.config.Enabled
}

// Wrap wraps an http.Handler with authentication checks
func (a *AuthMiddleware) Wrap(next http.Handler) http.Handler {
	if !a.IsEnabled() {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.authenticate(r) {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="DockerImageSave"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

// WrapFunc wraps an http.HandlerFunc with authentication checks
func (a *AuthMiddleware) WrapFunc(next http.HandlerFunc) http.HandlerFunc {
	if !a.IsEnabled() {
		return next
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if a.authenticate(r) {
			next(w, r)
			return
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="DockerImageSave"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
}

// authenticate checks if the request is properly authenticated
func (a *AuthMiddleware) authenticate(r *http.Request) bool {
	// First, try API key from header
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return a.validateAPIKey(apiKey)
	}

	// Also check query parameter for API key (useful for curl/wget).
	// NOTE: Query params are less secure (logged in URLs); prefer X-API-Key header in production.
	if apiKey := r.URL.Query().Get("api_key"); apiKey != "" {
		return a.validateAPIKey(apiKey)
	}

	// Fall back to Basic Auth
	username, password, ok := r.BasicAuth()
	if !ok {
		return false
	}

	return a.validateBasicAuth(username, password)
}

// validateAPIKey checks if the provided API key is valid
func (a *AuthMiddleware) validateAPIKey(key string) bool {
	for _, validKey := range a.config.APIKeys {
		if secureCompare(key, validKey) {
			return true
		}
	}
	return false
}

// validateBasicAuth checks username and password
func (a *AuthMiddleware) validateBasicAuth(username, password string) bool {
	if a.config.Username == "" {
		return false
	}

	usernameMatch := secureCompare(username, a.config.Username)
	passwordMatch := secureCompare(password, a.config.Password)

	return usernameMatch && passwordMatch
}

// secureCompare performs a constant-time comparison to prevent timing attacks
func secureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
