package postgres

import "errors"

var (
	ErrFilePasswordSourceNotImplemented   = errors.New("file password source not implemented")
	ErrSecretPasswordSourceNotImplemented = errors.New("secret password source not implemented")
	ErrDatabaseConnectionNotInitialized   = errors.New("database connection not initialized")
	ErrUnsupportedIsolation               = errors.New(
		"postgres: unsupported transaction isolation",
	)
	ErrSharedResultCacheNeedsRedis = errors.New(
		"database.cache.shared.enabled needs the Redis connection (the cache section)",
	)
)
