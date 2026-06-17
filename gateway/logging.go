package main

import (
	"log"
	"net/http"
	"time"
)

// LoggingMiddleware logs each request in a structured format.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap ResponseWriter to capture status code
		wrapped := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		log.Printf(`{"time":"%s","method":"%s","path":"%s","status":%d,"duration_ms":%d,"remote":"%s","content_length":%d}`,
			start.Format(time.RFC3339),
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			duration.Milliseconds(),
			r.RemoteAddr,
			r.ContentLength,
		)
	})
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}
