package iamports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/shared/pulid"
)

const AuthEventProviderPassword = "password"

const AuthEventProviderMFA = "mfa"

const AuthEventProviderMFAStepUp = "mfa.step_up"

const AuthEventProviderLogout = "session.logout"

const AuthEventProviderSessionRevoked = "session.revoked"

const AuthEventProviderPasswordResetRequest = "password_reset.request"

const AuthEventProviderPasswordResetConfirm = "password_reset.confirm"

const AuthEventProviderInviteAccept = "invite.accept"

const AuthEventProviderSignupRequested = "signup_requested"

const AuthEventProviderSignupRejected = "signup_rejected"

const AuthEventProviderSignupVerified = "signup_verified"

const AuthEventProviderSignupProvisioned = "signup_provisioned"

const AuthEventProviderLoginThrottled = "login_throttled"

func NonSignInAuthEventProviders() []string {
	return []string{
		AuthEventProviderLogout,
		AuthEventProviderSessionRevoked,
		AuthEventProviderPasswordResetRequest,
		AuthEventProviderPasswordResetConfirm,
		AuthEventProviderInviteAccept,
		AuthEventProviderSignupRequested,
		AuthEventProviderSignupRejected,
		AuthEventProviderSignupVerified,
		AuthEventProviderSignupProvisioned,
	}
}

type AuthEventRecord struct {
	Provider         string
	Outcome          iam.AuthEventOutcome
	UserID           pulid.ID
	OrganizationID   pulid.ID
	BusinessUnitID   pulid.ID
	AuthenticatorAAL int
	FederationFAL    int
	MFAState         string
	RiskOutcome      iam.RiskOutcome
	RiskSignals      []string
	ErrorCode        string
}

type AuthEventRecorder interface {
	Record(ctx context.Context, rec *AuthEventRecord)
}
