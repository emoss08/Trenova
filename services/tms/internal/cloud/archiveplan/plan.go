package archiveplan

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
)

type FilterKind int

const (
	FilterOrganization FilterKind = iota + 1
	FilterOrganizationOrUnit
	FilterUnit
	FilterID
	FilterTwoParty
)

type PlannedTable struct {
	Name        string
	Position    int
	Columns     []string
	Types       []string
	PrimaryKey  []string
	Filter      FilterKind
	Parties     [2]string
	Deferred    []string
	Nulled      []string
	Mapped      map[string]string
	Sequences   []Sequence
	Partitioned bool
	Envelope    []string
}

func (t *PlannedTable) Part() string {
	return tenantarchive.TablePart(t.Position, t.Name)
}

func (t *PlannedTable) ColumnIndex(name string) int {
	return slices.Index(t.Columns, name)
}

func (t *PlannedTable) Rewrites() bool {
	return len(t.Nulled) > 0 || len(t.Mapped) > 0 || len(t.Envelope) > 0
}

type Sequence struct {
	Column string
	Name   string
}

type ColumnUse struct {
	Table  string
	Column string
}

type GlobalReference struct {
	Table string
	Key   []string
	Uses  []ColumnUse
}

type Plan struct {
	Tables      []*PlannedTable
	Omitted     []tenantarchive.Omission
	Left        []tenantarchive.Omission
	References  []*GlobalReference
	ForeignKeys []*ForeignKey
	Outside     []*ForeignKey
	byName      map[string]*PlannedTable
}

func (p *Plan) Table(name string) (*PlannedTable, bool) {
	t, ok := p.byName[name]
	return t, ok
}

type Input struct {
	Catalog  *Catalog
	Purpose  tenantarchive.Purpose
	Envelope map[string][]string
}

type builder struct {
	in       *Input
	plan     *Plan
	refs     map[string]*GlobalReference
	problems []string
}

func Build(in *Input) (*Plan, error) {
	b := &builder{
		in:   in,
		plan: &Plan{byName: make(map[string]*PlannedTable)},
		refs: make(map[string]*GlobalReference),
	}
	names := make([]string, 0, len(in.Catalog.Tables))
	for name := range in.Catalog.Tables {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		b.classify(in.Catalog.Tables[name])
	}
	for _, fk := range in.Catalog.ForeignKeys {
		b.foreignKey(fk)
	}
	if len(b.problems) == 0 {
		if err := b.order(); err != nil {
			b.problems = append(b.problems, err.Error())
		}
	}
	if len(b.problems) > 0 {
		return nil, fmt.Errorf("the schema cannot be archived as written: %s",
			strings.Join(b.problems, "; "))
	}
	refNames := make([]string, 0, len(b.refs))
	for name := range b.refs {
		refNames = append(refNames, name)
	}
	slices.Sort(refNames)
	for _, name := range refNames {
		b.plan.References = append(b.plan.References, b.refs[name])
	}
	for _, list := range [][]tenantarchive.Omission{b.plan.Omitted, b.plan.Left} {
		slices.SortFunc(list, func(a, c tenantarchive.Omission) int {
			return strings.Compare(a.Table+"."+a.Column, c.Table+"."+c.Column)
		})
	}
	return b.plan, nil
}

func (b *builder) omit(table, column, reason string) {
	b.plan.Left = append(b.plan.Left, tenantarchive.Omission{
		Table: tenantarchive.QualifiedTable(table), Column: column, Reason: reason,
	})
}

func (b *builder) secret(table, column, reason string) {
	b.plan.Omitted = append(b.plan.Omitted, tenantarchive.Omission{
		Table: tenantarchive.QualifiedTable(table), Column: column, Reason: reason,
	})
}

