package passwordresetservice

import (
	"context"
	"errors"
	"html/template"
	"net/url"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
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
	"This link is no longer valid. Request a new one.",
)

type Params struct {
	fx.In

	UserRepository   tenantports.UserRepository
	TokenRepository  tenantports.PasswordResetTokenRepository
	SessionRepo      sessionports.SessionRepository
	OrganizationRepo tenantports.OrganizationRepository
	EmailService     emailports.EmailService
	Templates        documenttemplateports.DocumentTemplateResolver
	AuditService     auditports.AuditService
	AuthEvents       iamports.AuthEventRecorder
	Plans            platformplanports.PlanService           `optional:"true"`
	PlatformEmail    kernelports.PlatformEmailService        `optional:"true"`
	PlatformInvites  kernelports.PlatformAccountInviteSender `optional:"true"`
	Config           *config.Config
	Logger           *zap.Logger
}

type Service struct {
	ur           tenantports.UserRepository
	tokens       tenantports.PasswordResetTokenRepository
	sessions     sessionports.SessionRepository
	orgs         tenantports.OrganizationRepository
	emailService emailports.EmailService
	templates    documenttemplateports.DocumentTemplateResolver
	audit        auditports.AuditService
	authEvents   iamports.AuthEventRecorder
	plans        platformplanports.PlanService
	platform     kernelports.PlatformEmailService
	invites      kernelports.PlatformAccountInviteSender
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
		authEvents:   p.AuthEvents,
		plans:        p.Plans,
		platform:     p.PlatformEmail,
		invites:      p.PlatformInvites,
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
func (s *Service) RequestReset(ctx context.Context, emailAddress string) (err error) {
	address := strings.TrimSpace(emailAddress)
	if address == "" {
		return nil
	}

	event := resetEvent{provider: iamports.AuthEventProviderPasswordResetRequest}
	defer func() { s.recordResetEvent(ctx, &event, err) }()

	log := s.l.With(zap.String("operation", "RequestReset"))

	user, err := s.ur.FindByEmail(ctx, address)
	if err != nil || user == nil {
		// Not found is the common case and is not an error worth surfacing; a real
		// failure is logged but still answered identically.
		log.Debug("password reset requested for an unknown address")
		event.deny(iam.AuthEventOutcomeFailed, resetErrorUnknownAccount)
		return nil
	}
	event.user = user

	if !user.IsActive() || user.IsLocked {
		log.Info("password reset requested for an inactive or locked account",
			zap.String("userId", user.ID.String()))
		event.deny(iam.AuthEventOutcomeDenied, resetErrorAccountUnavailable)
		return nil
	}

	scoped := dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: user.CurrentOrganizationID,
		BusinessUnitID: user.BusinessUnitID,
		UserID:         user.ID,
	})

	allowed, err := s.withinRequestAllowance(scoped, user)
	if err != nil {
		return err
	}
	if !allowed {
		event.deny(iam.AuthEventOutcomeDenied, resetErrorRateLimited)
		return nil
	}

	// Only the newest link may work: somebody who asks twice because the first email
	// was slow should not be left with two live credentials in their mailbox.
	if err = s.issueToken(scoped, &linkRequest{
		user:    user,
		purpose: tenant.PasswordResetPurposeReset,
	}); err != nil {
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

	user, err := s.ur.GetByID(ctx, tenantports.GetUserByIDRequest{
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

	link := &linkRequest{user: user, purpose: tenant.PasswordResetPurposeReset}
	comment := "Password reset link sent by an administrator"
	if user.AwaitingAccountSetup() {
		link.purpose = tenant.PasswordResetPurposeInvite
		link.invitedBy = s.actorName(ctx, req.Actor)
		comment = "Account invitation resent by an administrator"
	}

	if err = s.issueToken(ctx, link); err != nil {
		log.Error("failed to issue an administrator reset link", zap.Error(err))
		return err
	}

	s.auditAdminLink(ctx, &adminLinkAudit{
		user:    user,
		actor:   req.Actor,
		purpose: link.purpose,
		comment: comment,
	})

	return nil
}

type InviteRequest struct {
	User  *tenant.User
	Actor pagination.TenantInfo
}

func (s *Service) SendInvite(ctx context.Context, req InviteRequest) error {
	if req.User == nil {
		return errortypes.NewBusinessError("There is no account to invite.")
	}

	if !req.User.IsActive() {
		return errortypes.NewBusinessError(
			"This account is not active, so an invitation would be unusable.",
		)
	}

	if err := s.issueToken(ctx, &linkRequest{
		user:      req.User,
		purpose:   tenant.PasswordResetPurposeInvite,
		invitedBy: s.actorName(ctx, req.Actor),
	}); err != nil {
		s.l.Error("failed to issue an account invitation",
			zap.String("userId", req.User.ID.String()),
			zap.Error(err))
		return err
	}

	return nil
}

type adminLinkAudit struct {
	user    *tenant.User
	actor   pagination.TenantInfo
	purpose tenant.PasswordResetPurpose
	comment string
}

func (s *Service) auditAdminLink(ctx context.Context, entry *adminLinkAudit) {
	if s.audit == nil {
		return
	}

	if err := s.audit.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceUser,
		ResourceID:     entry.user.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         entry.actor.UserID,
		OrganizationID: entry.actor.OrgID,
		BusinessUnitID: entry.actor.BuID,
		Critical:       true,
	},
		auditservice.WithComment(entry.comment),
		auditservice.WithCategory(audit.CategoryUser),
		auditservice.WithMetadata(map[string]any{"linkPurpose": string(entry.purpose)}),
		auditservice.WithRequest(ctx),
	); err != nil {
		s.l.Error("failed to audit the administrator link",
			zap.String("userId", entry.user.ID.String()),
			zap.Error(err))
	}
}

