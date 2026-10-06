package authservice

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"go.uber.org/zap"
)

const (
	mfaChallengeTTL         = 5 * time.Minute
	maxMFAChallengeFailures = 5
	mfaAssuranceLevel       = 2

	mfaStateChallenged = "challenged"
	riskDecisionAllow  = "allow"
	mfaStateRejected   = "rejected"

	authErrorMFAChallenge   = "mfa_challenge_invalid"
	authErrorMFAInvalidCode = "mfa_invalid_code"
	authErrorMFAExhausted   = "mfa_attempts_exhausted"
)

var (
	errMFAChallengeExpired = errortypes.NewAuthenticationError(
		"Your sign-in has expired. Sign in again.",
	)
	errMFAChallengeExhausted = errortypes.NewAuthenticationError(
		"Too many incorrect codes. Sign in again.",
	)
)

func (s *Service) issueMFAChallenge(
	ctx context.Context,
	usr *tenant.User,
	targetOrg *tenant.Organization,
	attempt *authAttempt,
) (*services.LoginResponse, error) {
	if s.mfa == nil || s.challenges == nil {
		return nil, nil //nolint:nilnil // no second factor is configured for this process
	}

	active, err := s.mfa.HasActiveFactor(ctx, usr.ID)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, nil //nolint:nilnil // the account has no second factor to ask for
	}

	token, tokenHash, err := tokenutils.New()
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	challenge := &repositories.MFAChallenge{
		UserID:       usr.ID,
		EmailAddress: usr.EmailAddress,
		IssuedAt:     now,
		ExpiresAt:    now + int64(mfaChallengeTTL.Seconds()),
	}
	if targetOrg != nil {
		challenge.OrganizationID = targetOrg.ID
		challenge.BusinessUnitID = targetOrg.BusinessUnitID
		challenge.SwitchOrg = true
	}

	if err = s.challenges.Save(ctx, tokenHash, challenge, mfaChallengeTTL); err != nil {
		return nil, err
	}

	attempt.mfaState = mfaStateChallenged

	return &services.LoginResponse{
		MFARequired:       true,
		MFAChallengeToken: token,
		MFAMethods:        []string{services.MFAMethodTOTP, services.MFAMethodRecoveryCode},
		ExpiresAt:         challenge.ExpiresAt,
	}, nil
}

func (s *Service) VerifyMFAChallenge(
	ctx context.Context,
	req services.VerifyMFAChallengeRequest,
) (resp *services.LoginResponse, err error) {
	if err = req.Validate(); err != nil {
		return nil, err
	}

	attempt := newAuthAttempt(services.AuthEventProviderMFA)
	attempt.aal = mfaAssuranceLevel
	defer func() { s.recordAuthAttempt(ctx, attempt, err) }()

	if s.mfa == nil || s.challenges == nil {
		attempt.fail(authErrorMFAChallenge)
		return nil, errMFAChallengeExpired
	}

	tokenHash := tokenutils.Hash(strings.TrimSpace(req.ChallengeToken))
	challenge, err := s.challenges.Get(ctx, tokenHash)
	if err != nil || challenge == nil || challenge.ExpiresAt < timeutils.NowUnix() {
		attempt.fail(authErrorMFAChallenge)
		return nil, errMFAChallengeExpired
	}

	usr, err := s.ur.FindByIDForLogin(ctx, challenge.UserID)
	if err != nil {
		attempt.fail(authErrorMFAChallenge)
		return nil, errMFAChallengeExpired
	}
	attempt.forUser(usr)
	if challenge.SwitchOrg {
		attempt.forOrganization(challenge.OrganizationID, challenge.BusinessUnitID)
	}

	if err = usr.ValidateStatus(); err != nil {
		attempt.fail(authErrorAccountUnavailable)
		return nil, err
	}

	throttleKey := loginThrottleKey(ctx, usr.EmailAddress)
	if err = s.checkLoginThrottle(ctx, throttleKey, attempt); err != nil {
		return nil, err
	}

	scoped := userScope(ctx, usr)
	method, err := s.mfa.VerifySecondFactor(scoped, &services.VerifySecondFactorRequest{
		UserID:       usr.ID,
		Code:         req.Code,
		RecoveryCode: req.RecoveryCode,
	})
	if err != nil {
		attempt.mfaState = mfaStateRejected
		return nil, s.rejectMFAAttempt(ctx, tokenHash, throttleKey, attempt, err)
	}
	attempt.mfaState = method

	if delErr := s.challenges.Delete(ctx, tokenHash); delErr != nil {
		s.l.Warn("failed to delete a redeemed sign-in challenge", zap.Error(delErr))
	}
	s.resetLoginThrottle(ctx, throttleKey)

	var targetOrg *tenant.Organization
	if challenge.SwitchOrg {
		targetOrg = &tenant.Organization{
			ID:             challenge.OrganizationID,
			BusinessUnitID: challenge.BusinessUnitID,
		}
	}

	now := timeutils.NowUnix()
	return s.finishLogin(scoped, usr, targetOrg, loginSessionContext{
		AuthProvider:          services.AuthEventProviderPassword,
		AuthenticatorAAL:      mfaAssuranceLevel,
		FederationFAL:         1,
		MFAAuthenticatedAt:    now,
		LastReauthenticatedAt: now,
		RiskDecision:          riskDecisionAllow,
	})
}

func (s *Service) rejectMFAAttempt(
	ctx context.Context,
	tokenHash string,
	throttleKey repositories.LoginThrottleKey,
	attempt *authAttempt,
	cause error,
) error {
	s.recordLoginFailure(ctx, throttleKey)

	failures, err := s.challenges.RecordFailure(ctx, tokenHash, mfaChallengeTTL)
	if err != nil {
		s.l.Warn("failed to count a rejected second factor", zap.Error(err))
	}

	if err != nil || failures >= maxMFAChallengeFailures {
		if delErr := s.challenges.Delete(ctx, tokenHash); delErr != nil {
			s.l.Warn("failed to delete an exhausted sign-in challenge", zap.Error(delErr))
		}
		attempt.fail(authErrorMFAExhausted)
		return errMFAChallengeExhausted
	}

	attempt.fail(authErrorMFAInvalidCode)
	return cause
}
