package archiveplan

import (
	"slices"
	"strings"
)

const (
	OrganizationColumn = "organization_id"
	BusinessUnitColumn = "business_unit_id"
	BusinessUnits      = "business_units"
	Organizations      = "organizations"
	Users              = "users"
	IDColumn           = "id"
)

type Column struct {
	Name      string `db:"column_name"`
	Type      string `db:"column_type"`
	Nullable  bool   `db:"nullable"`
	Generated bool   `db:"generated"`
	Identity  bool   `db:"identity"`
	Serial    string `db:"serial_sequence"`
}

type Table struct {
	Name       string
	Partitions bool
	Columns    []Column
	PrimaryKey []string
}

func (t *Table) Column(name string) (*Column, bool) {
	for i := range t.Columns {
		if t.Columns[i].Name == name {
			return &t.Columns[i], true
		}
	}
	return nil, false
}

func (t *Table) Has(name string) bool {
	_, ok := t.Column(name)
	return ok
}

func (t *Table) Tenant() bool {
	return t.Has(OrganizationColumn) || t.Has(BusinessUnitColumn)
}

type ForeignKey struct {
	Name          string   `db:"constraint_name"`
	Child         string   `db:"child_table"`
	Parent        string   `db:"parent_table"`
	ChildColumns  []string `db:"child_columns"`
	ParentColumns []string `db:"parent_columns"`
}

func (fk *ForeignKey) SelfReferencing() bool {
	return fk.Child == fk.Parent
}

func (fk *ForeignKey) ScopedToTenant() bool {
	return slices.Contains(fk.ChildColumns, OrganizationColumn) &&
		parentOf(fk, OrganizationColumn) == OrganizationColumn
}

func parentOf(fk *ForeignKey, child string) string {
	for i, c := range fk.ChildColumns {
		if c == child && i < len(fk.ParentColumns) {
			return fk.ParentColumns[i]
		}
	}
	return ""
}

func (fk *ForeignKey) ownColumns() []string {
	out := make([]string, 0, len(fk.ChildColumns))
	for _, c := range fk.ChildColumns {
		if c != OrganizationColumn && c != BusinessUnitColumn {
			out = append(out, c)
		}
	}
	return out
}

type Catalog struct {
	Tables      map[string]*Table
	ForeignKeys []*ForeignKey
	Rollups     []string
	Globals     map[string]string
}

func (c *Catalog) Table(name string) (*Table, bool) {
	t, ok := c.Tables[name]
	return t, ok
}

func QuoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func QuoteLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func JoinIdents(columns []string, alias string) string {
	quoted := make([]string, 0, len(columns))
	for _, c := range columns {
		if alias != "" {
			quoted = append(quoted, alias+"."+QuoteIdent(c))
			continue
		}
		quoted = append(quoted, QuoteIdent(c))
	}
	return strings.Join(quoted, ", ")
}
