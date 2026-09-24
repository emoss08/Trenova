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
