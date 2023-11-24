package middleware

import (
	"log"
	"net/http"
	"time"
)

// LoggingMiddleware logs the incoming HTTP request & its duration.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Start timer
		start := time.Now()

		// Wrap the response writer to capture the status code
		wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(wrappedWriter, r)

		// Log request details
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, wrappedWriter.statusCode, time.Since(start))
	})
}

// responseWriter is a minimal wrapper for http.ResponseWriter that allows the status code to be captured
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code and calls the original WriteHeader
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
