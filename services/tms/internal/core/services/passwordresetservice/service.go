package passwordresetservice

import (
	"context"
	"errors"
	"html/template"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// errInvalidToken is the single answer to every way a link can fail: unknown digest,
// already redeemed, superseded by a newer request, or expired. Telling those apart
// would let somebody probe which links once existed and how far back.
var errInvalidToken = errortypes.NewValidationError(
	"token",
	errortypes.ErrInvalid,
	"This password reset link is no longer valid. Request a new one.",
)

type Params struct {
	fx.In

	UserRepository   repositories.UserRepository
	TokenRepository  repositories.PasswordResetTokenRepository
	SessionRepo      repositories.SessionRepository
	OrganizationRepo repositories.OrganizationRepository
	EmailService     serviceports.EmailService
	Templates        serviceports.DocumentTemplateResolver
	AuditService     serviceports.AuditService
	Config           *config.Config
	Logger           *zap.Logger
}

type Service struct {
	ur           repositories.UserRepository
	tokens       repositories.PasswordResetTokenRepository
	sessions     repositories.SessionRepository
	orgs         repositories.OrganizationRepository
	emailService serviceports.EmailService
	templates    serviceports.DocumentTemplateResolver
	audit        serviceports.AuditService
	cfg          *config.Config
	l            *zap.Logger
}

func New(p Params) *Service {
	return &Service{
		ur:           p.UserRepository,
		tokens:       p.TokenRepository,
		sessions:     p.SessionRepo,
		orgs:         p.OrganizationRepo,
		emailService: p.EmailService,
		templates:    p.Templates,
		audit:        p.AuditService,
		cfg:          p.Config,
		l:            p.Logger.Named("service.password-reset"),
	}
}

// RequestReset mints a reset link and emails it.
//
// It returns nil for an address that belongs to nobody, to a deactivated account, or
// to an account that has already been sent its hourly allowance of links. The caller
// answers the same way in every case: an endpoint that distinguishes them is an
// account-existence oracle for anyone with a word list.
//
// Nothing about the account changes here. The password on file keeps working until
// somebody who can read that mailbox redeems the token, which is what stops a stranger
// from locking a user out by typing their address.
func (s *Service) RequestReset(ctx context.Context, emailAddress string) error {
	address := strings.TrimSpace(emailAddress)
	if address == "" {
		return nil
	}

	log := s.l.With(zap.String("operation", "RequestReset"))

	user, err := s.ur.FindByEmail(ctx, address)
	if err != nil || user == nil {
		// Not found is the common case and is not an error worth surfacing; a real
		// failure is logged but still answered identically.
		log.Debug("password reset requested for an unknown address")
		return nil
	}

	if !user.IsActive() || user.IsLocked {
		log.Info("password reset requested for an inactive or locked account",
			zap.String("userId", user.ID.String()))
		return nil
	}

	allowed, err := s.withinRequestAllowance(ctx, user)
	if err != nil {
		return err
	}
	if !allowed {
		return nil
	}

	// Only the newest link may work: somebody who asks twice because the first email
	// was slow should not be left with two live credentials in their mailbox.
	if err = s.issueToken(ctx, user); err != nil {
		log.Error("failed to issue a reset link", zap.Error(err))
		return err
	}

	return nil
}

// AdminResetRequest is an administrator sending a reset link to somebody else.
type AdminResetRequest struct {
	// TargetUserID is whose password is being reset.
	TargetUserID pulid.ID
	// Actor is the administrator making the request, and the tenant the lookup is
	// scoped to. A user outside it will not be found.
	Actor pagination.TenantInfo
}

// RequestResetForUser sends a reset link on an administrator's behalf.
//
// It differs from the self-service path in three deliberate ways, all of which follow
// from the caller already being authenticated and permission-checked:
//
//   - Failures are reported. There is no account-existence oracle to protect: an admin
//     who mistypes an id should be told, not left watching a button succeed.
//   - The hourly per-account cap does not apply. That cap exists to stop an anonymous
//     stranger flooding an inbox; the usual reason an admin is clicking this is that
//     the user already burned their own allowance and is stuck.
//   - The action is audited against the administrator who took it.
//
// What does not differ is the important part: this still only mints a link. It does not
// set a password, so an administrator cannot lock somebody out by clicking it, and
// cannot learn or choose the password that results.
func (s *Service) RequestResetForUser(ctx context.Context, req AdminResetRequest) error {
	log := s.l.With(
		zap.String("operation", "RequestResetForUser"),
		zap.String("targetUserId", req.TargetUserID.String()),
	)

	user, err := s.ur.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo:   req.Actor,
		LookupUserID: req.TargetUserID,
	})
	if err != nil {
		return err
	}

	if !user.IsActive() {
		return errortypes.NewBusinessError(
			"This account is not active, so a reset link would be unusable.",
		)
	}

	if err = s.issueToken(ctx, user); err != nil {
		log.Error("failed to issue an administrator reset link", zap.Error(err))
		return err
	}

	if s.audit != nil {
		if auditErr := s.audit.LogAction(&serviceports.LogActionParams{
			Resource:       permission.ResourceUser,
			ResourceID:     user.ID.String(),
			Operation:      permission.OpUpdate,
			UserID:         req.Actor.UserID,
			OrganizationID: req.Actor.OrgID,
			BusinessUnitID: req.Actor.BuID,
			Critical:       true,
		}, auditservice.WithComment("Password reset link sent by an administrator")); auditErr != nil {
			// The link is already out; failing the request now would tell the admin it
			// did not happen when it did.
			log.Error("failed to audit the administrator reset", zap.Error(auditErr))
		}
	}

	return nil
}

