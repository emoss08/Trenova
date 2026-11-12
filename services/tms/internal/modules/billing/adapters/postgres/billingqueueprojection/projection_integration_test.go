//go:build integration

package billingqueueprojection_test

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/billingqueueprojection"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type harness struct {
	ctx  context.Context
	db   *keel.DB
	item *billingqueue.BillingQueueItem
}

func newHarness(t *testing.T, status billingqueue.Status) *harness {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenant := data.Info()
	fixture := testutil.SeedShipmentIntegrationFixture(t, ctx, db, data, tenant)
	graph := testutil.CreateShipmentGraph(t, ctx, db, fixture, tenant, testutil.ShipmentGraphParams{
		BOL:        "BOL-BQPROJ",
		ProNumber:  "PRO-BQPROJ",
		ShipmentID: pulid.MustNew("shp_"),
	})

	tenantCtx := data.Ctx(ctx)
	item := &billingqueue.BillingQueueItem{
		ShipmentID:       graph.Shipment.ID,
		BillToCustomerID: graph.Shipment.CustomerID,
		Number:           "BQ-PROJ-1",
		Status:           status,
		BillType:         billingvocab.BillTypeInvoice,
	}
	require.NoError(t, billingqueuedb.BillingQueueItems.Create(tenantCtx, db, item))

	return &harness{ctx: tenantCtx, db: db, item: item}
}

func (h *harness) lane(t *testing.T) billingqueue.Lane {
	t.Helper()

	bq := billingqueuedb.BillingQueueItems
	lane, err := keel.IntoOne[billingqueue.Lane](
		h.ctx,
		h.db,
		bq.Pick(bq.Lane).Where(bq.ID.Eq(h.item.ID)),
	)
	require.NoError(t, err)

	return lane
}

func TestCreate_LeavesTheLaneToTheProjection(t *testing.T) {
	h := newHarness(t, billingqueue.StatusPosted)

	assert.Equal(t, billingqueue.LaneChecking, h.lane(t),
		"lane is readonly: an insert takes the column default until Refresh runs")
}

func TestRefresh_PlacesAnItemInItsLaneUnderATenantScope(t *testing.T) {
	h := newHarness(t, billingqueue.StatusPosted)

	require.NoError(t, billingqueueprojection.Refresh(h.ctx, h.db, h.item.ID))

	assert.Equal(t, billingqueue.LaneClosed, h.lane(t))
}

func TestRefresh_PlacesAnItemInItsLaneUnderASystemScope(t *testing.T) {
	h := newHarness(t, billingqueue.StatusPosted)

	system := dbscope.WithSystem(context.Background(), "refresh billing queue lanes in a test")
	require.NoError(t, billingqueueprojection.Refresh(system, h.db, h.item.ID))

	assert.Equal(t, billingqueue.LaneClosed, h.lane(t))
}

func TestMarkChecked_MovesAnUnassignedItemToNeedsYouUnderASystemScope(t *testing.T) {
	h := newHarness(t, billingqueue.StatusReadyForReview)

	system := dbscope.WithSystem(context.Background(), "mark billing queue items checked in a test")
	require.NoError(t, billingqueueprojection.MarkChecked(system, h.db, 1_000, h.item.ID))

	assert.Equal(t, billingqueue.LaneNeedsYou, h.lane(t))
}
