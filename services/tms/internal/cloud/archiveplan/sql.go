package archiveplan

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/pkg/dbscope"
)

const schemaPrefix = `"public".`

var SessionSettings = [][2]string{
	{"DateStyle", "ISO, YMD"},
	{"IntervalStyle", "postgres"},
	{"TimeZone", "UTC"},
	{"extra_float_digits", "1"},
	{"bytea_output", "hex"},
	{"client_encoding", "UTF8"},
	{"standard_conforming_strings", "on"},
	{"search_path", "pg_catalog, public"},
}

func Qualified(table string) string {
	return schemaPrefix + QuoteIdent(table)
}

func (t *PlannedTable) Where(alias string, tenant dbscope.Tenant) string {
	col := func(name string) string {
		if alias == "" {
			return QuoteIdent(name)
		}
		return alias + "." + QuoteIdent(name)
	}
	org := QuoteLiteral(tenant.OrganizationID.String())
	bu := QuoteLiteral(tenant.BusinessUnitID.String())
	switch t.Filter {
	case FilterOrganization:
		return col(OrganizationColumn) + " = " + org
	case FilterOrganizationOrUnit:
		return "(" + col(OrganizationColumn) + " = " + org + " OR (" + col(OrganizationColumn) +
			" IS NULL AND " + col(BusinessUnitColumn) + " = " + bu + "))"
	case FilterUnit:
		return col(BusinessUnitColumn) + " = " + bu
	case FilterID:
		return col(IDColumn) + " = " + bu
	case FilterTwoParty:
		return "(" + col(
			t.Parties[0],
		) + " = " + org + " OR " + col(
			t.Parties[1],
		) + " = " + org + ")"
	default:
		return "FALSE"
	}
}

func (t *PlannedTable) CopyOut(tenant dbscope.Tenant) string {
	return "COPY (SELECT " + JoinIdents(t.Columns, "a") + " FROM " + Qualified(t.Name) +
		" a WHERE " + t.Where("a", tenant) + ") TO STDOUT (FORMAT binary)"
}

func (t *PlannedTable) StageName() string {
	return fmt.Sprintf("archive_stage_%04d", t.Position)
}

func (t *PlannedTable) Staged() bool {
	return len(t.Deferred) > 0
}

func (t *PlannedTable) CopyIn() string {
	return "COPY " + QuoteIdent(t.StageName()) + " (" + JoinIdents(t.Columns, "") +
		") FROM STDIN (FORMAT binary)"
}

func (t *PlannedTable) DropStage() string {
	return "DROP TABLE " + QuoteIdent(t.StageName())
}

func (t *PlannedTable) CreateStage() string {
	return "CREATE TEMP TABLE " + QuoteIdent(t.StageName()) + " ON COMMIT DROP AS SELECT " +
		JoinIdents(t.Columns, "") + " FROM " + Qualified(t.Name) + " WITH NO DATA"
}

func (t *PlannedTable) InsertFromStage() string {
	selected := make([]string, 0, len(t.Columns))
	for _, c := range t.Columns {
		if slices.Contains(t.Deferred, c) {
			selected = append(selected, "NULL")
			continue
		}
		selected = append(selected, QuoteIdent(c))
	}
	return "INSERT INTO " + Qualified(t.Name) + " (" + JoinIdents(t.Columns, "") + ") SELECT " +
		strings.Join(selected, ", ") + " FROM " + QuoteIdent(t.StageName())
}

func (t *PlannedTable) RestoreDeferred() string {
	sets := make([]string, 0, len(t.Deferred))
	present := make([]string, 0, len(t.Deferred))
	for _, c := range t.Deferred {
		sets = append(sets, QuoteIdent(c)+" = s."+QuoteIdent(c))
		present = append(present, "s."+QuoteIdent(c)+" IS NOT NULL")
	}
	match := make([]string, 0, len(t.PrimaryKey))
	for _, c := range t.PrimaryKey {
		match = append(match, "t."+QuoteIdent(c)+" = s."+QuoteIdent(c))
	}
	return "UPDATE " + Qualified(t.Name) + " AS t SET " + strings.Join(sets, ", ") + " FROM " +
		QuoteIdent(t.StageName()) + " AS s WHERE " + strings.Join(match, " AND ") + " AND (" +
		strings.Join(present, " OR ") + ")"
}

func (t *PlannedTable) AdvanceSequences() []string {
	out := make([]string, 0, len(t.Sequences))
	for _, seq := range t.Sequences {
		out = append(
			out,
			"SELECT setval("+QuoteLiteral(seq.Name)+"::regclass, GREATEST((SELECT COALESCE(max("+
				QuoteIdent(
					seq.Column,
				)+"), 1) FROM "+Qualified(t.Name)+"), (SELECT last_value FROM "+
				seq.Name+")), true)",
		)
	}
	return out
}

func (p *Plan) CountQuery(tenant dbscope.Tenant) string {
	parts := make([]string, 0, len(p.Tables))
	for _, t := range p.Tables {
		parts = append(
			parts,
			"SELECT "+QuoteLiteral(
				t.Name,
			)+"::text AS table_name, count(*)::bigint AS row_count FROM "+
				Qualified(
					t.Name,
				)+" a WHERE "+t.Where(
				"a",
				tenant,
			),
		)
	}
	return strings.Join(parts, "\nUNION ALL\n")
}

