package journalreversalrepository

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalreversal"
	"github.com/emoss08/trenova/pkg/buncolgen/journalreversalcol"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

const originalEntryJoin = `LEFT JOIN journal_entries AS orig_je ON jr.original_journal_entry_id = orig_je.id AND jr.organization_id = orig_je.organization_id AND jr.business_unit_id = orig_je.business_unit_id`

func TestFilteringReversalsByOriginalEntryNumberMatchesTheEntryInTheSameTenant(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(&sql.DB{}, pgdialect.New())
	filter := &pagination.QueryOptions{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.ID("org_1"),
			BuID:  pulid.ID("bu_1"),
		},
		FieldFilters: []domaintypes.FieldFilter{
			{
				Field:    "originalJournalEntry.entryNumber",
				Operator: dbtype.OpContains,
				Value:    "000418",
			},
		},
		Sort: []domaintypes.SortField{
			{Field: "originalJournalEntry.entryNumber", Direction: dbtype.SortDirectionAsc},
		},
	}

	q, err := querybuilder.ApplyCursorFilters(
		db.NewSelect().Model((*journalreversal.Reversal)(nil)),
		journalreversalcol.ReversalTable.Alias,
		filter,
		pagination.CursorInfo{Limit: 25},
		(*journalreversal.Reversal)(nil),
	)
	require.NoError(t, err)

	query := q.String()
	assert.Equal(t, 1, strings.Count(query, originalEntryJoin))
	assert.Contains(t, query, `"orig_je"."entry_number" ILIKE '%000418%'`)
	assert.Contains(t, query, `"orig_je"."entry_number" ASC`)
	assert.Contains(t, query, `jr.organization_id = 'org_1'`)
	assert.Contains(t, query, `jr.business_unit_id = 'bu_1'`)
}

func TestFilteringReversalsByAnotherOriginalEntryFieldStillMatchesOnlyTheEntryNumber(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(&sql.DB{}, pgdialect.New())
	filter := &pagination.QueryOptions{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.ID("org_1"),
			BuID:  pulid.ID("bu_1"),
		},
		FieldFilters: []domaintypes.FieldFilter{
			{
				Field:    "originalJournalEntry.totalDebit",
				Operator: dbtype.OpEqual,
				Value:    "125000",
			},
		},
	}

	q, err := querybuilder.ApplyCursorFilters(
		db.NewSelect().Model((*journalreversal.Reversal)(nil)),
		journalreversalcol.ReversalTable.Alias,
		filter,
		pagination.CursorInfo{Limit: 25},
		(*journalreversal.Reversal)(nil),
	)
	require.NoError(t, err)

	query := q.String()
	assert.NotContains(t, query, "total_debit")
	assert.Contains(t, query, `"orig_je"."entry_number" = '125000'`)
}
