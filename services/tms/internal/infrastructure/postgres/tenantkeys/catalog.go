package tenantkeys

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/jackc/pgx/v5"
)

const (
	OrganizationColumn = "organization_id"
	BusinessUnitColumn = "business_unit_id"
	Organizations      = "organizations"

	unitSuffix         = "_" + BusinessUnitColumn
	organizationSuffix = "_" + OrganizationColumn

	tablesQuery = `SELECT c.relname::text AS table_name,
       c.relkind = 'p' AS partitioned,
       EXISTS (SELECT 1 FROM trenova_rls.global_tables g WHERE g.table_name = c.relname) AS global,
       array_agg(a.attname::text ORDER BY a.attnum) AS column_names,
       array_agg(a.attnotnull ORDER BY a.attnum) AS not_null
FROM pg_catalog.pg_class c
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
JOIN pg_catalog.pg_attribute a ON a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped
WHERE n.nspname = 'public'
  AND c.relkind IN ('r', 'p')
  AND NOT c.relispartition
GROUP BY c.oid, c.relname, c.relkind
ORDER BY c.relname`

	foreignKeysQuery = `SELECT con.conname::text AS constraint_name,
       child.relname::text AS child_table,
       parent.relname::text AS parent_table,
       ARRAY(
           SELECT att.attname::text
           FROM unnest(con.conkey) WITH ORDINALITY AS k(attnum, ord)
           JOIN pg_catalog.pg_attribute att ON att.attrelid = con.conrelid AND att.attnum = k.attnum
           ORDER BY k.ord
       ) AS child_columns,
       ARRAY(
           SELECT att.attname::text
           FROM unnest(con.confkey) WITH ORDINALITY AS k(attnum, ord)
           JOIN pg_catalog.pg_attribute att ON att.attrelid = con.confrelid AND att.attnum = k.attnum
           ORDER BY k.ord
       ) AS parent_columns,
       ARRAY(
           SELECT att.attname::text
           FROM unnest(COALESCE(con.confdelsetcols, '{}'::int2[])) WITH ORDINALITY AS k(attnum, ord)
           JOIN pg_catalog.pg_attribute att ON att.attrelid = con.conrelid AND att.attnum = k.attnum
           ORDER BY k.ord
       ) AS set_columns,
       con.confdeltype::text AS on_delete,
       con.confupdtype::text AS on_update,
       con.confmatchtype::text AS match_type,
       con.condeferrable AS deferrable,
       con.condeferred AS deferred,
       con.convalidated AS validated
FROM pg_catalog.pg_constraint con
JOIN pg_catalog.pg_class child ON child.oid = con.conrelid
JOIN pg_catalog.pg_class parent ON parent.oid = con.confrelid
JOIN pg_catalog.pg_namespace n ON n.oid = child.relnamespace
WHERE con.contype = 'f'
  AND n.nspname = 'public'
  AND NOT child.relispartition
ORDER BY con.conname`

	indexesQuery = `SELECT i.relname::text AS index_name,
       t.relname::text AS table_name,
       x.indisunique AS is_unique,
       COALESCE(pg_get_expr(x.indpred, x.indrelid), '') AS predicate,
       x.indexprs IS NOT NULL AS has_expressions,
       ARRAY(
           SELECT a.attname::text
           FROM unnest(x.indkey::int2[]) WITH ORDINALITY AS k(attnum, ord)
           JOIN pg_catalog.pg_attribute a ON a.attrelid = t.oid AND a.attnum = k.attnum
           WHERE k.ord <= x.indnkeyatts
           ORDER BY k.ord
       ) AS index_columns
FROM pg_catalog.pg_index x
JOIN pg_catalog.pg_class i ON i.oid = x.indexrelid
JOIN pg_catalog.pg_class t ON t.oid = x.indrelid
JOIN pg_catalog.pg_namespace n ON n.oid = t.relnamespace
WHERE n.nspname = 'public'
  AND x.indisvalid
  AND x.indisready
  AND NOT t.relispartition
ORDER BY t.relname, i.relname`

	relationsQuery = `SELECT c.relname::text
FROM pg_catalog.pg_class c
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public'
UNION
SELECT con.conname::text
FROM pg_catalog.pg_constraint con
JOIN pg_catalog.pg_namespace n ON n.oid = con.connamespace
WHERE n.nspname = 'public'`
)

type Table struct {
	Name        string
	Partitioned bool
	Global      bool
	Columns     map[string]bool
	PartyUnit   string
}

func (t *Table) Has(column string) bool {
	_, ok := t.Columns[column]
	return ok
}

