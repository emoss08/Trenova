package buncolgen

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
)

// Column represents a database column with pre-computed SQL expression fragments.
// All expressions are computed once at init time via [NewColumn], so every method
// is a zero-allocation field return.
//
// The key distinction between methods:
//   - [Column.String] returns the bare column name — use for Bun's Column() and model-aware contexts
//   - [Column.Qualified] returns alias.column — use when Bun can't infer the table (joins, subqueries, raw expressions)
//   - Expression methods (Eq, OrderDesc, etc.) return ready-to-use SQL fragments with the alias baked in
//   - [Column.Expr] embeds the column into a larger SQL template — use for ColumnExpr with functions like COALESCE, BTRIM, etc.
//
// Example showing the difference:
//
//	col := NewColumn("first_name", "wrk")
//
//	col.String()                          // "first_name"
//	col.Qualified()                       // "wrk.first_name"
//	col.Eq()                              // "wrk.first_name = ?"
//	col.OrderDesc()                       // "wrk.first_name DESC"
//	col.Expr("LOWER({})")                 // "LOWER(wrk.first_name)"
type Column struct {
	// Name is the bare database column name (e.g. "first_name").
	Name string

	// Alias is the table alias from the bun:"alias:xx" tag (e.g. "wrk").
	Alias string

	qualified string
	eq        string
	ne        string
	notEq     string
	gt        string
	gte       string
	lt        string
	lte       string
	in        string
	notIn     string
	isNull    string
	isNotNull string
	orderAsc  string
	orderDesc string
	like      string
	ilike     string
	notLike   string
	notILike  string
	between   string
}

// TableInfo holds metadata about a database table parsed from the bun.BaseModel tag.
type TableInfo struct {
	// Name is the PostgreSQL table name (e.g. "workers").
	Name string

	// Alias is the short alias used in all generated SQL fragments (e.g. "wrk").
	Alias string

	// PrimaryKey lists the columns that form the composite primary key.
	PrimaryKey []string
}

func assertValidIdentifier(s, label string) {
	if s == "" {
		panic("buncolgen: " + label + " must not be empty")
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			panic(
				"buncolgen: " + label + " contains invalid character '" + string(
					r,
				) + "' in \"" + s + "\"",
			)
		}
	}
}

// NewColumn creates a Column with all SQL expression fragments pre-computed.
// This is called by generated code at package init time — not by application code.
// Panics if name or alias contain characters outside [a-zA-Z0-9_].
func NewColumn(name, alias string) Column {
	assertValidIdentifier(name, "column name")
	assertValidIdentifier(alias, "table alias")
	q := alias + "." + name
	return Column{
		Name:      name,
		Alias:     alias,
		qualified: q,
		eq:        q + " = ?",
		ne:        q + " != ?",
		notEq:     q + " <> ?",
		gt:        q + " > ?",
		gte:       q + " >= ?",
		lt:        q + " < ?",
		lte:       q + " <= ?",
		in:        q + " IN (?)",
		notIn:     q + " NOT IN (?)",
		isNull:    q + " IS NULL",
		isNotNull: q + " IS NOT NULL",
		orderAsc:  q + " ASC",
		orderDesc: q + " DESC",
		like:      q + " LIKE ?",
		ilike:     q + " ILIKE ?",
		notLike:   q + " NOT LIKE ?",
		notILike:  q + " NOT ILIKE ?",
		between:   q + " BETWEEN ? AND ?",
	}
}

// ---------------------------------------------------------------------------
// Basic accessors
// ---------------------------------------------------------------------------

// String returns the bare column name without the table alias.
// Use this in Bun's model-aware methods where Bun handles qualification automatically.
//
//	q.Column(WorkerColumns.FirstName.String(), WorkerColumns.LastName.String())
//	// Bun generates: SELECT wrk.first_name, wrk.last_name FROM workers AS wrk
func (c Column) String() string { return c.Name }

