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

var keptTables = map[string]string{
	"organizations": "deleted last by DeleteOrganization, whose cascade removes the append-only " +
		"audit rows no direct delete may touch",
	"business_units": "deleted by DeleteBusinessUnit once nothing of the tenant references it",
}

type tenantTable struct {
	Name   string `db:"table_name"`
	HasOrg bool   `db:"has_org"`
	HasBU  bool   `db:"has_bu"`
}

type foreignKey struct {
	Name          string   `db:"constraint_name"`
	ChildTable    string   `db:"child_table"`
	ParentTable   string   `db:"parent_table"`
	OnDelete      string   `db:"on_delete"`
	ChildColumns  []string `db:"child_columns"`
	ParentColumns []string `db:"parent_columns"`
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

	for _, target := range targetsOf(in) {
		plan.byTable[target.Table] = target
	}

	for _, fk := range in.ForeignKeys {
		plan.constraints[fk.Name] = fk
		if _, isTarget := plan.byTable[fk.ParentTable]; !isTarget {
			continue
		}
		if _, childIsTarget := plan.byTable[fk.ChildTable]; childIsTarget || fk.selfReferencing() {
			continue
		}
		if !fk.restricts() || Kept(fk.ChildTable) {
			continue
		}
		plan.dependents[fk.ParentTable] = append(plan.dependents[fk.ParentTable], fk)
	}

	plan.targets = orderTargets(plan.byTable, in.ForeignKeys)

	return plan
}

func targetsOf(in *planInput) []purgeTarget {
	out := make([]purgeTarget, 0, len(in.Tables))
	for i := range in.Tables {
		table := in.Tables[i]
		if Kept(table.Name) {
			continue
		}
		switch {
		case table.HasOrg:
			out = append(out, purgeTarget{
				Table:  table.Name,
				Column: organizationColumn,
				Value:  in.OrganizationID,
			})
		case table.HasBU && in.ExclusiveBU:
			out = append(out, purgeTarget{
				Table:  table.Name,
				Column: businessUnitColumn,
				Value:  in.BusinessUnitID,
			})
		}
	}
	return out
}

func Kept(table string) bool {
	_, ok := keptTables[table]
	return ok
}

func orderTargets(byTable map[string]purgeTarget, fks []*foreignKey) []purgeTarget {
	names := make([]string, 0, len(byTable))
	for name := range byTable {
		names = append(names, name)
	}
	slices.Sort(names)

	parentsOf, childrenCount := parentGraph(byTable, fks)

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
			ready = append(ready, firstUnplaced(names, placed))
		}
		name := ready[0]
		ready = ready[1:]
		if _, done := placed[name]; done {
			continue
		}
		placed[name] = struct{}{}
		ordered = append(ordered, byTable[name])
		ready = append(ready, release(parentsOf[name], childrenCount)...)
	}

	return ordered
}

func parentGraph(
	byTable map[string]purgeTarget,
	fks []*foreignKey,
) (parentsOf map[string]map[string]struct{}, childrenCount map[string]int) {
	parentsOf = make(map[string]map[string]struct{}, len(byTable))
	childrenCount = make(map[string]int, len(byTable))
	for _, fk := range fks {
		_, childIn := byTable[fk.ChildTable]
		_, parentIn := byTable[fk.ParentTable]
		if fk.selfReferencing() || !childIn || !parentIn {
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
	return parentsOf, childrenCount
}

func firstUnplaced(names []string, placed map[string]struct{}) string {
	for _, name := range names {
		if _, done := placed[name]; !done {
			return name
		}
	}
	return ""
}

func release(parents map[string]struct{}, childrenCount map[string]int) []string {
	released := make([]string, 0)
	for parent := range parents {
		childrenCount[parent]--
		if childrenCount[parent] == 0 {
			released = append(released, parent)
		}
	}
	slices.Sort(released)
	return released
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func joinedColumns(columns []string, alias string) string {
	quoted := make([]string, 0, len(columns))
	for _, column := range columns {
		if alias != "" {
			quoted = append(quoted, alias+"."+quoteIdent(column))
			continue
		}
		quoted = append(quoted, quoteIdent(column))
	}

	return strings.Join(quoted, ", ")
}
