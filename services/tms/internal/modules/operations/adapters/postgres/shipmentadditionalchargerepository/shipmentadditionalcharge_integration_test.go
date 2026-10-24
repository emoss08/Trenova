//go:build integration

package shipmentadditionalchargerepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentadditionalchargerepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type harness struct {
	ctx         context.Context
	db          *keel.DB
	repo        shipmentports.ShipmentAdditionalChargeRepository
	entity      *shipment.Shipment
	accessorial *accessorialcharge.AccessorialCharge
	other       *accessorialcharge.AccessorialCharge
	otherCtx    context.Context
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenant := data.Info()
	fixture := testutil.SeedShipmentIntegrationFixture(t, ctx, db, data, tenant)
	graph := testutil.CreateShipmentGraph(t, ctx, db, fixture, tenant, testutil.ShipmentGraphParams{
		BOL:          "BOL-ACC",
		ProNumber:    "PRO-ACC",
		ShipmentID:   pulid.MustNew("shp_"),
		MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusNew},
	})
	graph.Shipment.FreightChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(100))

	return &harness{
		ctx: data.Ctx(ctx),
		db:  db,
		repo: shipmentadditionalchargerepository.New(shipmentadditionalchargerepository.Params{
			DB:     postgres.NewTestConnection(db),
			Logger: zap.NewNop(),
		}),
		entity: graph.Shipment,
		accessorial: testutil.MustCreateAccessorialCharge(
			t, ctx, db, tenant, "LUMP", accessorialcharge.MethodFlat, "", "25",
		),
		other: testutil.MustCreateAccessorialCharge(
			t, ctx, db, tenant, "DET", accessorialcharge.MethodFlat, "", "40",
		),
		otherCtx: seedtest.SeedAdditionalTenant(t, ctx, db, "other").Ctx(ctx),
	}
}

func (h *harness) stored(t *testing.T, ctx context.Context) []*shipment.AdditionalCharge {
	t.Helper()

	ac := shipmentdb.AdditionalCharges
	rows, err := ac.Where(ac.ShipmentID.Eq(h.entity.ID)).All(ctx, h.db)
	require.NoError(t, err)
	return rows
}

func (h *harness) header(t *testing.T) *shipment.Shipment {
	t.Helper()

	stored, err := shipmentdb.Shipments.Get(h.ctx, h.db, h.entity.ID)
	require.NoError(t, err)
	return stored
}

func flat(id pulid.ID, amount int64) *shipment.AdditionalCharge {
	return &shipment.AdditionalCharge{
		AccessorialChargeID: id,
		Method:              accessorialcharge.MethodFlat,
		Amount:              decimal.NewFromInt(amount),
		Unit:                1,
	}
}

func TestSyncForShipment_InsertsChargesAndReconcilesTheHeader(t *testing.T) {
	h := newHarness(t)
	h.entity.AdditionalCharges = []*shipment.AdditionalCharge{
		flat(h.accessorial.ID, 10),
		flat(h.other.ID, 5),
	}

	require.NoError(t, h.repo.SyncForShipment(h.ctx, h.entity))

	assert.Len(t, h.stored(t, h.ctx), 2)
	header := h.header(t)
	assert.True(t, header.OtherChargeAmount.Decimal.Equal(decimal.NewFromInt(15)))
	assert.True(t, header.TotalChargeAmount.Decimal.Equal(decimal.NewFromInt(115)))
	assert.Equal(t, h.entity.Version, header.Version, "reconciling totals is not an edit")
	assert.Empty(t, h.stored(t, h.otherCtx), "another tenant never sees the rows")
}

func TestSyncForShipment_UpdatesTheAccessorialAndDeletesRemovedCharges(t *testing.T) {
	h := newHarness(t)
	keep := flat(h.accessorial.ID, 10)
	drop := flat(h.accessorial.ID, 20)
	h.entity.AdditionalCharges = []*shipment.AdditionalCharge{keep, drop}
	require.NoError(t, h.repo.SyncForShipment(h.ctx, h.entity))

	keep.AccessorialChargeID = h.other.ID
	keep.Amount = decimal.NewFromInt(30)
	h.entity.AdditionalCharges = []*shipment.AdditionalCharge{keep}
	require.NoError(t, h.repo.SyncForShipment(h.ctx, h.entity))

	rows := h.stored(t, h.ctx)
	require.Len(t, rows, 1)
	assert.Equal(t, keep.ID, rows[0].ID)
	assert.Equal(t, h.other.ID, rows[0].AccessorialChargeID)
	assert.True(t, rows[0].Amount.Equal(decimal.NewFromInt(30)))
	assert.Equal(t, int64(1), rows[0].Version)
	assert.True(t, h.header(t).OtherChargeAmount.Decimal.Equal(decimal.NewFromInt(30)))
}

func TestSyncForShipment_LeavesRowsAloneWhenChargesWereNeverLoaded(t *testing.T) {
	h := newHarness(t)
	h.entity.AdditionalCharges = []*shipment.AdditionalCharge{flat(h.accessorial.ID, 10)}
	require.NoError(t, h.repo.SyncForShipment(h.ctx, h.entity))

	h.entity.AdditionalCharges = nil
	require.NoError(t, h.repo.SyncForShipment(h.ctx, h.entity))
	assert.Len(t, h.stored(t, h.ctx), 1)

	h.entity.AdditionalCharges = []*shipment.AdditionalCharge{}
	require.NoError(t, h.repo.SyncForShipment(h.ctx, h.entity))
	assert.Empty(t, h.stored(t, h.ctx))
	assert.True(t, h.header(t).OtherChargeAmount.Decimal.IsZero())
}

func TestSyncForShipment_RejectsAnUnknownCharge(t *testing.T) {
	h := newHarness(t)
	unknown := flat(h.accessorial.ID, 10)
	unknown.ID = pulid.MustNew("ac_")
	h.entity.AdditionalCharges = []*shipment.AdditionalCharge{unknown}

	err := h.repo.SyncForShipment(h.ctx, h.entity)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestSyncForShipment_MapsAMissingAccessorialCharge(t *testing.T) {
	h := newHarness(t)
	h.entity.AdditionalCharges = []*shipment.AdditionalCharge{flat(pulid.MustNew("acc_"), 10)}

	err := h.repo.SyncForShipment(h.ctx, h.entity)
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
}
