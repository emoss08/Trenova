//go:build integration

package dispatchplanpricerepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/dispatchplanpricerepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type scope struct {
	ctx    context.Context
	tenant pagination.TenantInfo
	moves  []pulid.ID
}

type harness struct {
	repo  dispatchplanports.PriceRepository
	own   scope
	other scope
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	seed := func(data *seedtest.TestData, code string) scope {
		tenant := data.Info()
		fixture := testutil.SeedShipmentIntegrationFixture(t, ctx, db, data, tenant)
		graph := testutil.CreateShipmentGraph(
			t,
			ctx,
			db,
			fixture,
			tenant,
			testutil.ShipmentGraphParams{
				BOL:        "BOL-" + code,
				ProNumber:  "PRO-" + code,
				ShipmentID: pulid.MustNew("shp_"),
				MoveStatuses: []shipmentvocab.MoveStatus{
					shipmentvocab.MoveStatusNew,
					shipmentvocab.MoveStatusNew,
					shipmentvocab.MoveStatusNew,
				},
			},
		)
		moves := make([]pulid.ID, 0, len(graph.Moves))
		for _, move := range graph.Moves {
			moves = append(moves, move.ID)
		}

		return scope{ctx: data.Ctx(ctx), tenant: tenant, moves: moves}
	}

	return &harness{
		repo: dispatchplanpricerepository.New(dispatchplanpricerepository.Params{
			DB:     postgres.WrapDB(db),
			Logger: zap.NewNop(),
		}),
		own:   seed(seedtest.SeedFullTestData(t, ctx, db), "DPP"),
		other: seed(seedtest.SeedAdditionalTenant(t, ctx, db, "DP"), "DPP-OT"),
	}
}

func covered(
	s scope,
	moveID pulid.ID,
	workerName string,
	costCents, generatedAt int64,
) *dispatchplanprice.DispatchPlanPrice {
	return &dispatchplanprice.DispatchPlanPrice{
		OrganizationID:     s.tenant.OrgID,
		BusinessUnitID:     s.tenant.BuID,
		MoveID:             moveID,
		WorkerID:           pulid.MustNew("wrk_"),
		WorkerName:         workerName,
		CostCents:          costCents,
		CoveragePriceCents: 40_000,
		DriverValueCents:   1_500,
		Alternatives:       []dispatchplanprice.Alternative{},
		GeneratedAt:        generatedAt,
	}
}

func uncovered(s scope, moveID pulid.ID, generatedAt int64) *dispatchplanprice.DispatchPlanPrice {
	return &dispatchplanprice.DispatchPlanPrice{
		OrganizationID:     s.tenant.OrgID,
		BusinessUnitID:     s.tenant.BuID,
		MoveID:             moveID,
		WorkerName:         "Nearest Driver",
		CostCents:          90_000,
		CoveragePriceCents: 40_000,
		UncoveredReason:    dispatchplanprice.UncoveredNotWorthIt,
		Alternatives:       []dispatchplanprice.Alternative{},
		GeneratedAt:        generatedAt,
	}
}

func (h *harness) replace(
	t *testing.T,
	s scope,
	generatedAt int64,
	moveIDs []pulid.ID,
	prices ...*dispatchplanprice.DispatchPlanPrice,
) {
	t.Helper()

	require.NoError(t, h.repo.Replace(s.ctx, &dispatchplanports.ReplacePricesRequest{
		TenantInfo:  s.tenant,
		GeneratedAt: generatedAt,
		MoveIDs:     moveIDs,
		Prices:      prices,
	}))
}

func (h *harness) stored(
	t *testing.T,
	s scope,
	moveIDs ...pulid.ID,
) map[pulid.ID]*dispatchplanprice.DispatchPlanPrice {
	t.Helper()

	prices, err := h.repo.ListByMoves(s.ctx, &dispatchplanports.ListPricesRequest{
		TenantInfo: s.tenant,
		MoveIDs:    moveIDs,
	})
	require.NoError(t, err)

	byMove := make(map[pulid.ID]*dispatchplanprice.DispatchPlanPrice, len(prices))
	for _, price := range prices {
		_, duplicate := byMove[price.MoveID]
		require.False(t, duplicate, "one price per move")
		byMove[price.MoveID] = price
	}

	return byMove
}

func TestReplace_RewritesTheMovesItPlansAndLeavesTheRest(t *testing.T) {
	h := newHarness(t)
	m0, m1, m2 := h.own.moves[0], h.own.moves[1], h.own.moves[2]

	h.replace(t, h.own, 1_000, []pulid.ID{m0, m1},
		covered(h.own, m0, "Ada", 1_200, 1_000),
		uncovered(h.own, m1, 1_000),
	)
	first := h.stored(t, h.own, m0, m1, m2)
	require.Len(t, first, 2)
	assert.Equal(t, "Ada", first[m0].WorkerName)
	assert.Equal(t, dispatchplanprice.UncoveredNotWorthIt, first[m1].UncoveredReason)

	h.replace(t, h.own, 2_000, []pulid.ID{m0}, covered(h.own, m0, "Bo", 800, 2_000))
	second := h.stored(t, h.own, m0, m1, m2)
	require.Len(t, second, 2)
	assert.Equal(t, "Bo", second[m0].WorkerName, "the newer plan replaces the move's price")
	assert.Equal(t, int64(800), second[m0].CostCents)
	assert.Equal(t, int64(2_000), second[m0].GeneratedAt)
	assert.Equal(t, int64(1_000), second[m1].GeneratedAt,
		"a move outside the newer plan keeps its price")

	h.replace(t, h.own, 3_000, []pulid.ID{m1})
	third := h.stored(t, h.own, m0, m1, m2)
	require.Len(t, third, 1, "a planned move with no price row is cleared")
	assert.Contains(t, third, m0)
}

