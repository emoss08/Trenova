package samsaratypes

import (
	"errors"
	"fmt"
	"net/http"
)

type APIError struct {
	StatusCode int
	Message    string
	RequestID  string
	RawBody    []byte
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}

	if e.RequestID != "" {
		return fmt.Sprintf(
			"samsara API error (status=%d, requestId=%s): %s",
			e.StatusCode,
			e.RequestID,
			e.Message,
		)
	}

	return fmt.Sprintf("samsara API error (status=%d): %s", e.StatusCode, e.Message)
}

func IsRateLimit(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == http.StatusTooManyRequests
}

func IsUnauthorized(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == http.StatusUnauthorized
}

func IsNotFound(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == http.StatusNotFound
}

func Temporary(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}

	return apiErr.StatusCode == http.StatusTooManyRequests ||
		(apiErr.StatusCode >= http.StatusInternalServerError &&
			apiErr.StatusCode <= http.StatusNetworkAuthenticationRequired)
}
