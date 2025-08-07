<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
# `/internal/api` Directory Documentation

## Overview

The `api` directory contains all HTTP-related code including handlers, middleware, and routing. This layer translates HTTP requests into domain operations and domain responses back to HTTP responses.

## Directory Structure

```markdown
/internal/api/
├── handlers/                # HTTP request handlers
│   ├── auth/               # Authentication handlers
│   │   ├── login.go
│   │   └── refresh.go
│   ├── shipment/           # Shipment-related handlers
│   │   ├── create.go
│   │   ├── update.go
│   │   └── types.go        # Request/Response types
│   ├── vehicle/            # Vehicle-related handlers
│   └── driver/             # Driver-related handlers
├── middleware/             # HTTP middleware
│   ├── auth.go            # Authentication middleware
│   ├── logging.go         # Request logging
│   ├── metrics.go         # Metrics collection
│   └── ratelimit.go       # Rate limiting
├── routes/                # Route definitions
│   └── router.go         # Main router setup
└── server/               # Server configuration
    └── server.go        # HTTP server setup
```

## Guidelines & Rules

### Handler Structure

```go
// Handler type should encapsulate its dependencies
type ShipmentHandler struct {
    shipmentService ports.ShipmentService
    logger         logger.Logger
    validator      validator.Validator
}

// Constructor follows dependency injection pattern
func NewShipmentHandler(
    shipmentService ports.ShipmentService,
    logger logger.Logger,
    validator validator.Validator,
) *ShipmentHandler {
    return &ShipmentHandler{
        shipmentService: shipmentService,
        logger:         logger,
        validator:      validator,
    }
}
```

### Request/Response Types

```go
// Types should be defined per handler
type CreateShipmentRequest struct {
    Origin      AddressDTO `json:"origin" validate:"required"`
    Destination AddressDTO `json:"destination" validate:"required"`
    Weight      float64    `json:"weight" validate:"required,gt=0"`
}

type CreateShipmentResponse struct {
    ID           string    `json:"id"`
    TrackingCode string    `json:"trackingCode"`
    Status       string    `json:"status"`
    CreatedAt    time.Time `json:"createdAt"`
}
```

## Best Practices

1. **Handler Organization**
   - Group handlers by domain concept
   - Keep handlers focused and simple
   - Handle one type of request per handler
   - Separate request/response types

2. **Error Handling**
   - Use consistent error responses
   - Map domain errors to HTTP status codes
   - Include appropriate error details
   - Log errors with context

3. **Validation**
   - Validate requests early
   - Use structured validation (e.g., validator tags)
   - Return clear validation errors
   - Sanitize inputs

4. **Middleware**
   - Keep middleware focused
   - Order middleware properly
   - Handle middleware errors
   - Document middleware effects

## What Does NOT Belong Here

1. **Business Logic**
   - Complex calculations
   - Business rules
   - Domain operations
   - Data transformations

2. **Infrastructure Concerns**
   - Direct database access
   - External service calls
   - Cache operations
   - File system operations

3. **Domain Logic**
   - Entity manipulation
   - Business validations
   - Domain events
   - Complex state management

## Example Implementations

### Handler Implementation

```go
// handlers/shipment/create.go
func (h *ShipmentHandler) Create(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    var req CreateShipmentRequest
    
    // Parse and validate request
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.renderError(w, errors.NewBadRequestError("invalid request body"))
        return
    }
    
    if err := h.validator.Validate(req); err != nil {
        h.renderError(w, errors.NewValidationError(err))
        return
    }
    
    // Map to domain model
    shipment := mapToDomain(req)
    
    // Call domain service
    result, err := h.shipmentService.CreateShipment(ctx, shipment)
    if err != nil {
        h.handleError(w, err)
        return
    }
    
    // Map to response
    response := mapToResponse(result)
    h.renderJSON(w, http.StatusCreated, response)
}

func (h *ShipmentHandler) handleError(w http.ResponseWriter, err error) {
    switch {
    case errors.IsNotFound(err):
        h.renderError(w, errors.NewNotFoundError(err.Error()))
    case errors.IsValidation(err):
        h.renderError(w, errors.NewValidationError(err))
    default:
        h.logger.Error("unexpected error", "error", err)
        h.renderError(w, errors.NewInternalError())
    }
}
```

### Middleware Implementation

```go
// middleware/logging.go
func RequestLogging(logger logger.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            
            // Create wrapped response writer to capture status
            ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
            
            // Process request
            next.ServeHTTP(ww, r)
            
            // Log request details
            logger.Info("http request completed",
                "method", r.Method,
                "path", r.URL.Path,
                "status", ww.Status(),
                "duration", time.Since(start),
                "bytes", ww.BytesWritten(),
            )
        })
    }
}
```

### Router Setup

```go
// routes/router.go
func NewRouter(
    auth *AuthHandler,
    shipments *ShipmentHandler,
    middleware *Middleware,
) *chi.Mux {
    r := chi.NewRouter()
    
    // Global middleware
    r.Use(middleware.RequestID)
    r.Use(middleware.RealIP)
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    
    // Routes
    r.Route("/api/v1", func(r chi.Router) {
        // Public routes
        r.Post("/auth/login", auth.Login)
        r.Post("/auth/refresh", auth.Refresh)
        
        // Protected routes
        r.Group(func(r chi.Router) {
            r.Use(middleware.Authenticate)
            
            r.Route("/shipments", func(r chi.Router) {
                r.Post("/", shipments.Create)
                r.Get("/{id}", shipments.GetByID)
                r.Put("/{id}", shipments.Update)
            })
        })
    })
    
    return r
}
```

## Testing Guidelines

1. **Handler Testing**
   - Test request validation
   - Test response mapping
   - Test error scenarios
   - Use table-driven tests

2. **Middleware Testing**
   - Test middleware chain
   - Test error handling
   - Test context modifications
   - Test with various requests

3. **Integration Testing**
   - Test complete request flow
   - Test middleware integration
   - Test error handling
   - Test actual HTTP requests
