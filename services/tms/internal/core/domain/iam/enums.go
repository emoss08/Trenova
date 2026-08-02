package iam

type IdentityProviderProtocol string

const (
	IdentityProviderProtocolOIDC IdentityProviderProtocol = "OIDC"
	IdentityProviderProtocolSAML IdentityProviderProtocol = "SAML"
)

func (p IdentityProviderProtocol) IsValid() bool {
	switch p {
	case IdentityProviderProtocolOIDC, IdentityProviderProtocolSAML:
		return true
	default:
		return false
	}
}

type MFAAuthenticatorType string

const (
	MFAAuthenticatorTypeWebAuthn MFAAuthenticatorType = "webauthn"
	MFAAuthenticatorTypeTOTP     MFAAuthenticatorType = "totp"
)

type AuthEventOutcome string

const (
	AuthEventOutcomeSuccess   AuthEventOutcome = "success"
	AuthEventOutcomeChallenge AuthEventOutcome = "challenge"
	AuthEventOutcomeDenied    AuthEventOutcome = "denied"
	AuthEventOutcomeFailed    AuthEventOutcome = "failed"
)

type RiskOutcome string

const (
	RiskOutcomeAllow     RiskOutcome = "allow"
	RiskOutcomeChallenge RiskOutcome = "challenge"
	RiskOutcomeDeny      RiskOutcome = "deny"
)

type SCIMTokenStatus string

const (
	SCIMTokenStatusActive  SCIMTokenStatus = "active"
	SCIMTokenStatusRevoked SCIMTokenStatus = "revoked"
)

type ProvisioningAction string

const (
	ProvisioningActionCreate     ProvisioningAction = "create"
	ProvisioningActionUpdate     ProvisioningAction = "update"
	ProvisioningActionDeactivate ProvisioningAction = "deactivate"
	ProvisioningActionDelete     ProvisioningAction = "delete"
)

type PolicyEffect string

const (
	PolicyEffectAllow PolicyEffect = "allow"
	PolicyEffectDeny  PolicyEffect = "deny"
)

func (v MFAAuthenticatorType) IsValid() bool {
	switch v {
	case MFAAuthenticatorTypeWebAuthn,
		MFAAuthenticatorTypeTOTP:
		return true
	default:
		return false
	}
}

func (v AuthEventOutcome) IsValid() bool {
	switch v {
	case AuthEventOutcomeSuccess,
		AuthEventOutcomeChallenge,
		AuthEventOutcomeDenied,
		AuthEventOutcomeFailed:
		return true
	default:
		return false
	}
}

func (v RiskOutcome) IsValid() bool {
	switch v {
	case RiskOutcomeAllow,
		RiskOutcomeChallenge,
		RiskOutcomeDeny:
		return true
	default:
		return false
	}
}

func (v SCIMTokenStatus) IsValid() bool {
	switch v {
	case SCIMTokenStatusActive,
		SCIMTokenStatusRevoked:
		return true
	default:
		return false
	}
}

func (v ProvisioningAction) IsValid() bool {
	switch v {
	case ProvisioningActionCreate,
		ProvisioningActionUpdate,
		ProvisioningActionDeactivate,
		ProvisioningActionDelete:
		return true
	default:
		return false
	}
}

func (v PolicyEffect) IsValid() bool {
	switch v {
	case PolicyEffectAllow,
		PolicyEffectDeny:
		return true
	default:
		return false
	}
}
