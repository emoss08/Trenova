package config

import "errors"

var InsecureDefaultValues = []string{"change", "secret", "example"}

var (
	ErrDatabasePasswordNotSet       = errors.New("database password not set in environment")
	ErrDatabasePasswordFileNotSet   = errors.New("database password file not set in environment")
	ErrDatabasePasswordSecretNotSet = errors.New("database password secret not set specified")
	ErrSessionSecretIsRequired      = errors.New("session secret is required")
	ErrSessionSecretIsInsecure      = errors.New(
		"session secret contains insecure default value",
	)
	ErrMaxIdleConnsExceedsMaxOpenConns = errors.New(
		"max idle connections cannot exceed max open connections",
	)
	ErrCacheMinIdleConnsExceedsPoolSize = errors.New(
		"cache min idle connections cannot exceed pool size",
	)
	ErrCorsEnabledButNoAllowedOrigins = errors.New(
		"CORS enabled but no allowed origins specified",
	)
	ErrLoggingOutputIsFileButFileConfigIsMissing = errors.New(
		"logging output is 'file' but file config is missing",
	)
)
