package bankreceiptbatchrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptbatch/bankreceiptbatchdb"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
)

func TestSourceOptions_PagesTheDistinctSourcesInSQL(t *testing.T) {
	t.Parallel()

	page := sourceOptions(" lockbox ").
		Order(bankreceiptbatchdb.BankReceiptBatches.Source.Asc()).
		Limit(20).
		Offset(40)
	sql, args := keeltest.Render(t, page)

	assert.Contains(t, sql, `GROUP BY`)
	assert.Contains(t, sql, `"source"`)
	assert.Contains(t, sql, "LIMIT 20")
	assert.Contains(t, sql, "OFFSET 40")
	assert.Contains(t, args, "%lockbox%", "the search is trimmed before it narrows the sources")
}

func TestSourceOptions_ListsEverySourceWithoutASearch(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, sourceOptions("  "))

	assert.NotContains(t, sql, "ILIKE")
	assert.Contains(t, sql, `GROUP BY`)
}
