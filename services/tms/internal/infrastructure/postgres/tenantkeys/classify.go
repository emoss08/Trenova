package tenantkeys

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
)

const maxIdentifier = 63

type Class int

const (
	ClassTenantColumn Class = iota + 1
	ClassHeld
	ClassConvertible
	ClassDeclared
	ClassUndeclared
)

func (c Class) String() string {
	switch c {
	case ClassTenantColumn:
		return "tenant column"
	case ClassHeld:
		return "held"
	case ClassConvertible:
		return "convertible"
	case ClassDeclared:
		return "declared exception"
	case ClassUndeclared:
		return "undeclared"
	default:
		return "unknown"
	}
}

type Form struct {
	Child  string
	Parent string
}

type IndexNeed struct {
	Table     string
	Name      string
	Columns   []string
	Predicate string
}

type UniqueNeed struct {
	Table   string
	Name    string
	Columns []string
}

type Key struct {
	FK     *ForeignKey
	Class  Class
	Reason string
	Form   Form
	Index  *IndexNeed
}

func (k *Key) Columns() []string {
	if k.Class != ClassConvertible {
		return slices.Clone(k.FK.ChildColumns)
	}
	return append(slices.Clone(k.FK.ChildColumns), k.Form.Child)
}

func (k *Key) ParentColumns() []string {
	if k.Class != ClassConvertible {
		return slices.Clone(k.FK.ParentColumns)
	}
	return append(slices.Clone(k.FK.ParentColumns), k.Form.Parent)
}

type Inventory struct {
	Keys        []*Key
	Uniques     []*UniqueNeed
	Indexes     []*IndexNeed
	Stale       []string
	Unsupported []string
}

func (inv *Inventory) Count(class Class) int {
	n := 0
	for _, k := range inv.Keys {
		if k.Class == class {
			n++
		}
	}
	return n
}

func (inv *Inventory) Of(class Class) []*Key {
	out := make([]*Key, 0)
	for _, k := range inv.Keys {
		if k.Class == class {
			out = append(out, k)
		}
	}
	return out
}

func Classify(cat *Catalog) *Inventory {
	return ClassifyWith(cat, exceptions)
}

type classifier struct {
	cat      *Catalog
	declared map[string]string
	inv      *Inventory
	names    map[string]struct{}
	indexes  map[string]*IndexNeed
	uniques  map[string]*UniqueNeed
}

func ClassifyWith(cat *Catalog, declared map[string]string) *Inventory {
	c := &classifier{
		cat:      cat,
		declared: declared,
		inv:      &Inventory{},
		names:    make(map[string]struct{}),
		indexes:  make(map[string]*IndexNeed),
		uniques:  make(map[string]*UniqueNeed),
	}
	seen := make(map[string]struct{}, len(declared))
	for _, fk := range cat.ForeignKeys {
		child, parent, ok := c.between(fk)
		if !ok {
			continue
		}
		seen[fk.Name] = struct{}{}
		key := c.classify(fk, child, parent)
		key.Index = c.indexFor(key, child)
		if key.Class == ClassConvertible {
			c.uniqueFor(key, parent)
			c.supported(key, child, parent)
		}
		c.inv.Keys = append(c.inv.Keys, key)
	}
	for name := range declared {
		if _, ok := seen[name]; !ok {
			c.inv.Stale = append(
				c.inv.Stale,
				name+": no foreign key between tenant tables has this name",
			)
		}
	}
	slices.Sort(c.inv.Stale)
	slices.SortFunc(
		c.inv.Keys,
		func(a, b *Key) int { return strings.Compare(a.FK.Name, b.FK.Name) },
	)
	slices.SortFunc(
		c.inv.Indexes,
		func(a, b *IndexNeed) int { return strings.Compare(a.Name, b.Name) },
	)
	slices.SortFunc(
		c.inv.Uniques,
		func(a, b *UniqueNeed) int { return strings.Compare(a.Name, b.Name) },
	)
	slices.Sort(c.inv.Unsupported)
	return c.inv
}

