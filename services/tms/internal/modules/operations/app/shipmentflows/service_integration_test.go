//go:build integration

package shipmentflows

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/chargeallocationrepository"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/shipmentcontrolrepository"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol/shipmentcontroldb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/orderrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentadditionalchargerepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentcommodityrepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentmoverepository"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentrepository"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcommercial"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentservicetest"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentvalidator"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/operationstest"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest"
	"github.com/emoss08/trenova/internal/modules/reference/adapters/postgres/accessorialchargerepository"
	"github.com/emoss08/trenova/internal/modules/reference/adapters/postgres/commodityrepository"
	"github.com/emoss08/trenova/internal/modules/reference/adapters/postgres/hazmatsegregationrulerepository"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazmatsegregationrule"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/testutil"
	internaltestutil "github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCreateIntegration_PersistsAdditionalChargesAndCommodities(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, shipmentRepo, controlRepo, _, tenantInfo, fixture, data := newIntegrationShipmentService(
		t,
		ctx,
		db,
	)
	ctx = integrationTenantContext(ctx, tenantInfo)
	configureShipmentControl(
		t,
		ctx,
		controlRepo,
		tenantInfo,
		func(sc *shipmentcontrol.ShipmentControl) {
			sc.CheckForDuplicateBOLs = true
			sc.CheckHazmatSegregation = false
		},
	)

	accessorial := testutil.MustCreateAccessorialCharge(
		t,
		ctx,
		db,
		tenantInfo,
		"LUMPER",
		accessorialcharge.MethodFlat,
		"",
		"15",
	)
	commodityEntity := testutil.MustCreateCommodity(
		t,
		ctx,
		db,
		tenantInfo,
		"General Freight",
		pulid.Nil,
	)

	entity := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	entity.BOL = "BOL-CREATE-CHARGE-COMMODITY"
	entity.AdditionalCharges = []*shipment.AdditionalCharge{
		{
			AccessorialChargeID: accessorial.ID,
			Method:              accessorialcharge.MethodFlat,
			Amount:              decimal.NewFromInt(15),
			Unit:                1,
		},
	}
	entity.Commodities = []*shipment.ShipmentCommodity{
		{
			CommodityID: commodityEntity.ID,
			Pieces:      10,
			Weight:      1000,
		},
	}

	created, err := svc.writer.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID),
	)
	require.NoError(t, err)
	require.NotNil(t, created)

	persisted, err := shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         created.ID,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	require.NoError(t, err)
	require.Len(t, persisted.AdditionalCharges, 1)
	require.Len(t, persisted.Commodities, 1)
	assert.Equal(t, accessorial.ID, persisted.AdditionalCharges[0].AccessorialChargeID)
	assert.Equal(t, commodityEntity.ID, persisted.Commodities[0].CommodityID)
	assert.True(t, decimal.NewFromInt(15).Equal(persisted.OtherChargeAmount.Decimal))
	assert.True(t, decimal.NewFromInt(15).Equal(persisted.TotalChargeAmount.Decimal))
	require.NotNil(t, persisted.AdditionalCharges[0].AccessorialCharge)
	require.NotNil(t, persisted.Commodities[0].Commodity)
}

