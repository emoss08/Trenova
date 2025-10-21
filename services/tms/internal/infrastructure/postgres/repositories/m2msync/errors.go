package m2msync

import "errors"

var (
	// ErrInvalidInput indicates the input is not a slice
	ErrInvalidInput = errors.New("input must be a slice")

	// ErrNoIDField indicates the entity doesn't have an ID field
	ErrNoIDField = errors.New("entity must have an ID field")

	// ErrInvalidIDType indicates the ID field is not of type pulid.ID
	ErrInvalidIDType = errors.New("ID field must be of type pulid.ID")
)