// Qualified returns "alias.column" — the fully qualified column reference.
// Use this inside raw SQL expressions, ColumnExpr, or anywhere Bun won't
// automatically prepend the table alias.
//
//	q.ColumnExpr("CONCAT(" + WorkerColumns.FirstName.Qualified() + ", ' ', " + WorkerColumns.LastName.Qualified() + ")")
//	// CONCAT(wrk.first_name, ' ', wrk.last_name)
func (c Column) Qualified() string { return c.qualified }

// ---------------------------------------------------------------------------
// WHERE clause fragments — each returns a string with ? bind placeholders
// ---------------------------------------------------------------------------

// Eq returns a "alias.column = ?" fragment for WHERE clauses.
//
//	q.Where(WorkerColumns.Status.Eq(), domaintypes.StatusActive)
//	// WHERE wrk.status = 'Active'
func (c Column) Eq() string { return c.eq }

// Ne returns a "alias.column != ?" fragment for WHERE clauses.
func (c Column) Ne() string { return c.ne }

// NotEq returns a "alias.column <> ?" fragment for WHERE clauses.
func (c Column) NotEq() string { return c.notEq }

// Gt returns a "alias.column > ?" fragment for WHERE clauses.
func (c Column) Gt() string { return c.gt }

// Gte returns a "alias.column >= ?" fragment for WHERE clauses.
func (c Column) Gte() string { return c.gte }

// Lt returns a "alias.column < ?" fragment for WHERE clauses.
func (c Column) Lt() string { return c.lt }

// Lte returns a "alias.column <= ?" fragment for WHERE clauses.
func (c Column) Lte() string { return c.lte }

// In returns a "alias.column IN (?)" fragment for WHERE clauses with multiple values.
// Bun expands the single ? placeholder to match the slice length.
//
//	q.Where(WorkerColumns.Status.In(), bun.In([]string{"Active", "Inactive"}))
//	// WHERE wrk.status IN ('Active', 'Inactive')
func (c Column) In() string { return c.in }

// IsNull returns a "alias.column IS NULL" fragment. No bind parameter needed.
//
//	q.Where(WorkerColumns.FleetCodeID.IsNull())
//	// WHERE wrk.fleet_code_id IS NULL
func (c Column) IsNull() string { return c.isNull }

// IsNotNull returns a "alias.column IS NOT NULL" fragment. No bind parameter needed.
func (c Column) IsNotNull() string { return c.isNotNull }

// Like returns a "alias.column LIKE ?" fragment for case-sensitive pattern matching.
//
//	q.Where(WorkerColumns.FirstName.Like(), "%john%")
//	// WHERE wrk.first_name LIKE '%john%'
func (c Column) Like() string { return c.like }

// ILike returns a "alias.column ILIKE ?" fragment for case-insensitive pattern matching (PostgreSQL-specific).
func (c Column) ILike() string { return c.ilike }

// Between returns a "alias.column BETWEEN ? AND ?" fragment requiring two bind parameters.
//
//	q.Where(WorkerColumns.CreatedAt.Between(), startTime, endTime)
//	// WHERE wrk.created_at BETWEEN 1711234567 AND 1711320967
func (c Column) Between() string { return c.between }

// NotIn returns a "alias.column NOT IN (?)" fragment for exclusion filtering.
func (c Column) NotIn() string { return c.notIn }

// NotLike returns a "alias.column NOT LIKE ?" fragment.
func (c Column) NotLike() string { return c.notLike }

// NotILike returns a "alias.column NOT ILIKE ?" fragment.
func (c Column) NotILike() string { return c.notILike }

// ---------------------------------------------------------------------------
// ORDER BY fragments
// ---------------------------------------------------------------------------

// OrderAsc returns a "alias.column ASC" fragment for ORDER BY clauses.
//
//	q.Order(WorkerColumns.LastName.OrderAsc())
//	// ORDER BY wrk.last_name ASC
func (c Column) OrderAsc() string { return c.orderAsc }

// OrderDesc returns a "alias.column DESC" fragment for ORDER BY clauses.
//
//	q.Order(WorkerColumns.CreatedAt.OrderDesc())
//	// ORDER BY wrk.created_at DESC
func (c Column) OrderDesc() string { return c.orderDesc }

