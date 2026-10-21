package config

import "errors"

var InsecureDefaultValues = []string{"change", "secret", "example"}

var (
	ErrRLSScopeKeyIDInvalid = errors.New(
		"database.rls.scopeKeyId must be 1-32 characters of letters, digits, '_' or '-'",
	)
	ErrRLSScopeKeyRequired = errors.New(
		"database.rls.scopeKey is required when row-level security is on",
	)
	ErrRLSScopeKeyInvalid = errors.New(
		"database.rls.scopeKey must be base64 encoding at least 32 random bytes",
	)
	ErrRLSSystemRoleRequired = errors.New(
		"database.rls.mode=enforce requires database.system.user and database.system.password",
	)
	ErrRLSSystemRoleMustDiffer = errors.New(
		"database.system.user must be a different role from database.user",
	)
	ErrRLSMigratorRoleMustDiffer = errors.New(
		"database.migrator.user must be a different role from database.user when row-level security is enforced",
	)
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
		"production and staging require security.encryption.keyManager=gcp-autokey, or keyManager=local with security.encryption.allowLocalKeyManagerInProduction",
	)
	ErrProductionLocalEncryptionKeyRequired = errors.New(
		"security.encryption.allowLocalKeyManagerInProduction requires security.encryption.key of at least 32 characters",
	)
	ErrProductionRecoveryCodePepperRequired = errors.New(
		"security.mfa.recoveryCodePepper must be set to a random value of at least 32 characters in production",
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
	ErrProductionContentScanFailOpen = errors.New(
		"production and staging refuse contentScan.failOpen: a file is held until the scanner can check it",
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
	ErrInvalidTrustedProxy = errors.New(
		"server.trustedProxies entries must be an IP address or CIDR block",
	)
	ErrInvalidHostPrefixCookie = errors.New(
		"__Host- session cookies require secure=true, httpOnly=true, sameSite=strict, path=/, and an empty domain",
	)
	ErrSessionIdleTimeoutExceedsMaxAge = errors.New(
		"security.session.idleTimeout cannot be longer than security.session.maxAge",
	)
	ErrSessionIdleWarningOutOfRange = errors.New(
		"security.session.idleWarning must be set and shorter than security.session.idleTimeout",
	)
	ErrLoggingOutputIsFileButFileConfigIsMissing = errors.New(
		"logging output is 'file' but file config is missing",
	)
	ErrRequestTimeoutExceedsWriteTimeout = errors.New(
		"server request timeout must be shorter than server write timeout",
	)
	ErrPlatformModeRequiresEdition = errors.New(
		"platform.mode is provided by an edition that is not part of this build",
	)
	ErrSectionRequiresEdition = errors.New(
		"configuration section belongs to an edition that is not part of this build",
	)
	ErrSectionNameRequired      = errors.New("configuration section name is required")
	ErrSectionPathsRequired     = errors.New("configuration section must own at least one path")
	ErrSectionAlreadyRegistered = errors.New("configuration section is already registered")
	ErrSectionPathConflict      = errors.New(
		"configuration section path is already owned by another section or by the base configuration",
	)
)
