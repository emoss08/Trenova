package models

import "encoding/json"

type RequestData struct {
	Name string `json:"name" validate:"required"`
}

type FieldError struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Attr   string `json:"attr,omitempty"`
}

type APIErrorResponse struct {
	Type    string       `json:"type"`
	Message string       `json:"message"`
	Errors  []FieldError `json:"errors"`
}

func NewAPIError(statusCode int, errorType, message string, fieldErrors []FieldError) error {
	return &APIErrorResponse{
		Type:    errorType,
		Message: message,
		Errors:  fieldErrors,
	}
}

func (r *APIErrorResponse) Error() string {
	// Convert the APIErrorResponse to a JSON string for the error message.
	// Error messages are typically strings, so you might just return a simple message
	// or format the struct as a JSON string.
	b, err := json.Marshal(r)
	if err != nil {
		return "error marshalling APIErrorResponse"
	}
	return string(b)
}
