package cloudsignupservice

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/cloudsignup"
	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/domain/onboarding"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/emailutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	provisioningScopeReason = "cloud signup provisioning"
	maxTokenLength          = 128
	usersEmailIndex         = "idx_users_email_address"
	provisioningLockTimeout = 10 * time.Second
)

var (
	errInvalidVerificationToken = errortypes.NewValidationError(
		"token",
		errortypes.ErrInvalid,
		"This verification link is invalid, expired or already used. Sign up again to get a new one.",
	)
	errEmailInUse = errortypes.NewValidationError(
		"token",
		errortypes.ErrAlreadyExists,
		"An account already exists for this email address. Sign in instead.",
	)
	errSignupExpired = errors.New("cloud signup verification token expired")
	errEmailTaken    = errors.New("cloud signup email already belongs to a user")
)

type provisioned struct {
	signup *cloudsignup.CloudSignup
	result *repositories.BootstrapTenantResult
	sub    *subscription.Subscription
}

func (s *Service) Verify(
	ctx context.Context,
	req *services.CloudSignupVerifyRequest,
) (*services.LoginResponse, error) {
	if !s.Enabled() {
		return nil, errSignupDisabled
	}

	token := strings.TrimSpace(req.Token)
	if token == "" || len(token) > maxTokenLength {
		s.rejected(ctx, reasonInvalidToken)
		return nil, errInvalidVerificationToken
	}

	out, err := s.provision(ctx, tokenutils.Hash(token))
	if err != nil {
		return nil, s.verifyFailure(ctx, err)
	}

	s.afterProvisioning(ctx, out)

	resp, err := s.auth.CreateSessionForUser(ctx, &services.CreateSessionForUserRequest{
		User:         out.result.Owner,
		AuthProvider: services.AuthEventProviderPassword,
	})
	if err != nil {
		s.l.Error("workspace was provisioned but the first session could not be created",
			zap.String("organizationId", out.result.Organization.ID.String()),
			zap.Error(err),
		)
		return nil, err
	}

	return resp, nil
}

