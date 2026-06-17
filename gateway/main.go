package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Load configuration
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create reverse proxy to llama-server
	proxy, err := NewProxy(cfg.LlamaServerURL)
	if err != nil {
		log.Fatalf("Failed to create proxy: %v", err)
	}

	// Set up routes
	mux := http.NewServeMux()

	// Health check (no auth required)
	mux.HandleFunc("GET /health", HealthHandler(cfg.LlamaServerURL))

	// Hardware discovery
	mux.HandleFunc("GET /v1/hardware", HardwareHandler())

	// OpenAI-compatible endpoints → proxy to llama-server
	mux.Handle("POST /v1/chat/completions", proxy)
	mux.Handle("POST /v1/completions", proxy)
	mux.Handle("GET /v1/models", proxy)

	// Catch-all for unmatched routes
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error": {"message": "Not found", "type": "invalid_request_error"}}`, http.StatusNotFound)
	})

	// Apply middleware: Logging → Auth → Routes
	handler := LoggingMiddleware(AuthMiddleware(cfg.APIKey, mux))

	// Start server
	addr := fmt.Sprintf(":%d", cfg.GatewayPort)
	log.Printf(`{"event":"startup","port":%d,"llama_server":"%s"}`, cfg.GatewayPort, cfg.LlamaServerURL)
	log.Printf(`{"event":"ready","message":"AI Gateway is running on %s"}`, addr)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
