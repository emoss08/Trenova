package accessorialchargeports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype"
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
