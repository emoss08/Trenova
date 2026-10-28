package mocks

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/permtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var _ auditports.AuditService = (*NoopAuditService)(nil)

type NoopAuditService struct{}

func (n *NoopAuditService) List(
	_ context.Context,
	_ *auditports.ListAuditEntriesRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	return &pagination.ListResult[*audit.Entry]{TotalExact: true}, nil
}

func (n *NoopAuditService) ListConnection(
	_ context.Context,
	_ *auditports.ListAuditEntriesConnectionRequest,
) (*pagination.CursorListResult[*audit.Entry], error) {
	return &pagination.CursorListResult[*audit.Entry]{}, nil
}

func (n *NoopAuditService) ListByResourceID(
	_ context.Context,
	_ *auditports.ListByResourceIDRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	return &pagination.ListResult[*audit.Entry]{TotalExact: true}, nil
}

func (n *NoopAuditService) GetByID(
	_ context.Context,
	_ auditports.GetAuditEntryByIDOptions,
) (*audit.Entry, error) {
	return nil, nil
}

func (n *NoopAuditService) LogAction(
	_ *auditports.LogActionParams,
	_ ...auditports.LogOption,
) error {
	return nil
}

func (n *NoopAuditService) LogActions(_ []auditports.BulkLogEntry) error {
	return nil
}

func (n *NoopAuditService) RegisterSensitiveFields(
	_ permission.Resource,
	_ []auditports.SensitiveField,
) error {
	return nil
}

var _ permissionports.PermissionEngine = (*AllowAllPermissionEngine)(nil)

type AllowAllPermissionEngine struct{}

func (a *AllowAllPermissionEngine) Check(
	_ context.Context,
	_ *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	return &permissionports.PermissionCheckResult{Allowed: true}, nil
}

func (a *AllowAllPermissionEngine) CheckBatch(
	_ context.Context,
	_ *permissionports.BatchPermissionCheckRequest,
) (*permissionports.BatchPermissionCheckResult, error) {
	return &permissionports.BatchPermissionCheckResult{}, nil
}

func (a *AllowAllPermissionEngine) GetLightManifest(
	_ context.Context,
	_, _ pulid.ID,
) (*permissionports.LightPermissionManifest, error) {
	return nil, nil
}

func (a *AllowAllPermissionEngine) GetResourcePermissions(
	_ context.Context,
	_, _ pulid.ID,
	_ string,
) (*permissionports.ResourcePermissionDetail, error) {
	return nil, nil
}

func (a *AllowAllPermissionEngine) GetResourcePermissionsBatch(
	ctx context.Context,
	userID, orgID pulid.ID,
	resources []string,
) (map[string]*permissionports.ResourcePermissionDetail, error) {
	return permtest.Batch(ctx, a, userID, orgID, resources)
}

func (a *AllowAllPermissionEngine) InvalidateUser(_ context.Context, _, _ pulid.ID) error {
	return nil
}

func (a *AllowAllPermissionEngine) GetEffectivePermissions(
	_ context.Context,
	_, _ pulid.ID,
) (*permissionports.EffectivePermissions, error) {
	registry := permissioncatalog.NewRegistry()
	resources := make(map[string]permissionports.EffectiveResourcePermission, len(registry.All()))
	for _, def := range registry.All() {
		operations := make([]permission.Operation, len(def.Operations))
		for i, op := range def.Operations {
			operations[i] = op.Operation
		}

		resources[def.Resource] = permissionports.EffectiveResourcePermission{
			Operations: operations,
			DataScope:  permission.DataScopeAll,
		}
	}

	return &permissionports.EffectivePermissions{
		MaxSensitivity: permission.SensitivityConfidential,
		Resources:      resources,
	}, nil
}

func (a *AllowAllPermissionEngine) SimulatePermissions(
	_ context.Context,
	_ *permissionports.SimulatePermissionsRequest,
) (*permissionports.EffectivePermissions, error) {
	return nil, nil
}

func (a *AllowAllPermissionEngine) AgentsUsable(
	_ context.Context,
	_ *permissionports.RequestActor,
	_ permission.Operation,
) (*permissionports.UsableAgents, error) {
	return &permissionports.UsableAgents{Assistant: true}, nil
}