// issueToken retires whatever was outstanding, stores a fresh digest and mails the
// link. Shared by both entry points so the two cannot drift on expiry, on the
// one-live-link rule, or on what actually reaches the mailbox.
func (s *Service) issueToken(ctx context.Context, user *tenant.User) error {
	token, tokenHash, err := tokenutils.New()
	if err != nil {
		return err
	}

	now := timeutils.NowUnix()
	expiresAt := now + int64(s.cfg.Security.PasswordReset.GetTokenTTL().Seconds())

	if err = s.tokens.InvalidateOutstanding(ctx, user.ID, now); err != nil {
		return err
	}

	if err = s.tokens.Create(ctx, &tenant.PasswordResetToken{
		UserID:         user.ID,
		BusinessUnitID: user.BusinessUnitID,
		OrganizationID: user.CurrentOrganizationID,
		TokenHash:      tokenHash,
		ExpiresAt:      expiresAt,
	}); err != nil {
		return err
	}

	s.sendResetEmail(ctx, user, token, expiresAt)
	return nil
}

func (s *Service) withinRequestAllowance(
	ctx context.Context,
	user *tenant.User,
) (bool, error) {
	since := timeutils.NowUnix() - int64(time.Hour.Seconds())
	issued, err := s.tokens.CountSince(ctx, user.ID, since)
	if err != nil {
		s.l.Error("failed to count recent reset requests", zap.Error(err))
		return false, err
	}

	if issued >= s.cfg.Security.PasswordReset.GetMaxRequestsPerHour() {
		s.l.Warn("password reset rate limit reached",
			zap.String("userId", user.ID.String()),
			zap.Int("issued", issued))
		return false, nil
	}

	return true, nil
}

// ResetPassword redeems a link and sets the new password.
func (s *Service) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	if strings.TrimSpace(rawToken) == "" {
		return errInvalidToken
	}

	if err := validatePasswordStrength(newPassword); err != nil {
		return err
	}

	now := timeutils.NowUnix()
	token, err := s.tokens.FindRedeemableByHash(ctx, tokenutils.Hash(rawToken), now)
	if err != nil {
		return errInvalidToken
	}

	// The repository loads the owning user alongside the token; a row that somehow
	// arrives without one cannot be acted on.
	user := token.User
	if user == nil {
		s.l.Error("reset token has no user", zap.String("tokenId", token.ID.String()))
		return errInvalidToken
	}

	if !user.IsActive() {
		return errInvalidToken
	}

	// Redeem before writing the password. Two tabs opened from the same email both
	// reach here; the guard lives in the UPDATE, so exactly one of them proceeds.
	redeemed, err := s.tokens.MarkUsed(ctx, token.ID, now)
	if err != nil {
		return err
	}
	if !redeemed {
		return errInvalidToken
	}

	hashed, err := user.GeneratePassword(newPassword)
	if err != nil {
		s.l.Error("failed to hash the new password", zap.Error(err))
		return err
	}

	if err = s.ur.UpdatePassword(ctx, repositories.UpdateUserPasswordRequest{
		UserID:         user.ID,
		OrganizationID: token.OrganizationID,
		BusinessUnitID: token.BusinessUnitID,
		Password:       hashed,
		// The user just chose this password, so any standing demand that they change
		// one is satisfied.
		MustChangePassword: false,
	}); err != nil {
		s.l.Error("failed to write the new password", zap.Error(err))
		return err
	}

	if err = s.tokens.InvalidateOutstanding(ctx, user.ID, now); err != nil {
		// The redeemed token is already spent; a stale sibling left live is worth
		// logging but not worth failing a completed reset over.
		s.l.Error("failed to retire sibling reset tokens", zap.Error(err))
	}

	// Whoever was holding a session on this account is not necessarily the person who
	// just proved they own the mailbox. A reset that leaves those sessions alive does
	// not actually take the account back.
	if err = s.sessions.DeleteAllForUser(ctx, user.ID); err != nil {
		s.l.Error("failed to end existing sessions after a password reset",
			zap.String("userId", user.ID.String()), zap.Error(err))
	}

	s.l.Info("password reset completed", zap.String("userId", user.ID.String()))
	return nil
}

