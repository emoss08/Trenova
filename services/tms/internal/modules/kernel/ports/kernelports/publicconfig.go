package kernelports

type PublicSessionPolicy struct {
	IdleTimeoutSeconds int64 `json:"idleTimeoutSeconds"`
	IdleWarningSeconds int64 `json:"idleWarningSeconds"`
	KeepAliveSeconds   int64 `json:"keepAliveSeconds"`
}

type PublicConfig struct {
	PlatformMode string              `json:"platformMode"`
	Session      PublicSessionPolicy `json:"session"`
}
