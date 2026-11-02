//go:build integration

package billingqueuerepository

import (
	"cmp"
	"slices"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const windowStart = int64(2_100_000_000)

type windowShipment struct {
	id         pulid.ID
	customerID pulid.ID
	payerID    pulid.ID
}

func (f *syncFixture) unqueuedShipments(t *testing.T, n int) []windowShipment {
	t.Helper()

	rows, err := keel.Pool(f.db).Query(f.ctx,
		`SELECT s.id, s.customer_id, COALESCE(s.bill_to_customer_id, s.customer_id) FROM shipments s
		WHERE s.organization_id = $1 AND s.business_unit_id = $2
			AND NOT EXISTS (SELECT 1 FROM billing_queue_items b WHERE b.shipment_id = s.id)
		ORDER BY s.id LIMIT $3`,
		f.tenant.OrgID, f.tenant.BuID, n,
	)
	require.NoError(t, err)
	defer rows.Close()

	shipments := make([]windowShipment, 0, n)
	for rows.Next() {
		var s windowShipment
		require.NoError(t, rows.Scan(&s.id, &s.customerID, &s.payerID))
		shipments = append(shipments, s)
	}
	require.NoError(t, rows.Err())
	require.Len(t, shipments, n, "the seed has shipments with no queue item")

	return shipments
}

func (f *syncFixture) queueDelivered(
	t *testing.T,
	s windowShipment,
	number string,
	delivered int64,
) *billingqueue.BillingQueueItem {
	t.Helper()

	_, err := keel.Pool(f.db).Exec(f.ctx,
		`UPDATE shipments SET actual_delivery_date = $2 WHERE id = $1`, s.id, delivered)
	require.NoError(t, err)

	item, err := f.repo.Create(f.ctx, &billingqueue.BillingQueueItem{
		OrganizationID:   f.tenant.OrgID,
		BusinessUnitID:   f.tenant.BuID,
		ShipmentID:       s.id,
		BillToCustomerID: s.payerID,
		Status:           billingqueue.StatusReadyForReview,
		BillType:         billingvocab.BillTypeInvoice,
		Number:           number,
	})
	require.NoError(t, err)

	return item
}

type itemCopies struct {
	CustomerID  *string
	DeliveredAt *int64
	ShippedAt   *int64
}

func (f *syncFixture) copiesOf(t *testing.T, id pulid.ID) itemCopies {
	t.Helper()

	var c itemCopies
	require.NoError(t, keel.Pool(f.db).QueryRow(f.ctx,
		`SELECT shipment_customer_id, shipment_delivered_at, shipment_shipped_at
		FROM billing_queue_items WHERE id = $1`, id,
	).Scan(&c.CustomerID, &c.DeliveredAt, &c.ShippedAt))

	return c
}

func TestShipmentCopies_AreSetOnInsertAndFollowTheShipment(t *testing.T) {
	t.Parallel()

	f := setupSyncFixture(t)
	s := f.unqueuedShipments(t, 1)[0]
	item := f.queueDelivered(t, s, "BQ-COPY-1", windowStart)

	copies := f.copiesOf(t, item.ID)
	require.NotNil(t, copies.CustomerID)
	assert.Equal(t, s.customerID.String(), *copies.CustomerID, "the item copies its shipment's customer")
	require.NotNil(t, copies.DeliveredAt)
	assert.Equal(t, windowStart, *copies.DeliveredAt)
	assert.Nil(t, copies.ShippedAt)

	_, err := keel.Pool(f.db).Exec(f.ctx,
		`UPDATE shipments SET actual_delivery_date = $2, actual_ship_date = $3 WHERE id = $1`,
		s.id, windowStart+86_400, windowStart-86_400)
	require.NoError(t, err)

	copies = f.copiesOf(t, item.ID)
	require.NotNil(t, copies.DeliveredAt)
	assert.Equal(t, windowStart+86_400, *copies.DeliveredAt, "a changed delivery reaches the item")
	require.NotNil(t, copies.ShippedAt)
	assert.Equal(t, windowStart-86_400, *copies.ShippedAt, "a recorded pickup reaches the item")

	_, err = keel.Pool(f.db).Exec(f.ctx,
		`UPDATE shipments SET actual_delivery_date = NULL WHERE id = $1`, s.id)
	require.NoError(t, err)
	assert.Nil(t, f.copiesOf(t, item.ID).DeliveredAt, "a cleared delivery clears the copy")
}

func TestConnection_ADeliveryWindowPagesInQueueOrderUnderAndOverItsCap(t *testing.T) {
	t.Parallel()

	f := setupSyncFixture(t)
	shipments := f.unqueuedShipments(t, 4)
	inWindow := []*billingqueue.BillingQueueItem{
		f.queueDelivered(t, shipments[0], "BQ-WINDOW-1", windowStart+3),
		f.queueDelivered(t, shipments[1], "BQ-WINDOW-2", windowStart+1),
		f.queueDelivered(t, shipments[2], "BQ-WINDOW-3", windowStart+2),
	}
	outside := f.queueDelivered(t, shipments[3], "BQ-WINDOW-4", windowStart+5*86_400)

	slices.SortFunc(inWindow, func(a, b *billingqueue.BillingQueueItem) int {
		return cmp.Or(cmp.Compare(a.CreatedAt, b.CreatedAt), cmp.Compare(a.ID, b.ID))
	})
	want := make([]pulid.ID, 0, len(inWindow))
	for _, item := range inWindow {
		want = append(want, item.ID)
	}

	window := domaintypes.FieldFilter{
		Field:    "shipment.actualDeliveryDate",
		Operator: dbtype.OpDateRange,
		Value:    map[string]any{"from": float64(windowStart), "to": float64(windowStart + 10)},
	}
	pageAll := func(t *testing.T, windowCap int) []pulid.ID {
		t.Helper()

		var ids []pulid.ID
		after := ""
		for range len(want) + 1 {
			cursor, err := pagination.NewCursorInfo(2, after)
			require.NoError(t, err)
			result, err := f.repo.connection(f.ctx, &billingqueueports.ListBillingQueueItemsRequest{
				Filter: &pagination.QueryOptions{
					TenantInfo:   f.tenant,
					FieldFilters: []domaintypes.FieldFilter{window},
				},
				Cursor: cursor,
			}, windowCap)
			require.NoError(t, err)
			for _, item := range result.Items {
				ids = append(ids, item.ID)
			}
			if !result.HasNextPage {
				return ids
			}
			last, ok := result.EdgeCursors.EdgeCursorAt(len(result.Items) - 1)
			require.True(t, ok)
			after = last
		}
		t.Fatalf("the window never ran out of pages")
		return nil
	}

	cases := []struct {
		name      string
		windowCap int
	}{
		{name: "under the cap the page is read from the window's ids", windowCap: 10},
		{name: "at the cap the window still fits", windowCap: len(want)},
		{name: "over the cap the page is read in queue order", windowCap: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pageAll(t, tc.windowCap)
			assert.Equal(t, want, got, "oldest first across pages")
			assert.NotContains(t, got, outside.ID)
		})
	}
}
