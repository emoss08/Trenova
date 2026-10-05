package tenantpurgerepository

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/shared/pulid"
)

const (
	organizationColumn = "organization_id"
	businessUnitColumn = "business_unit_id"

	onDeleteNoAction = "a"
	onDeleteRestrict = "r"
)

var retainedTables = map[string]struct{}{
	"organizations":              {},
	"business_units":             {},
	"users":                      {},
	"organization_subscriptions": {},
	"audit_entries":              {},
	"auth_events":                {},
	"ai_audit_events":            {},
	"ai_audit_chain_heads":       {},
	"ai_audit_seals":             {},
	"ai_audit_exports":           {},
	"ai_logs":                    {},
}

type tenantTable struct {
	Name   string `bun:"table_name"`
	HasOrg bool   `bun:"has_org"`
	HasBU  bool   `bun:"has_bu"`
}

type foreignKey struct {
	Name          string   `bun:"constraint_name"`
	ChildTable    string   `bun:"child_table"`
	ParentTable   string   `bun:"parent_table"`
	OnDelete      string   `bun:"on_delete"`
	ChildColumns  []string `bun:"child_columns,array"`
	ParentColumns []string `bun:"parent_columns,array"`
}

func (fk *foreignKey) restricts() bool {
	return fk.OnDelete == onDeleteNoAction || fk.OnDelete == onDeleteRestrict
}

func (fk *foreignKey) selfReferencing() bool {
	return fk.ChildTable == fk.ParentTable
}

type purgeTarget struct {
	Table  string
	Column string
	Value  pulid.ID
}

type purgePlan struct {
	targets     []purgeTarget
	byTable     map[string]purgeTarget
	constraints map[string]*foreignKey
	dependents  map[string][]*foreignKey
}

type planInput struct {
	Tables         []tenantTable
	ForeignKeys    []*foreignKey
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	ExclusiveBU    bool
}

func buildPlan(in *planInput) *purgePlan {
	plan := &purgePlan{
		byTable:     make(map[string]purgeTarget, len(in.Tables)),
		constraints: make(map[string]*foreignKey, len(in.ForeignKeys)),
		dependents:  make(map[string][]*foreignKey),
	}

	for i := range in.Tables {
		table := in.Tables[i]
		if Retained(table.Name) {
			continue
		}

		switch {
		case table.HasOrg:
			plan.byTable[table.Name] = purgeTarget{
				Table:  table.Name,
				Column: organizationColumn,
				Value:  in.OrganizationID,
			}
		case table.HasBU && in.ExclusiveBU:
			plan.byTable[table.Name] = purgeTarget{
				Table:  table.Name,
				Column: businessUnitColumn,
				Value:  in.BusinessUnitID,
			}
		}
	}

	for _, fk := range in.ForeignKeys {
		plan.constraints[fk.Name] = fk
		if _, isTarget := plan.byTable[fk.ParentTable]; !isTarget {
			continue
		}
		if _, childIsTarget := plan.byTable[fk.ChildTable]; childIsTarget || fk.selfReferencing() {
			continue
		}
		if !fk.restricts() {
			continue
		}
		plan.dependents[fk.ParentTable] = append(plan.dependents[fk.ParentTable], fk)
	}

	plan.targets = orderTargets(plan.byTable, in.ForeignKeys)

	return plan
}

func Retained(table string) bool {
	_, ok := retainedTables[table]
	return ok
}

func orderTargets(byTable map[string]purgeTarget, fks []*foreignKey) []purgeTarget {
	names := make([]string, 0, len(byTable))
	for name := range byTable {
		names = append(names, name)
	}
	slices.Sort(names)

	parentsOf := make(map[string]map[string]struct{}, len(names))
	childrenCount := make(map[string]int, len(names))
	for _, fk := range fks {
		if fk.selfReferencing() {
			continue
		}
		if _, ok := byTable[fk.ChildTable]; !ok {
			continue
		}
		if _, ok := byTable[fk.ParentTable]; !ok {
			continue
		}
		parents := parentsOf[fk.ChildTable]
		if parents == nil {
			parents = make(map[string]struct{})
			parentsOf[fk.ChildTable] = parents
		}
		if _, seen := parents[fk.ParentTable]; seen {
			continue
		}
		parents[fk.ParentTable] = struct{}{}
		childrenCount[fk.ParentTable]++
	}

	ordered := make([]purgeTarget, 0, len(names))
	placed := make(map[string]struct{}, len(names))
	ready := make([]string, 0, len(names))
	for _, name := range names {
		if childrenCount[name] == 0 {
			ready = append(ready, name)
		}
	}

	for len(ordered) < len(names) {
		if len(ready) == 0 {
			for _, name := range names {
				if _, done := placed[name]; !done {
					ready = append(ready, name)
					break
				}
			}
		}

		name := ready[0]
		ready = ready[1:]
		if _, done := placed[name]; done {
			continue
		}
		placed[name] = struct{}{}
		ordered = append(ordered, byTable[name])

		released := make([]string, 0)
		for parent := range parentsOf[name] {
			childrenCount[parent]--
			if childrenCount[parent] == 0 {
				released = append(released, parent)
			}
		}
		slices.Sort(released)
		ready = append(ready, released...)
	}

	return ordered
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func joinedColumns(columns []string) string {
	quoted := make([]string, 0, len(columns))
	for _, column := range columns {
		quoted = append(quoted, quoteIdent(column))
	}

	return strings.Join(quoted, ", ")
}

func columnList(columns []string) string {
	return "(" + joinedColumns(columns) + ")"
}
