package bankreceiptrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/keeltest"
)

func TestSummaryQuery(t *testing.T) {
	keeltest.Snapshot(t, summaryQuery(1000))
}
