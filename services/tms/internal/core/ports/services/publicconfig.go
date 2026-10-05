package services

import "context"

type PublicFreePlanSummary struct {
	Limits map[string]int64 `json:"limits"`
}

type PublicConfig struct {
	PlatformMode     string                `json:"platformMode"`
	SignupEnabled    bool                  `json:"signupEnabled"`
	TurnstileSiteKey string                `json:"turnstileSiteKey"`
	TermsURL         string                `json:"termsUrl"`
	PrivacyURL       string                `json:"privacyUrl"`
	FreePlan         PublicFreePlanSummary `json:"freePlan"`
}

type PublicConfigContributor interface {
	ContributePublicConfig(ctx context.Context, cfg *PublicConfig) error
}
