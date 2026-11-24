package settlementreleaserepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/settlementreleaseports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestASettlementIsReadOnlyThroughItsDriver(t *testing.T) {
	t.Parallel()

	repo := &repository{}
	for _, disputes := range []bool{false, true} {
		sql, _ := keeltest.Render(t, repo.settlementQuery(disputes))

		assert.Contains(t, sql, `JOIN "workers"`,
			"a settlement whose driver is outside the reader's row scope reads as nothing")
		assert.NotContains(t, sql, `LEFT JOIN "workers"`)
	}
}

func TestOnlyADisputeReadDecidesThatADisputeKeepsASettlement(t *testing.T) {
	t.Parallel()

	repo := &repository{}
	without, _ := keeltest.Render(t, repo.settlementQuery(false))
	with, _ := keeltest.Render(t, repo.settlementQuery(true))

	assert.NotContains(t, without, `"settlement_disputes"`,
		"a reader not told it may read disputes never reads them")
	assert.Contains(t, with, `"settlement_disputes"`)
}

func TestABatchRegeneratedInAnotherZoneIsTheSamePayPeriod(t *testing.T) {
	t.Parallel()

	periods := periodsOf([]batchRow{
		{Name: "Pay period ending Nov 14, 2026", PeriodStart: 1_794_124_800},
		{Name: "Pay period ending Nov 14, 2026", PeriodStart: 1_794_096_000},
		{Name: "Pay period ending Nov 7, 2026", PeriodStart: 1_793_516_400},
	})

	require.Len(t, periods, 2)
	assert.Len(t, periods[0], 2, "both batches of the week of Nov 8 are one period")
	assert.Len(t, periods[1], 1)
}

func TestAWeekAddsItsBatchesStoredTotalsAndNamesWhatCameOff(t *testing.T) {
	t.Parallel()

	week := weekOf([]batchRow{
		{
			PeriodStart: 100, PeriodEnd: 700, PayDate: 900, Currency: "USD",
			GrossMinor: 19_699_593, ReimbursementsMinor: 448_200, NetMinor: 17_069_215,
		},
		{
			PeriodStart: 90, PeriodEnd: 690, PayDate: 880, Currency: "USD",
			GrossMinor: 880_174, NetMinor: 792_074,
		},
	})

	assert.Equal(t, int64(2), week.Batches)
	assert.Equal(t, int64(90), week.PeriodStart)
	assert.Equal(t, int64(900), week.PayDate)
	assert.Equal(t, int64(21_027_967), week.EarnedMinor())
	assert.Equal(t, int64(21_027_967-17_861_289), week.DeductedMinor())
	assert.Equal(t, int64(1), week.Currencies)
}

func TestABatchesNamesAreSaidOnceAndItsBlocksAreBothKinds(t *testing.T) {
	t.Parallel()

	batch := batchOf([]batchRow{
		{Name: "Pay period ending Nov 14, 2026", PeriodEnd: 10, PayDate: 20},
		{Name: "Pay period ending Nov 14, 2026", PeriodEnd: 12, PayDate: 19},
	}, &splitRow{Settlements: 3, Critical: 1, CriticalNet: 0, Disputed: 1, DisputedNet: 141_250})

	assert.Equal(t, "Pay period ending Nov 14, 2026", batch.Name)
	assert.Equal(t, int64(12), batch.PeriodEnd)
	assert.Equal(t, int64(20), batch.PayDate)
	assert.Equal(t, settlementreleaseports.Tally{Count: 2, NetMinor: 141_250}, batch.BlockedTotal())
	for _, block := range driversettlement.ReleaseBlocks() {
		assert.Contains(t, batch.Blocked, block)
	}
}
