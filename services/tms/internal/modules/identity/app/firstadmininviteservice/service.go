package firstadmininviteservice

import (
	"context"
	"regexp"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/instancebootstrap"
	"github.com/emoss08/trenova/internal/modules/identity/domain/passwordreset"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/instancebootstrapports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	installScopeReason = "install the first administrator's account setup invite from the instance's operator, which has no tenant session"
	auditSource        = "first_administrator_invite"
	fieldTokenSHA256   = "tokenSha256"
	fieldExpiresAt     = "expiresAt"
)

var tokenSHA256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type bootstrapRecords interface {
	GetRecord(ctx context.Context) (*instancebootstrap.InstanceBootstrap, error)
}

type administrators interface {
	FindByIDForLogin(ctx context.Context, userID pulid.ID) (*tenant.User, error)
}

type inviteTokens interface {
	FindByUserAndHash(
		ctx context.Context,
		userID pulid.ID,
		tokenHash string,
	) (*passwordreset.PasswordResetToken, error)
	InvalidateOutstanding(ctx context.Context, userID pulid.ID, now int64) error
	Create(ctx context.Context, token *passwordreset.PasswordResetToken) error
}

type Params struct {
	fx.In

	DB             kernelports.Transactor
	Bootstraps     instancebootstrapports.InstanceBootstrapRepository
	Users          tenantports.UserRepository
	Tokens         tenantports.PasswordResetTokenRepository
	SecurityConfig *securityconfig.Config
	Auditor        permissionports.SecurityAuditor
	Logger         *zap.Logger
}

type Service struct {
	db         kernelports.Transactor
	bootstraps bootstrapRecords
	users      administrators
	tokens     inviteTokens
	inviteTTL  time.Duration
	auditor    permissionports.SecurityAuditor
	now        func() int64
	l          *zap.Logger
}

//nolint:gocritic // this is dependency injection
func New(p Params) *Service {
	return &Service{
		db:         p.DB,
		bootstraps: p.Bootstraps,
		users:      p.Users,
		tokens:     p.Tokens,
		inviteTTL:  p.SecurityConfig.PasswordReset.GetInviteTokenTTL(),
		auditor:    p.Auditor,
		now:        timeutils.NowUnix,
		l:          p.Logger.Named("service.first-administrator-invite"),
	}
}

var _ instancebootstrapports.FirstAdministratorInviteService = (*Service)(nil)

func (s *Service) Install(
	ctx context.Context,
	invite *instancebootstrapports.FirstAdministratorInvite,
) (*instancebootstrapports.FirstAdministratorInviteResult, error) {
	now := s.now()
	if err := validate(invite, now); err != nil {
		return nil, err
	}

	record, err := s.bootstraps.GetRecord(ctx)
	switch {
	case notFound(err):
		return nil, instancebootstrapports.ErrNotBootstrapped
	case err != nil:
		return nil, err
	}

	admin, err := s.users.FindByIDForLogin(ctx, record.AdminUserID)
	if err != nil {
		return nil, err
	}
	if !admin.AwaitingAccountSetup() {
		return &instancebootstrapports.FirstAdministratorInviteResult{SetUp: true}, nil
	}

	expiresAt := min(invite.ExpiresAt, now+int64(s.inviteTTL.Seconds()))
	var (
		result *instancebootstrapports.FirstAdministratorInviteResult
		issued bool
	)
	sysCtx := dbscope.WithSystem(ctx, installScopeReason)
	err = s.db.Transact(sysCtx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		existing, findErr := s.tokens.FindByUserAndHash(txCtx, admin.ID, invite.TokenSHA256)
		switch {
		case findErr == nil:
			result = answer(existing.IsRedeemable(now), invite.TokenSHA256)
			return nil
		case !notFound(findErr):
			return findErr
		}

		if invErr := s.tokens.InvalidateOutstanding(txCtx, admin.ID, now); invErr != nil {
			return invErr
		}
		if createErr := s.tokens.Create(txCtx, &passwordreset.PasswordResetToken{
			UserID:         admin.ID,
			BusinessUnitID: admin.BusinessUnitID,
			OrganizationID: admin.CurrentOrganizationID,
			TokenHash:      invite.TokenSHA256,
			Purpose:        passwordreset.PasswordResetPurposeInvite,
			ExpiresAt:      expiresAt,
		}); createErr != nil {
			return createErr
		}
		issued = true
		result = answer(true, invite.TokenSHA256)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if issued {
		s.audit(ctx, admin, expiresAt)
	}

	return result, nil
}

func notFound(err error) bool {
	return dberror.IsNotFoundError(err) || errortypes.IsNotFoundError(err)
}

func answer(live bool, tokenSHA256 string) *instancebootstrapports.FirstAdministratorInviteResult {
	if !live {
		return &instancebootstrapports.FirstAdministratorInviteResult{}
	}
	return &instancebootstrapports.FirstAdministratorInviteResult{InviteTokenSHA256: tokenSHA256}
}

func validate(invite *instancebootstrapports.FirstAdministratorInvite, now int64) error {
	multiErr := errortypes.NewMultiError()
	if invite == nil {
		multiErr.Add(fieldTokenSHA256, errortypes.ErrRequired, "An invite is required")
		return multiErr
	}
	if !tokenSHA256Pattern.MatchString(invite.TokenSHA256) {
		multiErr.Add(
			fieldTokenSHA256,
			errortypes.ErrInvalid,
			"tokenSha256 must be 64 lowercase hexadecimal characters",
		)
	}
	if invite.ExpiresAt <= now {
		multiErr.Add(fieldExpiresAt, errortypes.ErrInvalid, "expiresAt must be in the future")
	}
	return multiErr.Err()
}

func (s *Service) audit(ctx context.Context, admin *tenant.User, expiresAt int64) {
	s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceUser,
		ResourceID:     admin.ID.String(),
		Operation:      permission.OpUpdate,
		Actor:          permissionports.SystemAuditActor(),
		OrganizationID: admin.CurrentOrganizationID,
		BusinessUnitID: admin.BusinessUnitID,
		After: map[string]any{
			"userId":          admin.ID.String(),
			"accountSetupBy":  "invite",
			"inviteExpiresAt": expiresAt,
		},
		Comment:  "Trenova Cloud issued the first administrator a one-time account setup link",
		Metadata: map[string]any{"source": auditSource},
	})
	s.l.Info("installed the first administrator's account setup invite",
		zap.String("userId", admin.ID.String()),
		zap.Int64("expiresAt", expiresAt),
	)
}
