package testutil

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/locationcategory"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

type ShipmentIntegrationFixture struct {
	ServiceType      *servicetype.ServiceType
	ShipmentType     *shipmenttype.ShipmentType
	Customer         *customer.Customer
	FormulaTemplate  *formulatemplate.FormulaTemplate
	LocationCategory *locationcategory.LocationCategory
}

type ShipmentGraph struct {
	Shipment *shipment.Shipment
	Moves    []*shipment.ShipmentMove
}

type ShipmentGraphParams struct {
	BOL          string
	ProNumber    string
	ShipmentID   pulid.ID
	MoveStatuses []shipment.MoveStatus
}

func SeedShipmentIntegrationFixture(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	data *seedtest.TestData,
	tenantInfo pagination.TenantInfo,
) *ShipmentIntegrationFixture {
	t.Helper()

	return &ShipmentIntegrationFixture{
		ServiceType:      mustCreateServiceType(t, ctx, db, tenantInfo),
		ShipmentType:     mustCreateShipmentType(t, ctx, db, tenantInfo),
		Customer:         mustCreateCustomer(t, ctx, db, data, tenantInfo),
		FormulaTemplate:  mustCreateFormulaTemplate(t, ctx, db, tenantInfo),
		LocationCategory: mustCreateLocationCategory(t, ctx, db, tenantInfo),
	}
}

func CreateShipmentGraph(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	fixture *ShipmentIntegrationFixture,
	tenantInfo pagination.TenantInfo,
	params ShipmentGraphParams,
) *ShipmentGraph {
	t.Helper()

	now := timeutils.NowUnix() + 86_400
	origin := MustCreateLocation(t, ctx, db, fixture, tenantInfo, params.BOL+"-O", "Origin")
	delivery := MustCreateLocation(t, ctx, db, fixture, tenantInfo, params.BOL+"-D", "Delivery")

	shipmentEntity := &shipment.Shipment{
		ID:                params.ShipmentID,
		BusinessUnitID:    tenantInfo.BuID,
		OrganizationID:    tenantInfo.OrgID,
		ServiceTypeID:     fixture.ServiceType.ID,
		ShipmentTypeID:    fixture.ShipmentType.ID,
		CustomerID:        fixture.Customer.ID,
		FormulaTemplateID: fixture.FormulaTemplate.ID,
		Status:            shipment.StatusNew,
		ProNumber:         params.ProNumber,
		BOL:               params.BOL,
		RatingUnit:        1,
	}
	_, err := db.NewInsert().Model(shipmentEntity).Exec(ctx)
	require.NoError(t, err)

	moves := make([]*shipment.ShipmentMove, 0, len(params.MoveStatuses))
	for idx, status := range params.MoveStatuses {
		move := &shipment.ShipmentMove{
			ID:             pulid.MustNew("sm_"),
			BusinessUnitID: tenantInfo.BuID,
			OrganizationID: tenantInfo.OrgID,
			ShipmentID:     shipmentEntity.ID,
			Status:         status,
			Loaded:         true,
			Sequence:       int64(idx),
		}
		_, err = db.NewInsert().Model(move).Exec(ctx)
		require.NoError(t, err)

		stops := []*shipment.Stop{
			{
				ID:                   pulid.MustNew("stp_"),
				BusinessUnitID:       tenantInfo.BuID,
				OrganizationID:       tenantInfo.OrgID,
				ShipmentMoveID:       move.ID,
				LocationID:           origin.ID,
				Status:               shipment.StopStatusNew,
				Type:                 shipment.StopTypePickup,
				ScheduleType:         shipment.StopScheduleTypeOpen,
				Sequence:             0,
				ScheduledWindowStart: now + int64(idx*10_000),
				ScheduledWindowEnd:   int64Ptr(now + int64(idx*10_000) + 1_800),
			},
			{
				ID:                   pulid.MustNew("stp_"),
				BusinessUnitID:       tenantInfo.BuID,
				OrganizationID:       tenantInfo.OrgID,
				ShipmentMoveID:       move.ID,
				LocationID:           delivery.ID,
				Status:               shipment.StopStatusNew,
				Type:                 shipment.StopTypeDelivery,
				ScheduleType:         shipment.StopScheduleTypeOpen,
				Sequence:             1,
				ScheduledWindowStart: now + int64(idx*10_000) + 7_200,
				ScheduledWindowEnd:   int64Ptr(now + int64(idx*10_000) + 9_000),
			},
		}
		_, err = db.NewInsert().Model(&stops).Exec(ctx)
		require.NoError(t, err)
		move.Stops = stops
		moves = append(moves, move)
	}

	return &ShipmentGraph{
		Shipment: shipmentEntity,
		Moves:    moves,
	}
}

