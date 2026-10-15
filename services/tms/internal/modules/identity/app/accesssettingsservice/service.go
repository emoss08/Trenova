package accesssettingsservice

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const policyTTL = 30 * time.Second

type Params struct {
	fx.In

	Logger   *zap.Logger
	Repo     iamports.AccessSettingsRepository
	RBAC     permissionports.RBACRepository
	Auditor  permissionports.SecurityAuditor
	Notifier permissionports.PermissionChangeNotifier `optional:"true"`
}

type cachedPolicy struct {
	required  bool
	expiresAt time.Time
}

type Service struct {
	l        *zap.Logger
	repo     iamports.AccessSettingsRepository
	rbac     permissionports.RBACRepository
	auditor  permissionports.SecurityAuditor
	notifier permissionports.PermissionChangeNotifier
	now      func() time.Time

	mu       sync.RWMutex
	policies map[pagination.TenantInfo]cachedPolicy
}

func New(p Params) *Service {
	return &Service{
		l:        p.Logger.Named("service.access-settings"),
		repo:     p.Repo,
		rbac:     p.RBAC,
		auditor:  p.Auditor,
		notifier: p.Notifier,
		now:      time.Now,
		policies: make(map[pagination.TenantInfo]cachedPolicy),
	}
}

func (s *Service) Get(ctx context.Context, tenant pagination.TenantInfo) (*iam.AccessSettings, error) {
	return s.repo.Get(ctx, tenantKey(tenant))
}

func (s *Service) Update(
	ctx context.Context,
	req *iamports.UpdateAccessSettingsRequest,
) (*iam.AccessSettings, error) {
	tenant := tenantKey(req.TenantInfo)
	before, err := s.repo.Get(ctx, tenant)
	if err != nil {
		return nil, err
	}

	if !req.RequireRoleActivation {
		dynamic, dErr := s.rbac.ListRoleConstraints(ctx, permissionports.ListRoleConstraintsRequest{
			OrganizationID: tenant.OrgID,
			Type:           permission.RoleConstraintTypeDSD,
		})
		if dErr != nil {
			return nil, fmt.Errorf("read dynamic separation of duty constraints: %w", dErr)
		}
		if len(dynamic) > 0 {
			return nil, errortypes.NewValidationError(
				"requireRoleActivation",
				errortypes.ErrInvalid,
				"Role activation enforces your dynamic separation of duty constraints. Remove them before turning it off.",
			)
		}
	}

	next := &iam.AccessSettings{
		OrganizationID:        tenant.OrgID,
		BusinessUnitID:        tenant.BuID,
		RequireRoleActivation: req.RequireRoleActivation,
		UpdatedByID:           req.Actor.UserIDOrNil(),
		Version:               req.Version,
		CreatedAt:             before.CreatedAt,
	}
	saved, err := s.repo.Save(ctx, next)
	if err != nil {
		return nil, err
	}

	s.forget(tenant)
	if before.RequireRoleActivation != saved.RequireRoleActivation {
		if s.notifier != nil {
			s.notifier.OrganizationChanged(ctx, req.TenantInfo)
		}
		s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
			Resource:       permission.ResourceAccessPolicy,
			ResourceID:     tenant.OrgID.String(),
			Operation:      permission.OpUpdate,
			Actor:          req.Actor.AuditActorOrSystem(),
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
			Before:         before,
			After:          saved,
			Comment:        "Role activation setting changed",
		})
	}

	return saved, nil
}

func (s *Service) RoleActivationRequired(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (bool, error) {
	key := tenantKey(tenant)
	now := s.now()

	s.mu.RLock()
	cached, ok := s.policies[key]
	s.mu.RUnlock()
	if ok && now.Before(cached.expiresAt) {
		return cached.required, nil
	}

	settings, err := s.repo.Get(dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: key.OrgID,
		BusinessUnitID: key.BuID,
	}), key)
	if err != nil {
		return false, err
	}

	s.mu.Lock()
	s.policies[key] = cachedPolicy{
		required:  settings.RequireRoleActivation,
		expiresAt: now.Add(policyTTL),
	}
	s.mu.Unlock()

	return settings.RequireRoleActivation, nil
}

func (s *Service) forget(tenant pagination.TenantInfo) {
	s.mu.Lock()
	delete(s.policies, tenant)
	s.mu.Unlock()
}

func tenantKey(tenant pagination.TenantInfo) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: tenant.OrgID, BuID: tenant.BuID}
}