func TestUpdateIntegration_SyncsNestedChargesAndCommodities(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, shipmentRepo, controlRepo, _, tenantInfo, fixture, data := newIntegrationShipmentService(
		t,
		ctx,
		db,
	)
	ctx = integrationTenantContext(ctx, tenantInfo)
	configureShipmentControl(
		t,
		ctx,
		controlRepo,
		tenantInfo,
		func(sc *shipmentcontrol.ShipmentControl) {
			sc.CheckForDuplicateBOLs = true
			sc.CheckHazmatSegregation = false
		},
	)

	flatCharge := testutil.MustCreateAccessorialCharge(
		t,
		ctx,
		db,
		tenantInfo,
		"FLAT1",
		accessorialcharge.MethodFlat,
		"",
		"10",
	)
	perUnitCharge := testutil.MustCreateAccessorialCharge(
		t,
		ctx,
		db,
		tenantInfo,
		"LABOR",
		accessorialcharge.MethodPerUnit,
		accessorialcharge.RateUnitHour,
		"5",
	)
	commodityOne := testutil.MustCreateCommodity(t, ctx, db, tenantInfo, "Widgets", pulid.Nil)
	commodityTwo := testutil.MustCreateCommodity(t, ctx, db, tenantInfo, "Bolts", pulid.Nil)
	commodityThree := testutil.MustCreateCommodity(t, ctx, db, tenantInfo, "Pallets", pulid.Nil)

	entity := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	entity.BOL = "BOL-UPDATE-CHARGE-COMMODITY"
	entity.AdditionalCharges = []*shipment.AdditionalCharge{
		{
			AccessorialChargeID: flatCharge.ID,
			Method:              accessorialcharge.MethodFlat,
			Amount:              decimal.NewFromInt(10),
			Unit:                1,
		},
		{
			AccessorialChargeID: perUnitCharge.ID,
			Method:              accessorialcharge.MethodPerUnit,
			Amount:              decimal.NewFromInt(3),
			Unit:                2,
		},
	}
	entity.Commodities = []*shipment.ShipmentCommodity{
		{CommodityID: commodityOne.ID, Pieces: 10, Weight: 100},
		{CommodityID: commodityTwo.ID, Pieces: 20, Weight: 200},
	}

	created, err := svc.writer.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID),
	)
	require.NoError(t, err)

	loaded, err := shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         created.ID,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	require.NoError(t, err)
	require.Len(t, loaded.AdditionalCharges, 2)
	require.Len(t, loaded.Commodities, 2)

	removedChargeID := loaded.AdditionalCharges[1].ID
	removedCommodityID := loaded.Commodities[1].ID

	loaded.AdditionalCharges = []*shipment.AdditionalCharge{
		{
			ID:                  loaded.AdditionalCharges[0].ID,
			ShipmentID:          loaded.ID,
			AccessorialChargeID: loaded.AdditionalCharges[0].AccessorialChargeID,
			Method:              loaded.AdditionalCharges[0].Method,
			Amount:              decimal.NewFromInt(20),
			Unit:                1,
		},
		{
			AccessorialChargeID: perUnitCharge.ID,
			Method:              accessorialcharge.MethodPerUnit,
			Amount:              decimal.NewFromInt(5),
			Unit:                2,
		},
	}
	loaded.Commodities = []*shipment.ShipmentCommodity{
		{
			ID:          loaded.Commodities[0].ID,
			ShipmentID:  loaded.ID,
			CommodityID: loaded.Commodities[0].CommodityID,
			Pieces:      11,
			Weight:      110,
		},
		{
			CommodityID: commodityThree.ID,
			Pieces:      30,
			Weight:      300,
		},
	}

	updated, err := svc.writer.Update(
		ctx,
		loaded,
		internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID),
	)
	require.NoError(t, err)
	require.NotNil(t, updated)

	persisted, err := shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         created.ID,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	require.NoError(t, err)
	require.Len(t, persisted.AdditionalCharges, 2)
	require.Len(t, persisted.Commodities, 2)
	assert.True(t, decimal.NewFromInt(30).Equal(persisted.OtherChargeAmount.Decimal))
	assert.True(t, decimal.NewFromInt(30).Equal(persisted.TotalChargeAmount.Decimal))
	assert.Equal(t, loaded.AdditionalCharges[0].ID, persisted.AdditionalCharges[0].ID)
	assert.Equal(t, commodityOne.ID, persisted.Commodities[0].CommodityID)
	assert.Equal(t, int64(110), persisted.Commodities[0].Weight)
	assert.Equal(t, commodityThree.ID, persisted.Commodities[1].CommodityID)
	assert.NotEqual(t, removedChargeID, persisted.AdditionalCharges[1].ID)
	assert.NotEqual(t, removedCommodityID, persisted.Commodities[1].ID)
}