func MustCreateLocation(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	fixture *ShipmentIntegrationFixture,
	tenantInfo pagination.TenantInfo,
	code string,
	name string,
) *location.Location {
	t.Helper()

	entity := &location.Location{
		BusinessUnitID:     tenantInfo.BuID,
		OrganizationID:     tenantInfo.OrgID,
		LocationCategoryID: fixture.LocationCategory.ID,
		StateID:            fixture.Customer.StateID,
		Status:             domaintypes.StatusActive,
		Code:               code,
		Name:               name,
		AddressLine1:       "100 Integration Way",
		City:               "Charlotte",
		PostalCode:         "28202",
	}
	_, err := db.NewInsert().Model(entity).Exec(ctx)
	require.NoError(t, err)
	return entity
}

func mustCreateServiceType(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	tenantInfo pagination.TenantInfo,
) *servicetype.ServiceType {
	t.Helper()

	entity := &servicetype.ServiceType{
		BusinessUnitID: tenantInfo.BuID,
		OrganizationID: tenantInfo.OrgID,
		Status:         domaintypes.StatusActive,
		Code:           "LTL",
		Color:          "#112233",
	}
	_, err := db.NewInsert().Model(entity).Exec(ctx)
	require.NoError(t, err)
	return entity
}

func mustCreateShipmentType(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	tenantInfo pagination.TenantInfo,
) *shipmenttype.ShipmentType {
	t.Helper()

	entity := &shipmenttype.ShipmentType{
		BusinessUnitID: tenantInfo.BuID,
		OrganizationID: tenantInfo.OrgID,
		Status:         domaintypes.StatusActive,
		Code:           "STD",
		Color:          "#334455",
	}
	_, err := db.NewInsert().Model(entity).Exec(ctx)
	require.NoError(t, err)
	return entity
}

func mustCreateCustomer(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	data *seedtest.TestData,
	tenantInfo pagination.TenantInfo,
) *customer.Customer {
	t.Helper()

	entity := &customer.Customer{
		BusinessUnitID:     tenantInfo.BuID,
		OrganizationID:     tenantInfo.OrgID,
		StateID:            data.State.ID,
		Status:             domaintypes.StatusActive,
		Code:               "CUST1",
		Name:               "Integration Customer",
		AddressLine1:       "100 Customer Way",
		City:               "Charlotte",
		PostalCode:         "28202",
		AllowConsolidation: true,
	}
	_, err := db.NewInsert().Model(entity).Exec(ctx)
	require.NoError(t, err)
	return entity
}

func mustCreateFormulaTemplate(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	tenantInfo pagination.TenantInfo,
) *formulatemplate.FormulaTemplate {
	t.Helper()

	entity := &formulatemplate.FormulaTemplate{
		BusinessUnitID:      tenantInfo.BuID,
		OrganizationID:      tenantInfo.OrgID,
		Name:                "Integration Freight Template",
		Type:                formulatemplate.TemplateTypeFreightCharge,
		Expression:          "1",
		Status:              formulatemplate.StatusActive,
		SchemaID:            "shipment",
		VariableDefinitions: []*formulatypes.VariableDefinition{},
	}
	_, err := db.NewInsert().Model(entity).Exec(ctx)
	require.NoError(t, err)
	return entity
}

