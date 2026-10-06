//go:build integration

package billingqueuereviewrepository

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type neighborShipment struct {
	ID      pulid.ID `bun:"id"`
	PayerID pulid.ID `bun:"payer_id"`
}

// The queue reads newest first. Stepping from the middle item lands on the
// one queued after it going up and the one queued before it going down, its
// position counts from the top, and a filter that drops posted items drops
// them from the count and the steps alike.
func TestGetNeighborsWalksTheQueueInItsOwnOrder(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)
	_, err := seeder.NewEngine(
		db, registry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	).Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	var org struct {
		ID             pulid.ID `bun:"id"`
		BusinessUnitID pulid.ID `bun:"business_unit_id"`
	}
	require.NoError(t, db.NewSelect().
		Table("organizations").Column("id", "business_unit_id").Limit(1).Scan(ctx, &org))
	tenant := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}

	var shipments []neighborShipment
	require.NoError(t, db.NewSelect().
		Table("shipments").
		Column("id").
		ColumnExpr("COALESCE(bill_to_customer_id, customer_id) AS payer_id").
		Where("organization_id = ?", org.ID).
		Where("business_unit_id = ?", org.BusinessUnitID).
		Order("created_at ASC").
		Limit(4).
		Scan(ctx, &shipments))
	require.Len(t, shipments, 4)

	statuses := []billingqueue.Status{
		billingqueue.StatusReadyForReview,
		billingqueue.StatusPosted,
		billingqueue.StatusInReview,
		billingqueue.StatusReadyForReview,
	}
	ids := make([]pulid.ID, len(shipments))
	for i, shp := range shipments {
		item := &billingqueue.BillingQueueItem{
			OrganizationID:   org.ID,
			BusinessUnitID:   org.BusinessUnitID,
			ShipmentID:       shp.ID,
			BillToCustomerID: shp.PayerID,
			Status:           statuses[i],
			BillType:         billingqueue.BillTypeInvoice,
			Number:           fmt.Sprintf("BQ-NB-%d", i),
		}
		_, err = db.NewInsert().Model(item).Exec(ctx)
		require.NoError(t, err)
		// Oldest first: item 0 was queued first, item 3 last, and all of
		// them after anything the seed queued.
		_, err = db.NewUpdate().Table("billing_queue_items").
			Set("created_at = ?", 4_000_000_000+int64(i)*60).
			Where("id = ?", item.ID).Exec(ctx)
		require.NoError(t, err)
		ids[i] = item.ID
	}

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	filter := &pagination.QueryOptions{TenantInfo: tenant}

	// With posted items: 3, 2, 1, 0 — item 2 sits second.
	all, err := repo.GetNeighbors(ctx, &repositories.GetBillingQueueNeighborsRequest{
		ItemID: ids[2], Filter: filter, IncludePosted: true,
	})
	require.NoError(t, err)
	require.NotNil(t, all.PrevID)
	require.NotNil(t, all.NextID)
	require.Equal(t, ids[3], *all.PrevID)
	require.Equal(t, ids[1], *all.NextID)
	require.Equal(t, 2, all.Position)

	// Without posted items: 3, 2, 0 — item 1 is skipped over.
	open, err := repo.GetNeighbors(ctx, &repositories.GetBillingQueueNeighborsRequest{
		ItemID: ids[2], Filter: filter,
	})
	require.NoError(t, err)
	require.Equal(t, ids[0], *open.NextID)
	require.Equal(t, 2, open.Position)

	first, err := repo.GetNeighbors(ctx, &repositories.GetBillingQueueNeighborsRequest{
		ItemID: ids[3], Filter: filter,
	})
	require.NoError(t, err)
	require.Nil(t, first.PrevID)
	require.Equal(t, 1, first.Position)
}