func TestUpdateIntegration_ReplacesExistingCommodityInPlace(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, shipmentRepo, controlRepo, _, tenantInfo, fixture, data := newIntegrationShipmentService(
		t,
		ctx,
		db,
	)
	ctx = integrationTenantContext(ctx, tenantInfo)
	configureShipmentControl(
		t,
		ctx,
		controlRepo,
		tenantInfo,
		func(sc *shipmentcontrol.ShipmentControl) {
			sc.CheckForDuplicateBOLs = true
			sc.CheckHazmatSegregation = false
		},
	)

	commodityOne := testutil.MustCreateCommodity(
		t,
		ctx,
		db,
		tenantInfo,
		"Replacement One",
		pulid.Nil,
	)
	commodityTwo := testutil.MustCreateCommodity(
		t,
		ctx,
		db,
		tenantInfo,
		"Replacement Two",
		pulid.Nil,
	)

	entity := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	entity.BOL = "BOL-UPDATE-COMMODITY-REPLACE"
	entity.Commodities = []*shipment.ShipmentCommodity{
		{
			CommodityID: commodityOne.ID,
			Pieces:      4,
			Weight:      80,
		},
	}

	created, err := svc.writer.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID),
	)
	require.NoError(t, err)

	loaded, err := shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         created.ID,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	require.NoError(t, err)
	require.Len(t, loaded.Commodities, 1)

	existingCommodityRowID := loaded.Commodities[0].ID

	loaded.Commodities[0].CommodityID = commodityTwo.ID
	loaded.Commodities[0].Pieces = 6
	loaded.Commodities[0].Weight = 120

	updated, err := svc.writer.Update(
		ctx,
		loaded,
		internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID),
	)
	require.NoError(t, err)
	require.NotNil(t, updated)

	persisted, err := shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         created.ID,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	require.NoError(t, err)
	require.Len(t, persisted.Commodities, 1)
	assert.Equal(t, existingCommodityRowID, persisted.Commodities[0].ID)
	assert.Equal(t, commodityTwo.ID, persisted.Commodities[0].CommodityID)
	assert.Equal(t, int64(6), persisted.Commodities[0].Pieces)
	assert.Equal(t, int64(120), persisted.Commodities[0].Weight)
	require.NotNil(t, persisted.Commodities[0].Commodity)
	assert.Equal(t, commodityTwo.ID, persisted.Commodities[0].Commodity.ID)
}

func TestUpdateIntegration_ReplacesExistingAdditionalChargeInPlace(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, shipmentRepo, controlRepo, _, tenantInfo, fixture, data := newIntegrationShipmentService(
		t,
		ctx,
		db,
	)
	ctx = integrationTenantContext(ctx, tenantInfo)
	configureShipmentControl(
		t,
		ctx,
		controlRepo,
		tenantInfo,
		func(sc *shipmentcontrol.ShipmentControl) {
			sc.CheckForDuplicateBOLs = true
			sc.CheckHazmatSegregation = false
		},
	)

	flatCharge := testutil.MustCreateAccessorialCharge(
		t,
		ctx,
		db,
		tenantInfo,
		"RFLAT",
		accessorialcharge.MethodFlat,
		"",
		"10",
	)
	perUnitCharge := testutil.MustCreateAccessorialCharge(
		t,
		ctx,
		db,
		tenantInfo,
		"RUNIT",
		accessorialcharge.MethodPerUnit,
		accessorialcharge.RateUnitHour,
		"5",
	)

	entity := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	entity.BOL = "BOL-UPDATE-CHARGE-REPLACE"
	entity.AdditionalCharges = []*shipment.AdditionalCharge{
		{
			AccessorialChargeID: flatCharge.ID,
			Method:              accessorialcharge.MethodFlat,
			Amount:              decimal.NewFromInt(10),
			Unit:                1,
		},
	}

	created, err := svc.writer.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID),
	)
	require.NoError(t, err)

	loaded, err := shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         created.ID,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	require.NoError(t, err)
	require.Len(t, loaded.AdditionalCharges, 1)

	existingChargeRowID := loaded.AdditionalCharges[0].ID

	loaded.AdditionalCharges[0].AccessorialChargeID = perUnitCharge.ID
	loaded.AdditionalCharges[0].Method = accessorialcharge.MethodPerUnit
	loaded.AdditionalCharges[0].Amount = decimal.NewFromInt(7)
	loaded.AdditionalCharges[0].Unit = 2

	updated, err := svc.writer.Update(
		ctx,
		loaded,
		internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID),
	)
	require.NoError(t, err)
	require.NotNil(t, updated)

	persisted, err := shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         created.ID,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	require.NoError(t, err)
	require.Len(t, persisted.AdditionalCharges, 1)
	assert.Equal(t, existingChargeRowID, persisted.AdditionalCharges[0].ID)
	assert.Equal(t, perUnitCharge.ID, persisted.AdditionalCharges[0].AccessorialChargeID)
	assert.Equal(t, accessorialcharge.MethodPerUnit, persisted.AdditionalCharges[0].Method)
	assert.True(t, decimal.NewFromInt(7).Equal(persisted.AdditionalCharges[0].Amount))
	assert.Equal(t, int16(2), persisted.AdditionalCharges[0].Unit)
	require.NotNil(t, persisted.AdditionalCharges[0].AccessorialCharge)
	assert.Equal(t, perUnitCharge.ID, persisted.AdditionalCharges[0].AccessorialCharge.ID)
}

