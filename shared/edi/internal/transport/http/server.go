package http

import (
	"context"
	"net/http"
	"time"

	"github.com/emoss08/trenova/shared/edi/internal/core/services"
	"github.com/emoss08/trenova/shared/edi/internal/transport/endpoints"
	"github.com/emoss08/trenova/shared/edi/internal/transport/middleware"
	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/google/uuid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Address              string
	ReadTimeout          time.Duration
	WriteTimeout         time.Duration
	IdleTimeout          time.Duration
	MaxHeaderBytes       int
	RateLimitPerSecond   int
	RateLimitBurst       int
	EnableDetailedLogging bool
}

// DefaultServerConfig returns default server configuration
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Address:              ":8080",
		ReadTimeout:          15 * time.Second,
		WriteTimeout:         15 * time.Second,
		IdleTimeout:          60 * time.Second,
		MaxHeaderBytes:       1 << 20, // 1 MB
		RateLimitPerSecond:   100,
		RateLimitBurst:       200,
		EnableDetailedLogging: false,
	}
}

// NewHTTPServerParams holds dependencies for the HTTP server
type NewHTTPServerParams struct {
	fx.In
	Logger         *zap.Logger
	EDIProcessor   *services.EDIProcessorService
	ProfileService *services.ProfileService
	Config         *ServerConfig `optional:"true"`
}

// encodeErrorFunc is the error encoder for go-kit HTTP transport
func encodeErrorFunc(ctx context.Context, err error, w http.ResponseWriter) {
	encodeError(ctx, err, w)
}

// NewHTTPServer creates a new HTTP server with go-kit transport
func NewHTTPServer(params NewHTTPServerParams) *http.Server {
	config := params.Config
	if config == nil {
		cfg := DefaultServerConfig()
		config = &cfg
	}

	// Create validator
	validator := middleware.NewStructValidator()

	// Create endpoints
	ediEndpoints := endpoints.NewEndpoints(params.EDIProcessor)
	profileEndpoints := endpoints.NewProfileEndpoints(params.ProfileService)

	// Apply middleware chains
	commonMiddleware := []endpoint.Middleware{
		middleware.LoggingMiddleware(params.Logger),
		middleware.ValidationMiddleware(validator),
		middleware.TimeoutMiddleware(30 * time.Second),
		middleware.BulkheadMiddleware(100), // Max 100 concurrent requests
	}

	// Rate limiting middleware
	rateLimitMiddleware := middleware.RateLimitMiddleware(middleware.RateLimiterConfig{
		RequestsPerSecond: config.RateLimitPerSecond,
		Burst:            config.RateLimitBurst,
		PerPartner:       true,
	})

	// Circuit breaker for external calls
	circuitBreakerMiddleware := middleware.CircuitBreakerMiddleware(
		middleware.DefaultCircuitBreakerConfig("edi-processor"),
	)

	// Apply middleware to EDI endpoints
	ediMiddleware := append(commonMiddleware, rateLimitMiddleware, circuitBreakerMiddleware)
	ediEndpoints = ediEndpoints.Chain(ediMiddleware...)

	// Apply middleware to profile endpoints (no circuit breaker needed)
	profileMiddleware := append(commonMiddleware, rateLimitMiddleware)
	profileEndpoints = profileEndpoints.Chain(profileMiddleware...)

	// Create HTTP handlers with options
	options := []httptransport.ServerOption{
		httptransport.ServerBefore(extractRequestID),
		httptransport.ServerBefore(extractHeaders),
		httptransport.ServerErrorEncoder(encodeErrorFunc),
		httptransport.ServerAfter(setResponseHeaders),
	}

	// Create router
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", healthCheckHandler)

	// EDI processing endpoints
	mux.Handle("/api/v1/process", httptransport.NewServer(
		ediEndpoints.ProcessEDIEndpoint,
		DecodeProcessEDIRequest,
		EncodeProcessEDIResponse,
		options...,
	))

	mux.Handle("/api/v1/documents", httptransport.NewServer(
		ediEndpoints.ListDocumentsEndpoint,
		DecodeListDocumentsRequest,
		EncodeListDocumentsResponse,
		options...,
	))

	mux.Handle("/api/v1/documents/get", httptransport.NewServer(
		ediEndpoints.GetDocumentEndpoint,
		DecodeGetDocumentRequest,
		EncodeGetDocumentResponse,
		options...,
	))

	// Profile management endpoints
	mux.Handle("/api/v1/profiles", httptransport.NewServer(
		profileEndpoints.ListProfilesEndpoint,
		DecodeListProfilesRequest,
		EncodeListProfilesResponse,
		options...,
	))

	mux.Handle("/api/v1/profiles/import", httptransport.NewServer(
		profileEndpoints.ImportProfileEndpoint,
		DecodeImportProfileRequest,
		EncodeImportProfileResponse,
		options...,
	))

	mux.Handle("/api/v1/profiles/get", httptransport.NewServer(
		profileEndpoints.GetProfileEndpoint,
		DecodeGetProfileRequest,
		EncodeGetProfileResponse,
		options...,
	))

	mux.Handle("/api/v1/profiles/delete", httptransport.NewServer(
		profileEndpoints.DeleteProfileEndpoint,
		DecodeDeleteProfileRequest,
		EncodeDeleteProfileResponse,
		options...,
	))

	// Add middleware for all routes
	handler := loggingMiddleware(params.Logger)(mux)
	handler = recoveryMiddleware(params.Logger)(handler)
	handler = corsMiddleware()(handler)

	// Create HTTP server
	server := &http.Server{
		Addr:           config.Address,
		Handler:        handler,
		ReadTimeout:    config.ReadTimeout,
		WriteTimeout:   config.WriteTimeout,
		IdleTimeout:    config.IdleTimeout,
		MaxHeaderBytes: config.MaxHeaderBytes,
	}

	return server
}

// extractRequestID extracts or generates a request ID
func extractRequestID(ctx context.Context, r *http.Request) context.Context {
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.New().String()
	}
	return context.WithValue(ctx, "request_id", requestID)
}

// extractHeaders extracts useful headers into context
func extractHeaders(ctx context.Context, r *http.Request) context.Context {
	// Extract partner ID if present
	if partnerID := r.Header.Get("X-Partner-ID"); partnerID != "" {
		ctx = context.WithValue(ctx, "partner_id", partnerID)
	}
	
	// Extract API key if present (for future use)
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		ctx = context.WithValue(ctx, "api_key", apiKey)
	}

	return ctx
}

// setResponseHeaders sets common response headers
func setResponseHeaders(ctx context.Context, w http.ResponseWriter) context.Context {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	return ctx
}

// healthCheckHandler handles health check requests
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"edi-processor"}`))
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			// Wrap response writer to capture status code
			wrapped := wrapResponseWriter(w)
			
			// Process request
			next.ServeHTTP(wrapped, r)
			
			// Log request
			logger.Info("http request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", wrapped.status),
				zap.Duration("duration", time.Since(start)),
				zap.String("remote_addr", r.RemoteAddr),
			)
		})
	}
}

// recoveryMiddleware recovers from panics
func recoveryMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic recovered",
						zap.Any("error", err),
						zap.String("path", r.URL.Path),
						zap.String("method", r.Method),
					)
					
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error":"internal server error"}`))
				}
			}()
			
			next.ServeHTTP(w, r)
		})
	}
}

// corsMiddleware adds CORS headers
func corsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID, X-Partner-ID, X-API-Key")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// responseWriter wrapper to capture status code
type responseWriter struct {
	http.ResponseWriter
	status int
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}