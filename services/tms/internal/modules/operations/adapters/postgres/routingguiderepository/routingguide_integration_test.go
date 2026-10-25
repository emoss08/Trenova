//go:build integration

package routingguiderepository

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender/tenderdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func seedCarrier(
	t *testing.T,
	ctx context.Context,
	db *keel.DB,
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
	require.NoError(
		t,
		carrierdb.Carriers.Create(data.Ctx(ctx), db, entity),
		"failed to seed carrier",
	)
	return entity
}

func TestUpdateSwapsEntryRanksWithoutUniqueViolation(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	first := seedCarrier(t, ctx, db, data, "RGSWA")
	second := seedCarrier(t, ctx, db, data, "RGSWB")
	ctx = data.Ctx(ctx)

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
	e := tenderdb.RoutingGuideEntries
	entries, err := e.Where(e.RoutingGuideID.Eq(updated.ID)).All(ctx, db)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	for _, entry := range entries {
		ranks[entry.ID] = entry.Rank
	}

	require.Equal(t, int16(2), ranks[firstEntryID], "first entry keeps its identity at rank 2")
	require.Equal(t, int16(1), ranks[secondEntryID], "second entry keeps its identity at rank 1")
}

func TestRoutingGuidesStayInsideTheirTenant(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	other := seedtest.SeedAdditionalTenant(t, ctx, db, "RG")
	repo := New(Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})

	guide, err := repo.Create(data.Ctx(ctx), &tender.RoutingGuide{
		BusinessUnitID:   data.BusinessUnit.ID,
		OrganizationID:   data.Organization.ID,
		Name:             "Tenant guide",
		OriginState:      "GA",
		DestinationState: "TX",
		Specificity:      tender.SpecificityStatePair,
	})
	require.NoError(t, err)

	_, err = repo.GetByID(other.Ctx(ctx), tenderports.GetRoutingGuideByIDRequest{
		ID:         guide.ID,
		TenantInfo: other.Info(),
	})
	require.True(t, errortypes.IsNotFoundError(err))

	found, err := repo.GetByID(data.Ctx(ctx), tenderports.GetRoutingGuideByIDRequest{
		ID:         guide.ID,
		TenantInfo: data.Info(),
	})
	require.NoError(t, err)
	require.Equal(t, guide.ID, found.ID)
}