func (a *AllowAllPermissionEngine) MayUseAgent(
	_ context.Context,
	_ *permissionports.RequestActor,
	_ *agentdefinition.Definition,
) (bool, error) {
	return true, nil
}

func (a *AllowAllPermissionEngine) RoleCoverage(
	_ context.Context,
	req *permissionports.RoleCoverageRequest,
) ([]permissionports.RoleCoverage, error) {
	return permtest.FullCoverage(req), nil
}

var _ permissionports.PermissionEngine = (*DenyResourcesPermissionEngine)(nil)

type DenyResourcesPermissionEngine struct {
	AllowAllPermissionEngine

	Denied []string
}

func (d *DenyResourcesPermissionEngine) Check(
	ctx context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	for _, resource := range d.Denied {
		if resource == req.Resource {
			return &permissionports.PermissionCheckResult{Allowed: false}, nil
		}
	}

	return d.AllowAllPermissionEngine.Check(ctx, req)
}

func (d *DenyResourcesPermissionEngine) CheckBatch(
	_ context.Context,
	req *permissionports.BatchPermissionCheckRequest,
) (*permissionports.BatchPermissionCheckResult, error) {
	results := make([]permissionports.PermissionCheckResult, len(req.Checks))
	for i, check := range req.Checks {
		allowed := !slices.Contains(d.Denied, check.Resource)
		results[i] = permissionports.PermissionCheckResult{Allowed: allowed}
		if allowed {
			results[i].DataScope = permission.DataScopeOrganization
		}
	}

	return &permissionports.BatchPermissionCheckResult{Results: results}, nil
}

var _ permissionports.PermissionEngine = (*FailingPermissionEngine)(nil)

type FailingPermissionEngine struct {
	AllowAllPermissionEngine

	Failing []string
	Err     error
}

func (f *FailingPermissionEngine) Check(
	ctx context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	for _, resource := range f.Failing {
		if resource == req.Resource {
			return nil, f.Err
		}
	}

	return f.AllowAllPermissionEngine.Check(ctx, req)
}

var _ accessorialchargeports.DataTransformer = (*NoopDataTransformer)(nil)

type NoopDataTransformer struct{}

func (n *NoopDataTransformer) TransformAccessorialCharge(
	_ context.Context,
	_ *accessorialcharge.AccessorialCharge,
) error {
	return nil
}

func (n *NoopDataTransformer) TransformFleetCode(_ context.Context, _ *fleetcode.FleetCode) error {
	return nil
}

func (n *NoopDataTransformer) TransformEquipmentType(
	_ context.Context,
	_ *equipmenttype.EquipmentType,
) error {
	return nil
}

func (n *NoopDataTransformer) TransformServiceType(
	_ context.Context,
	_ *servicetype.ServiceType,
) error {
	return nil
}

func (n *NoopDataTransformer) TransformShipmentType(
	_ context.Context,
	_ *shipmenttype.ShipmentType,
) error {
	return nil
}

func (n *NoopDataTransformer) TransformHazardousMaterial(
	_ context.Context,
	_ *hazardousmaterial.HazardousMaterial,
) error {
	return nil
}

func (n *NoopDataTransformer) TransformCommodity(
	_ context.Context,
	_ *commodity.Commodity,
) error {
	return nil
}

func (n *NoopDataTransformer) TransformCustomer(
	_ context.Context,
	_ *customer.Customer,
) error {
	return nil
}

func (n *NoopDataTransformer) TransformAccountType(
	_ context.Context,
	_ *accounttype.AccountType,
) error {
	return nil
}

func (n *NoopDataTransformer) TransformGLAccount(
	_ context.Context,
	_ *glaccount.GLAccount,
) error {
	return nil
}

func (n *NoopDataTransformer) TransformFiscalYear(
	_ context.Context,
	_ *fiscalyear.FiscalYear,
) error {
	return nil
}

func (n *NoopDataTransformer) TransformLocation(
	_ context.Context,
	_ *location.Location,
) error {
	return nil
}

func (n *NoopDataTransformer) TransformDocumentType(
	_ context.Context,
	_ *documenttype.DocumentType,
) error {
	return nil
}