func TestCreateIntegration_RejectsManualDetentionChargeWhenAutoGenerated(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, _, controlRepo, _, tenantInfo, fixture, data := newIntegrationShipmentService(t, ctx, db)
	ctx = integrationTenantContext(ctx, tenantInfo)
	detentionCharge := testutil.MustCreateAccessorialCharge(
		t,
		ctx,
		db,
		tenantInfo,
		"DETM",
		accessorialcharge.MethodFlat,
		"",
		"50",
	)
	configureShipmentControl(
		t,
		ctx,
		controlRepo,
		tenantInfo,
		func(sc *shipmentcontrol.ShipmentControl) {
			sc.TrackDetentionTime = true
			sc.AutoGenerateDetentionCharges = true
			sc.DetentionThreshold = int16PtrForIntegration(30)
			sc.DetentionChargeID = &detentionCharge.ID
		},
	)

	entity := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	entity.BOL = "BOL-MANUAL-DETENTION"
	entity.AdditionalCharges = []*shipment.AdditionalCharge{
		{
			AccessorialChargeID: detentionCharge.ID,
			Method:              accessorialcharge.MethodFlat,
			Amount:              decimal.NewFromInt(50),
			Unit:                1,
		},
	}

	created, err := svc.writer.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID),
	)
	require.Nil(t, created)
	require.Error(t, err)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "additionalCharges[0].accessorialChargeId")
}

func TestUpdateIntegration_RejectsManualDetentionChargeWhenAutoGenerated(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, shipmentRepo, controlRepo, _, tenantInfo, fixture, data := newIntegrationShipmentService(
		t,
		ctx,
		db,
	)
	ctx = integrationTenantContext(ctx, tenantInfo)
	manualCharge := testutil.MustCreateAccessorialCharge(
		t,
		ctx,
		db,
		tenantInfo,
		"MISC",
		accessorialcharge.MethodFlat,
		"",
		"25",
	)
	detentionCharge := testutil.MustCreateAccessorialCharge(
		t,
		ctx,
		db,
		tenantInfo,
		"DETU",
		accessorialcharge.MethodFlat,
		"",
		"50",
	)
	configureShipmentControl(
		t,
		ctx,
		controlRepo,
		tenantInfo,
		func(sc *shipmentcontrol.ShipmentControl) {
			sc.TrackDetentionTime = true
			sc.AutoGenerateDetentionCharges = true
			sc.DetentionThreshold = int16PtrForIntegration(30)
			sc.DetentionChargeID = &detentionCharge.ID
		},
	)

	entity := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	entity.BOL = "BOL-MANUAL-DETENTION-UPD"
	entity.AdditionalCharges = []*shipment.AdditionalCharge{
		{
			AccessorialChargeID: manualCharge.ID,
			Method:              accessorialcharge.MethodFlat,
			Amount:              decimal.NewFromInt(25),
			Unit:                1,
		},
	}

	created, err := svc.writer.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID),
	)
	require.NoError(t, err)

	loaded, err := shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         created.ID,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	require.NoError(t, err)
	require.Len(t, loaded.AdditionalCharges, 1)

	loaded.AdditionalCharges[0].AccessorialChargeID = detentionCharge.ID
	loaded.AdditionalCharges[0].Amount = decimal.NewFromInt(50)

	updated, err := svc.writer.Update(
		ctx,
		loaded,
		internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID),
	)
	require.Nil(t, updated)
	require.Error(t, err)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "additionalCharges[0].accessorialChargeId")
}

