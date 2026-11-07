package scimservice

import (
	"context"
	"crypto/subtle"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/app/directoryaccounts"
	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/roleservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/domain/scim"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/scimports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/scimproto"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const touchInterval = 60

var errUnauthorized = scimproto.Unauthorized("The bearer token is not valid for this directory")

type RoleAssigner interface {
	AssignRole(ctx context.Context, req roleservice.AssignRoleRequest) error
	UnassignRole(ctx context.Context, req roleservice.UnassignRoleRequest) error
	RequireCanGrantRole(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		roleID pulid.ID,
	) error
}

type Accounts interface {
	Create(ctx context.Context, req *directoryaccounts.CreateRequest) (*tenant.User, error)
	Update(
		ctx context.Context,
		req *directoryaccounts.UpdateRequest,
	) (*directoryaccounts.UpdateResult, error)
	Revoke(ctx context.Context, req *directoryaccounts.AccessRequest) error
	Restore(ctx context.Context, req *directoryaccounts.AccessRequest) error
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	Repo          scimports.Repository
	Users         tenantports.UserRepository
	Organizations tenantports.OrganizationRepository
	Accounts      Accounts
	Roles         RoleAssigner
	Plans         platformplanports.PlanService
	Requester     scimports.ReconcileRequester
	Notifier      permissionports.PermissionChangeNotifier
	Auditor       permissionports.SecurityAuditor
	DB            kernelports.Transactor
}

type Service struct {
	l             *zap.Logger
	repo          scimports.Repository
	users         tenantports.UserRepository
	organizations tenantports.OrganizationRepository
	accounts      Accounts
	roles         RoleAssigner
	plans         platformplanports.PlanService
	requester     scimports.ReconcileRequester
	notifier      permissionports.PermissionChangeNotifier
	auditor       permissionports.SecurityAuditor
	db            kernelports.Transactor
}

var (
	_ scimports.Provisioner     = (*Service)(nil)
	_ scimports.GrantReconciler = (*Service)(nil)
	_ scimports.DirectoryGrants = (*Service)(nil)
)

func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.scim"),
		repo:          p.Repo,
		users:         p.Users,
		organizations: p.Organizations,
		accounts:      p.Accounts,
		roles:         p.Roles,
		plans:         p.Plans,
		requester:     p.Requester,
		notifier:      p.Notifier,
		auditor:       p.Auditor,
		db:            p.DB,
	}
}

func (s *Service) Authenticate(
	ctx context.Context,
	req scimports.AuthenticateRequest,
) (*scimports.Directory, error) {
	token := strings.TrimSpace(req.Token)
	prefix, ok := scim.TokenLookupPrefix(token)
	if !ok {
		return nil, errUnauthorized
	}

	credential, err := s.repo.FindCredential(ctx, prefix)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errUnauthorized
		}
		return nil, err
	}

	if subtle.ConstantTimeCompare(
		[]byte(scim.HashToken(token)),
		[]byte(credential.TokenHash),
	) != 1 {
		return nil, errUnauthorized
	}

	now := timeutils.NowUnix()
	if !credentialAdmits(credential, req.TenantSlug, now) {
		return nil, errUnauthorized
	}

	dir := &scimports.Directory{
		ID:             credential.DirectoryID,
		OrganizationID: credential.OrganizationID,
		BusinessUnitID: credential.BusinessUnitID,
		TokenID:        credential.TokenID,
		TenantSlug:     credential.TenantSlug,
	}
	scoped := scopedTo(ctx, dir)

	if err = planservice.RequireCapability(
		scoped,
		s.plans,
		dir.TenantInfo(),
		platformplan.CapabilitySSO,
	); err != nil {
		if errortypes.IsPlanRestrictionError(err) {
			return nil, scimproto.Forbidden("The organization's plan does not include provisioning")
		}
		return nil, err
	}

	if credential.LastUsedAt < now-touchInterval {
		if touchErr := s.repo.TouchCredential(scoped, dir.TokenID, now); touchErr != nil {
			s.l.Warn("could not record a SCIM token's use",
				zap.String("tokenId", dir.TokenID.String()),
				zap.Error(touchErr))
		}
	}

	return dir, nil
}

func credentialAdmits(credential *scimports.Credential, tenantSlug string, now int64) bool {
	switch {
	case credential.Status != iam.SCIMTokenStatusActive:
		return false
	case credential.ExpiresAt > 0 && credential.ExpiresAt <= now:
		return false
	case !credential.DirectoryEnabled:
		return false
	default:
		return strings.EqualFold(strings.TrimSpace(tenantSlug), credential.TenantSlug)
	}
}

func scopedTo(ctx context.Context, dir *scimports.Directory) context.Context {
	return dbscope.WithTenant(ctx, dir.DBTenant())
}

func actorOf(dir *scimports.Directory) directoryaccounts.Actor {
	return directoryaccounts.Actor{
		DirectoryID:    dir.ID,
		TokenID:        dir.TokenID,
		OrganizationID: dir.OrganizationID,
		BusinessUnitID: dir.BusinessUnitID,
	}
}

func pageBounds(startIndex, count int) (start, size int) {
	start = max(startIndex, 1)
	size = min(max(count, 0), scimports.MaxPageSize)

	return start, size
}

func parseIDs(raw []string) []pulid.ID {
	seen := make(map[pulid.ID]struct{}, len(raw))
	ids := make([]pulid.ID, 0, len(raw))
	for _, value := range raw {
		id := pulid.ID(strings.TrimSpace(value))
		if id.IsNil() {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	return ids
}