// ---------------------------------------------------------------------------
// SELECT and UPDATE helpers
// ---------------------------------------------------------------------------

// As returns a "alias.column AS label" fragment for aliasing a column in SELECT.
//
//	q.ColumnExpr(WorkerColumns.FirstName.As("name"))
//	// wrk.first_name AS name
func (c Column) As(label string) string { return c.qualified + " AS " + label }

// Bare returns the unqualified column name. This is an alias for [Column.String]
// that reads more clearly at call sites where the intent is to reference the bare column.
//
//	Columns: []string{WorkerColumns.ID.Bare(), WorkerColumns.FirstName.Bare()}
func (c Column) Bare() string { return c.Name }

// Set returns a "column = ?" fragment (without the table alias) for UPDATE SET clauses.
// Bun's SET clause operates on bare column names, not alias-qualified ones.
//
//	q.Set(WorkerColumns.Status.Set(), domaintypes.StatusInactive)
//	// SET status = 'Inactive'
func (c Column) Set() string { return c.Name + " = ?" }

// SetNull returns a "column = NULL" fragment for UPDATE SET clauses.
// Use this instead of raw string literals when clearing nullable columns.
//
//	q.Set(ShipmentColumns.CanceledByID.SetNull())
//	// SET canceled_by_id = NULL
func (c Column) SetNull() string { return c.Name + " = NULL" }

// SetExcluded returns a "column = EXCLUDED.column" fragment for PostgreSQL upserts.
// This is useful inside ON CONFLICT DO UPDATE clauses to copy the incoming value
// from the excluded row without repeating the column name as a string literal.
//
//	q.Set(DocumentContentColumns.Status.SetExcluded())
//	// SET status = EXCLUDED.status
func (c Column) SetExcluded() string { return c.Name + " = EXCLUDED." + c.Name }

// SetExpr returns a "column = <expr>" fragment for UPDATE SET clauses.
// Pass any additional bind arguments to Bun's Set method as usual.
//
//	q.Set(WorkerColumns.Status.SetExpr("CASE WHEN {} = ? THEN ? ELSE {} END"), oldStatus, nextStatus, fallbackStatus)
//	// SET status = CASE WHEN status = ? THEN ? ELSE status END
//
// The expression may include "{}" placeholders, which are replaced with the bare
// column name to avoid repeating it in common self-referential updates.
func (c Column) SetExpr(expr string) string {
	return c.Name + " = " + strings.ReplaceAll(expr, "{}", c.Name)
}

// Inc returns a "column = column + n" fragment for UPDATE SET clauses that increment a value.
// Use this for version bumps and counter updates.
//
//	q.Set(ShipmentColumns.Version.Inc(1))
//	// SET version = version + 1
func (c Column) Inc(n int) string {
	return c.Name + " = " + c.Name + " + " + strconv.Itoa(n)
}

// Dec returns a "column = column - n" fragment for UPDATE SET clauses that decrement a value.
//
//	q.Set(WorkerColumns.RemainingPTO.Dec(8))
//	// SET remaining_pto = remaining_pto - 8
func (c Column) Dec(n int) string {
	return c.Name + " = " + c.Name + " - " + strconv.Itoa(n)
}

// ---------------------------------------------------------------------------
// SQL expression templates
// ---------------------------------------------------------------------------

// Expr replaces every occurrence of {} in the template with the column's qualified name.
// Use this to embed a column reference into a larger SQL expression without string concatenation.
//
//	WorkerColumns.ExternalID.Expr("NULLIF(BTRIM({}), '') IS NOT NULL")
//	// "NULLIF(BTRIM(wrk.external_id), '') IS NOT NULL"
//
//	WorkerColumns.FirstName.Expr("LOWER({})")
//	// "LOWER(wrk.first_name)"
func (c Column) Expr(template string) string {
	return strings.ReplaceAll(template, "{}", c.qualified)
}