func TestCreateIntegration_RejectsHazmatSegregationConflicts(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, _, controlRepo, _, tenantInfo, fixture, data := newIntegrationShipmentService(t, ctx, db)
	ctx = integrationTenantContext(ctx, tenantInfo)
	configureShipmentControl(
		t,
		ctx,
		controlRepo,
		tenantInfo,
		func(sc *shipmentcontrol.ShipmentControl) {
			sc.CheckHazmatSegregation = true
		},
	)

	leftHazmat := testutil.MustCreateHazardousMaterial(
		t,
		ctx,
		db,
		tenantInfo,
		"HM1",
		"Explosive",
		hazardousmaterial.HazardousClass1,
	)
	rightHazmat := testutil.MustCreateHazardousMaterial(
		t,
		ctx,
		db,
		tenantInfo,
		"HM3",
		"Paint",
		hazardousmaterial.HazardousClass3,
	)
	leftCommodity := testutil.MustCreateCommodity(
		t,
		ctx,
		db,
		tenantInfo,
		"Explosives",
		leftHazmat.ID,
	)
	rightCommodity := testutil.MustCreateCommodity(t, ctx, db, tenantInfo, "Paint", rightHazmat.ID)
	testutil.MustCreateHazmatSegregationRule(
		t,
		ctx,
		db,
		tenantInfo,
		"No explosives with flammables",
		hazardousmaterial.HazardousClass1,
		hazardousmaterial.HazardousClass3,
		hazmatsegregationrule.SegregationTypeProhibited,
		nil,
		nil,
	)

	entity := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	entity.BOL = "BOL-HAZMAT-CONFLICT"
	entity.Commodities = []*shipment.ShipmentCommodity{
		{CommodityID: leftCommodity.ID, Pieces: 10, Weight: 100},
		{CommodityID: rightCommodity.ID, Pieces: 12, Weight: 120},
	}

	created, err := svc.writer.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID),
	)
	require.Nil(t, created)
	require.Error(t, err)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	shipmentservicetest.AssertErrorField(t, multiErr, "commodities[0].commodityId")
	shipmentservicetest.AssertErrorField(t, multiErr, "commodities[1].commodityId")

	count, getErr := keel.QueryValue[int64](
		ctx,
		db,
		"SELECT count(*) FROM shipments WHERE organization_id = $1 AND business_unit_id = $2 AND bol = $3",
		tenantInfo.OrgID.String(),
		tenantInfo.BuID.String(),
		entity.BOL,
	)
	require.NoError(t, getErr)
	assert.Equal(t, int64(0), count)
}

func TestUpdateIntegration_ReconcilesDetentionCharge(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, shipmentRepo, controlRepo, _, tenantInfo, fixture, data := newIntegrationShipmentService(
		t,
		ctx,
		db,
	)
	ctx = integrationTenantContext(ctx, tenantInfo)
	detentionCharge := testutil.MustCreateAccessorialCharge(
		t,
		ctx,
		db,
		tenantInfo,
		"DETN",
		accessorialcharge.MethodPerUnit,
		accessorialcharge.RateUnitHour,
		"25",
	)
	configureShipmentControl(
		t,
		ctx,
		controlRepo,
		tenantInfo,
		func(sc *shipmentcontrol.ShipmentControl) {
			sc.TrackDetentionTime = true
			sc.AutoGenerateDetentionCharges = true
			sc.DetentionThreshold = int16PtrForIntegration(30)
			sc.DetentionChargeID = &detentionCharge.ID
		},
	)

	now := timeutilsNow()
	entity := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	entity.BOL = "BOL-DETENTION-RECALC"
	entity.Moves[0].Stops[0].ActualArrival = int64PtrForIntegration(now - (30*60 + 90*60))

	created, err := svc.writer.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID),
	)
	require.NoError(t, err)

	loaded, err := shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         created.ID,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	require.NoError(t, err)
	require.Len(t, loaded.AdditionalCharges, 1)
	assert.True(t, loaded.AdditionalCharges[0].IsSystemGenerated)
	assert.Equal(t, detentionCharge.ID, loaded.AdditionalCharges[0].AccessorialChargeID)
	assert.Equal(t, int16(2), loaded.AdditionalCharges[0].Unit)
	assert.True(t, decimal.NewFromInt(50).Equal(loaded.OtherChargeAmount.Decimal))

	actualDeparture := *loaded.Moves[0].Stops[0].ActualArrival + 600
	loaded.Moves[0].Stops[0].ActualDeparture = &actualDeparture

	updated, err := svc.writer.Update(
		ctx,
		loaded,
		internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID),
	)
	require.NoError(t, err)
	require.NotNil(t, updated)

	persisted, err := shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         created.ID,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	require.NoError(t, err)
	assert.Empty(t, persisted.AdditionalCharges)
	assert.True(t, decimal.NewFromInt(0).Equal(persisted.OtherChargeAmount.Decimal))
	assert.True(t, decimal.NewFromInt(0).Equal(persisted.TotalChargeAmount.Decimal))
}

