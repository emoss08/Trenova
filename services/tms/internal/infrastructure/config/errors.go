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
	ErrEncryptionKeyIsInsecure = errors.New(
		"encryption key contains insecure default value",
	)
	ErrProductionEncryptionModeRequired = errors.New(
		"production and staging require security.encryption.mode=envelope",
	)
	ErrProductionKMSRequired = errors.New(
		"production and staging require security.encryption.keyManager=gcp-autokey",
	)
	ErrProductionGCPKMSConfigRequired = errors.New(
		"production and staging require a GCP KMS crypto key resource",
	)
	ErrProductionDatabaseSSLRequired = errors.New(
		"production and staging require database.sslMode other than disable",
	)
	ErrProductionSessionCookieRequired = errors.New(
		"production and staging require secure, httpOnly, sameSite=strict session cookies",
	)
	ErrProductionStorageTLSRequired = errors.New(
		"production and staging require TLS for non-local object storage endpoints",
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
	ErrCredentialedWildcardCORS = errors.New(
		"production and staging cannot allow wildcard CORS with credentials",
	)
	ErrInvalidHostPrefixCookie = errors.New(
		"__Host- session cookies require secure=true, httpOnly=true, sameSite=strict, path=/, and an empty domain",
	)
	ErrLoggingOutputIsFileButFileConfigIsMissing = errors.New(
		"logging output is 'file' but file config is missing",
	)
	ErrRequestTimeoutExceedsWriteTimeout = errors.New(
		"server request timeout must be shorter than server write timeout",
	)
)
