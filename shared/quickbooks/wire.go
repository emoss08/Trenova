package quickbooks

import (
	"errors"
	"strings"
)

const (
	QueryResource      = "query"
	MaxRequestIDLength = 50
)

var (
	ErrRequestIDRequired = errors.New(
		"quickbooks: a request id of at most 50 characters is required",
	)
)

type Ref struct {
	Value string `json:"value"`
}

func ValidateRequestID(requestID string) error {
	trimmed := strings.TrimSpace(requestID)
	if trimmed == "" || len(trimmed) > MaxRequestIDLength || trimmed != requestID {
		return ErrRequestIDRequired
	}
	return nil
}
