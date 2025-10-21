package fileutils

import "errors"

var (
	ErrFileHasNoExtension      = errors.New("file has no extension")
	ErrCouldNotFindProjectRoot = errors.New("could not find project root (no go.mod found)")
)