func (p *Plan) HoldingQuery(tenant dbscope.Tenant) string {
	parts := make([]string, 0, len(p.Tables))
	for _, t := range p.Tables {
		parts = append(parts, "SELECT "+QuoteLiteral(t.Name)+"::text WHERE EXISTS (SELECT 1 FROM "+
			Qualified(t.Name)+" a WHERE "+t.Where("a", tenant)+")")
	}
	return strings.Join(parts, "\nUNION ALL\n")
}

func notNull(columns []string, alias string) string {
	conds := make([]string, 0, len(columns))
	for _, c := range columns {
		conds = append(conds, alias+"."+QuoteIdent(c)+" IS NOT NULL")
	}
	return strings.Join(conds, " AND ")
}

func matches(fk *ForeignKey) string {
	conds := make([]string, 0, len(fk.ChildColumns))
	for i, c := range fk.ChildColumns {
		conds = append(conds, "p."+QuoteIdent(fk.ParentColumns[i])+" = c."+QuoteIdent(c))
	}
	return strings.Join(conds, " AND ")
}

func (p *Plan) OutsideQueries(tenant dbscope.Tenant, batch int) []string {
	return p.referenceQueries(p.Outside, tenant, batch, true)
}

func (p *Plan) DanglingQueries(tenant dbscope.Tenant, batch int) []string {
	return p.referenceQueries(p.ForeignKeys, tenant, batch, true)
}

func (p *Plan) referenceQueries(
	fks []*ForeignKey,
	tenant dbscope.Tenant,
	batch int,
	withinParent bool,
) []string {
	out := make([]string, 0)
	parts := make([]string, 0, batch)
	flush := func() {
		if len(parts) > 0 {
			out = append(out, strings.Join(parts, "\nUNION ALL\n"))
			parts = parts[:0]
		}
	}
	for _, fk := range fks {
		child, ok := p.byName[fk.Child]
		if !ok {
			continue
		}
		parent, ok := p.byName[fk.Parent]
		if !ok {
			continue
		}
		parentWhere := ""
		if withinParent {
			parentWhere = " AND " + parent.Where("p", tenant)
		}
		parts = append(parts, "SELECT "+QuoteLiteral(fk.Name)+"::text AS constraint_name, "+
			QuoteLiteral(fk.Child)+"::text AS table_name, "+
			QuoteLiteral(strings.Join(fk.ChildColumns, ", "))+"::text AS column_names, "+
			"count(*)::bigint AS row_count FROM "+
			Qualified(fk.Child)+" c WHERE "+child.Where("c", tenant)+" AND "+
			notNull(fk.ChildColumns, "c")+" AND NOT EXISTS (SELECT 1 FROM "+Qualified(fk.Parent)+
			" p WHERE "+matches(fk)+parentWhere+")")
		if len(parts) >= batch {
			flush()
		}
	}
	flush()
	return out
}

func (r *GlobalReference) CollectQuery(p *Plan, tenant dbscope.Tenant) string {
	sources := make([]string, 0, len(r.Uses))
	for _, use := range r.Uses {
		t := p.byName[use.Table]
		sources = append(sources, "SELECT c."+QuoteIdent(use.Column)+"::text AS id FROM "+
			Qualified(use.Table)+" c WHERE "+t.Where("c", tenant)+" AND c."+
			QuoteIdent(use.Column)+" IS NOT NULL")
	}
	keys := make([]string, 0, len(r.Key))
	for _, k := range r.Key {
		keys = append(keys, "g."+QuoteIdent(k)+"::text")
	}
	return "SELECT g." + QuoteIdent(IDColumn) + "::text AS id, ARRAY[" + strings.Join(keys, ", ") +
		"] AS natural_key FROM " + Qualified(r.Table) + " g WHERE g." + QuoteIdent(IDColumn) +
		"::text IN (" + strings.Join(sources, " UNION ") + ") ORDER BY 1"
}

func ResolveQuery(table string, key []string) string {
	keys := make([]string, 0, len(key))
	args := make([]string, 0, len(key))
	for i, k := range key {
		keys = append(keys, "g."+QuoteIdent(k)+"::text")
		args = append(args, fmt.Sprintf("$%d::text[]", i+1))
	}
	return "SELECT g." + QuoteIdent(IDColumn) + "::text AS id, ARRAY[" + strings.Join(keys, ", ") +
		"] AS natural_key FROM " + Qualified(table) + " g WHERE (" + strings.Join(keys, ", ") +
		") IN (SELECT * FROM unnest(" + strings.Join(args, ", ") + "))"
}

func (t *PlannedTable) ScopedKey() bool {
	return slices.Contains(t.PrimaryKey, OrganizationColumn) ||
		slices.Contains(t.PrimaryKey, BusinessUnitColumn)
}

func (t *PlannedTable) ExistingKeysQuery() string {
	cols := make([]string, 0, len(t.PrimaryKey))
	args := make([]string, 0, len(t.PrimaryKey))
	for i, c := range t.PrimaryKey {
		cols = append(cols, "t."+QuoteIdent(c)+"::text")
		args = append(args, fmt.Sprintf("$%d::text[]", i+1))
	}
	if len(t.PrimaryKey) == 1 {
		return "SELECT count(*)::bigint FROM " + Qualified(t.Name) + " t WHERE t." +
			QuoteIdent(t.PrimaryKey[0]) + "::text = ANY($1::text[])"
	}
	return "SELECT count(*)::bigint FROM " + Qualified(
		t.Name,
	) + " t WHERE (" + strings.Join(
		cols,
		", ",
	) +
		") IN (SELECT * FROM unnest(" + strings.Join(
		args,
		", ",
	) + "))"
}
