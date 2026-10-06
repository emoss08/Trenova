package supportaccess

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type PermissionSourceParams struct {
	fx.In

	Registry *permission.Registry
}

type PermissionSource struct {
	registry  *permission.Registry
	once      sync.Once
	readOnly  map[string]*repositories.CachedResourcePermission
	readWrite map[string]*repositories.CachedResourcePermission
}

var _ services.DelegatedPermissionSource = (*PermissionSource)(nil)

func NewPermissionSource(p PermissionSourceParams) *PermissionSource {
	return &PermissionSource{registry: p.Registry}
}

func (s *PermissionSource) DelegatedPermissions(
	ctx context.Context,
	userID, orgID pulid.ID,
) (*repositories.CachedPermissions, bool, error) {
	active, ok := ActiveFrom(ctx)
	if !ok || active.PrincipalUserID != userID || active.OrganizationID != orgID {
		return nil, false, nil
	}

	s.once.Do(s.build)

	resources := s.readOnly
	if active.WriteActive() {
		resources = s.readWrite
	}

	return &repositories.CachedPermissions{
		MaxSensitivity: string(permission.SensitivityConfidential),
		Resources:      resources,
		ExpiresAt:      active.ExpiresAt,
	}, true, nil
}

func (s *PermissionSource) build() {
	definitions := s.registry.All()
	s.readOnly = make(map[string]*repositories.CachedResourcePermission, len(definitions))
	s.readWrite = make(map[string]*repositories.CachedResourcePermission, len(definitions))

	for _, def := range definitions {
		readable := false
		all := make([]string, 0, len(def.Operations))
		for _, op := range def.Operations {
			all = append(all, string(op.Operation))
			if op.Operation == permission.OpRead {
				readable = true
			}
		}

		if readable {
			read := &repositories.CachedResourcePermission{
				Operations: []string{string(permission.OpRead)},
				DataScope:  string(permission.DataScopeOrganization),
			}
			s.readOnly[def.Resource] = read
			s.readWrite[def.Resource] = read
		}

		if IsDeniedResource(def.Resource) {
			continue
		}

		s.readWrite[def.Resource] = &repositories.CachedResourcePermission{
			Operations: all,
			DataScope:  string(permission.DataScopeOrganization),
		}
	}
}
