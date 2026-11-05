package authservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	riskSignalAccountThrottle = "login_throttle_account"
	riskSignalIPThrottle      = "login_throttle_ip"
)

var errSubscriptionExpired = errortypes.NewAuthorizationError(
	"This workspace's free trial has ended and it is scheduled for deletion, so sign-in is " +
		"no longer available. Start a new trial or contact support to recover it.",
)

func (s *Service) beginLoginAttempt(
	ctx context.Context,
	key sessionports.LoginThrottleKey,
	attempt *authAttempt,
) (*sessionports.LoginThrottleDecision, error) {
	decision, err := s.throttle.Begin(ctx, key)
	if err != nil {
		s.l.Error("login throttle check failed; refusing the attempt", zap.Error(err))
		attempt.fail(authErrorThrottleUnavailable)
		return nil, errortypes.NewRateLimitError(
			"emailAddress",
			"Sign-in is briefly unavailable. Try again in a moment.",
		).WithRetryAfter(sessionports.LoginThrottleUnavailableRetry).WithInternal(err)
	}
	if !decision.Blocked {
		return decision, nil
	}

	attempt.provider = iamports.AuthEventProviderLoginThrottled
	if decision.Scope == sessionports.LoginThrottleScopeIP {
		attempt.fail(authErrorThrottledIP)
		attempt.riskSignals = []string{riskSignalIPThrottle}
	} else {
		attempt.fail(authErrorThrottledAccount)
		attempt.riskSignals = []string{riskSignalAccountThrottle}
	}

	return nil, errortypes.NewRateLimitError(
		"emailAddress",
		"Too many sign-in attempts. Wait a moment before trying again.",
	).WithRetryAfter(decision.RetryAfter)
}

func (s *Service) recordLoginFailure(
	ctx context.Context,
	decision *sessionports.LoginThrottleDecision,
) {
	if !decision.Tripped {
		return
	}

	rec := &iamports.AuthEventRecord{
		Provider:    iamports.AuthEventProviderLoginThrottled,
		Outcome:     authOutcome(errortypes.NewRateLimitError("", "throttled")),
		RiskSignals: []string{riskSignalAccountThrottle},
		ErrorCode:   authErrorThrottledAccount,
	}
	if decision.TrippedScope == sessionports.LoginThrottleScopeIP {
		rec.RiskSignals = []string{riskSignalIPThrottle}
		rec.ErrorCode = authErrorThrottledIP
	}
	s.authEvents.Record(ctx, rec)
}

func (s *Service) resetLoginThrottle(ctx context.Context, key sessionports.LoginThrottleKey) {
	if err := s.throttle.Succeed(ctx, key); err != nil {
		s.l.Warn("failed to clear the sign-in failure counter", zap.Error(err))
	}
}

func (s *Service) enforcePlanLogin(
	ctx context.Context,
	usr *tenant.User,
	targetOrg *tenant.Organization,
) (*tenant.Organization, error) {
	if !s.plans.EnforcesPlans() {
		return targetOrg, nil
	}

	if targetOrg != nil {
		allowed, err := s.organizationAllowsLogin(ctx, targetOrg.ID, targetOrg.BusinessUnitID)
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, errSubscriptionExpired
		}
		return targetOrg, nil
	}

	allowed, err := s.organizationAllowsLogin(ctx, usr.CurrentOrganizationID, usr.BusinessUnitID)
	if err != nil {
		return nil, err
	}
	if allowed {
		return nil, nil //nolint:nilnil // no organization switch is needed
	}

	memberships, err := s.ur.GetOrganizations(ctx, usr.ID)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	for _, membership := range memberships {
		if membership == nil || membership.OrganizationID == usr.CurrentOrganizationID {
			continue
		}
		if membership.ExpiresAt != nil && *membership.ExpiresAt <= now {
			continue
		}

		allowed, err = s.organizationAllowsLogin(
			ctx,
			membership.OrganizationID,
			membership.BusinessUnitID,
		)
		if err != nil {
			return nil, err
		}
		if allowed {
			return &tenant.Organization{
				ID:             membership.OrganizationID,
				BusinessUnitID: membership.BusinessUnitID,
			}, nil
		}
	}

	return nil, errSubscriptionExpired
}

func (s *Service) organizationAllowsLogin(
	ctx context.Context,
	orgID, buID pulid.ID,
) (bool, error) {
	resolved, err := s.plans.Resolve(ctx, orgID, buID)
	if err != nil {
		return false, err
	}

	return resolved.AllowsLogin(), nil
}
