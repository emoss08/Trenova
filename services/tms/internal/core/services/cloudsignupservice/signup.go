package cloudsignupservice

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/cloudsignup"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/requestmeta"
	"github.com/emoss08/trenova/shared/emailutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var errSignupDisabled = errortypes.NewNotFoundError("Signup is not available")

func (s *Service) Signup(
	ctx context.Context,
	req *services.CloudSignupRequest,
) (*services.CloudSignupAccepted, error) {
	if !s.Enabled() {
		return nil, errSignupDisabled
	}

	if strings.TrimSpace(req.Website) != "" {
		s.rejected(ctx, reasonHoneypot)
		return accepted(), nil
	}

	if err := s.verifyTurnstile(ctx, req.TurnstileToken, services.TurnstileActionSignup); err != nil {
		return nil, err
	}

	input, err := s.validateSignup(req)
	if err != nil {
		s.rejected(ctx, reasonInvalidInput)
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	existing, err := s.findUser(ctx, input.email.Lower)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		s.rejected(ctx, reasonExistingAccount)
		s.sendExistingAccount(ctx, input.email.Lower, existing.Name)
		return accepted(), nil
	}

	token, tokenHash, err := tokenutils.New()
	if err != nil {
		return nil, err
	}
	expiresAt := timeutils.NowUnix() + int64(s.tokenTTL().Seconds())

	signup, sent, err := s.storeSignup(ctx, &storeSignupParams{
		input:        input,
		passwordHash: string(passwordHash),
		tokenHash:    tokenHash,
		expiresAt:    expiresAt,
	})
	if err != nil {
		return nil, err
	}
	if !sent {
		s.rejected(ctx, reasonSendLimit)
		return accepted(), nil
	}

	s.requested(ctx, "")
	s.sendVerification(ctx, signup, token)

	return accepted(), nil
}

func (s *Service) Resend(
	ctx context.Context,
	req *services.CloudSignupResendRequest,
) (*services.CloudSignupAccepted, error) {
	if !s.Enabled() {
		return nil, errSignupDisabled
	}

	if err := s.verifyTurnstile(
		ctx,
		req.TurnstileToken,
		services.TurnstileActionSignupResend,
	); err != nil {
		return nil, err
	}

	address, emailErr := emailutils.Parse(req.EmailAddress)
	if emailErr != nil {
		return nil, errortypes.NewValidationError(
			"emailAddress",
			errortypes.ErrInvalidFormat,
			"Enter a valid email address",
		)
	}

	signup, err := s.signups.GetPendingByEmail(ctx, address.Normalized)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			s.rejected(ctx, reasonUnknownSignup)
			return accepted(), nil
		}
		return nil, err
	}

	if signup.Attempts >= maxSendAttempts {
		s.rejected(ctx, reasonSendLimit)
		return accepted(), nil
	}

	token, tokenHash, err := tokenutils.New()
	if err != nil {
		return nil, err
	}

	meta := metaFrom(ctx)
	touched, err := s.signups.Touch(ctx, &repositories.TouchCloudSignupRequest{
		ID:        signup.ID,
		TokenHash: tokenHash,
		ExpiresAt: timeutils.NowUnix() + int64(s.tokenTTL().Seconds()),
		ClientIP:  meta.ClientIP,
		UserAgent: meta.UserAgent,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return accepted(), nil
		}
		return nil, err
	}

	s.requested(ctx, "resend")
	s.sendVerification(ctx, touched, token)

	return accepted(), nil
}

type storeSignupParams struct {
	input        *signupInput
	passwordHash string
	tokenHash    string
	expiresAt    int64
}

func (s *Service) storeSignup(
	ctx context.Context,
	params *storeSignupParams,
) (*cloudsignup.CloudSignup, bool, error) {
	pending, err := s.signups.GetPendingByEmail(ctx, params.input.email.Normalized)
	switch {
	case err == nil:
		return s.refreshSignup(ctx, pending, params)
	case !errortypes.IsNotFoundError(err):
		return nil, false, err
	}

	meta := metaFrom(ctx)
	entity := &cloudsignup.CloudSignup{
		EmailAddress:    params.input.email.Lower,
		EmailNormalized: params.input.email.Normalized,
		Name:            params.input.name,
		CompanyName:     params.input.companyName,
		PasswordHash:    params.passwordHash,
		TokenHash:       params.tokenHash,
		Status:          cloudsignup.StatusPending,
		ClientIP:        stringutils.TruncateRunes(meta.ClientIP, cloudsignup.MaxClientIPLength),
		UserAgent:       stringutils.TruncateRunes(meta.UserAgent, cloudsignup.MaxUserAgentLength),
		Attempts:        1,
		ExpiresAt:       params.expiresAt,
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, false, multiErr
	}

	created, err := s.signups.Create(ctx, entity)
	if err == nil {
		return created, true, nil
	}
	if !dberror.IsUniqueConstraintViolation(err) {
		return nil, false, err
	}

	pending, err = s.signups.GetPendingByEmail(ctx, params.input.email.Normalized)
	if err != nil {
		return nil, false, err
	}

	return s.refreshSignup(ctx, pending, params)
}

