package variables

import "errors"

var (
	ErrNoResolverConfigured = errors.New("no resolver configured")
	ErrVariableNameEmpty    = errors.New("variable name cannot be empty")
	ErrVariableNotFound     = errors.New("variable not found")
)
