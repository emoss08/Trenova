package permtest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreach"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
)

var _ permissionports.PermissionEngine = (*Engine)(nil)

type Grant struct {
	Operations     []permission.Operation
	DataScope      permission.DataScope
	MaxSensitivity permission.FieldSensitivity
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
) (*permissionports.ResourcePermissionDetail, error) {
	grant := e.DefaultGrant
	if override, ok := e.Overrides[resource]; ok {
		grant = override
	}
	return &permissionports.ResourcePermissionDetail{
		Resource:       resource,
		Operations:     grant.Operations,
		DataScope:      grant.DataScope,
		MaxSensitivity: grant.MaxSensitivity,
	}, nil
}

func (e *Engine) Check(
	context.Context,
	*permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	return &permissionports.PermissionCheckResult{Allowed: true}, nil
}

func (e *Engine) CheckBatch(
	context.Context,
	*permissionports.BatchPermissionCheckRequest,
) (*permissionports.BatchPermissionCheckResult, error) {
	return &permissionports.BatchPermissionCheckResult{}, nil
}

func (e *Engine) GetLightManifest(
	context.Context,
	pulid.ID,
	pulid.ID,
) (*permissionports.LightPermissionManifest, error) {
	return &permissionports.LightPermissionManifest{}, nil
}

func (e *Engine) InvalidateUser(context.Context, pulid.ID, pulid.ID) error { return nil }

func (e *Engine) GetEffectivePermissions(
	context.Context,
	pulid.ID,
	pulid.ID,
) (*permissionports.EffectivePermissions, error) {
	return &permissionports.EffectivePermissions{}, nil
}

func (e *Engine) SimulatePermissions(
	context.Context,
	*permissionports.SimulatePermissionsRequest,
) (*permissionports.EffectivePermissions, error) {
	return &permissionports.EffectivePermissions{}, nil
}

func (e *Engine) AgentsUsable(
	context.Context,
	*permissionports.RequestActor,
	permission.Operation,
) (*permissionports.UsableAgents, error) {
	return &permissionports.UsableAgents{Assistant: true}, nil
}

func (e *Engine) MayUseAgent(
	context.Context,
	*permissionports.RequestActor,
	*agentdefinition.Definition,
) (bool, error) {
	return true, nil
}

func (e *Engine) RoleCoverage(
	_ context.Context,
	req *permissionports.RoleCoverageRequest,
) ([]permissionports.RoleCoverage, error) {
	return FullCoverage(req), nil
}

// FullCoverage is every requested role covering everything, for an engine
// that allows all.
func FullCoverage(req *permissionports.RoleCoverageRequest) []permissionports.RoleCoverage {
	if req == nil {
		return []permissionports.RoleCoverage{}
	}

	out := make([]permissionports.RoleCoverage, 0, len(req.RoleIDs))
	for _, id := range req.RoleIDs {
		out = append(out, permissionports.RoleCoverage{
			RoleID:           id,
			Coverage:         agentreach.CoverageFull,
			MissingResources: []permission.Resource{},
		})
	}

	return out
}

func (e *Engine) GetResourcePermissionsBatch(
	ctx context.Context,
	userID, orgID pulid.ID,
	resources []string,
) (map[string]*permissionports.ResourcePermissionDetail, error) {
	return Batch(ctx, e, userID, orgID, resources)
}

// DetailReader is the one engine read Batch answers a batch from.
type DetailReader interface {
	GetResourcePermissions(
		ctx context.Context,
		userID, orgID pulid.ID,
		resource string,
	) (*permissionports.ResourcePermissionDetail, error)
}

// Batch answers GetResourcePermissionsBatch for a test engine from its own
// GetResourcePermissions, one resource at a time.
func Batch(
	ctx context.Context,
	engine DetailReader,
	userID, orgID pulid.ID,
	resources []string,
) (map[string]*permissionports.ResourcePermissionDetail, error) {
	details := make(map[string]*permissionports.ResourcePermissionDetail, len(resources))
	for _, resource := range resources {
		detail, err := engine.GetResourcePermissions(ctx, userID, orgID, resource)
		if err != nil {
			return nil, err
		}
		if detail != nil {
			details[resource] = detail
		}
	}
	return details, nil
}
