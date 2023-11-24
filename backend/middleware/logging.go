package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/fatih/color"
)

// LoggingMiddleware logs the incoming HTTP request & its duration.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the response writer to capture the status code
		wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(wrappedWriter, r)

		duration := time.Since(start)

		statusColor := colorForStatus(wrappedWriter.statusCode)
		methodColor := colorForMethod(r.Method)

		// Using Sprintf to format colored strings
		status := statusColor.Sprintf("%3d", wrappedWriter.statusCode)
		method := methodColor.Sprint(r.Method)

		log.Printf("[MONTA] %v | %s | %13v | %15s | %-7s %s",
			start.Format("2006/01/02 - 15:04:05"),
			status,
			duration,
			r.RemoteAddr,
			method,
			r.URL.Path,
		)
	})
}

func colorForStatus(code int) *color.Color {
	switch {
	case code >= 200 && code < 300:
		return color.New(color.FgGreen)
	case code >= 300 && code < 400:
		return color.New(color.FgCyan)
	case code >= 400 && code < 500:
		return color.New(color.FgYellow)
	default:
		return color.New(color.FgRed)
	}
}

func colorForMethod(method string) *color.Color {
	switch method {
	case "GET":
		return color.New(color.FgBlue)
	case "POST":
		return color.New(color.FgCyan)
	case "PUT":
		return color.New(color.FgYellow)
	case "DELETE":
		return color.New(color.FgRed)
	case "PATCH":
		return color.New(color.FgGreen)
	case "HEAD":
		return color.New(color.FgMagenta)
	default:
		return color.New(color.FgWhite)
	}
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