func (s *Service) actorName(ctx context.Context, actor pagination.TenantInfo) string {
	if actor.UserID.IsNil() {
		return ""
	}

	user, err := s.ur.GetByID(ctx, tenantports.GetUserByIDRequest{TenantInfo: actor})
	if err != nil || user == nil {
		return ""
	}

	return user.Name
}

// issueToken retires whatever was outstanding, stores a fresh digest and mails the
// link. Shared by both entry points so the two cannot drift on expiry, on the
// one-live-link rule, or on what actually reaches the mailbox.
type linkRequest struct {
	user      *tenant.User
	purpose   tenant.PasswordResetPurpose
	invitedBy string
}

func (s *Service) issueToken(ctx context.Context, link *linkRequest) error {
	token, tokenHash, err := tokenutils.New()
	if err != nil {
		return err
	}

	now := timeutils.NowUnix()
	expiresAt := now + int64(s.linkTTL(link.purpose).Seconds())

	if err = s.tokens.InvalidateOutstanding(ctx, link.user.ID, now); err != nil {
		return err
	}

	if err = s.tokens.Create(ctx, &tenant.PasswordResetToken{
		UserID:         link.user.ID,
		BusinessUnitID: link.user.BusinessUnitID,
		OrganizationID: link.user.CurrentOrganizationID,
		TokenHash:      tokenHash,
		Purpose:        link.purpose,
		ExpiresAt:      expiresAt,
	}); err != nil {
		return err
	}

	if link.purpose.IsInvite() {
		s.sendInviteEmail(ctx, link, token, expiresAt)
		return nil
	}

	s.sendResetEmail(ctx, link.user, token, expiresAt)
	return nil
}

