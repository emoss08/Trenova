package accountingsync

type ConnectionStatus string

const (
	ConnectionStatusConnected    = ConnectionStatus("Connected")
	ConnectionStatusDegraded     = ConnectionStatus("Degraded")
	ConnectionStatusFailing      = ConnectionStatus("Failing")
	ConnectionStatusRevoked      = ConnectionStatus("Revoked")
	ConnectionStatusDisconnected = ConnectionStatus("Disconnected")
)

func (s ConnectionStatus) String() string { return string(s) }

func (s ConnectionStatus) IsValid() bool {
	switch s {
	case ConnectionStatusConnected,
		ConnectionStatusDegraded,
		ConnectionStatusFailing,
		ConnectionStatusRevoked,
		ConnectionStatusDisconnected:
		return true
	default:
		return false
	}
}

func AllConnectionStatuses() []ConnectionStatus {
	return []ConnectionStatus{
		ConnectionStatusConnected,
		ConnectionStatusDegraded,
		ConnectionStatusFailing,
		ConnectionStatusRevoked,
		ConnectionStatusDisconnected,
	}
}

type ErrorCategory string

const (
	ErrorCategoryTransient     = ErrorCategory("Transient")
	ErrorCategoryRateLimited   = ErrorCategory("RateLimited")
	ErrorCategoryUnauthorized  = ErrorCategory("Unauthorized")
	ErrorCategoryRevoked       = ErrorCategory("Revoked")
	ErrorCategoryConfiguration = ErrorCategory("Configuration")
	ErrorCategoryUnknown       = ErrorCategory("Unknown")
)

func (c ErrorCategory) String() string { return string(c) }

func (c ErrorCategory) IsValid() bool {
	switch c {
	case ErrorCategoryTransient,
		ErrorCategoryRateLimited,
		ErrorCategoryUnauthorized,
		ErrorCategoryRevoked,
		ErrorCategoryConfiguration,
		ErrorCategoryUnknown:
		return true
	default:
		return false
	}
}

func AllErrorCategories() []ErrorCategory {
	return []ErrorCategory{
		ErrorCategoryTransient,
		ErrorCategoryRateLimited,
		ErrorCategoryUnauthorized,
		ErrorCategoryRevoked,
		ErrorCategoryConfiguration,
		ErrorCategoryUnknown,
	}
}

const (
	maxAppClientIDLength = 255
	appFingerprintLength = 32
)

type AppSource string

const (
	AppSourceInstance = AppSource("Instance")
	AppSourceTenant   = AppSource("Tenant")
)

func (s AppSource) String() string { return string(s) }

func (s AppSource) IsValid() bool {
	switch s {
	case AppSourceInstance, AppSourceTenant:
		return true
	default:
		return false
	}
}

type AppEnvironment string

const (
	AppEnvironmentSandbox    = AppEnvironment("Sandbox")
	AppEnvironmentProduction = AppEnvironment("Production")
)

func (e AppEnvironment) String() string { return string(e) }

func (e AppEnvironment) IsValid() bool {
	switch e {
	case AppEnvironmentSandbox, AppEnvironmentProduction:
		return true
	default:
		return false
	}
}

const (
	FailingAfterConsecutiveFailures = 3
	RefreshTokenWarningWindow       = int64(14 * 24 * 60 * 60)
	maxPausedReasonLength           = 500
	maxChangesErrorMessage          = 2000
	maxErrorMessageLength           = 2000
)

type SyncMode string

const (
	SyncModeDocument = SyncMode("Document")
	SyncModeLedger   = SyncMode("Ledger")
)

func (m SyncMode) String() string { return string(m) }

func (m SyncMode) IsValid() bool {
	switch m {
	case SyncModeDocument, SyncModeLedger:
		return true
	default:
		return false
	}
}

func AllSyncModes() []SyncMode {
	return []SyncMode{SyncModeDocument, SyncModeLedger}
}

type LedgerGranularity string

const (
	LedgerDetailed     = LedgerGranularity("Detailed")
	LedgerDailySummary = LedgerGranularity("DailySummary")
)

func (g LedgerGranularity) String() string { return string(g) }

func (g LedgerGranularity) IsValid() bool {
	switch g {
	case LedgerDetailed, LedgerDailySummary:
		return true
	default:
		return false
	}
}

func AllLedgerGranularities() []LedgerGranularity {
	return []LedgerGranularity{LedgerDetailed, LedgerDailySummary}
}

const (
	JournalDayPrefix     = "jday_"
	JournalOpeningPrefix = "jopen_"
	journalDayLayout     = "20060102"
	JournalDaySQLLayout  = "YYYYMMDD"
)

const (
	PrecheckConfidence  = 0.95
	ConfidentConfidence = 0.70
	MaxModelConfidence  = 0.90
	MaxCandidates       = 3
	maxReasonLength     = 500
)