func (b *builder) classify(t *Table) {
	cat := b.in.Catalog
	_, global := cat.Globals[t.Name]
	switch {
	case t.Name == BusinessUnits:
		b.carry(t, FilterID, [2]string{})
		return
	case global:
		return
	}
	if parties, ok := TwoParty(t.Name); ok {
		b.carry(t, FilterTwoParty, parties)
		return
	}
	if !t.Tenant() {
		if _, ok := OutsideTenancy(t.Name); ok {
			return
		}
		for _, c := range t.Columns {
			if strings.HasSuffix(c.Name, "organization_id") {
				b.problems = append(b.problems, fmt.Sprintf(
					"%s names an organization in %s but has no organization_id or business_unit_id; "+
						"list it as two-party or outside tenancy in archiveplan",
					t.Name,
					c.Name,
				))
				return
			}
		}
		return
	}
	if reason, ok := Derived(t.Name, cat.Rollups); ok {
		b.omit(t.Name, "", "derived: "+reason)
		return
	}
	if reason, ok := NotCarried(t.Name); ok {
		b.omit(t.Name, "", "not carried: "+reason)
		return
	}
	if reason, ok := exportTables[t.Name]; ok && b.in.Purpose == tenantarchive.PurposeExport {
		b.secret(t.Name, "", reason)
		return
	}
	org, hasOrg := t.Column(OrganizationColumn)
	switch {
	case hasOrg && org.Nullable && t.Has(BusinessUnitColumn):
		b.carry(t, FilterOrganizationOrUnit, [2]string{})
	case hasOrg:
		b.carry(t, FilterOrganization, [2]string{})
	default:
		b.carry(t, FilterUnit, [2]string{})
	}
}

func (b *builder) carry(t *Table, filter FilterKind, parties [2]string) {
	planned := &PlannedTable{
		Name:        t.Name,
		PrimaryKey:  slices.Clone(t.PrimaryKey),
		Filter:      filter,
		Parties:     parties,
		Mapped:      make(map[string]string),
		Partitioned: t.Partitions,
	}
	envelope := b.in.Envelope[t.Name]
	for _, c := range t.Columns {
		if c.Generated {
			continue
		}
		secret := secretColumn(c.Name) || slices.Contains(envelope, c.Name)
		if b.in.Purpose == tenantarchive.PurposeExport && secret {
			b.secret(t.Name, c.Name, secretReason(c.Name))
			continue
		}
		planned.Columns = append(planned.Columns, c.Name)
		planned.Types = append(planned.Types, c.Type)
		if c.Serial != "" {
			planned.Sequences = append(planned.Sequences, Sequence{Column: c.Name, Name: c.Serial})
		}
		if slices.Contains(envelope, c.Name) {
			planned.Envelope = append(planned.Envelope, c.Name)
		}
	}
	if len(planned.PrimaryKey) == 0 {
		b.problems = append(b.problems, t.Name+" has no primary key")
		return
	}
	b.plan.Tables = append(b.plan.Tables, planned)
	b.plan.byName[t.Name] = planned
}

func (b *builder) foreignKey(fk *ForeignKey) {
	child, carried := b.plan.byName[fk.Child]
	if !carried {
		return
	}
	cat := b.in.Catalog
	if parent, ok := b.plan.byName[fk.Parent]; ok {
		b.plan.ForeignKeys = append(b.plan.ForeignKeys, fk)
		if !fk.SelfReferencing() && !withinTenant(fk, child, parent) &&
			!b.carriesUnit(fk) {
			b.plan.Outside = append(b.plan.Outside, fk)
		}
		return
	}
	if _, global := cat.Globals[fk.Parent]; global {
		b.global(fk, child)
		return
	}
	parentTable, known := cat.Tables[fk.Parent]
	if known && parentTable.Tenant() {
		b.nullOut(fk, child)
		return
	}
	b.problems = append(b.problems, fmt.Sprintf(
		"%s refers to %s, which is neither carried, a tenant table nor a global table",
		fk.Child, fk.Parent))
}