func (s *Service) linkTTL(purpose tenant.PasswordResetPurpose) time.Duration {
	if purpose.IsInvite() {
		return s.cfg.Security.PasswordReset.GetInviteTokenTTL()
	}
	return s.cfg.Security.PasswordReset.GetTokenTTL()
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
func (s *Service) ResetPassword(ctx context.Context, rawToken, newPassword string) (err error) {
	if strings.TrimSpace(rawToken) == "" {
		return errInvalidToken
	}

	if err = tenant.CheckNewPassword(resetPasswordField, newPassword, ""); err != nil {
		return err
	}

	event := resetEvent{provider: iamports.AuthEventProviderPasswordResetConfirm}
	defer func() { s.recordResetEvent(ctx, &event, err) }()

	now := timeutils.NowUnix()
	token, err := s.tokens.FindRedeemableByHash(ctx, tokenutils.Hash(rawToken), now)
	if err != nil {
		event.deny(iam.AuthEventOutcomeFailed, resetErrorInvalidToken)
		return errInvalidToken
	}

	// The repository loads the owning user alongside the token; a row that somehow
	// arrives without one cannot be acted on.
	user := token.User
	if user == nil {
		s.l.Error("reset token has no user", zap.String("tokenId", token.ID.String()))
		event.deny(iam.AuthEventOutcomeFailed, resetErrorInvalidToken)
		return errInvalidToken
	}
	event.user = user
	event.organizationID = token.OrganizationID
	event.businessUnitID = token.BusinessUnitID
	if token.Purpose.IsInvite() {
		event.provider = iamports.AuthEventProviderInviteAccept
	}

	if !user.IsActive() {
		event.deny(iam.AuthEventOutcomeDenied, resetErrorAccountUnavailable)
		return errInvalidToken
	}

	if err = tenant.CheckNewPassword(
		resetPasswordField,
		newPassword,
		user.EmailAddress,
	); err != nil {
		event.deny(iam.AuthEventOutcomeFailed, resetErrorWeakPassword)
		return err
	}

	scoped := dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: token.OrganizationID,
		BusinessUnitID: token.BusinessUnitID,
		UserID:         user.ID,
	})

	// Redeem before writing the password. Two tabs opened from the same email both
	// reach here; the guard lives in the UPDATE, so exactly one of them proceeds.
	redeemed, err := s.tokens.MarkUsed(scoped, token.ID, now)
	if err != nil {
		return err
	}
	if !redeemed {
		event.deny(iam.AuthEventOutcomeFailed, resetErrorInvalidToken)
		return errInvalidToken
	}

	hashed, err := user.GeneratePassword(newPassword)
	if err != nil {
		s.l.Error("failed to hash the new password", zap.Error(err))
		return err
	}

	if err = s.ur.UpdatePassword(scoped, tenantports.UpdateUserPasswordRequest{
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

	if err = s.tokens.InvalidateOutstanding(scoped, user.ID, now); err != nil {
		// The redeemed token is already spent; a stale sibling left live is worth
		// logging but not worth failing a completed reset over.
		s.l.Error("failed to retire sibling reset tokens", zap.Error(err))
	}

	// Whoever was holding a session on this account is not necessarily the person who
	// just proved they own the mailbox. A reset that leaves those sessions alive does
	// not actually take the account back.
	if err = s.sessions.DeleteAllForUser(scoped, user.ID); err != nil {
		s.l.Error("failed to end existing sessions after a password reset",
			zap.String("userId", user.ID.String()), zap.Error(err))
	}

	s.l.Info("password reset completed", zap.String("userId", user.ID.String()))
	return nil
}

const resetPasswordField = "newPassword"

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

	tenantInfo := pagination.TenantInfo{
		UserID: user.ID,
		OrgID:  user.CurrentOrganizationID,
		BuID:   user.BusinessUnitID,
	}

	resetURL, err := s.linkURL(rawToken, tenant.PasswordResetPurposeReset)
	if err != nil {
		log.Error("password reset email cannot be sent", zap.Error(err))
		return
	}

	ttl := s.cfg.Security.PasswordReset.GetTokenTTL()
	idempotencyKey := "password-reset-" + tokenutils.Hash(rawToken)

	if s.usesPlatformSender(ctx, tenantInfo) {
		if err = s.platform.SendPasswordReset(ctx, &kernelports.PasswordResetEmail{
			To:               user.EmailAddress,
			Locale:           i18n.Locale(user.Locale),
			Name:             user.Name,
			CompanyName:      s.companyName(ctx, user),
			ResetURL:         resetURL,
			ExpiresInMinutes: int(ttl.Minutes()),
			ExpiresAt:        expiresAt,
			Timezone:         user.Timezone,
			IdempotencyKey:   idempotencyKey,
		}); err != nil {
			log.Error("failed to send the password reset email through the platform sender",
				zap.Error(err))
		}
		return
	}

	if s.templates == nil || s.emailService == nil {
		log.Error("password reset email cannot be sent: templating or email is not configured")
		return
	}

	rendered, renderErr := s.templates.RenderMessage(
		ctx,
		&documenttemplateports.RenderMessageRequest{
			TenantInfo: tenantInfo,
			Kind:       documenttemplate.KindPasswordResetEmail,
			Data: documenttemplate.PasswordResetContext{
				FirstName:   stringutils.FirstName(user.Name),
				FullName:    user.Name,
				CompanyName: s.companyName(ctx, user),
				ResetURL: template.URL(
					resetURL,
				), //nolint:gosec // built from configured base URL + generated token
				ExpiresInMinutes: int(ttl.Minutes()),
				ExpiresAt:        timeutils.FormatStampIn(expiresAt, user.Timezone),
			},
			ReferenceID:       user.ID,
			Locale:            i18n.Locale(user.Locale),
			FallbackToBuiltIn: true,
		},
	)
	if renderErr != nil {
		log.Error("failed to render the password reset email", zap.Error(renderErr))
		return
	}

	if _, err = s.emailService.Send(ctx, &emailports.SendEmailRequest{
		TenantInfo: tenantInfo,
		Purpose:    email.PurposeGeneral,
		To:         []string{user.EmailAddress},
		Subject:    rendered.Subject,
		HTML:       rendered.HTML,
		Text:       rendered.Text,
		// Keyed on the token hash so a retry of the same send is deduplicated while two
		// genuinely separate requests each go out.
		IdempotencyKey: idempotencyKey,
	}); err != nil {
		log.Error("failed to send the password reset email", zap.Error(err))
	}
}