func (c *classifier) between(fk *ForeignKey) (child, parent *Table, ok bool) {
	child, ok = c.cat.Table(fk.Child)
	if !ok || !child.Tenant() {
		return nil, nil, false
	}
	parent, ok = c.cat.Table(fk.Parent)
	if !ok || !parent.Tenant() {
		return nil, nil, false
	}
	return child, parent, true
}

func (c *classifier) classify(fk *ForeignKey, child, parent *Table) *Key {
	key := &Key{FK: fk}
	reason, declared := c.declared[fk.Name]
	switch {
	case tenantColumnKey(fk):
		key.Class = ClassTenantColumn
		if declared {
			c.inv.Stale = append(c.inv.Stale, fk.Name+": it is the row's own tenant column")
		}
	case held(fk, child):
		key.Class = ClassHeld
		if declared {
			c.inv.Stale = append(c.inv.Stale, fk.Name+": it already carries the tenant")
		}
	default:
		form, convertible := formFor(fk, child, parent)
		switch {
		case convertible:
			key.Class = ClassConvertible
			key.Form = form
			if declared {
				c.inv.Stale = append(c.inv.Stale, fk.Name+": it can carry "+form.Child+" now")
			}
		case declared:
			key.Class = ClassDeclared
			key.Reason = reason
		default:
			key.Class = ClassUndeclared
		}
	}
	return key
}

func tenantColumnKey(fk *ForeignKey) bool {
	return fk.Parent == Organizations &&
		slices.Equal(fk.ChildColumns, []string{OrganizationColumn}) &&
		slices.Equal(fk.ParentColumns, []string{"id"})
}

type tenantKind int

const (
	kindNone tenantKind = iota
	kindUnit
	kindOrganization
)

func kindOf(column string) tenantKind {
	switch {
	case column == BusinessUnitColumn || strings.HasSuffix(column, unitSuffix):
		return kindUnit
	case column == OrganizationColumn || strings.HasSuffix(column, organizationSuffix):
		return kindOrganization
	default:
		return kindNone
	}
}

func held(fk *ForeignKey, child *Table) bool {
	for i, column := range fk.ChildColumns {
		if i >= len(fk.ParentColumns) {
			return false
		}
		kind := kindOf(column)
		if kind != kindNone && kind == kindOf(fk.ParentColumns[i]) && child.Required(column) {
			return true
		}
	}
	return false
}

func formFor(fk *ForeignKey, child, parent *Table) (Form, bool) {
	cu, pu := child.UnitColumn(), parent.UnitColumn()
	if cu != "" && pu != "" && child.Required(cu) && parent.Required(pu) &&
		!slices.Contains(fk.ChildColumns, cu) && !slices.Contains(fk.ParentColumns, pu) {
		return Form{Child: cu, Parent: pu}, true
	}
	if child.Required(OrganizationColumn) && parent.Required(OrganizationColumn) &&
		!slices.Contains(fk.ChildColumns, OrganizationColumn) &&
		!slices.Contains(fk.ParentColumns, OrganizationColumn) {
		return Form{Child: OrganizationColumn, Parent: OrganizationColumn}, true
	}
	return Form{}, false
}

func (c *classifier) supported(key *Key, child, parent *Table) {
	fk := key.FK
	problems := make([]string, 0)
	if child.Partitioned || parent.Partitioned {
		problems = append(problems, "a partitioned table cannot take a NOT VALID foreign key")
	}
	if fk.OnUpdate == actionSetNull || fk.OnUpdate == actionSetDefault {
		problems = append(
			problems,
			"ON UPDATE SET NULL or SET DEFAULT would change "+key.Form.Child,
		)
	}
	if fk.Match == matchFull {
		problems = append(problems, "MATCH FULL would refuse a reference left empty")
	}
	if len(problems) > 0 {
		c.inv.Unsupported = append(c.inv.Unsupported, fk.Name+": "+strings.Join(problems, "; "))
	}
}

func ownColumns(columns []string, child *Table) []string {
	out := make([]string, 0, len(columns))
	for _, column := range columns {
		if column == OrganizationColumn || column == BusinessUnitColumn ||
			(child.PartyUnit != "" && column == child.PartyUnit) {
			continue
		}
		out = append(out, column)
	}
	return out
}

