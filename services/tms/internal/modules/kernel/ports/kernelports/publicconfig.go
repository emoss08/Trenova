package kernelports

type PublicSessionPolicy struct {
	IdleTimeoutSeconds int64 `json:"idleTimeoutSeconds"`
	IdleWarningSeconds int64 `json:"idleWarningSeconds"`
	KeepAliveSeconds   int64 `json:"keepAliveSeconds"`
}

type PublicTrenovaSupport struct {
	RequestsURL   string `json:"requestsUrl,omitempty"`
	NewRequestURL string `json:"newRequestUrl,omitempty"`
}

type PublicConfig struct {
	PlatformMode   string                `json:"platformMode"`
	Managed        bool                  `json:"managed"`
	TrenovaSupport *PublicTrenovaSupport `json:"trenovaSupport,omitempty"`
	Session        PublicSessionPolicy   `json:"session"`
}
