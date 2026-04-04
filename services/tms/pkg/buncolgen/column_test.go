package buncolgen

import (
	"testing"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testCol = NewColumn("first_name", "wrk")
var testCol2 = NewColumn("last_name", "wrk")
var testCol3 = NewColumn("status", "wrk")

func TestColumnBasicAccessors(t *testing.T) {
	assert.Equal(t, "first_name", testCol.String())
	assert.Equal(t, "first_name", testCol.Bare())
	assert.Equal(t, "wrk.first_name", testCol.Qualified())
}

func TestColumnWhereFragments(t *testing.T) {
	assert.Equal(t, "wrk.first_name = ?", testCol.Eq())
	assert.Equal(t, "wrk.first_name != ?", testCol.Ne())
	assert.Equal(t, "wrk.first_name <> ?", testCol.NotEq())
	assert.Equal(t, "wrk.first_name > ?", testCol.Gt())
	assert.Equal(t, "wrk.first_name >= ?", testCol.Gte())
	assert.Equal(t, "wrk.first_name < ?", testCol.Lt())
	assert.Equal(t, "wrk.first_name <= ?", testCol.Lte())
	assert.Equal(t, "wrk.first_name IN (?)", testCol.In())
	assert.Equal(t, "wrk.first_name NOT IN (?)", testCol.NotIn())
	assert.Equal(t, "wrk.first_name IS NULL", testCol.IsNull())
	assert.Equal(t, "wrk.first_name IS NOT NULL", testCol.IsNotNull())
	assert.Equal(t, "wrk.first_name LIKE ?", testCol.Like())
	assert.Equal(t, "wrk.first_name ILIKE ?", testCol.ILike())
	assert.Equal(t, "wrk.first_name NOT LIKE ?", testCol.NotLike())
	assert.Equal(t, "wrk.first_name NOT ILIKE ?", testCol.NotILike())
	assert.Equal(t, "wrk.first_name BETWEEN ? AND ?", testCol.Between())
}

func TestColumnOrderFragments(t *testing.T) {
	assert.Equal(t, "wrk.first_name ASC", testCol.OrderAsc())
	assert.Equal(t, "wrk.first_name DESC", testCol.OrderDesc())
}

func TestColumnSelectAndUpdateHelpers(t *testing.T) {
	assert.Equal(t, "wrk.first_name AS name", testCol.As("name"))
	assert.Equal(t, "first_name = ?", testCol.Set())
}

func TestColumnInc(t *testing.T) {
	col := NewColumn("version", "wrk")
	assert.Equal(t, "version = version + 1", col.Inc(1))
	assert.Equal(t, "version = version + 5", col.Inc(5))
}

func TestColumnDec(t *testing.T) {
	col := NewColumn("remaining_pto", "wrk")
	assert.Equal(t, "remaining_pto = remaining_pto - 8", col.Dec(8))
}

func TestColumnExpr(t *testing.T) {
	t.Run("replaces single placeholder", func(t *testing.T) {
		result := testCol.Expr("LOWER({})")
		assert.Equal(t, "LOWER(wrk.first_name)", result)
	})

	t.Run("replaces multiple occurrences", func(t *testing.T) {
		result := testCol.Expr("COALESCE({}, '') || ' ' || {}")
		assert.Equal(t, "COALESCE(wrk.first_name, '') || ' ' || wrk.first_name", result)
	})

	t.Run("no placeholder returns unchanged", func(t *testing.T) {
		result := testCol.Expr("COUNT(*)")
		assert.Equal(t, "COUNT(*)", result)
	})

	t.Run("real-world NULLIF BTRIM pattern", func(t *testing.T) {
		col := NewColumn("external_id", "wrk")
		result := col.Expr("NULLIF(BTRIM({}), '') IS NOT NULL")
		assert.Equal(t, "NULLIF(BTRIM(wrk.external_id), '') IS NOT NULL", result)
	})
}

func TestExpr(t *testing.T) {
	t.Run("positional placeholders", func(t *testing.T) {
		result := Expr("CONCAT({0}, ' ', {1})", testCol, testCol2)
		assert.Equal(t, "CONCAT(wrk.first_name, ' ', wrk.last_name)", result)
	})

	t.Run("single column with braces shorthand", func(t *testing.T) {
		result := Expr("LOWER({})", testCol)
		assert.Equal(t, "LOWER(wrk.first_name)", result)
	})

	t.Run("single column with positional", func(t *testing.T) {
		result := Expr("LOWER({0})", testCol)
		assert.Equal(t, "LOWER(wrk.first_name)", result)
	})

	t.Run("three columns", func(t *testing.T) {
		result := Expr("{0} = ? AND {1} LIKE ? AND {2} IS NOT NULL", testCol, testCol2, testCol3)
		assert.Equal(t, "wrk.first_name = ? AND wrk.last_name LIKE ? AND wrk.status IS NOT NULL", result)
	})

	t.Run("repeated placeholder", func(t *testing.T) {
		result := Expr("{0} = {0}", testCol)
		assert.Equal(t, "wrk.first_name = wrk.first_name", result)
	})

	t.Run("no columns passthrough", func(t *testing.T) {
		result := Expr("COUNT(*)")
		assert.Equal(t, "COUNT(*)", result)
	})
}

func TestCount(t *testing.T) {
	assert.Equal(t, "COUNT(*) AS total_workers", Count("total_workers"))
}

func TestCountDistinct(t *testing.T) {
	result := CountDistinct(testCol3, "unique_statuses")
	assert.Equal(t, "COUNT(DISTINCT wrk.status) AS unique_statuses", result)
}

func TestCountFilter(t *testing.T) {
	t.Run("zero conditions falls back to plain COUNT", func(t *testing.T) {
		result := CountFilter("total")
		assert.Equal(t, "COUNT(*) AS total", result)
	})

	t.Run("single condition", func(t *testing.T) {
		result := CountFilter("active_workers", testCol3.Eq())
		assert.Equal(t, "COUNT(*) FILTER (WHERE wrk.status = ?) AS active_workers", result)
	})

	t.Run("multiple conditions joined with AND", func(t *testing.T) {
		result := CountFilter("synced",
			testCol3.Eq(),
			testCol.Expr("NULLIF(BTRIM({}), '') IS NOT NULL"),
		)
		assert.Equal(t,
			"COUNT(*) FILTER (WHERE wrk.status = ? AND NULLIF(BTRIM(wrk.first_name), '') IS NOT NULL) AS synced",
			result,
		)
	})
}

func TestSum(t *testing.T) {
	col := NewColumn("weight", "sp")
	assert.Equal(t, "SUM(sp.weight) AS total_weight", Sum(col, "total_weight"))
}

func TestMin(t *testing.T) {
	col := NewColumn("created_at", "wrk")
	assert.Equal(t, "MIN(wrk.created_at) AS earliest", Min(col, "earliest"))
}

func TestMax(t *testing.T) {
	col := NewColumn("created_at", "wrk")
	assert.Equal(t, "MAX(wrk.created_at) AS latest", Max(col, "latest"))
}

func TestCoalesce(t *testing.T) {
	t.Run("empty string fallback", func(t *testing.T) {
		col := NewColumn("name", "loc")
		result := Coalesce(col, "''", "location_name")
		assert.Equal(t, "COALESCE(loc.name, '') AS location_name", result)
	})

	t.Run("numeric fallback", func(t *testing.T) {
		col := NewColumn("weight", "sp")
		result := Coalesce(col, "0", "weight")
		assert.Equal(t, "COALESCE(sp.weight, 0) AS weight", result)
	})
}

func TestNewColumnPanicsOnInvalidIdentifier(t *testing.T) {
	t.Run("empty name", func(t *testing.T) {
		assert.Panics(t, func() { NewColumn("", "wrk") })
	})

	t.Run("empty alias", func(t *testing.T) {
		assert.Panics(t, func() { NewColumn("id", "") })
	})

	t.Run("name with SQL injection", func(t *testing.T) {
		assert.Panics(t, func() { NewColumn("id; DROP TABLE--", "wrk") })
	})

	t.Run("alias with dot", func(t *testing.T) {
		assert.Panics(t, func() { NewColumn("id", "wrk.evil") })
	})

	t.Run("valid identifiers pass", func(t *testing.T) {
		assert.NotPanics(t, func() { NewColumn("first_name", "wrk") })
		assert.NotPanics(t, func() { NewColumn("id", "sm_orig") })
	})
}

func TestRel(t *testing.T) {
	t.Run("single segment", func(t *testing.T) {
		assert.Equal(t, "Profile", Rel("Profile"))
	})

	t.Run("two segments", func(t *testing.T) {
		assert.Equal(t, "Memberships.Organization", Rel("Memberships", "Organization"))
	})

	t.Run("three segments", func(t *testing.T) {
		assert.Equal(t, "Memberships.Organization.State", Rel("Memberships", "Organization", "State"))
	})
}

func TestApplyTenant(t *testing.T) {
	orgCol := NewColumn("organization_id", "wrk")
	buCol := NewColumn("business_unit_id", "wrk")
	ti := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	fn := ApplyTenant(orgCol, buCol, ti)
	require.NotNil(t, fn)
}
