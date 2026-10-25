package journalreversalrepository

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalreversal/journalreversaldb"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
)

func TestFilteringReversalsByOriginalEntryNumberJoinsTheEntryInTheSameTenant(t *testing.T) {
	t.Parallel()

	jr := journalreversaldb.Reversals
	q := listquery.Filter(jr.Query(), &pagination.QueryOptions{
		FieldFilters: []domaintypes.FieldFilter{
			{
				Field:    "originalJournalEntry.entryNumber",
				Operator: dbtype.OpContains,
				Value:    "000418",
			},
		},
	})

	sql, args := keeltest.Render(t, q.Limit(25))
	assert.Equal(t, 1, strings.Count(sql, `LEFT JOIN "journal_entries"`))
	assert.Contains(t, sql, `"entry_number"`)
	assert.Contains(t, sql, `"original_journal_entry_id"`)
	assert.Contains(t, args, "%000418%")
}
