package types

import (
	"fmt"
	"strings"
)

type HTTPResponse struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  any    `json:"results"`
}

type ValidationErrorDetail struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Attr   string `json:"attr"`
}

type ValidationErrorResponse struct { //nolint:errname // JSON structure
	Type   string                  `json:"type"`
	Errors []ValidationErrorDetail `json:"errors"`
}

// Error implements error interface for ValidationErrorResponse.
func (v *ValidationErrorResponse) Error() string {
	errs := make([]string, len(v.Errors))
	for i, e := range v.Errors {
		errs[i] = fmt.Sprintf("%s: %s (attribute: %s)", e.Code, e.Detail, e.Attr)
	}
	return fmt.Sprintf("validation error: [%s]", strings.Join(errs, ", "))
}
