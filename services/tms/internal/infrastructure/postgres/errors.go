package postgres

import "errors"

var (
	ErrFilePasswordSourceNotImplemented   = errors.New("file password source not implemented")
	ErrSecretPasswordSourceNotImplemented = errors.New("secret password source not implemented")
	ErrDatabaseConnectionNotInitialized   = errors.New("database connection not initialized")
)
