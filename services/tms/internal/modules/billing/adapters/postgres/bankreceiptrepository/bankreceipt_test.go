package bankreceiptrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/keeltest"
)

func TestSummaryQuery_Dump(t *testing.T) {
	sql, args := keeltest.Render(t, summaryQuery(1000))
	t.Log(sql)
	t.Log(args)
}
