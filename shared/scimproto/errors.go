package scimproto

import (
	"errors"
	"net/http"
	"strconv"
)

const (
	ScimTypeInvalidFilter = "invalidFilter"
	ScimTypeTooMany       = "tooMany"
	ScimTypeUniqueness    = "uniqueness"
	ScimTypeMutability    = "mutability"
	ScimTypeInvalidSyntax = "invalidSyntax"
	ScimTypeInvalidPath   = "invalidPath"
	ScimTypeNoTarget      = "noTarget"
	ScimTypeInvalidValue  = "invalidValue"
)

type Error struct {
	Status   int
	ScimType string
	Detail   string
}

func (e *Error) Error() string {
	return e.Detail
}

type ErrorBody struct {
	Schemas  []string `json:"schemas"`
	Status   string   `json:"status"`
	ScimType string   `json:"scimType,omitempty"`
	Detail   string   `json:"detail,omitempty"`
}

func (e *Error) Body() *ErrorBody {
	return &ErrorBody{
		Schemas:  []string{SchemaError},
		Status:   strconv.Itoa(e.Status),
		ScimType: e.ScimType,
		Detail:   e.Detail,
	}
}

func AsError(err error) (*Error, bool) {
	var scimErr *Error
	if errors.As(err, &scimErr) {
		return scimErr, true
	}

	return nil, false
}

func InvalidFilter(detail string) *Error {
	return &Error{Status: http.StatusBadRequest, ScimType: ScimTypeInvalidFilter, Detail: detail}
}

func InvalidSyntax(detail string) *Error {
	return &Error{Status: http.StatusBadRequest, ScimType: ScimTypeInvalidSyntax, Detail: detail}
}

func InvalidPath(detail string) *Error {
	return &Error{Status: http.StatusBadRequest, ScimType: ScimTypeInvalidPath, Detail: detail}
}

func InvalidValue(detail string) *Error {
	return &Error{Status: http.StatusBadRequest, ScimType: ScimTypeInvalidValue, Detail: detail}
}

func NoTarget(detail string) *Error {
	return &Error{Status: http.StatusBadRequest, ScimType: ScimTypeNoTarget, Detail: detail}
}

func Mutability(detail string) *Error {
	return &Error{Status: http.StatusBadRequest, ScimType: ScimTypeMutability, Detail: detail}
}

func TooMany(detail string) *Error {
	return &Error{Status: http.StatusBadRequest, ScimType: ScimTypeTooMany, Detail: detail}
}

func Uniqueness(detail string) *Error {
	return &Error{Status: http.StatusConflict, ScimType: ScimTypeUniqueness, Detail: detail}
}

func NotFound(detail string) *Error {
	return &Error{Status: http.StatusNotFound, Detail: detail}
}

func Unauthorized(detail string) *Error {
	return &Error{Status: http.StatusUnauthorized, Detail: detail}
}

func Forbidden(detail string) *Error {
	return &Error{Status: http.StatusForbidden, Detail: detail}
}

func Unsupported(detail string) *Error {
	return &Error{Status: http.StatusNotImplemented, Detail: detail}
}

func PayloadTooLarge(detail string) *Error {
	return &Error{Status: http.StatusRequestEntityTooLarge, Detail: detail}
}
