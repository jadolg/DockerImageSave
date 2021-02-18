package dockerimagesave

import docker "github.com/fsouza/go-dockerclient"

// PullResponse is response format when calling pull image
type PullResponse struct {
	ID     string `json:"id,omitempty"`
	Status string `json:"status,omitempty"`
	Error  string `json:"error,omitempty"`
}

// SearchResponse
type SearchResponse struct {
	Term         string                  `json:"term"`
	Error        string                  `json:"error,omitempty"`
	Status       string                  `json:"status,omitempty"`
	SearchResult []docker.APIImageSearch `json:"search_result,omitempty"`
}

// SaveResponse is response format when calling save image
type SaveResponse struct {
	ID     string `json:"id,omitempty"`
	URL    string `json:"url,omitempty"`
	Error  string `json:"error,omitempty"`
	Size   int64  `json:"size,omitempty"`
	Status string `json:"status,omitempty"`
}

// HealthCheckResponse is response format for healthcheck method
type HealthCheckResponse struct {
	Memory     uint64 `json:"memory,omitempty"`
	UsedMemory uint64 `json:"used_memory,omitempty"`
	OS         string `json:"os,omitempty"`
	Platform   string `json:"platform,omitempty"`
	Error      string `json:"error,omitempty"`
}