func TestCreateIntegration_RatesWithHydratedCommodityDetails(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, shipmentRepo, controlRepo, _, tenantInfo, fixture, data := newIntegrationShipmentService(
		t,
		ctx,
		db,
	)
	ctx = integrationTenantContext(ctx, tenantInfo)
	configureShipmentControl(
		t,
		ctx,
		controlRepo,
		tenantInfo,
		func(sc *shipmentcontrol.ShipmentControl) {
			sc.CheckForDuplicateBOLs = true
			sc.CheckHazmatSegregation = false
		},
	)

	hazmat := testutil.MustCreateHazardousMaterial(
		t,
		ctx,
		db,
		tenantInfo,
		"HM-RATE",
		"Rate Hazmat",
		hazardousmaterial.HazardousClass3,
	)
	commodityOne := testutil.MustCreateCommodity(
		t,
		ctx,
		db,
		tenantInfo,
		"Rate Commodity One",
		hazmat.ID,
	)
	commodityTwo := testutil.MustCreateCommodity(
		t,
		ctx,
		db,
		tenantInfo,
		"Rate Commodity Two",
		pulid.Nil,
	)

	_, err := keel.Exec(ctx, db,
		"UPDATE commodities SET linear_feet_per_unit = $1 WHERE id = $2",
		1.25, commodityOne.ID.String())
	require.NoError(t, err)

	_, err = keel.Exec(ctx, db,
		"UPDATE commodities SET linear_feet_per_unit = $1 WHERE id = $2",
		2.0, commodityTwo.ID.String())
	require.NoError(t, err)

	entity := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	entity.BOL = "BOL-INTEGRATION-RATE-CREATE"
	entity.Weight = nil
	entity.Pieces = nil
	entity.Commodities = []*shipment.ShipmentCommodity{
		{CommodityID: commodityOne.ID, Pieces: 4, Weight: 800},
		{CommodityID: commodityTwo.ID, Pieces: 2, Weight: 200},
	}

	created, err := svc.writer.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID),
	)
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.True(t, decimal.NewFromFloat(34).Equal(created.FreightChargeAmount.Decimal))
	assert.True(t, decimal.NewFromFloat(34).Equal(created.TotalChargeAmount.Decimal))

	persisted, err := shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         created.ID,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	require.NoError(t, err)
	assert.True(t, decimal.NewFromFloat(34).Equal(persisted.FreightChargeAmount.Decimal))
	assert.True(t, decimal.NewFromFloat(34).Equal(persisted.TotalChargeAmount.Decimal))
	require.NotNil(t, persisted.Commodities[0].Commodity)
}

func TestUpdateIntegration_RecalculatesWithHydratedCommodityDetails(t *testing.T) {
	t.Parallel()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	svc, shipmentRepo, controlRepo, _, tenantInfo, fixture, data := newIntegrationShipmentService(
		t,
		ctx,
		db,
	)
	ctx = integrationTenantContext(ctx, tenantInfo)
	configureShipmentControl(
		t,
		ctx,
		controlRepo,
		tenantInfo,
		func(sc *shipmentcontrol.ShipmentControl) {
			sc.CheckForDuplicateBOLs = true
			sc.CheckHazmatSegregation = false
		},
	)

	commodityOne := testutil.MustCreateCommodity(
		t,
		ctx,
		db,
		tenantInfo,
		"Rate Update One",
		pulid.Nil,
	)
	commodityTwo := testutil.MustCreateCommodity(
		t,
		ctx,
		db,
		tenantInfo,
		"Rate Update Two",
		pulid.Nil,
	)

	_, err := keel.Exec(ctx, db,
		"UPDATE commodities SET linear_feet_per_unit = $1 WHERE id = ANY($2)",
		1.0, []string{commodityOne.ID.String(), commodityTwo.ID.String()})
	require.NoError(t, err)

	entity := makeIntegrationShipment(t, ctx, db, fixture, tenantInfo, data.User.ID)
	entity.BOL = "BOL-INTEGRATION-RATE-UPDATE"
	entity.Weight = nil
	entity.Pieces = nil
	entity.Commodities = []*shipment.ShipmentCommodity{
		{CommodityID: commodityOne.ID, Pieces: 2, Weight: 100},
		{CommodityID: commodityTwo.ID, Pieces: 3, Weight: 150},
	}

	created, err := svc.writer.Create(
		ctx,
		entity,
		internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID),
	)
	require.NoError(t, err)
	assert.True(t, decimal.NewFromFloat(5).Equal(created.FreightChargeAmount.Decimal))

	loaded, err := shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         created.ID,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	require.NoError(t, err)

	_, err = keel.Exec(ctx, db,
		"UPDATE commodities SET linear_feet_per_unit = $1 WHERE id = $2",
		2.0, commodityOne.ID.String())
	require.NoError(t, err)

	loaded.Commodities = []*shipment.ShipmentCommodity{
		{
			ID:          loaded.Commodities[0].ID,
			ShipmentID:  loaded.ID,
			CommodityID: commodityOne.ID,
			Pieces:      4,
			Weight:      120,
		},
		{
			ID:          loaded.Commodities[1].ID,
			ShipmentID:  loaded.ID,
			CommodityID: commodityTwo.ID,
			Pieces:      1,
			Weight:      80,
		},
	}

	updated, err := svc.writer.Update(
		ctx,
		loaded,
		internaltestutil.NewSessionActor(data.User.ID, tenantInfo.OrgID, tenantInfo.BuID),
	)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.True(t, decimal.NewFromFloat(9).Equal(updated.FreightChargeAmount.Decimal))

	persisted, err := shipmentRepo.GetByID(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         created.ID,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	require.NoError(t, err)
	assert.True(t, decimal.NewFromFloat(9).Equal(persisted.FreightChargeAmount.Decimal))
	assert.True(t, decimal.NewFromFloat(9).Equal(persisted.TotalChargeAmount.Decimal))
}

