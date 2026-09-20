package dbhelper_test

import (
	"database/sql"
	"testing"

	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

type arrayRow struct {
	bun.BaseModel `bun:"table:t"`

	ID    int      `bun:"id"`
	Tasks []string `bun:"tasks,type:TEXT[],array,nullzero"`
}

func render(t *testing.T, value any) string {
	t.Helper()

	db := bun.NewDB(new(sql.DB), pgdialect.New())
	query := db.NewUpdate().Model(&arrayRow{}).
		Set("tasks = ?", value).
		Where("id = ?", 1)

	rendered, err := query.AppendQuery(db.QueryGen(), nil)
	require.NoError(t, err)

	return string(rendered)
}

/*
A Go slice and a pgdialect.Array are indistinguishable at the call site.

Bun renders a bare slice passed to Set as JSON, because Set takes an untyped
value and never sees the field's `array` tag — the tag only steers model-based
inserts. Postgres rejects the JSON against a text[] column, and it does so at
runtime on a user's save, not at compile time:

	malformed array literal: "["DocumentClassification","AssistantChat"]"

That shipped twice: once in a WHERE clause, and once in the UPDATE that saves
an AI provider and an agent definition. Nothing but the rendered SQL can tell
the two apart, so that is what this asserts.
*/
func TestTextArray_RendersAPostgresArrayNotJSON(t *testing.T) {
	t.Parallel()

	sql := render(t, dbhelper.TextArray([]string{"DocumentClassification", "AssistantChat"}))

	assert.Contains(t, sql, `'{"DocumentClassification","AssistantChat"}'`)
	assert.NotContains(t, sql, `["DocumentClassification"`,
		"a JSON array is what Postgres rejects")
}

// The bug, so the test is anchored to the thing it prevents rather than to the
// fix. A bare slice renders as JSON; that is not a supposition.
func TestTextArray_ABareSliceWouldHaveRenderedJSON(t *testing.T) {
	t.Parallel()

	sql := render(t, []string{"DocumentClassification", "AssistantChat"})

	assert.Contains(t, sql, `["DocumentClassification","AssistantChat"]`,
		"this is what the code used to send, and what Postgres refused")
}

// A named string type is what the columns actually hold — Task, EventKind,
// ContextProvider — so the generic parameter is load-bearing, not decoration.
func TestTextArray_AcceptsANamedStringType(t *testing.T) {
	t.Parallel()

	type Task string

	sql := render(t, dbhelper.TextArray([]Task{"AssistantChat"}))

	assert.Contains(t, sql, `'{"AssistantChat"}'`)
}

// Clearing a list writes NULL, which is what the nullzero columns expect. An
// empty array would be a different value: "configured as none" rather than
// "not configured".
func TestTextArray_EmptyBecomesNull(t *testing.T) {
	t.Parallel()

	assert.Nil(t, dbhelper.TextArray([]string{}))
	assert.Nil(t, dbhelper.TextArray[string](nil))
	assert.Contains(t, render(t, dbhelper.TextArray([]string{})), "NULL")
}
