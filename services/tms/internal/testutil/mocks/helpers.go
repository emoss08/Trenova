package mocks

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var _ services.AuditService = (*NoopAuditService)(nil)

type NoopAuditService struct{}

func (n *NoopAuditService) List(
	_ context.Context,
	_ *repositories.ListAuditEntriesRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	return &pagination.ListResult[*audit.Entry]{}, nil
}

func (n *NoopAuditService) ListByResourceID(
	_ context.Context,
	_ *repositories.ListByResourceIDRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	return &pagination.ListResult[*audit.Entry]{}, nil
}

func (n *NoopAuditService) GetByID(
	_ context.Context,
	_ repositories.GetAuditEntryByIDOptions,
) (*audit.Entry, error) {
	return nil, nil
}

func (n *NoopAuditService) LogAction(_ *services.LogActionParams, _ ...services.LogOption) error {
	return nil
}

func (n *NoopAuditService) LogActions(_ []services.BulkLogEntry) error {
	return nil
}

func (n *NoopAuditService) RegisterSensitiveFields(
	_ permission.Resource,
	_ []services.SensitiveField,
) error {
	return nil
}

var _ services.PermissionEngine = (*AllowAllPermissionEngine)(nil)

type AllowAllPermissionEngine struct{}

func (a *AllowAllPermissionEngine) Check(
	_ context.Context,
	_ *services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	return &services.PermissionCheckResult{Allowed: true}, nil
}

func (a *AllowAllPermissionEngine) CheckBatch(
	_ context.Context,
	_ *services.BatchPermissionCheckRequest,
) (*services.BatchPermissionCheckResult, error) {
	return &services.BatchPermissionCheckResult{}, nil
}

func (a *AllowAllPermissionEngine) GetLightManifest(
	_ context.Context,
	_, _ pulid.ID,
) (*services.LightPermissionManifest, error) {
	return nil, nil
}

func (a *AllowAllPermissionEngine) GetResourcePermissions(
	_ context.Context,
	_, _ pulid.ID,
	_ string,
) (*services.ResourcePermissionDetail, error) {
	return nil, nil
}

func (a *AllowAllPermissionEngine) InvalidateUser(_ context.Context, _, _ pulid.ID) error {
	return nil
}

func (a *AllowAllPermissionEngine) GetEffectivePermissions(
	_ context.Context,
	_, _ pulid.ID,
) (*services.EffectivePermissions, error) {
	return nil, nil
}

func (a *AllowAllPermissionEngine) SimulatePermissions(
	_ context.Context,
	_ *services.SimulatePermissionsRequest,
) (*services.EffectivePermissions, error) {
	return nil, nil
}

var _ services.DataTransformer = (*NoopDataTransformer)(nil)

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
