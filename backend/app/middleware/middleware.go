package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"trenova-go-backend/app/models"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// Color codes for the terminal.
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
)

// ColoredMethod formats HTTP methods with color.
func ColoredMethod(method string) string {
	return fmt.Sprintf("%s%s%s", Blue, method, Reset)
}

// ColoredPath formats paths with color.
func ColoredPath(path string) string {
	return fmt.Sprintf("%s%s%s", Yellow, path, Reset)
}

// ColoredStatus formats the outcome status with color.
func ColoredStatus(statusCode int) (string, string) {
	outcome := "Success"
	color := Green
	switch {
	case statusCode >= 400:
		outcome = "Failure"
		color = Red
	case statusCode >= 300:
		outcome = "Redirect"
		color = Yellow
	case statusCode >= 200:
		outcome = "Success"
		color = Green
	default:
		outcome = "Unknown"
		color = Reset
	}
	return fmt.Sprintf("%s%s%s", color, outcome, Reset), color
}

// AdvancedLoggingMiddleware logs detailed request and response information with color
func AdvancedLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the ResponseWriter
		lrw := NewLoggingResponseWriter(w)

		// Perform the request
		next.ServeHTTP(lrw, r)

		// Calculate duration
		duration := time.Since(start)

		// Determine the outcome and set the color
		outcome, color := ColoredStatus(lrw.statusCode)

		// Log the request and response details with color
		log.Printf("%s %s -> Matched: %s %s %s\n%s >> Outcome: %s\n%s >> Response %s in %v.\n",
			ColoredMethod(r.Method), ColoredPath(r.URL.Path),
			ColoredMethod(r.Method), ColoredPath(r.URL.Path), r.Header.Get("Content-Type"),
			color, outcome, Reset, http.StatusText(lrw.statusCode), duration)
	})
}

// LoggingResponseWriter captures the status code for logging
type LoggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// NewLoggingResponseWriter creates a new response writer wrapper
func NewLoggingResponseWriter(w http.ResponseWriter) *LoggingResponseWriter {
	return &LoggingResponseWriter{w, http.StatusOK} // Default to 200 OK
}

// WriteHeader captures the status code
func (lrw *LoggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

const AuthenticatedUserKey = "authenticatedUser"

func AuthMiddleware(db *gorm.DB) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		type contextKey string

		const AuthenticatedUserKey contextKey = "authenticatedUser"

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractToken(r)

			// Validate the token and get user
			user, err := validateToken(db, tokenStr)
			if err != nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Add the user to the request context
			ctx := context.WithValue(r.Context(), AuthenticatedUserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
func extractToken(r *http.Request) string {
	bearToken := r.Header.Get("Authorization")
	strArr := strings.Split(bearToken, " ")
	if len(strArr) == 2 {
		return strArr[1]
	}

	// TODO(WOLFRED): Include the logic to retrieve the cookie token
	return ""
}

func validateToken(db *gorm.DB, tokenStr string) (*models.User, error) {
	// You'll need to implement the actual validation logic and user retrieval here.
	// This usually involves checking the token against the database, ensuring it's not expired,
	// and then fetching the user associated with the token.
	// This is a placeholder for your validation logic.

	var token models.Token
	if err := db.Where("key = ?", tokenStr).First(&token).Error; err != nil {
		return nil, err
	}

	// Check the token expiry and user's active status
	if token.User.Status == "A" || token.IsExpired() {
		return nil, fmt.Errorf("invalid token")
	}

	token.LastUsed = time.Now()
	if err := db.Save(&token).Error; err != nil {
		return nil, err
	}

	return &token.User, nil
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow all origins
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Setup allowed methods
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")

		// Setup allowed headers
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		// Check if it's a preflight request and handle it
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
