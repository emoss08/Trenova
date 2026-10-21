package mfaservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	stepUpErrorThrottled   = "step_up_throttled"
	stepUpErrorRejected    = "step_up_rejected"
	stepUpMFAStateVerified = "verified"
	stepUpMFAStateRejected = "rejected"
)

type stepUpAttempt struct {
	user     *tenant.User
	key      sessionports.LoginThrottleKey
	decision *sessionports.LoginThrottleDecision
	tenant   pagination.TenantInfo
}

func (s *Service) beginStepUp(
	ctx context.Context,
	userID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*stepUpAttempt, error) {
	usr, err := s.users.FindByIDForLogin(ctx, userID)
	if err != nil {
		return nil, err
	}

	attempt := &stepUpAttempt{
		user:   usr,
		key:    sessionports.NewLoginThrottleKey(ctx, usr.EmailAddress),
		tenant: tenantInfo,
	}
	if attempt.tenant.OrgID.IsNil() {
		attempt.tenant.OrgID = usr.CurrentOrganizationID
		attempt.tenant.BuID = usr.BusinessUnitID
	}
	attempt.tenant.UserID = usr.ID

	if s.throttle == nil {
		return attempt, nil
	}

	decision, err := s.throttle.Begin(ctx, attempt.key)
	if err != nil {
		s.l.Warn("two-factor step-up throttle check failed; allowing the attempt", zap.Error(err))
		return attempt, nil
	}
	if decision.Blocked {
		s.recordStepUp(ctx, attempt, iam.AuthEventOutcomeDenied, stepUpErrorThrottled)
		return nil, errortypes.NewRateLimitError(
			"code",
			"Too many attempts. Wait a moment before trying again.",
		).WithRetryAfter(decision.RetryAfter)
	}
	attempt.decision = decision

	return attempt, nil
}

func (s *Service) finishStepUp(ctx context.Context, attempt *stepUpAttempt, err error) {
	if err == nil {
		if s.throttle != nil {
			if resetErr := s.throttle.Succeed(ctx, attempt.key); resetErr != nil {
				s.l.Warn("failed to clear the two-factor step-up counter", zap.Error(resetErr))
			}
		}
		s.recordStepUp(ctx, attempt, iam.AuthEventOutcomeSuccess, "")
		return
	}

	s.recordStepUp(ctx, attempt, iam.AuthEventOutcomeFailed, stepUpErrorRejected)
	if attempt.decision != nil && attempt.decision.Tripped {
		s.recordStepUp(ctx, attempt, iam.AuthEventOutcomeDenied, stepUpErrorThrottled)
	}
}

func (s *Service) recordStepUp(
	ctx context.Context,
	attempt *stepUpAttempt,
	outcome iam.AuthEventOutcome,
	errorCode string,
) {
	if s.authEvents == nil {
		return
	}

	rec := &iamports.AuthEventRecord{
		Provider:         iamports.AuthEventProviderMFAStepUp,
		Outcome:          outcome,
		UserID:           attempt.user.ID,
		OrganizationID:   attempt.tenant.OrgID,
		BusinessUnitID:   attempt.tenant.BuID,
		AuthenticatorAAL: mfaAssuranceLevel,
		FederationFAL:    1,
		MFAState:         stepUpMFAStateVerified,
		RiskOutcome:      iam.RiskOutcomeAllow,
		ErrorCode:        errorCode,
	}
	if outcome != iam.AuthEventOutcomeSuccess {
		rec.AuthenticatorAAL = 1
		rec.MFAState = stepUpMFAStateRejected
	}
	if outcome == iam.AuthEventOutcomeDenied {
		rec.RiskOutcome = iam.RiskOutcomeDeny
	}
	s.authEvents.Record(ctx, rec)
}
