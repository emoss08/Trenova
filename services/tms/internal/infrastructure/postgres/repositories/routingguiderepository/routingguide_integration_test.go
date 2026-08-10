//go:build integration

package routingguiderepository

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func seedCarrier(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	data *seedtest.TestData,
	code string,
) *carrier.Carrier {
	t.Helper()

	entity := &carrier.Carrier{
		BusinessUnitID: data.BusinessUnit.ID,
		OrganizationID: data.Organization.ID,
		Status:         carrier.StatusActive,
		Code:           code,
		Name:           "Test Carrier " + code,
	}
	_, err := db.NewInsert().Model(entity).Exec(ctx)
	require.NoError(t, err, "failed to seed carrier")
	return entity
}

func TestUpdateSwapsEntryRanksWithoutUniqueViolation(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	first := seedCarrier(t, ctx, db, data, "RGSWA")
	second := seedCarrier(t, ctx, db, data, "RGSWB")

	repo := New(Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})

	guide := &tender.RoutingGuide{
		BusinessUnitID:   data.BusinessUnit.ID,
		OrganizationID:   data.Organization.ID,
		Name:             "Rank swap guide",
		OriginState:      "GA",
		DestinationState: "TX",
		Specificity:      tender.SpecificityStatePair,
		Entries: []*tender.RoutingGuideEntry{
			{
				CarrierID:       first.ID,
				Rank:            1,
				RateMethod:      "Flat",
				Rate:            decimal.NewFromInt(1000),
				OfferTTLSeconds: 900,
				Channel:         tender.ChannelEmail,
			},
			{
				CarrierID:       second.ID,
				Rank:            2,
				RateMethod:      "Flat",
				Rate:            decimal.NewFromInt(1200),
				OfferTTLSeconds: 900,
				Channel:         tender.ChannelEmail,
			},
		},
	}

	created, err := repo.Create(ctx, guide)
	require.NoError(t, err)
	require.Len(t, created.Entries, 2)

	firstEntryID := created.Entries[0].ID
	secondEntryID := created.Entries[1].ID
	require.False(t, firstEntryID.IsNil())
	require.False(t, secondEntryID.IsNil())

	created.Entries[0].Rank = 2
	created.Entries[1].Rank = 1

	updated, err := repo.Update(ctx, created)
	require.NoError(t, err, "rank swap must not trip uq_routing_guide_entries_rank")

	ranks := make(map[pulid.ID]int16, 2)
	entries := make([]*tender.RoutingGuideEntry, 0, 2)
	err = db.NewSelect().
		Model(&entries).
		Where("routing_guide_id = ?", updated.ID).
		Scan(ctx)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	for _, entry := range entries {
		ranks[entry.ID] = entry.Rank
	}

	require.Equal(t, int16(2), ranks[firstEntryID], "first entry keeps its identity at rank 2")
	require.Equal(t, int16(1), ranks[secondEntryID], "second entry keeps its identity at rank 1")
}
