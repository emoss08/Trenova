package readscopeservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/databaseconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/approvaldelegation"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type TeamDirectory interface {
	ManagerIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		userID pulid.ID,
		scope approvaldelegation.ApprovalScope,
		asOf int64,
	) ([]pulid.ID, map[pulid.ID]pulid.ID, error)
}

type Params struct {
	fx.In

	Config   *config.Config
	DB       kernelports.Detacher
	Engine   permissionports.PermissionEngine
	Teams    TeamDirectory
	Registry *permissioncatalog.Registry
}

type Service struct {
	db                kernelports.Detacher
	engine            permissionports.PermissionEngine
	teams             TeamDirectory
	registry          *permissioncatalog.Registry
	kinds             []string
	databaseEnforcing bool
}

func New(p Params) *Service {
	return &Service{
		db:                p.DB,
		engine:            p.Engine,
		teams:             p.Teams,
		registry:          p.Registry,
		kinds:             p.Registry.DatabaseScopedResources(),
		databaseEnforcing: databaseEnforcesScopes(p.Config),
	}
}

func databaseEnforcesScopes(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	switch cfg.Database.RLS.Mode {
	case databaseconfig.RLSModeObserve, databaseconfig.RLSModeEnforce:
		return true
	default:
		return false
	}
}

func (s *Service) DatabaseEnforcing() bool {
	return s.databaseEnforcing
}

func (s *Service) Enforced(resource string, scope permission.DataScope) bool {
	return s.registry.ScopeEnforced(resource, scope, s.databaseEnforcing)
}

func (s *Service) EnforceableScopes(resource string) []permission.DataScope {
	return s.registry.EnforceableScopes(resource, s.databaseEnforcing)
}

func (s *Service) Resolver(principal *permissionports.PermissionCheckRequest) readscope.Resolver {
	return func(ctx context.Context) (readscope.Grants, error) {
		return s.Grants(ctx, principal)
	}
}

func (s *Service) Grants(
	ctx context.Context,
	principal *permissionports.PermissionCheckRequest,
) (readscope.Grants, error) {
	if principal == nil || len(s.kinds) == 0 || !s.databaseEnforcing {
		return readscope.Grants{}, nil
	}

	var grants readscope.Grants
	err := s.db.RunDetached(ctx, true, func(ctx context.Context) error {
		var resolveErr error
		grants, resolveErr = s.resolve(ctx, principal)
		return resolveErr
	})
	if err != nil {
		return nil, err
	}

	return grants, nil
}

func (s *Service) resolve(
	ctx context.Context,
	principal *permissionports.PermissionCheckRequest,
) (readscope.Grants, error) {
	checks := make([]permissionports.ResourceOperationCheck, 0, len(s.kinds))
	for _, kind := range s.kinds {
		checks = append(checks, permissionports.ResourceOperationCheck{
			Resource:  kind,
			Operation: permission.OpRead,
		})
	}

	results, err := s.engine.CheckBatch(ctx, &permissionports.BatchPermissionCheckRequest{
		PrincipalType:     principal.PrincipalType,
		PrincipalID:       principal.PrincipalID,
		UserID:            principal.UserID,
		APIKeyID:          principal.APIKeyID,
		BusinessUnitID:    principal.BusinessUnitID,
		OrganizationID:    principal.OrganizationID,
		Checks:            checks,
		ContextAttributes: principal.ContextAttributes,
	})
	if err != nil {
		return nil, fmt.Errorf("check read scopes: %w", err)
	}
	if len(results.Results) != len(checks) {
		return nil, fmt.Errorf("check read scopes: %d results for %d checks",
			len(results.Results), len(checks))
	}

	grants := make(readscope.Grants, len(checks))
	var team []pulid.ID
	teamResolved := false
	for i, result := range results.Results {
		kind := s.kinds[i]
		if !result.Allowed || !s.registry.EnforcedByDatabase(kind, result.DataScope) {
			continue
		}

		grant := readscope.Grant{Scope: result.DataScope.String()}
		if result.DataScope == permission.DataScopeTeam {
			if !teamResolved {
				team, err = s.team(ctx, principal)
				if err != nil {
					return nil, err
				}
				teamResolved = true
			}
			grant.Team = team
		}
		grants[kind] = grant
	}

	return grants, nil
}

func (s *Service) team(
	ctx context.Context,
	principal *permissionports.PermissionCheckRequest,
) ([]pulid.ID, error) {
	if principal.UserID.IsNil() {
		return nil, nil
	}

	ids, _, err := s.teams.ManagerIDs(
		ctx,
		pagination.TenantInfo{
			OrgID:  principal.OrganizationID,
			BuID:   principal.BusinessUnitID,
			UserID: principal.UserID,
		},
		principal.UserID,
		approvaldelegation.ApprovalScopeAll,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("resolve the team a read is limited to: %w", err)
	}
	return ids, nil
}