func (t *Table) Required(column string) bool {
	return t.Columns[column]
}

func (t *Table) Tenant() bool {
	if t.Global {
		return false
	}
	return t.Has(OrganizationColumn) || t.Has(BusinessUnitColumn) || t.PartyUnit != ""
}

func (t *Table) UnitColumn() string {
	if t.Has(BusinessUnitColumn) {
		return BusinessUnitColumn
	}
	return t.PartyUnit
}

type ForeignKey struct {
	Name          string   `db:"constraint_name"`
	Child         string   `db:"child_table"`
	Parent        string   `db:"parent_table"`
	ChildColumns  []string `db:"child_columns"`
	ParentColumns []string `db:"parent_columns"`
	SetColumns    []string `db:"set_columns"`
	OnDelete      string   `db:"on_delete"`
	OnUpdate      string   `db:"on_update"`
	Match         string   `db:"match_type"`
	Deferrable    bool     `db:"deferrable"`
	Deferred      bool     `db:"deferred"`
	Validated     bool     `db:"validated"`
}

type Index struct {
	Name        string   `db:"index_name"`
	Table       string   `db:"table_name"`
	Unique      bool     `db:"is_unique"`
	Predicate   string   `db:"predicate"`
	Expressions bool     `db:"has_expressions"`
	Columns     []string `db:"index_columns"`
}

type Catalog struct {
	Tables      map[string]*Table
	ForeignKeys []*ForeignKey
	Indexes     map[string][]*Index
	Names       map[string]struct{}
}

func (c *Catalog) Table(name string) (*Table, bool) {
	t, ok := c.Tables[name]
	return t, ok
}

type tableRow struct {
	Name        string   `db:"table_name"`
	Partitioned bool     `db:"partitioned"`
	Global      bool     `db:"global"`
	ColumnNames []string `db:"column_names"`
	NotNull     []bool   `db:"not_null"`
}

func Load(ctx context.Context, db *keel.DB) (*Catalog, error) {
	rows, err := keel.QueryAll(ctx, db, pgx.RowToStructByName[tableRow], tablesQuery)
	if err != nil {
		return nil, fmt.Errorf("read the tables: %w", err)
	}
	fks, err := keel.QueryAll(ctx, db, pgx.RowToAddrOfStructByName[ForeignKey], foreignKeysQuery)
	if err != nil {
		return nil, fmt.Errorf("read the foreign keys: %w", err)
	}
	indexes, err := keel.QueryAll(ctx, db, pgx.RowToAddrOfStructByName[Index], indexesQuery)
	if err != nil {
		return nil, fmt.Errorf("read the indexes: %w", err)
	}
	names, err := keel.QueryAll(ctx, db, pgx.RowTo[string], relationsQuery)
	if err != nil {
		return nil, fmt.Errorf("read the relation names: %w", err)
	}
	return NewCatalog(tablesFrom(rows), fks, indexes, names), nil
}

func tablesFrom(rows []tableRow) []*Table {
	out := make([]*Table, 0, len(rows))
	for _, row := range rows {
		t := &Table{
			Name:        row.Name,
			Partitioned: row.Partitioned,
			Global:      row.Global,
			Columns:     make(map[string]bool, len(row.ColumnNames)),
		}
		for i, name := range row.ColumnNames {
			t.Columns[name] = i < len(row.NotNull) && row.NotNull[i]
		}
		out = append(out, t)
	}
	return out
}

func NewCatalog(tables []*Table, fks []*ForeignKey, indexes []*Index, names []string) *Catalog {
	cat := &Catalog{
		Tables:      make(map[string]*Table, len(tables)),
		ForeignKeys: fks,
		Indexes:     make(map[string][]*Index),
		Names:       make(map[string]struct{}, len(names)),
	}
	for _, t := range tables {
		if !t.Global {
			t.PartyUnit = partyUnit(t)
		}
		cat.Tables[t.Name] = t
	}
	for _, idx := range indexes {
		cat.Indexes[idx.Table] = append(cat.Indexes[idx.Table], idx)
	}
	for _, name := range names {
		cat.Names[name] = struct{}{}
	}
	return cat
}

func partyUnit(t *Table) string {
	if t.Has(BusinessUnitColumn) {
		return ""
	}
	candidates := make([]string, 0, 2)
	for name := range t.Columns {
		if !strings.HasSuffix(name, unitSuffix) {
			continue
		}
		twin := strings.TrimSuffix(name, BusinessUnitColumn) + OrganizationColumn
		if t.Has(twin) {
			candidates = append(candidates, name)
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	slices.Sort(candidates)
	return candidates[0]
}