func TestReplace_AnOlderPlanNeverOverwritesANewerPrice(t *testing.T) {
	h := newHarness(t)
	m0, m1 := h.own.moves[0], h.own.moves[1]

	h.replace(t, h.own, 2_000, []pulid.ID{m0}, covered(h.own, m0, "Newer", 200, 2_000))
	h.replace(t, h.own, 1_000, []pulid.ID{m0, m1},
		covered(h.own, m0, "Older", 100, 1_000),
		covered(h.own, m1, "Older", 100, 1_000),
	)

	prices := h.stored(t, h.own, m0, m1)
	require.Len(t, prices, 2)
	assert.Equal(t, "Newer", prices[m0].WorkerName, "a late-finishing older plan loses")
	assert.Equal(t, int64(2_000), prices[m0].GeneratedAt)
	assert.Equal(t, "Older", prices[m1].WorkerName,
		"the older plan still prices a move nothing newer has priced")
}

func TestReplace_StoresEveryPriceField(t *testing.T) {
	h := newHarness(t)
	m0 := h.own.moves[0]
	price := covered(h.own, m0, "Ada", 1_234, 5_000)
	price.Alternatives = []dispatchplanprice.Alternative{
		{WorkerID: pulid.MustNew("wrk_"), WorkerName: "Bo", ExtraCents: 650},
		{WorkerID: pulid.MustNew("wrk_"), WorkerName: "Cy", ExtraCents: 900},
	}

	h.replace(t, h.own, 5_000, []pulid.ID{m0}, price)

	stored := h.stored(t, h.own, m0)[m0]
	require.NotNil(t, stored)
	assert.False(t, stored.ID.IsNil())
	assert.Equal(t, price.WorkerID, stored.WorkerID)
	assert.True(t, stored.Covered())
	assert.Equal(t, int64(1_234), stored.CostCents)
	assert.Equal(t, int64(40_000), stored.CoveragePriceCents)
	assert.Equal(t, int64(1_500), stored.DriverValueCents)
	assert.Empty(t, stored.UncoveredReason)
	assert.Equal(t, price.Alternatives, stored.Alternatives)
	assert.Equal(t, int64(5_000), stored.GeneratedAt)
}

func TestReplace_StoresAnUncoveredMoveWithoutAWorker(t *testing.T) {
	h := newHarness(t)
	m0 := h.own.moves[0]

	h.replace(t, h.own, 1_000, []pulid.ID{m0}, uncovered(h.own, m0, 1_000))

	stored := h.stored(t, h.own, m0)[m0]
	require.NotNil(t, stored)
	assert.True(t, stored.WorkerID.IsNil())
	assert.False(t, stored.Covered())
	assert.Equal(t, "Nearest Driver", stored.WorkerName)
	assert.Equal(t, dispatchplanprice.UncoveredNotWorthIt, stored.UncoveredReason)
	assert.Empty(t, stored.Alternatives)
}

func TestListByMoves_ReturnsOnlyTheRequestedMovesOfTheTenant(t *testing.T) {
	h := newHarness(t)
	m0, m1, m2 := h.own.moves[0], h.own.moves[1], h.own.moves[2]
	h.replace(t, h.own, 1_000, []pulid.ID{m0, m1, m2},
		covered(h.own, m0, "Ada", 100, 1_000),
		covered(h.own, m1, "Bo", 200, 1_000),
		covered(h.own, m2, "Cy", 300, 1_000),
	)
	theirs := h.other.moves[0]
	h.replace(t, h.other, 1_000, []pulid.ID{theirs}, covered(h.other, theirs, "Dee", 400, 1_000))

	prices := h.stored(t, h.own, m0, m2)
	require.Len(t, prices, 2)
	assert.Contains(t, prices, m0)
	assert.Contains(t, prices, m2)

	assert.Empty(t, h.stored(t, h.own, theirs), "another tenant's price is never read")
	assert.Empty(t, h.stored(t, h.other, m0, m1, m2), "another tenant cannot read ours")
	require.Len(t, h.stored(t, h.other, theirs), 1)
}

func TestReplace_NeverClearsAnotherTenantsPrices(t *testing.T) {
	h := newHarness(t)
	m0 := h.own.moves[0]
	h.replace(t, h.own, 1_000, []pulid.ID{m0}, covered(h.own, m0, "Ada", 100, 1_000))

	h.replace(t, h.other, 2_000, []pulid.ID{m0})

	require.Len(t, h.stored(t, h.own, m0), 1,
		"a later plan in another tenant naming our move deletes nothing of ours")
}
