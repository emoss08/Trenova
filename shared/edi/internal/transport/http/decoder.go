package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/shared/edi/internal/transport"
	"github.com/emoss08/trenova/shared/edi/internal/transport/endpoints"
)

const (
	maxRequestSize = 10 * 1024 * 1024 // 10MB max request size
)

// DecodeProcessEDIRequest decodes the HTTP request for processing EDI
func DecodeProcessEDIRequest(_ context.Context, r *http.Request) (interface{}, error) {
	// Check content type
	contentType := r.Header.Get("Content-Type")
	
	var partnerID string
	var ediContent string

	// Get partner ID from query or header
	partnerID = r.URL.Query().Get("partner_id")
	if partnerID == "" {
		partnerID = r.Header.Get("X-Partner-ID")
	}
	
	if partnerID == "" {
		return nil, transport.NewServiceError(
			transport.ErrorTypeValidation,
			"partner_id is required (query param or X-Partner-ID header)",
		)
	}

	// Read body with size limit
	r.Body = http.MaxBytesReader(nil, r.Body, maxRequestSize)
	
	switch {
	case strings.Contains(contentType, "application/json"):
		var req struct {
			EDIContent string `json:"edi_content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if err.Error() == "http: request body too large" {
				return nil, transport.NewServiceError(
					transport.ErrorTypeValidation,
					fmt.Sprintf("request body exceeds maximum size of %d bytes", maxRequestSize),
				)
			}
			return nil, transport.NewServiceError(
				transport.ErrorTypeValidation,
				fmt.Sprintf("invalid JSON: %v", err),
			)
		}
		ediContent = req.EDIContent
		
	case strings.Contains(contentType, "text/plain"), 
	     strings.Contains(contentType, "application/edi"),
	     strings.Contains(contentType, "application/x12"):
		body, err := io.ReadAll(r.Body)
		if err != nil {
			if err.Error() == "http: request body too large" {
				return nil, transport.NewServiceError(
					transport.ErrorTypeValidation,
					fmt.Sprintf("request body exceeds maximum size of %d bytes", maxRequestSize),
				)
			}
			return nil, transport.NewServiceError(
				transport.ErrorTypeValidation,
				fmt.Sprintf("failed to read request body: %v", err),
			)
		}
		ediContent = string(body)
		
	default:
		return nil, transport.NewServiceError(
			transport.ErrorTypeValidation,
			fmt.Sprintf("unsupported content type: %s", contentType),
		)
	}

	if ediContent == "" {
		return nil, transport.NewServiceError(
			transport.ErrorTypeValidation,
			"EDI content is required",
		)
	}

	return endpoints.ProcessEDIRequest{
		PartnerID:  partnerID,
		EDIContent: ediContent,
	}, nil
}

// DecodeGetDocumentRequest decodes the HTTP request for getting a document
func DecodeGetDocumentRequest(_ context.Context, r *http.Request) (interface{}, error) {
	documentID := r.URL.Query().Get("id")
	if documentID == "" {
		// Try path parameter
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) > 0 {
			documentID = parts[len(parts)-1]
		}
	}
	
	if documentID == "" {
		return nil, transport.NewServiceError(
			transport.ErrorTypeValidation,
			"document ID is required",
		)
	}

	return endpoints.GetDocumentRequest{
		DocumentID: documentID,
	}, nil
}

// DecodeListDocumentsRequest decodes the HTTP request for listing documents
func DecodeListDocumentsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	query := r.URL.Query()
	
	// Parse pagination
	limit := 100 // default
	if l := query.Get("limit"); l != "" {
		parsedLimit, err := strconv.Atoi(l)
		if err != nil || parsedLimit < 1 || parsedLimit > 1000 {
			return nil, transport.NewServiceError(
				transport.ErrorTypeValidation,
				"limit must be between 1 and 1000",
			)
		}
		limit = parsedLimit
	}

	offset := 0
	if o := query.Get("offset"); o != "" {
		parsedOffset, err := strconv.Atoi(o)
		if err != nil || parsedOffset < 0 {
			return nil, transport.NewServiceError(
				transport.ErrorTypeValidation,
				"offset must be non-negative",
			)
		}
		offset = parsedOffset
	}

	return endpoints.ListDocumentsRequest{
		PartnerID: query.Get("partner_id"),
		Limit:     limit,
		Offset:    offset,
	}, nil
}

// DecodeImportProfileRequest decodes the HTTP request for importing a profile
func DecodeImportProfileRequest(_ context.Context, r *http.Request) (interface{}, error) {
	// Check content type
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return nil, transport.NewServiceError(
			transport.ErrorTypeValidation,
			"content type must be application/json",
		)
	}

	// Read body with size limit
	r.Body = http.MaxBytesReader(nil, r.Body, maxRequestSize)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		if err.Error() == "http: request body too large" {
			return nil, transport.NewServiceError(
				transport.ErrorTypeValidation,
				fmt.Sprintf("request body exceeds maximum size of %d bytes", maxRequestSize),
			)
		}
		return nil, transport.NewServiceError(
			transport.ErrorTypeValidation,
			fmt.Sprintf("failed to read request body: %v", err),
		)
	}

	// Validate it's valid JSON
	var test interface{}
	if err := json.Unmarshal(body, &test); err != nil {
		return nil, transport.NewServiceError(
			transport.ErrorTypeValidation,
			fmt.Sprintf("invalid JSON: %v", err),
		)
	}

	return endpoints.ImportProfileRequest{
		ProfileJSON: body,
	}, nil
}

// DecodeListProfilesRequest decodes the HTTP request for listing profiles
func DecodeListProfilesRequest(_ context.Context, r *http.Request) (interface{}, error) {
	activeOnly := true // default to active only
	if a := r.URL.Query().Get("include_inactive"); a == "true" {
		activeOnly = false
	}

	return endpoints.ListProfilesRequest{
		ActiveOnly: activeOnly,
	}, nil
}

// DecodeGetProfileRequest decodes the HTTP request for getting a profile
func DecodeGetProfileRequest(_ context.Context, r *http.Request) (interface{}, error) {
	partnerID := r.URL.Query().Get("partner_id")
	if partnerID == "" {
		// Try path parameter
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) > 0 {
			partnerID = parts[len(parts)-1]
		}
	}
	
	if partnerID == "" {
		return nil, transport.NewServiceError(
			transport.ErrorTypeValidation,
			"partner ID is required",
		)
	}

	return endpoints.GetProfileRequest{
		PartnerID: partnerID,
	}, nil
}

// DecodeDeleteProfileRequest decodes the HTTP request for deleting a profile
func DecodeDeleteProfileRequest(_ context.Context, r *http.Request) (interface{}, error) {
	partnerID := r.URL.Query().Get("partner_id")
	if partnerID == "" {
		// Try path parameter
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) > 0 {
			partnerID = parts[len(parts)-1]
		}
	}
	
	if partnerID == "" {
		return nil, transport.NewServiceError(
			transport.ErrorTypeValidation,
			"partner ID is required",
		)
	}

	return endpoints.DeleteProfileRequest{
		PartnerID: partnerID,
	}, nil
}