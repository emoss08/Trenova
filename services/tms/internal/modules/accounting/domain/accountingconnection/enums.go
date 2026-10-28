package accountingconnection

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

type AuthMethod string

const (
	AuthMethodAuthorization     = AuthMethod("Authorization")
	AuthMethodClientCredentials = AuthMethod("ClientCredentials")
)

func (m AuthMethod) String() string { return string(m) }

func AllAuthMethods() []AuthMethod {
	return []AuthMethod{AuthMethodAuthorization, AuthMethodClientCredentials}
}

func (m AuthMethod) IsValid() bool {
	switch m {
	case AuthMethodAuthorization, AuthMethodClientCredentials:
		return true
	default:
		return false
	}
}

const (
	FailingAfterConsecutiveFailures = 3
	maxPausedReasonLength           = 500
	maxChangesErrorMessage          = 2000
	maxErrorMessageLength           = 2000
)

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