func (s *Service) provision(ctx context.Context, tokenHash string) (*provisioned, error) {
	sysCtx := dbscope.WithSystem(ctx, provisioningScopeReason)

	var out *provisioned
	err := s.db.WithTx(sysCtx, ports.TxOptions{LockTimeout: provisioningLockTimeout}, func(
		txCtx context.Context,
		_ bun.Tx,
	) error {
		signup, err := s.signups.GetPendingByTokenHash(txCtx, tokenHash)
		if err != nil {
			if errortypes.IsNotFoundError(err) {
				return errInvalidVerificationToken
			}
			return err
		}

		now := timeutils.NowUnix()
		if signup.IsExpired(now) {
			return &signupFailure{signup: signup, cause: errSignupExpired}
		}

		if err = s.bootstrap.LockProvisioning(txCtx); err != nil {
			return err
		}

		if err = s.checkCapacity(txCtx, now); err != nil {
			return err
		}

		existing, err := s.findUser(txCtx, signup.EmailAddress)
		if err != nil {
			return err
		}
		if existing != nil {
			return &signupFailure{signup: signup, cause: errEmailTaken}
		}

		out, err = s.createTenant(txCtx, signup, now)
		if err != nil {
			if dberror.IsUniqueConstraintViolation(err) &&
				dberror.ExtractConstraintName(err) == usersEmailIndex {
				return &signupFailure{signup: signup, cause: errEmailTaken}
			}
			return err
		}

		orgID := out.result.Organization.ID
		ports.AfterCommit(txCtx, func(context.Context) { s.plans.Invalidate(orgID) })

		return nil
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (s *Service) createTenant(
	ctx context.Context,
	signup *cloudsignup.CloudSignup,
	now int64,
) (*provisioned, error) {
	address, err := emailutils.Parse(signup.EmailAddress)
	if err != nil {
		return nil, err
	}

	result, err := s.bootstrap.Bootstrap(ctx, &repositories.BootstrapTenantRequest{
		BusinessUnitName:  signup.CompanyName,
		Organization:      placeholderOrganization(signup.CompanyName),
		LoginSlugBase:     loginSlugBase(signup.CompanyName),
		StateAbbreviation: placeholderState,
		Owner: &tenant.User{
			Name:               signup.Name,
			EmailAddress:       address.Lower,
			Password:           signup.PasswordHash,
			Status:             domaintypes.StatusActive,
			Timezone:           defaultTimezone,
			Locale:             defaultLocale,
			TimeFormat:         domaintypes.TimeFormat12Hour,
			MustChangePassword: false,
		},
		UsernameBase: usernameBase(address),
		Now:          now,
	})
	if err != nil {
		return nil, err
	}

	trial := s.platform.Cloud.Trial
	trialEndsAt := now + int64(trial.GetLifetime().Seconds())
	sub := &subscription.Subscription{
		OrganizationID: result.Organization.ID,
		BusinessUnitID: result.BusinessUnit.ID,
		PlanKey:        string(platformplan.PlanKeyFreeDemo),
		Status:         subscription.StatusTrialing,
		TrialEndsAt:    trialEndsAt,
		ReadOnlyUntil:  trialEndsAt + int64(trial.GetReadOnlyGrace().Seconds()),
	}
	multiErr := errortypes.NewMultiError()
	sub.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	createdSub, err := s.subscriptions.Create(ctx, sub)
	if err != nil {
		return nil, err
	}

	if _, err = s.onboarding.Create(
		ctx,
		onboarding.NewPending(result.Organization.ID, result.BusinessUnit.ID),
	); err != nil {
		return nil, err
	}

	if err = s.signups.MarkProvisioned(ctx, &repositories.MarkCloudSignupProvisionedRequest{
		ID:             signup.ID,
		OrganizationID: result.Organization.ID,
		BusinessUnitID: result.BusinessUnit.ID,
		UserID:         result.Owner.ID,
		VerifiedAt:     now,
	}); err != nil {
		return nil, err
	}

	return &provisioned{signup: signup, result: result, sub: createdSub}, nil
}

func placeholderOrganization(companyName string) *tenant.Organization {
	return &tenant.Organization{
		Name:                   stringutils.TruncateRunes(companyName, maxCompanyNameLength),
		ScacCode:               onboarding.PlaceholderSCAC,
		DOTNumber:              onboarding.PlaceholderDOTNumber,
		City:                   onboarding.PlaceholderCity,
		PostalCode:             onboarding.PlaceholderPostalCode,
		Timezone:               defaultTimezone,
		Locale:                 defaultLocale,
		BrokerageEnabled:       true,
		AssetOperationsEnabled: true,
	}
}

func (s *Service) checkCapacity(ctx context.Context, now int64) error {
	signup := s.platform.Cloud.Signup

	if limit := signup.GetMaxSignupsPerDay(); limit > 0 {
		since := timeutils.DayStartUTC(now)
		count, err := s.signups.CountProvisionedSince(ctx, since)
		if err != nil {
			return err
		}
		if count >= limit {
			return signupsPaused()
		}
	}

	if limit := signup.GetMaxActiveTenants(); limit > 0 {
		count, err := s.subscriptions.CountByStatus(ctx, &repositories.CountSubscriptionsByStatusRequest{
			Statuses: []subscription.Status{subscription.StatusTrialing, subscription.StatusReadOnly},
		})
		if err != nil {
			return err
		}
		if count >= limit {
			return signupsPaused()
		}
	}

	return nil
}

func signupsPaused() error {
	return errortypes.NewPlanRestrictionError(
		"",
		errortypes.PlanRestrictionReasonSignupsPaused,
		string(platformplan.PlanKeyFreeDemo),
	)
}

type signupFailure struct {
	signup *cloudsignup.CloudSignup
	cause  error
}

func (f *signupFailure) Error() string {
	return f.cause.Error()
}

func (f *signupFailure) Unwrap() error {
	return f.cause
}

func (s *Service) verifyFailure(ctx context.Context, err error) error {
	var failure *signupFailure
	if errors.As(err, &failure) {
		switch {
		case errors.Is(failure.cause, errSignupExpired):
			s.rejectSignup(ctx, failure.signup, reasonExpiredToken)
			return errInvalidVerificationToken
		case errors.Is(failure.cause, errEmailTaken):
			s.rejectSignup(ctx, failure.signup, reasonEmailInUse)
			return errEmailInUse
		}
	}

	switch {
	case errors.Is(err, errInvalidVerificationToken):
		s.rejected(ctx, reasonInvalidToken)
	case errortypes.IsPlanRestrictionError(err):
		s.rejected(ctx, reasonSignupsPaused)
	default:
		s.l.Error("cloud signup provisioning failed", zap.Error(err))
		s.rejected(ctx, reasonProvisioningError)
	}

	return err
}

func (s *Service) rejectSignup(ctx context.Context, signup *cloudsignup.CloudSignup, reason string) {
	s.rejected(ctx, reason)

	if err := s.signups.Reject(ctx, &repositories.RejectCloudSignupRequest{
		ID:     signup.ID,
		Reason: reason,
	}); err != nil && !errortypes.IsNotFoundError(err) {
		s.l.Error("failed to reject a cloud signup",
			zap.String("signupId", signup.ID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) afterProvisioning(ctx context.Context, out *provisioned) {
	org := out.result.Organization
	owner := out.result.Owner

	s.record(ctx, &signupEvent{
		provider:       services.AuthEventProviderSignupVerified,
		outcome:        iam.AuthEventOutcomeSuccess,
		userID:         owner.ID,
		organizationID: org.ID,
		businessUnitID: org.BusinessUnitID,
	})
	s.record(ctx, &signupEvent{
		provider:       services.AuthEventProviderSignupProvisioned,
		outcome:        iam.AuthEventOutcomeSuccess,
		userID:         owner.ID,
		organizationID: org.ID,
		businessUnitID: org.BusinessUnitID,
	})

	s.auditOwnerGrant(ctx, out)

	if err := s.email.SendWelcome(ctx, &services.WelcomeEmail{
		To:          owner.EmailAddress,
		Name:        owner.Name,
		CompanyName: org.Name,
		TrialEndsAt: out.sub.TrialEndsAt,
		Timezone:    owner.Timezone,
	}); err != nil {
		s.l.Error("failed to send the welcome email",
			zap.String("organizationId", org.ID.String()),
			zap.Error(err),
		)
	}

	s.l.Info("cloud signup provisioned a workspace",
		zap.String("organizationId", org.ID.String()),
		zap.String("businessUnitId", org.BusinessUnitID.String()),
		zap.String("userId", owner.ID.String()),
	)
}

func (s *Service) auditOwnerGrant(ctx context.Context, out *provisioned) {
	if s.security == nil {
		return
	}

	org := out.result.Organization
	owner := out.result.Owner
	s.security.RecordChange(ctx, &services.SecurityChange{
		Resource:   permission.ResourceRole,
		ResourceID: out.result.AdminRoleID.String(),
		Operation:  permission.OpAssign,
		Actor: services.AuditActor{
			PrincipalType: services.PrincipalTypeUser,
			PrincipalID:   owner.ID,
			UserID:        owner.ID,
		},
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		After: map[string]any{
			"userId":         owner.ID.String(),
			"roleId":         out.result.AdminRoleID.String(),
			"organizationId": org.ID.String(),
			"membership":     "default",
		},
		Comment: "Cloud signup created the organization owner with the Organization Administrator role",
		Metadata: map[string]any{
			"signupId": out.signup.ID.String(),
			"source":   "cloud_signup",
		},
	})
}
