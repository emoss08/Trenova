package testutil

import "errors"

var (
	ErrTest         = errors.New("test error")
	ErrNotFound     = errors.New("not found")
	ErrDatabase     = errors.New("database error")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict")
	ErrInternal     = errors.New("internal error")
)