func (s *Service) usesPlatformSender(ctx context.Context, tenantInfo pagination.TenantInfo) bool {
	if s.plans == nil || s.platform == nil || !s.plans.EnforcesPlans() {
		return false
	}

	resolved, err := s.plans.Resolve(ctx, tenantInfo.OrgID, tenantInfo.BuID)
	if err != nil {
		s.l.Warn("could not resolve the plan for a password reset; using the tenant sender",
			zap.String("organizationId", tenantInfo.OrgID.String()),
			zap.Error(err))
		return false
	}

	return resolved.IsManaged()
}

var errNoResetBaseURL = errors.New(
	"security.passwordReset.baseUrl is not set, so a reset link cannot be built",
)

// linkURL refuses to guess. Deriving the origin from the request's Host header is how
// reset links end up pointing at an attacker's domain, so an unconfigured deployment
// gets a logged error instead of a link.
func (s *Service) linkURL(rawToken string, purpose tenant.PasswordResetPurpose) (string, error) {
	base := s.cfg.Security.PasswordReset.GetBaseURL()
	if base == "" {
		return "", errNoResetBaseURL
	}

	query := url.Values{"token": []string{rawToken}}
	if purpose.IsInvite() {
		query.Set("purpose", string(purpose))
	}

	return base + "/auth/reset?" + query.Encode(), nil
}

func (s *Service) sendInviteEmail(
	ctx context.Context,
	link *linkRequest,
	rawToken string,
	expiresAt int64,
) {
	user := link.user
	log := s.l.With(zap.String("userId", user.ID.String()))

	tenantInfo := pagination.TenantInfo{
		UserID: user.ID,
		OrgID:  user.CurrentOrganizationID,
		BuID:   user.BusinessUnitID,
	}

	setupURL, err := s.linkURL(rawToken, link.purpose)
	if err != nil {
		log.Error("account invitation email cannot be sent", zap.Error(err))
		return
	}

	hours := int(s.linkTTL(link.purpose).Hours())
	idempotencyKey := "account-invite-" + tokenutils.Hash(rawToken)

	if s.invites != nil && s.usesPlatformSender(ctx, tenantInfo) {
		if err = s.invites.SendAccountInvite(ctx, &kernelports.AccountInviteEmail{
			To:             user.EmailAddress,
			Locale:         i18n.Locale(user.Locale),
			Name:           user.Name,
			Username:       user.Username,
			CompanyName:    s.companyName(ctx, user),
			InvitedByName:  link.invitedBy,
			SetupURL:       setupURL,
			ExpiresInHours: hours,
			ExpiresAt:      expiresAt,
			Timezone:       user.Timezone,
			IdempotencyKey: idempotencyKey,
		}); err != nil {
			log.Error("failed to send the account invitation through the platform sender",
				zap.Error(err))
		}
		return
	}

	if s.templates == nil || s.emailService == nil {
		log.Error("account invitation email cannot be sent: templating or email is not configured")
		return
	}

	rendered, renderErr := s.templates.RenderMessage(
		ctx,
		&documenttemplateports.RenderMessageRequest{
			TenantInfo: tenantInfo,
			Kind:       documenttemplate.KindAccountInviteEmail,
			Data: documenttemplate.AccountInviteContext{
				FirstName:     stringutils.FirstName(user.Name),
				FullName:      user.Name,
				Username:      user.Username,
				EmailAddress:  user.EmailAddress,
				InvitedByName: link.invitedBy,
				CompanyName:   s.companyName(ctx, user),
				SetupURL: template.URL( //nolint:gosec // built from configured base URL + generated token
					setupURL,
				),
				ExpiresInDays:  wholeDays(hours),
				ExpiresInHours: hours,
				ExpiresAt:      timeutils.FormatStampIn(expiresAt, user.Timezone),
			},
			ReferenceID:       user.ID,
			Locale:            i18n.Locale(user.Locale),
			FallbackToBuiltIn: true,
		},
	)
	if renderErr != nil {
		log.Error("failed to render the account invitation email", zap.Error(renderErr))
		return
	}

	if _, err = s.emailService.Send(ctx, &emailports.SendEmailRequest{
		TenantInfo:     tenantInfo,
		Purpose:        email.PurposeGeneral,
		To:             []string{user.EmailAddress},
		Subject:        rendered.Subject,
		HTML:           rendered.HTML,
		Text:           rendered.Text,
		IdempotencyKey: idempotencyKey,
	}); err != nil {
		log.Error("failed to send the account invitation email", zap.Error(err))
	}
}

func wholeDays(hours int) int {
	if hours <= 0 || hours%24 != 0 {
		return 0
	}
	return hours / 24
}

func (s *Service) companyName(ctx context.Context, user *tenant.User) string {
	org, err := s.orgs.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
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
