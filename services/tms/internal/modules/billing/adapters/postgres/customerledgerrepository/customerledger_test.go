package customerledgerrepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
)

func TestTotalsReadTheMonthlyBalances(t *testing.T) {
	keeltest.Snapshot(t, customerTotalsQuery([]pulid.ID{"cus_1", "cus_2"}))
}

func TestFactoredBalancesReadOnlyInvoicesStillCarryingABalance(t *testing.T) {
	keeltest.Snapshot(t, nonRecourseFactoredQuery([]pulid.ID{"cus_1"}))
}

func TestBalanceAsOfReadsClosedMonthsAndThePartialMonth(t *testing.T) {
	cut := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC).Unix()
	t.Run("closed months", func(t *testing.T) {
		keeltest.Snapshot(t, closedMonthsBeforeQuery(cut))
	})
	t.Run("partial month", func(t *testing.T) {
		keeltest.Snapshot(
			t,
			entriesBetweenQuery(cut, time.Date(2026, time.March, 14, 12, 0, 0, 0, time.UTC).Unix()),
		)
	})
}