func withinTenant(fk *ForeignKey, child, parent *PlannedTable) bool {
	byOrg := func(t *PlannedTable) bool {
		return t.Filter == FilterOrganization || t.Filter == FilterOrganizationOrUnit
	}
	if fk.ScopedToTenant() && byOrg(child) && byOrg(parent) {
		return true
	}
	return slices.Contains(fk.ChildColumns, BusinessUnitColumn) &&
		parentOf(fk, BusinessUnitColumn) == BusinessUnitColumn &&
		child.Filter == FilterUnit && parent.Filter == FilterUnit
}

func (b *builder) carriesUnit(fk *ForeignKey) bool {
	table, ok := b.in.Catalog.Tables[fk.Child]
	if !ok {
		return false
	}
	for i, c := range fk.ChildColumns {
		if i >= len(fk.ParentColumns) || !unitColumn(c) || !unitColumn(fk.ParentColumns[i]) {
			continue
		}
		if column, found := table.Column(c); found && !column.Nullable {
			return true
		}
	}
	return false
}

func unitColumn(name string) bool {
	return name == BusinessUnitColumn || strings.HasSuffix(name, "_"+BusinessUnitColumn)
}

func (b *builder) global(fk *ForeignKey, child *PlannedTable) {
	key, ok := GlobalKey(fk.Parent)
	if !ok {
		b.problems = append(b.problems, fmt.Sprintf(
			"%s refers to the global table %s, which archiveplan has no natural key for", fk.Child,
			fk.Parent))
		return
	}
	if len(fk.ChildColumns) != 1 || len(fk.ParentColumns) != 1 || fk.ParentColumns[0] != IDColumn {
		b.problems = append(b.problems, fmt.Sprintf(
			"%s refers to the global table %s by more than its id", fk.Child, fk.Parent))
		return
	}
	column := fk.ChildColumns[0]
	if !slices.Contains(child.Columns, column) {
		return
	}
	ref := b.refs[fk.Parent]
	if ref == nil {
		ref = &GlobalReference{Table: fk.Parent, Key: key}
		b.refs[fk.Parent] = ref
	}
	use := ColumnUse{Table: fk.Child, Column: column}
	if !slices.Contains(ref.Uses, use) {
		ref.Uses = append(ref.Uses, use)
	}
	child.Mapped[column] = fk.Parent
}

func (b *builder) nullOut(fk *ForeignKey, child *PlannedTable) {
	own := fk.ownColumns()
	table := b.in.Catalog.Tables[fk.Child]
	if len(own) == 0 {
		b.problems = append(b.problems, fmt.Sprintf(
			"%s is carried but depends on %s, which is not", fk.Child, fk.Parent))
		return
	}
	for _, c := range own {
		column, ok := table.Column(c)
		if !ok || !column.Nullable {
			b.problems = append(b.problems, fmt.Sprintf(
				"%s.%s refers to %s, which is not carried, and cannot be left empty",
				fk.Child, c, fk.Parent))
			return
		}
	}
	for _, c := range own {
		if !slices.Contains(child.Nulled, c) && slices.Contains(child.Columns, c) {
			child.Nulled = append(child.Nulled, c)
			b.omit(fk.Child, c, "emptied: it refers to "+fk.Parent+", which is not carried")
		}
	}
}

func (p *Plan) Matches(m *tenantarchive.Manifest) error {
	if len(p.Tables) != len(m.Tables) {
		return tenantarchive.Refuse(tenantarchive.RefusalSchema, fmt.Sprintf(
			"the archive lists %d tables and this server carries %d at the same schema version",
			len(m.Tables), len(p.Tables)))
	}
	for i, t := range p.Tables {
		if got := m.Tables[i].Name; got != tenantarchive.QualifiedTable(t.Name) {
			return tenantarchive.Refuse(tenantarchive.RefusalSchema, fmt.Sprintf(
				"the archive's table %d is %s where this server loads %s", i+1, got, t.Name))
		}
	}
	return nil
}

func (t *PlannedTable) TypeOf(index int) string {
	return t.Types[index]
}