const minPasswordLength = 8

func validatePasswordStrength(password string) error {
	if len(password) < minPasswordLength {
		return errortypes.NewValidationError(
			"newPassword",
			errortypes.ErrInvalid,
			"Password must be at least 8 characters",
		)
	}
	return nil
}

// sendResetEmail renders the message through the organization's template and mails it.
//
// A failure here is logged, never returned: the caller's response must not differ
// between an address that received mail and one that did not.
func (s *Service) sendResetEmail(
	ctx context.Context,
	user *tenant.User,
	rawToken string,
	expiresAt int64,
) {
	log := s.l.With(zap.String("userId", user.ID.String()))

	if s.templates == nil || s.emailService == nil {
		log.Error("password reset email cannot be sent: templating or email is not configured")
		return
	}

	tenantInfo := pagination.TenantInfo{
		UserID: user.ID,
		OrgID:  user.CurrentOrganizationID,
		BuID:   user.BusinessUnitID,
	}

	resetURL, err := s.resetURL(rawToken)
	if err != nil {
		log.Error("password reset email cannot be sent", zap.Error(err))
		return
	}

	ttl := s.cfg.Security.PasswordReset.GetTokenTTL()
	rendered, renderErr := s.templates.RenderMessage(ctx, &serviceports.RenderMessageRequest{
		TenantInfo: tenantInfo,
		Kind:       documenttemplate.KindPasswordResetEmail,
		Data: documenttemplate.PasswordResetContext{
			FirstName:        firstName(user.Name),
			FullName:         user.Name,
			CompanyName:      s.companyName(ctx, user),
			ResetURL:         template.URL(resetURL), //nolint:gosec // built from configured base URL + generated token
			ExpiresInMinutes: int(ttl.Minutes()),
			ExpiresAt:        timeutils.FormatStampIn(expiresAt, user.Timezone),
		},
		ReferenceID:       user.ID,
		FallbackToBuiltIn: true,
	})
	if renderErr != nil {
		log.Error("failed to render the password reset email", zap.Error(renderErr))
		return
	}

	if _, err = s.emailService.Send(ctx, &serviceports.SendEmailRequest{
		TenantInfo: tenantInfo,
		Purpose:    email.PurposeGeneral,
		To:         []string{user.EmailAddress},
		Subject:    rendered.Subject,
		HTML:       rendered.HTML,
		Text:       rendered.Text,
		// Keyed on the token hash so a retry of the same send is deduplicated while two
		// genuinely separate requests each go out.
		IdempotencyKey: "password-reset-" + tokenutils.Hash(rawToken),
	}); err != nil {
		log.Error("failed to send the password reset email", zap.Error(err))
	}
}

var errNoResetBaseURL = errors.New(
	"security.passwordReset.baseUrl is not set, so a reset link cannot be built",
)

// resetURL refuses to guess. Deriving the origin from the request's Host header is how
// reset links end up pointing at an attacker's domain, so an unconfigured deployment
// gets a logged error instead of a link.
func (s *Service) resetURL(rawToken string) (string, error) {
	base := s.cfg.Security.PasswordReset.GetBaseURL()
	if base == "" {
		return "", errNoResetBaseURL
	}
	return base + "/auth/reset?token=" + rawToken, nil
}

func (s *Service) companyName(ctx context.Context, user *tenant.User) string {
	org, err := s.orgs.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: pagination.TenantInfo{
			UserID: user.ID,
			OrgID:  user.CurrentOrganizationID,
			BuID:   user.BusinessUnitID,
		},
	})
	if err != nil || org == nil {
		return "Trenova"
	}
	return org.Name
}

func firstName(fullName string) string {
	name := strings.TrimSpace(fullName)
	if name == "" {
		return ""
	}
	if space := strings.IndexByte(name, ' '); space > 0 {
		return name[:space]
	}
	return name
}
