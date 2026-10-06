package seedaccountrepository

import (
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestReferenceColumnsPicksTheColumnBoundToID(t *testing.T) {
	t.Parallel()

	columns := referenceColumns([]catalogReference{
		{ChildTable: "shipments", ChildColumns: []string{"created_by_id"}, ParentColumns: []string{"id"}},
		{ChildTable: "shipments", ChildColumns: []string{"created_by_id"}, ParentColumns: []string{"id"}},
		{
			ChildTable:    "job_positions",
			ChildColumns:  []string{"business_unit_id", "organization_id"},
			ParentColumns: []string{"business_unit_id", "id"},
		},
		{ChildTable: "odd", ChildColumns: []string{"code"}, ParentColumns: []string{"scac_code"}},
		{ChildTable: "broken", ChildColumns: []string{}, ParentColumns: []string{"id"}},
	})

	assert.Equal(t, []referenceColumn{
		{Table: "shipments", Column: "created_by_id"},
		{Table: "job_positions", Column: "organization_id"},
	}, columns)
}

func TestTableSetsDoNotOverlap(t *testing.T) {
	t.Parallel()

	defaults := seedDefaultTables()
	for table := range historyTables() {
		assert.NotContains(t, defaults, table, "history is never a seed default")
	}
	for table := range membershipTables() {
		assert.NotContains(t, defaults, table, "memberships are judged by their owner")
	}
	assert.NotContains(t, defaults, "users")
	assert.NotContains(t, defaults, "organizations")
	assert.Contains(t, ownedByUserTables(), "user_organization_memberships")
}

func TestRetainedByRule(t *testing.T) {
	t.Parallel()

	assert.True(t, retainedByRule(&pgconn.PgError{Code: pgerrcode.ForeignKeyViolation}))
	assert.True(t, retainedByRule(&pgconn.PgError{Code: pgerrcode.RaiseException}))
	assert.True(t, retainedByRule(&pgconn.PgError{Code: pgerrcode.RestrictViolation}))
	assert.False(t, retainedByRule(&pgconn.PgError{Code: pgerrcode.SerializationFailure}))
}
