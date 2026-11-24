package accountsreceivablerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/keeltest"
)

func TestAgingAtTwoDatesReadsOneWindowStepAsLongAsTheGapBetweenThem(t *testing.T) {
	keeltest.Snapshot(t, agingTrendQuery(openAsOfWindow{
		asOf:   1_795_449_600,
		step:   1_795_449_600 - 1_792_771_200,
		points: 2,
	}))
}

func TestCollectionTotalsReadTheOpenBalanceAsOfBothEndsOfThePeriod(t *testing.T) {
	keeltest.Snapshot(t, collectionTotalsQuery(1_787_587_200, 1_795_449_600))
}
