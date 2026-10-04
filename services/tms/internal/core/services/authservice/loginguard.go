package authservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/requestmeta"
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

func loginThrottleKey(ctx context.Context, emailAddress string) repositories.LoginThrottleKey {
	key := repositories.LoginThrottleKey{
		Account: strings.ToLower(strings.TrimSpace(emailAddress)),
	}
	if meta, ok := requestmeta.From(ctx); ok {
		key.ClientIP = strings.TrimSpace(meta.ClientIP)
	}

	return key
}

func (s *Service) checkLoginThrottle(
	ctx context.Context,
	key repositories.LoginThrottleKey,
	attempt *authAttempt,
) error {
	if s.throttle == nil {
		return nil
	}

	decision, err := s.throttle.Check(ctx, key)
	if err != nil {
		s.l.Warn("login throttle check failed; allowing the attempt", zap.Error(err))
		return nil
	}
	if !decision.Blocked {
		return nil
	}

	attempt.provider = services.AuthEventProviderLoginThrottled
	if decision.Scope == repositories.LoginThrottleScopeIP {
		attempt.fail(authErrorThrottledIP)
		attempt.riskSignals = []string{riskSignalIPThrottle}
	} else {
		attempt.fail(authErrorThrottledAccount)
		attempt.riskSignals = []string{riskSignalAccountThrottle}
	}

	return errortypes.NewRateLimitError(
		"emailAddress",
		"Too many sign-in attempts. Wait a moment before trying again.",
	).WithRetryAfter(decision.RetryAfter)
}

func (s *Service) recordLoginFailure(ctx context.Context, key repositories.LoginThrottleKey) {
	if s.throttle == nil {
		return
	}

	decision, err := s.throttle.RecordFailure(ctx, key)
	if err != nil {
		s.l.Warn("failed to record a failed sign-in", zap.Error(err))
		return
	}
	if !decision.Blocked || s.authEvents == nil {
		return
	}

	rec := &services.AuthEventRecord{
		Provider:    services.AuthEventProviderLoginThrottled,
		Outcome:     authOutcome(errortypes.NewRateLimitError("", "throttled")),
		RiskSignals: []string{riskSignalAccountThrottle},
		ErrorCode:   authErrorThrottledAccount,
	}
	if decision.Scope == repositories.LoginThrottleScopeIP {
		rec.RiskSignals = []string{riskSignalIPThrottle}
		rec.ErrorCode = authErrorThrottledIP
	}
	s.authEvents.Record(ctx, rec)
}

func (s *Service) resetLoginThrottle(ctx context.Context, key repositories.LoginThrottleKey) {
	if s.throttle == nil {
		return
	}

	if err := s.throttle.Reset(ctx, key.Account); err != nil {
		s.l.Warn("failed to clear the sign-in failure counter", zap.Error(err))
	}
}

func (s *Service) enforcePlanLogin(
	ctx context.Context,
	usr *tenant.User,
	targetOrg *tenant.Organization,
) (*tenant.Organization, error) {
	if s.plans == nil || !s.plans.IsCloud() {
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
	req *services.CreateSessionForUserRequest,
) (resp *services.LoginResponse, err error) {
	if req == nil || req.User == nil {
		return nil, errortypes.NewAuthenticationError("A user is required to start a session")
	}

	usr := req.User
	provider := strings.TrimSpace(req.AuthProvider)
	if provider == "" {
		provider = services.AuthEventProviderPassword
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
		RiskDecision:          "allow",
	})
}
