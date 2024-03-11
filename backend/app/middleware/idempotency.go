package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/http/httptest"
	"time"
)

var (
	logger            = logrus.New()
	ExcludedMethods   = getExcludedMethods()
	ExcludedPaths     = getExcludedPaths()
	IdempotencyKeyTTL = getIdempotencyKeyTTL()
)

func getExcludedMethods() []string {
	return []string{"GET", "OPTIONS", "HEAD", "TRACE", "CONNECT"}
}

func getExcludedPaths() []string {
	return []string{"/api/auth/login", "/api/auth/logout"}
}

func getIdempotencyKeyTTL() time.Duration {
	return 60 * 60 * 24 * 7
}

// StoredResponse represents the structure to store response data.
type StoredResponse struct {
	StatusCode int                 `json:"statusCode"`
	Headers    map[string][]string `json:"headers"`
	Body       []byte              `json:"body"`
}

func IdempotencyMiddleware(store *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if contains(ExcludedMethods, r.Method) || contains(ExcludedPaths, r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Safely assert the orgID from the context
			orgIDInterface := r.Context().Value(ContextKeyOrgID)

			if orgIDInterface == nil {
				logger.Error("Org ID not found in context")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)

				return
			}

			orgID, ok := orgIDInterface.(uuid.UUID)
			if !ok {
				logger.Error("Org ID is not of type uuid.UUID")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)

				return
			}

			idempotencyKey := r.Header.Get("X-Idempotency-Key")
			if idempotencyKey == "" {
				http.Error(w, "Idempotency key is required for this request", http.StatusBadRequest)
				return
			}

			// Prefix the idempotency key with the Org ID to ensure it's unique per organization
			prefixedKey := fmt.Sprintf("%s:%s", orgID, idempotencyKey)

			ctx := r.Context()
			val, err := store.Get(ctx, prefixedKey).Result()

			if errors.Is(err, redis.Nil) {
				recorder := httptest.NewRecorder()
				next.ServeHTTP(recorder, r)

				storedResponse := StoredResponse{
					StatusCode: recorder.Code,
					Headers:    recorder.Header(),
					Body:       recorder.Body.Bytes(),
				}

				responseData, err := json.Marshal(storedResponse)
				if err != nil {
					logger.WithError(err).Error("Failed to serialize response")
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)

					return
				}

				if err := store.Set(ctx, prefixedKey, responseData, time.Second*IdempotencyKeyTTL).Err(); err != nil {
					logger.WithError(err).Error("Failed to store response in Redis")
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)

					return
				}

				copyHeader(w.Header(), recorder.Header())
				w.WriteHeader(recorder.Code)

				if _, err := w.Write(recorder.Body.Bytes()); err != nil {
					logger.WithError(err).Error("Failed to write response to client")
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)

					return
				}

				return
			} else if err != nil {
				logger.WithError(err).Error("Failed to retrieve response from Redis")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)

				return
			}

			var storedResponse StoredResponse
			if err := json.Unmarshal([]byte(val), &storedResponse); err != nil {
				logger.WithError(err).Error("Failed to deserialize stored response")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)

				return
			}

			copyHeader(w.Header(), storedResponse.Headers)
			w.WriteHeader(storedResponse.StatusCode)

			if _, err := w.Write(storedResponse.Body); err != nil {
				logger.WithError(err).Error("Failed to write response to client")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)

				return
			}
		})
	}
}

func contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}

	return false
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