// Expr replaces positional placeholders {0}, {1}, etc. in the template with each column's
// qualified name. When exactly one column is provided, {} is also supported as shorthand for {0}.
//
// Use this for SQL expressions that reference multiple columns:
//
//	Expr("CONCAT({0}, ' ', {1})", WorkerColumns.FirstName, WorkerColumns.LastName)
//	// "CONCAT(wrk.first_name, ' ', wrk.last_name)"
//
//	Expr("{0} = ? AND NULLIF(BTRIM({1}), '') IS NOT NULL", wrk.Status, wrk.ExternalID)
//	// "wrk.status = ? AND NULLIF(BTRIM(wrk.external_id), '') IS NOT NULL"
func Expr(template string, cols ...Column) string {
	for i, col := range cols {
		template = strings.ReplaceAll(template, "{"+strconv.Itoa(i)+"}", col.Qualified())
	}
	if len(cols) == 1 {
		template = strings.ReplaceAll(template, "{}", cols[0].Qualified())
	}
	return template
}

// ---------------------------------------------------------------------------
// Aggregate helpers — return ColumnExpr-ready strings
// ---------------------------------------------------------------------------

// Count returns a "COUNT(*) AS alias" expression for use with ColumnExpr.
//
//	q.ColumnExpr(buncolgen.Count("total_workers"))
//	// COUNT(*) AS total_workers
func Count(alias string) string {
	return "COUNT(*) AS " + alias
}

// CountDistinct returns a "COUNT(DISTINCT col) AS alias" expression.
//
//	q.ColumnExpr(buncolgen.CountDistinct(WorkerColumns.Status, "unique_statuses"))
//	// COUNT(DISTINCT wrk.status) AS unique_statuses
func CountDistinct(col Column, alias string) string {
	return "COUNT(DISTINCT " + col.Qualified() + ") AS " + alias
}

// CountFilter returns a "COUNT(*) FILTER (WHERE cond AND ...) AS alias" expression.
// Each condition is a pre-built SQL fragment (e.g. Column.Eq()). Bind parameters
// are passed separately to ColumnExpr as usual.
//
//	q.ColumnExpr(buncolgen.CountFilter("active_workers", wrk.Status.Eq()), domaintypes.StatusActive)
//	// COUNT(*) FILTER (WHERE wrk.status = 'Active') AS active_workers
//
//	q.ColumnExpr(buncolgen.CountFilter("synced",
//	    wrk.Status.Eq(),
//	    wrk.ExternalID.Expr("NULLIF(BTRIM({}), '') IS NOT NULL"),
//	), domaintypes.StatusActive)
//	// COUNT(*) FILTER (WHERE wrk.status = ? AND NULLIF(BTRIM(wrk.external_id), '') IS NOT NULL) AS synced
func CountFilter(alias string, conditions ...string) string {
	if len(conditions) == 0 {
		return "COUNT(*) AS " + alias
	}
	return "COUNT(*) FILTER (WHERE " + strings.Join(conditions, " AND ") + ") AS " + alias
}

// Sum returns a "SUM(col) AS alias" expression.
//
//	q.ColumnExpr(buncolgen.Sum(ShipmentColumns.Weight, "total_weight"))
//	// SUM(sp.weight) AS total_weight
func Sum(col Column, alias string) string {
	return "SUM(" + col.Qualified() + ") AS " + alias
}

// Min returns a "MIN(col) AS alias" expression.
//
//	q.ColumnExpr(buncolgen.Min(WorkerColumns.CreatedAt, "earliest"))
//	// MIN(wrk.created_at) AS earliest
func Min(col Column, alias string) string {
	return "MIN(" + col.Qualified() + ") AS " + alias
}

// Max returns a "MAX(col) AS alias" expression.
//
//	q.ColumnExpr(buncolgen.Max(WorkerColumns.CreatedAt, "latest"))
//	// MAX(wrk.created_at) AS latest
func Max(col Column, alias string) string {
	return "MAX(" + col.Qualified() + ") AS " + alias
}

