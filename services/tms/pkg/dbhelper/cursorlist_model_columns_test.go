package dbhelper

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/journalreversal"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

/*
A Bun model query stops selecting the model's own columns the moment it is given
any explicit one. CursorList adds the cursor's sort values as explicit columns,
so a caller whose Query was a bare Model(...) scanned the sort values and nothing
else: every page had the right number of rows and every field on them empty.
That is how the inbox, journal reversals and EDI test cases drew blank tables.

These render the page query the way CursorList builds it, with no live database.
*/
func TestSelectWithCursorColumnsKeepsTheModelColumnsOfABareModelQuery(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(new(sql.DB), pgdialect.New())
	filter := &pagination.QueryOptions{}
	entities := make([]*journalreversal.Reversal, 0)

	q, err := querybuilder.ApplyCursorFilters(
		db.NewSelect().Model(&entities),
		"jr",
		filter,
		pagination.CursorInfo{},
		(*journalreversal.Reversal)(nil),
	)
	require.NoError(t, err)
	require.NotEmpty(t, filter.CursorColumns)

	sqlText := selectWithCursorColumns(q, filter.CursorColumns).String()

	assert.Contains(t, sqlText, `"jr"."id"`, sqlText)
	assert.Contains(t, sqlText, `"jr"."original_journal_entry_id"`, sqlText)
	assert.Contains(t, sqlText, `AS "__cursor_value_0"`, sqlText)
	assert.NotContains(t, sqlText, `"jr"."__cursor_value_0"`,
		"the scan-only cursor fields are not table columns and must not be selected as one")
}

func TestSelectWithCursorColumnsKeepsJoinedRelations(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(new(sql.DB), pgdialect.New())
	filter := &pagination.QueryOptions{}
	entities := make([]*edi.EDITestCase, 0)

	q, err := querybuilder.ApplyCursorFilters(
		db.NewSelect().Model(&entities).Relation("DocumentProfile"),
		"etc",
		filter,
		pagination.CursorInfo{},
		(*edi.EDITestCase)(nil),
	)
	require.NoError(t, err)

	sqlText := selectWithCursorColumns(q, filter.CursorColumns).String()

	assert.Contains(t, sqlText, `"etc"."id"`, sqlText)
	assert.Contains(t, sqlText, `"document_profile"."id" AS "document_profile__id"`, sqlText)
	assert.Contains(t, sqlText, "__cursor_value_0", sqlText)
}

func TestSelectWithCursorColumnsLeavesAnExplicitProjectionAlone(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(new(sql.DB), pgdialect.New())
	filter := &pagination.QueryOptions{}
	entities := make([]*journalreversal.Reversal, 0)

	q, err := querybuilder.ApplyCursorFilters(
		db.NewSelect().Model(&entities).Column("id", "status"),
		"jr",
		filter,
		pagination.CursorInfo{},
		(*journalreversal.Reversal)(nil),
	)
	require.NoError(t, err)

	sqlText := selectWithCursorColumns(q, filter.CursorColumns).String()
	selectList := sqlText[:strings.Index(sqlText, " FROM ")]

	assert.Contains(t, selectList, `"jr"."id"`)
	assert.Contains(t, selectList, `"jr"."status"`)
	assert.NotContains(t, selectList, "original_journal_entry_id",
		"a projection the caller chose must not be widened")
}
