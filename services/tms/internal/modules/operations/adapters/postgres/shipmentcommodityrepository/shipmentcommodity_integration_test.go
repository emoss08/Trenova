//go:build integration

package shipmentcommodityrepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentcommodityrepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type harness struct {
	ctx      context.Context
	db       *keel.DB
	repo     shipmentports.ShipmentCommodityRepository
	entity   *shipment.Shipment
	steel    *commodity.Commodity
	lumber   *commodity.Commodity
	otherCtx context.Context
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenant := data.Info()
	fixture := testutil.SeedShipmentIntegrationFixture(t, ctx, db, data, tenant)
	graph := testutil.CreateShipmentGraph(t, ctx, db, fixture, tenant, testutil.ShipmentGraphParams{
		BOL:          "BOL-COM",
		ProNumber:    "PRO-COM",
		ShipmentID:   pulid.MustNew("shp_"),
		MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusNew},
	})

	return &harness{
		ctx: data.Ctx(ctx),
		db:  db,
		repo: shipmentcommodityrepository.New(shipmentcommodityrepository.Params{
			DB:     postgres.NewTestConnection(db),
			Logger: zap.NewNop(),
		}),
		entity:   graph.Shipment,
		steel:    testutil.MustCreateCommodity(t, ctx, db, tenant, "Steel", pulid.Nil),
		lumber:   testutil.MustCreateCommodity(t, ctx, db, tenant, "Lumber", pulid.Nil),
		otherCtx: seedtest.SeedAdditionalTenant(t, ctx, db, "OT").Ctx(ctx),
	}
}

func (h *harness) stored(t *testing.T, ctx context.Context) []*shipment.ShipmentCommodity {
	t.Helper()

	sc := shipmentdb.ShipmentCommodities
	rows, err := sc.Where(sc.ShipmentID.Eq(h.entity.ID)).All(ctx, h.db)
	require.NoError(t, err)
	return rows
}

func TestSyncForShipment_InsertsUpdatesAndDeletesCommodities(t *testing.T) {
	h := newHarness(t)
	length := 12.5
	keep := &shipment.ShipmentCommodity{
		CommodityID: h.steel.ID,
		Weight:      100,
		Pieces:      2,
		LengthFeet:  &length,
	}
	drop := &shipment.ShipmentCommodity{CommodityID: h.lumber.ID, Weight: 50, Pieces: 5}
	h.entity.Commodities = []*shipment.ShipmentCommodity{keep, drop}

	require.NoError(t, h.repo.SyncForShipment(h.ctx, h.entity))
	require.Len(t, h.stored(t, h.ctx), 2)
	assert.Empty(t, h.stored(t, h.otherCtx), "another tenant never sees the rows")

	keep.CommodityID = h.lumber.ID
	keep.Weight = 300
	keep.LengthFeet = nil
	h.entity.Commodities = []*shipment.ShipmentCommodity{keep}
	require.NoError(t, h.repo.SyncForShipment(h.ctx, h.entity))

	rows := h.stored(t, h.ctx)
	require.Len(t, rows, 1)
	assert.Equal(t, keep.ID, rows[0].ID)
	assert.Equal(t, h.lumber.ID, rows[0].CommodityID)
	assert.Equal(t, int64(300), rows[0].Weight)
	assert.Nil(t, rows[0].LengthFeet, "a cleared dimension is stored as NULL")
	assert.Equal(t, int64(1), rows[0].Version)
	assert.Equal(t, int64(1), keep.Version)
}

func TestSyncForShipment_RejectsAnUnknownCommodityRow(t *testing.T) {
	h := newHarness(t)
	h.entity.Commodities = []*shipment.ShipmentCommodity{{
		ID:          pulid.MustNew("sc_"),
		CommodityID: h.steel.ID,
		Weight:      1,
		Pieces:      1,
	}}

	err := h.repo.SyncForShipment(h.ctx, h.entity)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestSyncForShipment_MapsAMissingCommodity(t *testing.T) {
	h := newHarness(t)
	h.entity.Commodities = []*shipment.ShipmentCommodity{{
		CommodityID: pulid.MustNew("com_"),
		Weight:      1,
		Pieces:      1,
	}}

	err := h.repo.SyncForShipment(h.ctx, h.entity)
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
}
