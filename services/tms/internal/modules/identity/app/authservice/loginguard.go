package authservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
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
	if s.throttle == nil {
		return nil, nil //nolint:nilnil // no throttle is configured
	}

	decision, err := s.throttle.Begin(ctx, key)
	if err != nil {
		s.l.Warn("login throttle check failed; allowing the attempt", zap.Error(err))
		return nil, nil //nolint:nilnil // the throttle fails open
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
	if decision == nil || !decision.Tripped || s.authEvents == nil {
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
	if s.throttle == nil {
		return
	}

	if err := s.throttle.Succeed(ctx, key); err != nil {
		s.l.Warn("failed to clear the sign-in failure counter", zap.Error(err))
	}
}

func (s *Service) enforcePlanLogin(
	ctx context.Context,
	usr *tenant.User,
	targetOrg *tenant.Organization,
) (*tenant.Organization, error) {
	if s.plans == nil || !s.plans.EnforcesPlans() {
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

func (s *Service) CreateSessionForUser(
	ctx context.Context,
	req *tenantports.CreateSessionForUserRequest,
) (resp *tenantports.LoginResponse, err error) {
	if req == nil || req.User == nil {
		return nil, errortypes.NewAuthenticationError("A user is required to start a session")
	}

	usr := req.User
	provider := strings.TrimSpace(req.AuthProvider)
	if provider == "" {
		provider = iamports.AuthEventProviderPassword
	}

	attempt := newAuthAttempt(provider)
	attempt.forUser(usr)
	defer func() { s.recordAuthAttempt(ctx, attempt, err) }()

	if err = usr.ValidateStatus(); err != nil {
		attempt.fail(authErrorAccountUnavailable)
		return nil, err
	}

	return s.createLoginResponse(userScope(ctx, usr), usr, loginSessionContext{
		AuthProvider:          provider,
		AuthenticatorAAL:      1,
		FederationFAL:         1,
		LastReauthenticatedAt: timeutils.NowUnix(),
		RiskDecision:          riskDecisionAllow,
	})
}
