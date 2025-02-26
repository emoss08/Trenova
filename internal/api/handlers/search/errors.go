package search

import (
	"errors"
	"fmt"
)

// Common errors that can be returned by the search service.
var (
	// ErrServiceStopped is returned when an operation is attempted on a stopped service
	ErrServiceStopped = errors.New("search service is stopped")

	// ErrIndexingInProgress is returned when an indexing operation conflicts with another one
	ErrIndexingInProgress = errors.New("indexing is already in progress")

	// ErrDocumentNotFound is returned when a document cannot be found
	ErrDocumentNotFound = errors.New("document not found")

	// ErrBatchProcessingFail is returned when batch processing fails
	ErrBatchProcessingFail = errors.New("batch processing failed")

	// ErrTaskTimeout is returned when a task times out
	ErrTaskTimeout = errors.New("task timed out")

	// ErrInvalidRequest is returned when a search request is invalid
	ErrInvalidRequest = errors.New("invalid search request")

	// ErrClientUnavailable is returned when the search client is unavailable
	ErrClientUnavailable = errors.New("search client is unavailable")

	// ErrRateLimited is returned when too many requests are made in a short time
	ErrRateLimited = errors.New("search rate limit exceeded")
)

// Error is a domain error specific to search operations
type Error struct {
	Code    string // Error code for clients to handle programmatically
	Message string // Human-readable error message
	Cause   error  // Underlying error, if any
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.Cause
}

// NewError creates a new search error
func NewError(code, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Error codes that can be used by clients
const (
	// ErrCodeServiceUnavailable indicates the search service is not available
	ErrCodeServiceUnavailable = "SEARCH_SERVICE_UNAVAILABLE"

	// ErrCodeIndexingFailed indicates a document indexing operation failed
	ErrCodeIndexingFailed = "INDEXING_FAILED"

	// ErrCodeSearchFailed indicates a search operation failed
	ErrCodeSearchFailed = "SEARCH_FAILED"

	// ErrCodeInvalidQuery indicates the search query is invalid
	ErrCodeInvalidQuery = "INVALID_QUERY"

	// ErrCodeInvalidFilter indicates the filter expression is invalid
	ErrCodeInvalidFilter = "INVALID_FILTER"

	// ErrCodeRateLimited indicates too many requests were made
	ErrCodeRateLimited = "RATE_LIMITED"

	// ErrCodeTimeout indicates an operation timed out
	ErrCodeTimeout = "TIMEOUT"
)

// IsIndexingError returns true if the error is related to indexing
func IsIndexingError(err error) bool {
	if err == nil {
		return false
	}

	var searchErr *Error
	if errors.As(err, &searchErr) {
		return searchErr.Code == ErrCodeIndexingFailed
	}

	return errors.Is(err, ErrIndexingInProgress) ||
		errors.Is(err, ErrBatchProcessingFail)
}

// IsSearchError returns true if the error is related to searching
func IsSearchError(err error) bool {
	if err == nil {
		return false
	}

	var searchErr *Error
	if errors.As(err, &searchErr) {
		return searchErr.Code == ErrCodeSearchFailed ||
			searchErr.Code == ErrCodeInvalidQuery ||
			searchErr.Code == ErrCodeInvalidFilter
	}

	return errors.Is(err, ErrInvalidRequest)
}

// IsServiceError returns true if the error is related to service availability
func IsServiceError(err error) bool {
	if err == nil {
		return false
	}

	var searchErr *Error
	if errors.As(err, &searchErr) {
		return searchErr.Code == ErrCodeServiceUnavailable ||
			searchErr.Code == ErrCodeTimeout ||
			searchErr.Code == ErrCodeRateLimited
	}

	return errors.Is(err, ErrServiceStopped) ||
		errors.Is(err, ErrTaskTimeout) ||
		errors.Is(err, ErrClientUnavailable) ||
		errors.Is(err, ErrRateLimited)
}