func (s *Service) refreshSignup(
	ctx context.Context,
	pending *cloudsignup.CloudSignup,
	params *storeSignupParams,
) (*cloudsignup.CloudSignup, bool, error) {
	if pending.Attempts >= maxSendAttempts {
		return pending, false, nil
	}

	meta := metaFrom(ctx)
	refreshed, err := s.signups.Refresh(ctx, &repositories.RefreshCloudSignupRequest{
		ID:           pending.ID,
		Name:         params.input.name,
		EmailAddress: params.input.email.Lower,
		CompanyName:  params.input.companyName,
		PasswordHash: params.passwordHash,
		TokenHash:    params.tokenHash,
		ExpiresAt:    params.expiresAt,
		ClientIP:     stringutils.TruncateRunes(meta.ClientIP, cloudsignup.MaxClientIPLength),
		UserAgent:    stringutils.TruncateRunes(meta.UserAgent, cloudsignup.MaxUserAgentLength),
	})
	if err != nil {
		return nil, false, err
	}

	return refreshed, true, nil
}

func (s *Service) verifyTurnstile(ctx context.Context, token, action string) error {
	if s.turnstile == nil || !s.turnstile.Enabled() {
		return nil
	}

	meta := metaFrom(ctx)
	err := s.turnstile.Verify(ctx, &services.TurnstileVerification{
		Token:          token,
		RemoteIP:       meta.ClientIP,
		ExpectedAction: action,
	})
	if err == nil {
		return nil
	}

	if errors.Is(err, services.ErrTurnstileUnavailable) {
		s.l.Error("turnstile verification is unavailable", zap.Error(err))
		s.rejected(ctx, reasonTurnstileDown)
		return errortypes.NewValidationError(
			"turnstileToken",
			errortypes.ErrInvalid,
			"The security check could not be verified right now. Try again in a moment",
		)
	}

	s.rejected(ctx, reasonTurnstile)
	return errortypes.NewValidationError(
		"turnstileToken",
		errortypes.ErrInvalid,
		"The security check failed or expired. Complete it again",
	)
}

func (s *Service) findUser(ctx context.Context, emailAddress string) (*userSummary, error) {
	user, err := s.users.FindByEmail(ctx, emailAddress)
	if err != nil {
		if isUserNotFound(err) {
			return nil, nil //nolint:nilnil // no user holds the address
		}
		return nil, err
	}
	if user == nil {
		return nil, nil //nolint:nilnil // no user holds the address
	}

	return &userSummary{Name: user.Name}, nil
}

type userSummary struct {
	Name string
}

func isUserNotFound(err error) bool {
	if errortypes.IsNotFoundError(err) {
		return true
	}

	var validationErr *errortypes.Error
	return errors.As(err, &validationErr) && validationErr.Code == errortypes.ErrNotFound
}

func (s *Service) sendVerification(
	ctx context.Context,
	signup *cloudsignup.CloudSignup,
	token string,
) {
	if err := s.email.SendSignupVerification(ctx, &services.SignupVerificationEmail{
		To:          signup.EmailAddress,
		Name:        signup.Name,
		CompanyName: signup.CompanyName,
		Token:       token,
		ExpiresAt:   signup.ExpiresAt,
	}); err != nil {
		s.l.Error("failed to send the signup verification email",
			zap.String("signupId", signup.ID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) sendExistingAccount(ctx context.Context, emailAddress, name string) {
	if err := s.email.SendSignupExistingAccount(ctx, &services.SignupExistingAccountEmail{
		To:   emailAddress,
		Name: name,
	}); err != nil {
		s.l.Error("failed to send the existing account notice", zap.Error(err))
	}
}

func (s *Service) tokenTTL() time.Duration {
	return s.platform.Cloud.Signup.GetVerificationTokenTTL()
}

func metaFrom(ctx context.Context) requestmeta.Meta {
	meta, _ := requestmeta.From(ctx)
	return meta
}
