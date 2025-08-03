/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package api

import (
	"errors"
	"fmt"
)

// Custom error types for better error handling
var (
	ErrInvalidZipCode     = errors.New("invalid zip code format")
	ErrZipCodeNotFound    = errors.New("zip code not found in database")
	ErrNoRouteAvailable   = errors.New("no route available between locations")
	ErrServiceUnavailable = errors.New("routing service temporarily unavailable")
)

// ClientError represents an error caused by client input
type ClientError struct {
	Code    string
	Message string
	Details map[string]string
}

func (e ClientError) Error() string {
	return e.Message
}

// IsClientError checks if an error is a client error
func IsClientError(err error) bool {
	var clientErr ClientError
	return errors.As(err, &clientErr)
}

// NewZipCodeError creates a client error for invalid zip codes
func NewZipCodeError(zipCode string) error {
	return ClientError{
		Code:    "INVALID_ZIP_CODE",
		Message: fmt.Sprintf("Invalid or unknown zip code: %s", zipCode),
		Details: map[string]string{
			"zip_code": zipCode,
		},
	}
}
