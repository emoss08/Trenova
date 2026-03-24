package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
)

type DataTransformer interface {
	TransformAccessorialCharge(
		ctx context.Context,
		entity *accessorialcharge.AccessorialCharge,
	) error
	TransformFleetCode(ctx context.Context, entity *fleetcode.FleetCode) error
	TransformEquipmentType(ctx context.Context, entity *equipmenttype.EquipmentType) error
	TransformServiceType(ctx context.Context, entity *servicetype.ServiceType) error
	TransformShipmentType(ctx context.Context, entity *shipmenttype.ShipmentType) error
	TransformHazardousMaterial(
		ctx context.Context,
		entity *hazardousmaterial.HazardousMaterial,
	) error
	TransformCommodity(
		ctx context.Context,
		entity *commodity.Commodity,
	) error
	TransformCustomer(
		ctx context.Context,
		entity *customer.Customer,
	) error
	TransformAccountType(
		ctx context.Context,
		entity *accounttype.AccountType,
	) error
	TransformGLAccount(
		ctx context.Context,
		entity *glaccount.GLAccount,
	) error
	TransformFiscalYear(
		ctx context.Context,
		entity *fiscalyear.FiscalYear,
	) error
	TransformLocation(
		ctx context.Context,
		entity *location.Location,
	) error
	TransformDocumentType(
		ctx context.Context,
		entity *documenttype.DocumentType,
	) error
}
