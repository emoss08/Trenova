package kernelports

import (
	"context"
)

type PublicFreePlanSummary struct {
	Limits map[string]int64 `json:"limits"`
}

type PublicSessionPolicy struct {
	IdleTimeoutSeconds int64 `json:"idleTimeoutSeconds"`
	IdleWarningSeconds int64 `json:"idleWarningSeconds"`
	KeepAliveSeconds   int64 `json:"keepAliveSeconds"`
}

type PublicConfig struct {
	PlatformMode     string                `json:"platformMode"`
	SignupEnabled    bool                  `json:"signupEnabled"`
	TurnstileSiteKey string                `json:"turnstileSiteKey"`
	TermsURL         string                `json:"termsUrl"`
	PrivacyURL       string                `json:"privacyUrl"`
	FreePlan         PublicFreePlanSummary `json:"freePlan"`
	Session          PublicSessionPolicy   `json:"session"`
}

type PublicConfigContributor interface {
	ContributePublicConfig(ctx context.Context, cfg *PublicConfig) error
}
