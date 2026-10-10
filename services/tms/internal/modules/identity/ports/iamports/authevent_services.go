package iamports

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/shared/pulid"
)

const AuthEventProviderPassword = "password"

const AuthEventProviderMFA = "mfa"

const AuthEventProviderLogout = "session.logout"

const AuthEventProviderPasswordResetRequest = "password_reset.request"

const AuthEventProviderPasswordResetConfirm = "password_reset.confirm"

const AuthEventProviderSignupRequested = "signup_requested"

const AuthEventProviderSignupRejected = "signup_rejected"

const AuthEventProviderSignupVerified = "signup_verified"

const AuthEventProviderSignupProvisioned = "signup_provisioned"

const AuthEventProviderLoginThrottled = "login_throttled"

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