// Coalesce returns a "COALESCE(col, fallback) AS alias" expression.
// The fallback parameter is a raw SQL literal (e.g. "”" for empty string, "0" for zero).
//
//	q.ColumnExpr(buncolgen.Coalesce(LocationColumns.Name, "''", "location_name"))
//	// COALESCE(loc.name, '') AS location_name
func Coalesce(col Column, fallback, alias string) string {
	return "COALESCE(" + col.Qualified() + ", " + fallback + ") AS " + alias
}

// ---------------------------------------------------------------------------
// Tenant scoping — shared implementations for all query types
// ---------------------------------------------------------------------------

// ScopeTenant adds WHERE organization_id = ? AND business_unit_id = ? to a SelectQuery.
// Prefer the generated per-entity helpers (e.g. WorkerScopeTenant).
func ScopeTenant(
	q *bun.SelectQuery,
	orgCol, buCol Column,
	ti pagination.TenantInfo,
) *bun.SelectQuery {
	return q.Where(orgCol.eq, ti.OrgID).Where(buCol.eq, ti.BuID)
}

// ScopeTenantUpdate adds WHERE organization_id = ? AND business_unit_id = ? to an UpdateQuery.
// Prefer the generated per-entity helpers (e.g. WorkerScopeTenantUpdate).
//
//	WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
//	    return buncolgen.TrailerScopeTenantUpdate(uq, req.TenantInfo).
//	        Where(buncolgen.TrailerColumns.ID.In(), bun.List(req.TrailerIDs))
//	})
func ScopeTenantUpdate(
	q *bun.UpdateQuery,
	orgCol, buCol Column,
	ti pagination.TenantInfo,
) *bun.UpdateQuery {
	return q.Where(orgCol.eq, ti.OrgID).Where(buCol.eq, ti.BuID)
}

// ScopeTenantDelete adds WHERE organization_id = ? AND business_unit_id = ? to a DeleteQuery.
// Prefer the generated per-entity helpers (e.g. WorkerScopeTenantDelete).
func ScopeTenantDelete(
	q *bun.DeleteQuery,
	orgCol, buCol Column,
	ti pagination.TenantInfo,
) *bun.DeleteQuery {
	return q.Where(orgCol.eq, ti.OrgID).Where(buCol.eq, ti.BuID)
}

// ApplyTenant returns a closure compatible with SelectQuery.Apply() that adds tenant
// WHERE clauses. Prefer the generated per-entity helpers (e.g. WorkerApplyTenant).
//
//	// Instead of wrapping ScopeTenant in an anonymous function:
//	.Apply(buncolgen.WorkerApplyTenant(tenantInfo))
func ApplyTenant(
	orgCol, buCol Column,
	ti pagination.TenantInfo,
) func(*bun.SelectQuery) *bun.SelectQuery {
	return func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.Where(orgCol.eq, ti.OrgID).Where(buCol.eq, ti.BuID)
	}
}

// ---------------------------------------------------------------------------
// Relation path helpers
// ---------------------------------------------------------------------------

// Rel joins relation name segments with "." for Bun's nested eager-loading syntax.
// Use this to build dot-separated relation paths in a type-safe way.
//
//	// Before:
//	q.Relation("Memberships.Organization.State")
//
//	// After:
//	q.Relation(buncolgen.Rel(
//	    buncolgen.UserRelations.Memberships,
//	    buncolgen.OrganizationMembershipRelations.Organization,
//	    buncolgen.OrganizationRelations.State,
//	))
//	// → "Memberships.Organization.State"
func Rel(segments ...string) string {
	return strings.Join(segments, ".")
}

// ---------------------------------------------------------------------------
// Filter construction
// ---------------------------------------------------------------------------

// NewFieldFilter creates a [domaintypes.FieldFilter] from a JSON field name, operator, and value.
// Prefer the generated per-entity filter builders (e.g. WorkerFilter.Status) which pass
// the correct JSON field name automatically.
func NewFieldFilter(jsonName string, op dbtype.Operator, value any) domaintypes.FieldFilter {
	return domaintypes.FieldFilter{
		Field:    jsonName,
		Operator: op,
		Value:    value,
	}
}
