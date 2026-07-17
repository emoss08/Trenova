package permtest

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

var _ services.PermissionEngine = (*Engine)(nil)

type Grant struct {
	Operations       []permission.Operation
	DataScope        permission.DataScope
	MaxSensitivity   permission.FieldSensitivity
	AccessibleFields []string
}

// Engine is a deterministic PermissionEngine for tests: every resource
// resolves to DefaultGrant unless overridden per resource key.
type Engine struct {
	DefaultGrant Grant
	Overrides    map[string]Grant
}

func AllowAll() *Engine {
	return &Engine{
		DefaultGrant: Grant{
			Operations: []permission.Operation{
				permission.OpRead,
				permission.OpExport,
			},
			DataScope:      permission.DataScopeOrganization,
			MaxSensitivity: permission.SensitivityConfidential,
		},
		Overrides: make(map[string]Grant),
	}
}

func (e *Engine) GetResourcePermissions(
	_ context.Context,
	_, _ pulid.ID,
	resource string,
) (*services.ResourcePermissionDetail, error) {
	grant := e.DefaultGrant
	if override, ok := e.Overrides[resource]; ok {
		grant = override
	}
	return &services.ResourcePermissionDetail{
		Resource:         resource,
		Operations:       grant.Operations,
		DataScope:        grant.DataScope,
		MaxSensitivity:   grant.MaxSensitivity,
		AccessibleFields: grant.AccessibleFields,
	}, nil
}

func (e *Engine) Check(
	context.Context,
	*services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	return &services.PermissionCheckResult{Allowed: true}, nil
}

func (e *Engine) CheckBatch(
	context.Context,
	*services.BatchPermissionCheckRequest,
) (*services.BatchPermissionCheckResult, error) {
	return &services.BatchPermissionCheckResult{}, nil
}

func (e *Engine) GetLightManifest(
	context.Context,
	pulid.ID,
	pulid.ID,
) (*services.LightPermissionManifest, error) {
	return &services.LightPermissionManifest{}, nil
}

func (e *Engine) InvalidateUser(context.Context, pulid.ID, pulid.ID) error { return nil }

func (e *Engine) GetEffectivePermissions(
	context.Context,
	pulid.ID,
	pulid.ID,
) (*services.EffectivePermissions, error) {
	return &services.EffectivePermissions{}, nil
}

func (e *Engine) SimulatePermissions(
	context.Context,
	*services.SimulatePermissionsRequest,
) (*services.EffectivePermissions, error) {
	return &services.EffectivePermissions{}, nil
}
