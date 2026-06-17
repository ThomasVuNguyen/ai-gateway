package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HealthResponse is the JSON structure returned by the /health endpoint.
type HealthResponse struct {
	Status      string            `json:"status"`
	Gateway     string            `json:"gateway"`
	LlamaServer LlamaServerHealth `json:"llama_server"`
	Timestamp   string            `json:"timestamp"`
}

// LlamaServerHealth represents the health of the llama-server backend.
type LlamaServerHealth struct {
	Status string `json:"status"`
	URL    string `json:"url"`
}

// HealthHandler checks the gateway and llama-server status.
func HealthHandler(llamaURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		llamaStatus := "ok"

		// Ping llama-server's health endpoint
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(llamaURL + "/health")
		if err != nil {
			llamaStatus = fmt.Sprintf("unreachable: %v", err)
		} else {
			defer resp.Body.Close()
			io.ReadAll(resp.Body) // drain body
			if resp.StatusCode != http.StatusOK {
				llamaStatus = fmt.Sprintf("unhealthy: status %d", resp.StatusCode)
			}
		}

		health := HealthResponse{
			Status:  "ok",
			Gateway: "running",
			LlamaServer: LlamaServerHealth{
				Status: llamaStatus,
				URL:    llamaURL,
			},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(health)
	}
}
