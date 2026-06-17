package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// NewProxy creates a reverse proxy to the llama-server.
// It handles both regular and streaming (SSE) responses.
func NewProxy(targetURL string) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host

			// Strip X-Hardware header before forwarding (it's our custom field)
			req.Header.Del("X-Hardware")

			log.Printf(`{"event":"proxy","target":"%s%s"}`, target.Host, req.URL.Path)
		},
		// Use a custom transport that disables response buffering for SSE streaming
		Transport: &streamingTransport{http.DefaultTransport},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf(`{"event":"proxy_error","error":"%v","path":"%s"}`, err, r.URL.Path)
			http.Error(w, `{"error": {"message": "Backend unavailable", "type": "server_error"}}`, http.StatusBadGateway)
		},
	}

	return proxy, nil
}

// streamingTransport wraps an http.RoundTripper to disable response buffering,
// which is essential for Server-Sent Events (SSE) streaming from llama-server.
type streamingTransport struct {
	http.RoundTripper
}

func (t *streamingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.RoundTripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// If the response is SSE (text/event-stream), ensure no buffering
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		resp.Body = &flushingReader{resp.Body}
	}

	return resp, nil
}

// flushingReader wraps an io.ReadCloser for streaming passthrough.
type flushingReader struct {
	io.ReadCloser
}
