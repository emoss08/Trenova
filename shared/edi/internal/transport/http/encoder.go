package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/emoss08/trenova/shared/edi/internal/transport"
)

// EncodeResponse is the common response encoder
func EncodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	// Set common headers
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	
	// Add request ID if available
	if reqID := ctx.Value("request_id"); reqID != nil {
		w.Header().Set("X-Request-ID", reqID.(string))
	}

	// Check for error response
	if e, ok := response.(error); ok {
		return encodeError(ctx, e, w)
	}

	// Check if response has an error field
	if resp, ok := response.(interface{ GetError() string }); ok && resp.GetError() != "" {
		// This is a business error wrapped in response
		w.WriteHeader(http.StatusBadRequest)
		return json.NewEncoder(w).Encode(response)
	}

	// Success response
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(response)
}

// encodeError encodes an error response
func encodeError(ctx context.Context, err error, w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	
	// Add request ID if available
	requestID := ""
	if reqID := ctx.Value("request_id"); reqID != nil {
		requestID = reqID.(string)
		w.Header().Set("X-Request-ID", requestID)
	}

	// Determine status code
	statusCode := transport.HTTPStatusFromError(err)
	w.WriteHeader(statusCode)

	// Create error response
	errResp := transport.NewErrorResponse(err, requestID)
	
	return json.NewEncoder(w).Encode(errResp)
}

// EncodeProcessEDIResponse encodes the process EDI response
func EncodeProcessEDIResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	return EncodeResponse(ctx, w, response)
}

// EncodeGetDocumentResponse encodes the get document response
func EncodeGetDocumentResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	return EncodeResponse(ctx, w, response)
}

// EncodeListDocumentsResponse encodes the list documents response
func EncodeListDocumentsResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	return EncodeResponse(ctx, w, response)
}

// EncodeImportProfileResponse encodes the import profile response
func EncodeImportProfileResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	return EncodeResponse(ctx, w, response)
}

// EncodeListProfilesResponse encodes the list profiles response
func EncodeListProfilesResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	return EncodeResponse(ctx, w, response)
}

// EncodeGetProfileResponse encodes the get profile response
func EncodeGetProfileResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	return EncodeResponse(ctx, w, response)
}

// EncodeDeleteProfileResponse encodes the delete profile response
func EncodeDeleteProfileResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	return EncodeResponse(ctx, w, response)
}

// EncodeHTTPGenericResponse is a generic response encoder
func EncodeHTTPGenericResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	return EncodeResponse(ctx, w, response)
}