func mustCreateLocationCategory(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	tenantInfo pagination.TenantInfo,
) *locationcategory.LocationCategory {
	t.Helper()

	entity := &locationcategory.LocationCategory{
		BusinessUnitID: tenantInfo.BuID,
		OrganizationID: tenantInfo.OrgID,
		Name:           "Customer Sites",
		Type:           locationcategory.CategoryCustomerLocation,
		Color:          "#445566",
	}
	_, err := db.NewInsert().Model(entity).Exec(ctx)
	require.NoError(t, err)
	return entity
}

func MustCreateAccessorialCharge(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	tenantInfo pagination.TenantInfo,
	code string,
	method accessorialcharge.Method,
	rateUnit accessorialcharge.RateUnit,
	amount string,
) *accessorialcharge.AccessorialCharge {
	t.Helper()

	entity := &accessorialcharge.AccessorialCharge{
		BusinessUnitID: tenantInfo.BuID,
		OrganizationID: tenantInfo.OrgID,
		Status:         domaintypes.StatusActive,
		Code:           code,
		Description:    code + " charge",
		Method:         method,
		RateUnit:       rateUnit,
		Amount:         decimal.RequireFromString(amount),
	}
	_, err := db.NewInsert().Model(entity).Exec(ctx)
	require.NoError(t, err)
	return entity
}

func MustCreateHazardousMaterial(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	tenantInfo pagination.TenantInfo,
	code string,
	name string,
	class hazardousmaterial.HazardousClass,
) *hazardousmaterial.HazardousMaterial {
	t.Helper()

	entity := &hazardousmaterial.HazardousMaterial{
		BusinessUnitID:     tenantInfo.BuID,
		OrganizationID:     tenantInfo.OrgID,
		Status:             domaintypes.StatusActive,
		Code:               code,
		Name:               name,
		Description:        name + " description",
		Class:              class,
		PackingGroup:       hazardousmaterial.PackingGroupII,
		ProperShippingName: name,
	}
	_, err := db.NewInsert().Model(entity).Exec(ctx)
	require.NoError(t, err)
	return entity
}

func MustCreateCommodity(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	tenantInfo pagination.TenantInfo,
	name string,
	hazmatID pulid.ID,
) *commodity.Commodity {
	t.Helper()

	entity := &commodity.Commodity{
		BusinessUnitID:      tenantInfo.BuID,
		OrganizationID:      tenantInfo.OrgID,
		HazardousMaterialID: hazmatID,
		Status:              domaintypes.StatusActive,
		Name:                name,
		Description:         name + " commodity",
	}
	_, err := db.NewInsert().Model(entity).Exec(ctx)
	require.NoError(t, err)
	return entity
}

func MustCreateHazmatSegregationRule(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	tenantInfo pagination.TenantInfo,
	name string,
	classA hazardousmaterial.HazardousClass,
	classB hazardousmaterial.HazardousClass,
	segType hazmatsegregationrule.SegregationType,
	hazmatAID *pulid.ID,
	hazmatBID *pulid.ID,
) *hazmatsegregationrule.HazmatSegregationRule {
	t.Helper()

	entity := &hazmatsegregationrule.HazmatSegregationRule{
		BusinessUnitID:  tenantInfo.BuID,
		OrganizationID:  tenantInfo.OrgID,
		Status:          domaintypes.StatusActive,
		Name:            name,
		ClassA:          classA,
		ClassB:          classB,
		SegregationType: segType,
		HazmatAID:       hazmatAID,
		HazmatBID:       hazmatBID,
		MinimumDistance: nil,
	}
	if segType == hazmatsegregationrule.SegregationTypeDistance {
		entity.MinimumDistance = float64Ptr(10)
		entity.DistanceUnit = "FT"
	}
	_, err := db.NewInsert().Model(entity).Exec(ctx)
	require.NoError(t, err)
	return entity
}

//go:fix inline
func float64Ptr(v float64) *float64 {
	return &v
}

//go:fix inline
func int64Ptr(v int64) *int64 {
	return &v
}
