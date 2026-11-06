package dispatchconsoleservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/internal/testutil/mocks/dispatchplanportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func boardMove(covered bool) *BoardMove {
	move := &dispatchboardports.BoardMove{
		MoveID:    pulid.MustNew("smv_"),
		ProNumber: "PRO-1",
	}
	if covered {
		move.AssignmentID = pulid.MustNew("asn_")
	}

	return decorateMove(move, testNow)
}

func priceService(t *testing.T) (*Service, *dispatchplanportsmock.MockPriceRepository) {
	t.Helper()

	prices := dispatchplanportsmock.NewMockPriceRepository(t)

	return &Service{prices: prices, l: zap.NewNop()}, prices
}

func TestAttachPlanPrices_ReadsOnlyUncoveredMovesAndAttachesByMove(t *testing.T) {
	t.Parallel()

	svc, prices := priceService(t)
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	first, covered, second, unpriced := boardMove(false), boardMove(true), boardMove(false),
		boardMove(false)
	moves := []*BoardMove{first, covered, second, unpriced}

	secondPrice := &dispatchplanprice.DispatchPlanPrice{MoveID: second.MoveID, CostCents: 200}
	firstPrice := &dispatchplanprice.DispatchPlanPrice{MoveID: first.MoveID, CostCents: 100}
	prices.EXPECT().
		ListByMoves(mock.Anything, &dispatchplanports.ListPricesRequest{
			TenantInfo: tenant,
			MoveIDs:    []pulid.ID{first.MoveID, second.MoveID, unpriced.MoveID},
		}).
		Return([]*dispatchplanprice.DispatchPlanPrice{secondPrice, firstPrice}, nil).
		Once()

	require.NoError(t, svc.attachPlanPrices(t.Context(), tenant, moves))

	assert.Same(t, firstPrice, first.PlanPrice)
	assert.Same(t, secondPrice, second.PlanPrice)
	assert.Nil(t, unpriced.PlanPrice, "a move the plan has not priced shows no price")
	assert.Nil(t, covered.PlanPrice, "a covered move is never priced")
}

func TestAttachPlanPrices_SkipsTheReadWhenEveryMoveIsCovered(t *testing.T) {
	t.Parallel()

	svc, prices := priceService(t)

	require.NoError(t, svc.attachPlanPrices(
		t.Context(),
		pagination.TenantInfo{},
		[]*BoardMove{boardMove(true), boardMove(true)},
	))
	require.NoError(t, svc.attachPlanPrices(t.Context(), pagination.TenantInfo{}, nil))

	prices.AssertNotCalled(t, "ListByMoves", mock.Anything, mock.Anything)
}

func TestAttachPlanPrices_ReturnsTheReadError(t *testing.T) {
	t.Parallel()

	svc, prices := priceService(t)
	boom := errors.New("read failed")
	prices.EXPECT().ListByMoves(mock.Anything, mock.Anything).Return(nil, boom)
	move := boardMove(false)

	err := svc.attachPlanPrices(t.Context(), pagination.TenantInfo{}, []*BoardMove{move})

	require.ErrorIs(t, err, boom)
	assert.Nil(t, move.PlanPrice)
}

func TestAttachPlannedMoves_PicksTheNewestPricePerDriverAndSkipsUncovered(t *testing.T) {
	t.Parallel()

	busy := &BoardDriver{
		BoardDriver: &dispatchboardports.BoardDriver{WorkerID: pulid.MustNew("wrk_")},
	}
	idle := &BoardDriver{
		BoardDriver: &dispatchboardports.BoardDriver{WorkerID: pulid.MustNew("wrk_")},
	}
	named := &BoardDriver{
		BoardDriver: &dispatchboardports.BoardDriver{WorkerID: pulid.MustNew("wrk_")},
	}

	newer, older, uncovered, unpriced := boardMove(false), boardMove(false), boardMove(false),
		boardMove(false)
	newer.PlanPrice = &dispatchplanprice.DispatchPlanPrice{
		MoveID:           newer.MoveID,
		WorkerID:         busy.WorkerID,
		CostCents:        1_200,
		DriverValueCents: 4_000,
		GeneratedAt:      2_000,
	}
	older.PlanPrice = &dispatchplanprice.DispatchPlanPrice{
		MoveID:      older.MoveID,
		WorkerID:    busy.WorkerID,
		CostCents:   900,
		GeneratedAt: 1_000,
	}
	uncovered.PlanPrice = &dispatchplanprice.DispatchPlanPrice{
		MoveID:          uncovered.MoveID,
		WorkerName:      "Named Driver",
		CostCents:       5_000,
		UncoveredReason: dispatchplanprice.UncoveredNotWorthIt,
		GeneratedAt:     3_000,
	}

	for _, order := range [][]*BoardMove{
		{newer, older, uncovered, unpriced},
		{older, newer, unpriced, uncovered},
	} {
		busy.PlannedMove, idle.PlannedMove, named.PlannedMove = nil, nil, nil

		attachPlannedMoves([]*BoardDriver{busy, idle, named}, order)

		require.NotNil(t, busy.PlannedMove)
		assert.Equal(t, &PlannedMove{
			MoveID:      newer.MoveID,
			ProNumber:   newer.ProNumber,
			CostCents:   1_200,
			ValueCents:  4_000,
			GeneratedAt: 2_000,
		}, busy.PlannedMove, "the newest plan's move wins whatever the board order")
		assert.Nil(t, idle.PlannedMove)
		assert.Nil(t, named.PlannedMove, "an uncovered move's best driver is not planned onto it")
	}
}

func TestAttachPlannedMoves_LeavesDriversAloneWithoutAnyPlan(t *testing.T) {
	t.Parallel()

	driver := &BoardDriver{
		BoardDriver: &dispatchboardports.BoardDriver{WorkerID: pulid.MustNew("wrk_")},
	}

	attachPlannedMoves([]*BoardDriver{driver}, []*BoardMove{boardMove(false)})

	assert.Nil(t, driver.PlannedMove)
}
