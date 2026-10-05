package cloudsignupservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	reasonHoneypot          = "honeypot"
	reasonTurnstile         = "turnstile_rejected"
	reasonTurnstileDown     = "turnstile_unavailable"
	reasonInvalidInput      = "invalid_input"
	reasonExistingAccount   = "existing_account"
	reasonSendLimit         = "send_limit_reached"
	reasonUnknownSignup     = "unknown_signup"
	reasonInvalidToken      = "invalid_token"
	reasonExpiredToken      = "expired_token"
	reasonSignupsPaused     = "signups_paused"
	reasonEmailInUse        = "email_in_use"
	reasonProvisioningError = "provisioning_failed"
)

type signupEvent struct {
	provider       string
	outcome        iam.AuthEventOutcome
	reason         string
	userID         pulid.ID
	organizationID pulid.ID
	businessUnitID pulid.ID
}

func (s *Service) record(ctx context.Context, event *signupEvent) {
	if s.authEvents == nil {
		return
	}

	rec := &services.AuthEventRecord{
		Provider:         event.provider,
		Outcome:          event.outcome,
		UserID:           event.userID,
		OrganizationID:   event.organizationID,
		BusinessUnitID:   event.businessUnitID,
		AuthenticatorAAL: signupAuthAAL,
		FederationFAL:    signupAuthAAL,
		RiskOutcome:      iam.RiskOutcomeAllow,
		ErrorCode:        event.reason,
	}
	if event.outcome == iam.AuthEventOutcomeDenied {
		rec.RiskOutcome = iam.RiskOutcomeDeny
	}

	s.authEvents.Record(ctx, rec)
}

func (s *Service) rejected(ctx context.Context, reason string) {
	s.record(ctx, &signupEvent{
		provider: services.AuthEventProviderSignupRejected,
		outcome:  iam.AuthEventOutcomeDenied,
		reason:   reason,
	})
}

func (s *Service) requested(ctx context.Context, reason string) {
	s.record(ctx, &signupEvent{
		provider: services.AuthEventProviderSignupRequested,
		outcome:  iam.AuthEventOutcomeSuccess,
		reason:   reason,
	})
}
