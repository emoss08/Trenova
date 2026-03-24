package m2msync

import "errors"

var (
	ErrInvalidInput  = errors.New("input must be a slice of pointers")
	ErrNoIDField     = errors.New("entity must have an ID field")
	ErrInvalidIDType = errors.New("id field must be of type pulid.ID")
)
