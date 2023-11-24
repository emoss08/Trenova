package models

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
