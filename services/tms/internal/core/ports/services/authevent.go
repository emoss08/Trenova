package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	AuthEventProviderPassword             = "password"
	AuthEventProviderLogout               = "session.logout"
	AuthEventProviderPasswordResetRequest = "password_reset.request"
	AuthEventProviderPasswordResetConfirm = "password_reset.confirm"
)

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
	Record(ctx context.Context, rec AuthEventRecord)
}
