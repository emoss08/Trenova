package tenantpurgerepository

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tableNames(targets []purgeTarget) []string {
	names := make([]string, 0, len(targets))
	for _, target := range targets {
		names = append(names, target.Table)
	}

	return names
}

func indexOf(names []string, name string) int {
	for i, candidate := range names {
		if candidate == name {
			return i
		}
	}

	return -1
}

func TestBuildPlanSkipsRetainedTablesAndKeysByOrganization(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	plan := buildPlan(&planInput{
		Tables: []tenantTable{
			{Name: "shipments", HasOrg: true, HasBU: true},
			{Name: "audit_entries", HasOrg: true, HasBU: true},
			{Name: "organization_subscriptions", HasOrg: true, HasBU: true},
			{Name: "users", HasBU: true},
			{Name: "bu_settings", HasBU: true},
		},
		OrganizationID: orgID,
		BusinessUnitID: buID,
	})

	assert.Equal(t, []string{"shipments"}, tableNames(plan.targets))
	assert.Equal(t, organizationColumn, plan.byTable["shipments"].Column)
	assert.Equal(t, orgID, plan.byTable["shipments"].Value)
}

func TestBuildPlanIncludesBusinessUnitTablesOnlyWhenTheUnitIsExclusive(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	input := &planInput{
		Tables:         []tenantTable{{Name: "bu_settings", HasBU: true}},
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: buID,
	}

	assert.Empty(t, buildPlan(input).targets)

	input.ExclusiveBU = true
	plan := buildPlan(input)
	require.Len(t, plan.targets, 1)
	assert.Equal(t, businessUnitColumn, plan.targets[0].Column)
	assert.Equal(t, buID, plan.targets[0].Value)
}

func TestBuildPlanOrdersChildrenBeforeParents(t *testing.T) {
	t.Parallel()

	plan := buildPlan(&planInput{
		Tables: []tenantTable{
			{Name: "customers", HasOrg: true},
			{Name: "shipments", HasOrg: true},
			{Name: "shipment_moves", HasOrg: true},
			{Name: "stops", HasOrg: true},
		},
		ForeignKeys: []*foreignKey{
			{Name: "fk_shipments_customer", ChildTable: "shipments", ParentTable: "customers", OnDelete: "a"},
			{Name: "fk_moves_shipment", ChildTable: "shipment_moves", ParentTable: "shipments", OnDelete: "c"},
			{Name: "fk_stops_move", ChildTable: "stops", ParentTable: "shipment_moves", OnDelete: "r"},
			{Name: "fk_stops_customer", ChildTable: "stops", ParentTable: "customers", OnDelete: "a"},
		},
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})

	names := tableNames(plan.targets)
	require.Len(t, names, 4)
	assert.Less(t, indexOf(names, "stops"), indexOf(names, "shipment_moves"))
	assert.Less(t, indexOf(names, "shipment_moves"), indexOf(names, "shipments"))
	assert.Less(t, indexOf(names, "shipments"), indexOf(names, "customers"))
	assert.Less(t, indexOf(names, "stops"), indexOf(names, "customers"))
}

func TestBuildPlanBreaksForeignKeyCyclesWithoutDroppingTables(t *testing.T) {
	t.Parallel()

	plan := buildPlan(&planInput{
		Tables: []tenantTable{
			{Name: "a", HasOrg: true},
			{Name: "b", HasOrg: true},
			{Name: "c", HasOrg: true},
		},
		ForeignKeys: []*foreignKey{
			{Name: "fk_a_b", ChildTable: "a", ParentTable: "b", OnDelete: "a"},
			{Name: "fk_b_a", ChildTable: "b", ParentTable: "a", OnDelete: "a"},
			{Name: "fk_c_c", ChildTable: "c", ParentTable: "c", OnDelete: "a"},
		},
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})

	assert.ElementsMatch(t, []string{"a", "b", "c"}, tableNames(plan.targets))
}

func TestBuildPlanRecordsRestrictingDependentsOutsideTheTenantTables(t *testing.T) {
	t.Parallel()

	plan := buildPlan(&planInput{
		Tables: []tenantTable{
			{Name: "customers", HasOrg: true},
			{Name: "shipments", HasOrg: true},
		},
		ForeignKeys: []*foreignKey{
			{Name: "fk_links_customer", ChildTable: "customer_links", ParentTable: "customers", OnDelete: "r"},
			{Name: "fk_notes_customer", ChildTable: "customer_notes", ParentTable: "customers", OnDelete: "c"},
			{Name: "fk_shipments_customer", ChildTable: "shipments", ParentTable: "customers", OnDelete: "a"},
			{Name: "fk_tree_customer", ChildTable: "customers", ParentTable: "customers", OnDelete: "a"},
		},
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})

	require.Len(t, plan.dependents["customers"], 1)
	assert.Equal(t, "customer_links", plan.dependents["customers"][0].ChildTable)
	assert.Contains(t, plan.constraints, "fk_tree_customer")
}

func TestRetainedCoversTheAppendOnlyAndAnchorTables(t *testing.T) {
	t.Parallel()

	for _, table := range []string{
		"audit_entries", "auth_events", "ai_audit_events", "ai_logs",
		"organizations", "business_units", "users", "organization_subscriptions",
	} {
		assert.True(t, Retained(table), table)
	}
	assert.False(t, Retained("shipments"))
}

func TestColumnListQuotesIdentifiers(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `("id", "organization_id")`, columnList([]string{"id", "organization_id"}))
	assert.Equal(t, `"we""ird"`, joinedColumns([]string{`we"ird`}))
}