func notNullPredicate(columns []string, child *Table) string {
	nullable := make([]string, 0, len(columns))
	for _, column := range columns {
		if !child.Required(column) {
			nullable = append(nullable, column)
		}
	}
	return notNull(nullable)
}

func notNull(columns []string) string {
	parts := make([]string, 0, len(columns))
	for _, column := range columns {
		parts = append(parts, "("+column+" IS NOT NULL)")
	}
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	default:
		return "(" + strings.Join(parts, " AND ") + ")"
	}
}

func (c *classifier) covered(child *Table, sets ...[]string) bool {
	for _, idx := range c.cat.Indexes[child.Name] {
		if idx.Expressions {
			continue
		}
		for _, set := range sets {
			if len(set) == 0 || len(idx.Columns) < len(set) {
				continue
			}
			if !sameSet(idx.Columns[:len(set)], set) {
				continue
			}
			if idx.Predicate == "" || idx.Predicate == notNullPredicate(set, child) ||
				idx.Predicate == notNull(set) {
				return true
			}
		}
	}
	return false
}

func (c *classifier) indexFor(key *Key, child *Table) *IndexNeed {
	columns := key.Columns()
	own := ownColumns(columns, child)
	if c.covered(child, columns, own) {
		return nil
	}
	target := own
	if len(target) == 0 {
		target = columns
	}
	predicate := notNullPredicate(target, child)
	dedupe := child.Name + "|" + strings.Join(target, ",") + "|" + predicate
	if need, ok := c.indexes[dedupe]; ok {
		return need
	}
	if child.Partitioned {
		c.inv.Unsupported = append(c.inv.Unsupported, key.FK.Name+": "+child.Name+
			" is partitioned, so its index is CREATE INDEX ON ONLY, one CONCURRENTLY per partition, "+
			"then ALTER INDEX ... ATTACH PARTITION, written by hand")
	}
	need := &IndexNeed{
		Table:     child.Name,
		Name:      c.name("idx_"+child.Name+"_fk_"+strings.Join(target, "_"), dedupe),
		Columns:   target,
		Predicate: predicate,
	}
	c.indexes[dedupe] = need
	c.inv.Indexes = append(c.inv.Indexes, need)
	return need
}

func (c *classifier) uniqueFor(key *Key, parent *Table) {
	columns := key.ParentColumns()
	for _, idx := range c.cat.Indexes[parent.Name] {
		if idx.Unique && !idx.Expressions && idx.Predicate == "" && sameSet(idx.Columns, columns) {
			return
		}
	}
	sorted := slices.Clone(columns)
	slices.Sort(sorted)
	dedupe := parent.Name + "|" + strings.Join(sorted, ",")
	if _, ok := c.uniques[dedupe]; ok {
		return
	}
	base := "uq_" + parent.Name + "_" + strings.Join(key.FK.ParentColumns, "_") + "_" +
		strings.TrimSuffix(key.Form.Parent, "_id")
	need := &UniqueNeed{Table: parent.Name, Name: c.name(base, dedupe), Columns: columns}
	c.uniques[dedupe] = need
	c.inv.Uniques = append(c.inv.Uniques, need)
}

func (c *classifier) name(base, seed string) string {
	_, taken := c.cat.Names[base]
	_, mine := c.names[base]
	if len(base) <= maxIdentifier && !taken && !mine {
		c.names[base] = struct{}{}
		return base
	}
	sum := sha256.Sum256([]byte(seed))
	suffix := "_" + hex.EncodeToString(sum[:])[:8]
	if len(base) > maxIdentifier-len(suffix) {
		base = base[:maxIdentifier-len(suffix)]
	}
	name := base + suffix
	c.names[name] = struct{}{}
	return name
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x, y := slices.Clone(a), slices.Clone(b)
	slices.Sort(x)
	slices.Sort(y)
	return slices.Equal(x, y)
}

func TemporaryName(constraint string) string {
	sum := sha256.Sum256([]byte(constraint))
	return fmt.Sprintf("tk_%s", hex.EncodeToString(sum[:])[:24])
}