func newIntegrationShipmentService(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
) (
	*rig,
	shipmentports.ShipmentRepository,
	tenantports.ShipmentControlRepository,
	accessorialchargeports.AccessorialChargeRepository,
	pagination.TenantInfo,
	*testutil.ShipmentIntegrationFixture,
	*seedtest.TestData,
) {
	t.Helper()
	return newIntegrationShipmentServiceWith(t, ctx, db, nil)
}

func newIntegrationShipmentServiceWith(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	override func(*rigParams),
) (
	*rig,
	shipmentports.ShipmentRepository,
	tenantports.ShipmentControlRepository,
	accessorialchargeports.AccessorialChargeRepository,
	pagination.TenantInfo,
	*testutil.ShipmentIntegrationFixture,
	*seedtest.TestData,
) {
	t.Helper()

	conn := postgres.NewTestConnection(db)
	moveRepo := shipmentmoverepository.New(shipmentmoverepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	additionalChargeRepo := shipmentadditionalchargerepository.New(
		shipmentadditionalchargerepository.Params{
			DB:     conn,
			Logger: zap.NewNop(),
		},
	)
	shipmentCommodityRepo := shipmentcommodityrepository.New(shipmentcommodityrepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	orderRepo := orderrepository.New(orderrepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	shipmentRepo := shipmentrepository.New(shipmentrepository.Params{
		DB:                         conn,
		Logger:                     zap.NewNop(),
		Generator:                  testutil.TestSequenceGenerator{SingleValue: "PRO-INTEGRATION"},
		MoveRepository:             moveRepo,
		AdditionalChargeRepository: additionalChargeRepo,
		CommodityRepository:        shipmentCommodityRepo,
		OrderRepository:            orderRepo,
		ChargeAllocationRepository: chargeallocationrepository.New(
			chargeallocationrepository.Params{DB: conn, Logger: zap.NewNop()},
		),
	})
	controlRepo := shipmentcontrolrepository.New(shipmentcontrolrepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	accessorialRepo := accessorialchargerepository.New(accessorialchargerepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	commodityRepo := commodityrepository.New(commodityrepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	hazmatRuleRepo := hazmatsegregationrulerepository.New(hazmatsegregationrulerepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	validator := shipmentvalidator.New(shipmentvalidator.Params{
		DB:                        conn,
		ControlRepo:               controlRepo,
		CustomerRepo:              operationstest.NewCustomerRepository(t),
		CommodityRepo:             commodityRepo,
		HazmatSegregationRuleRepo: hazmatRuleRepo,
		ShipmentRepo:              shipmentRepo,
	})

	audit := &mocks.NoopAuditService{}
	commercial := shipmentcommercial.New(shipmentcommercial.Params{
		Logger: zap.NewNop(),
		RateEngine: ratingtest.NewStack(t, ratingtest.StackOptions{
			Any: &tariffports.Template{
				TemplateID:    pulid.MustNew("ft_"),
				VersionNumber: 1,
				Name:          "Linear feet",
				Source:        integrationTariff,
			},
		}).Engine,
		AccessorialRepo: accessorialRepo,
	})

	params := rigParams{
		Logger:          zap.NewNop(),
		DB:              conn,
		Repo:            shipmentRepo,
		UserRepo:        tenantportsmock.NewMockUserRepository(t),
		ControlRepo:     controlRepo,
		CommodityRepo:   commodityRepo,
		AccessorialRepo: accessorialRepo,
		Permissions:     permissionportsmock.NewMockPermissionEngine(t),
		Validator:       validator,
		AuditService:    audit,
		EventService:    noopShipmentEventService{},
		Coordinator:     newStateCoordinator(),
		Commercial:      commercial,
	}
	if override != nil {
		override(&params)
	}
	svc := newRig(&params)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenantInfo := pagination.TenantInfo{
		OrgID: data.Organization.ID,
		BuID:  data.BusinessUnit.ID,
	}
	fixture := testutil.SeedShipmentIntegrationFixture(t, ctx, db, data, tenantInfo)
	insertShipmentControl(t, ctx, db, tenantInfo)

	return svc, shipmentRepo, controlRepo, accessorialRepo, tenantInfo, fixture, data
}

func insertShipmentControl(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	tenantInfo pagination.TenantInfo,
) {
	t.Helper()

	entity := &shipmentcontrol.ShipmentControl{
		OrganizationID:               tenantInfo.OrgID,
		BusinessUnitID:               tenantInfo.BuID,
		MaxShipmentWeightLimit:       80000,
		AutoDelayShipments:           true,
		AutoDelayShipmentsThreshold:  int16PtrForIntegration(30),
		DetentionThreshold:           int16PtrForIntegration(30),
		AutoCancelShipmentsThreshold: int8PtrForIntegration(30),
		CheckForDuplicateBOLs:        true,
		AllowMoveRemovals:            true,
		CheckHazmatSegregation:       true,
	}
	require.NoError(t, shipmentcontroldb.ShipmentControls.Create(
		integrationTenantContext(ctx, tenantInfo),
		db,
		entity,
	))
}

func configureShipmentControl(
	t *testing.T,
	ctx context.Context,
	repo tenantports.ShipmentControlRepository,
	tenantInfo pagination.TenantInfo,
	mutate func(*shipmentcontrol.ShipmentControl),
) {
	t.Helper()

	entity, err := repo.Get(ctx, tenantports.GetShipmentControlRequest{TenantInfo: tenantInfo})
	require.NoError(t, err)
	mutate(entity)
	_, err = repo.Update(ctx, entity)
	require.NoError(t, err)
}

func makeIntegrationShipment(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
	fixture *testutil.ShipmentIntegrationFixture,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) *shipment.Shipment {
	t.Helper()

	origin := testutil.MustCreateLocation(
		t,
		ctx,
		db,
		fixture,
		tenantInfo,
		pulid.MustNew("loc_").String(),
		"Origin",
	)
	destination := testutil.MustCreateLocation(
		t,
		ctx,
		db,
		fixture,
		tenantInfo,
		pulid.MustNew("loc_").String(),
		"Destination",
	)

	return &shipment.Shipment{
		BusinessUnitID:    tenantInfo.BuID,
		OrganizationID:    tenantInfo.OrgID,
		ServiceTypeID:     fixture.ServiceType.ID,
		ShipmentTypeID:    fixture.ShipmentType.ID,
		CustomerID:        fixture.Customer.ID,
		EnteredByID:       userID,
		FormulaTemplateID: fixture.FormulaTemplate.ID,
		Status:            shipmentvocab.StatusNew,
		BOL:               "BOL-INTEGRATION",
		Moves: []*shipment.ShipmentMove{
			{
				Status:   shipmentvocab.MoveStatusNew,
				Loaded:   true,
				Sequence: 0,
				Stops: []*shipment.Stop{
					{
						LocationID:           origin.ID,
						Status:               shipmentvocab.StopStatusNew,
						Type:                 shipmentvocab.StopTypePickup,
						ScheduleType:         shipmentvocab.StopScheduleTypeOpen,
						Sequence:             0,
						ScheduledWindowStart: 1_700_000_000,
						ScheduledWindowEnd:   int64PtrForIntegration(1_700_003_600),
					},
					{
						LocationID:           destination.ID,
						Status:               shipmentvocab.StopStatusNew,
						Type:                 shipmentvocab.StopTypeDelivery,
						ScheduleType:         shipmentvocab.StopScheduleTypeOpen,
						Sequence:             1,
						ScheduledWindowStart: 1_700_007_200,
						ScheduledWindowEnd:   int64PtrForIntegration(1_700_010_800),
					},
				},
			},
		},
	}
}

func integrationTenantContext(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return dbscope.WithTenant(ctx, tenantInfo.DBTenant())
}

func int16PtrForIntegration(v int16) *int16 {
	return &v
}

func int8PtrForIntegration(v int16) *int16 {
	return &v
}

func int64PtrForIntegration(v int64) *int64 {
	return &v
}

func timeutilsNow() int64 {
	return timeutils.NowUnix()
}

const integrationTariff = `tariff "Linear feet"

charge "Linear feet" (quantity: shipment.linearFeet, rate: $1/ft)
charge "Hazmat" = if shipment.hazmat { $25 } else { $0 }
`
