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
	stepUpErrorUnavailable = "step_up_throttle_unavailable"
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

	decision, err := s.throttle.Begin(ctx, attempt.key)
	if err != nil {
		s.l.Error("two-factor step-up throttle check failed; refusing the attempt", zap.Error(err))
		s.recordStepUp(ctx, attempt, iam.AuthEventOutcomeDenied, stepUpErrorUnavailable)
		return nil, errortypes.NewRateLimitError(
			"code",
			"Verification is briefly unavailable. Try again in a moment.",
		).WithRetryAfter(sessionports.LoginThrottleUnavailableRetry).WithInternal(err)
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

func (s *Service) GuardPassword(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	verify func() error,
) error {
	attempt, err := s.beginStepUp(ctx, tenantInfo.UserID, tenantInfo)
	if err != nil {
		return err
	}

	if err = verify(); err != nil {
		s.recordStepUpFailure(ctx, attempt)
	}

	return err
}

func (s *Service) finishStepUp(ctx context.Context, attempt *stepUpAttempt, err error) {
	if err == nil {
		if resetErr := s.throttle.Succeed(ctx, attempt.key); resetErr != nil {
			s.l.Warn("failed to clear the two-factor step-up counter", zap.Error(resetErr))
		}

		s.recordStepUp(ctx, attempt, iam.AuthEventOutcomeSuccess, "")
		return
	}

	s.recordStepUpFailure(ctx, attempt)
}

func (s *Service) recordStepUpFailure(ctx context.Context, attempt *stepUpAttempt) {
	s.recordStepUp(ctx, attempt, iam.AuthEventOutcomeFailed, stepUpErrorRejected)
	if attempt.decision.Tripped {
		s.recordStepUp(ctx, attempt, iam.AuthEventOutcomeDenied, stepUpErrorThrottled)
	}
}

func (s *Service) recordStepUp(
	ctx context.Context,
	attempt *stepUpAttempt,
	outcome iam.AuthEventOutcome,
	errorCode string,
) {
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
