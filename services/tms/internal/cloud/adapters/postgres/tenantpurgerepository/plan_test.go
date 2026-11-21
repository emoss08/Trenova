package tenantpurgerepository

import (
	"slices"
	"strings"
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

func TestBuildPlanKeysByOrganizationAndKeepsOnlyTheOrganizationAndItsUnit(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	plan := buildPlan(&planInput{
		Tables: []tenantTable{
			{Name: "shipments", HasOrg: true, HasBU: true},
			{Name: "audit_entries", HasOrg: true, HasBU: true},
			{Name: "organizations", HasBU: true},
			{Name: "users", HasBU: true},
		},
		OrganizationID: orgID,
		BusinessUnitID: buID,
	})

	assert.ElementsMatch(t, []string{"shipments", "audit_entries"}, tableNames(plan.targets))
	assert.Equal(t, organizationColumn, plan.byTable["shipments"].Column)
	assert.Equal(t, orgID, plan.byTable["shipments"].Value)
	assert.True(t, Kept("organizations"))
	assert.True(t, Kept("business_units"))
}

func TestBuildPlanIncludesBusinessUnitTablesOnlyWhenTheUnitIsExclusive(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	input := &planInput{
		Tables:         []tenantTable{{Name: "users", HasBU: true}},
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

func TestATableWhosePartiesNameTheUnitIsPurgedByTheFirstPartysUnit(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	transfers := tenantTable{
		Name: "edi_load_tender_transfers", PartyBU: "source_business_unit_id",
	}
	input := &planInput{
		Tables:         []tenantTable{transfers},
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: buID,
	}
	assert.Empty(t, buildPlan(input).targets, "not while another organization shares the unit")

	input.ExclusiveBU = true
	plan := buildPlan(input)
	require.Len(t, plan.targets, 1)
	assert.Equal(t, "source_business_unit_id", plan.targets[0].Column)
	assert.Equal(t, buID, plan.targets[0].Value)
	assert.Contains(t, RemainingQuery([]tenantTable{transfers}),
		`FROM "edi_load_tender_transfers" WHERE "source_business_unit_id" = $2`)
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
			{
				Name:        "fk_shipments_customer",
				ChildTable:  "shipments",
				ParentTable: "customers",
				OnDelete:    "a",
			},
			{
				Name:        "fk_moves_shipment",
				ChildTable:  "shipment_moves",
				ParentTable: "shipments",
				OnDelete:    "c",
			},
			{
				Name:        "fk_stops_move",
				ChildTable:  "stops",
				ParentTable: "shipment_moves",
				OnDelete:    "r",
			},
			{
				Name:        "fk_stops_customer",
				ChildTable:  "stops",
				ParentTable: "customers",
				OnDelete:    "a",
			},
		},
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})

	names := tableNames(plan.targets)
	require.Len(t, names, 4)
	assert.Less(t, slices.Index(names, "stops"), slices.Index(names, "shipment_moves"))
	assert.Less(t, slices.Index(names, "shipment_moves"), slices.Index(names, "shipments"))
	assert.Less(t, slices.Index(names, "shipments"), slices.Index(names, "customers"))
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
		Tables: []tenantTable{{Name: "shipments", HasOrg: true}},
		ForeignKeys: []*foreignKey{
			{
				Name:        "fk_notes_shipment",
				ChildTable:  "notes",
				ParentTable: "shipments",
				OnDelete:    "r",
			},
			{Name: "fk_tags_shipment", ChildTable: "tags", ParentTable: "shipments", OnDelete: "c"},
			{
				Name:        "fk_orgs_shipment",
				ChildTable:  "organizations",
				ParentTable: "shipments",
				OnDelete:    "a",
			},
		},
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})

	require.Len(t, plan.dependents["shipments"], 1)
	assert.Equal(t, "notes", plan.dependents["shipments"][0].ChildTable)
}

func TestRemainingQueryCountsEveryTenantTableAndTheTenantItself(t *testing.T) {
	t.Parallel()

	sql := RemainingQuery([]tenantTable{
		{Name: "shipments", HasOrg: true, HasBU: true},
		{Name: "users", HasBU: true},
		{Name: "odd\"name", HasOrg: true},
	})

	assert.Contains(t, sql, `FROM "organizations" WHERE "id" = $1`)
	assert.Contains(t, sql, `FROM "business_units" WHERE "id" = $2`)
	assert.Contains(t, sql, `FROM "shipments" WHERE "organization_id" = $1`)
	assert.Contains(t, sql, `FROM "users" WHERE "business_unit_id" = $2`)
	assert.Contains(t, sql, `FROM "odd""name" WHERE`)
	assert.Equal(t, 4, strings.Count(sql, "UNION ALL"))
}

func TestHoldingQueryAsksEachTableOnceAndBindsBothTenantValues(t *testing.T) {
	t.Parallel()

	tables := []tenantTable{
		{Name: "shipments", HasOrg: true, HasBU: true},
		{Name: "users", HasBU: true},
		{Name: "organizations", HasBU: true},
	}

	shared := HoldingQuery(tables, true)
	assert.Contains(t, shared, `EXISTS (SELECT 1 FROM "shipments" WHERE "organization_id" = $1)`)
	assert.Contains(t, shared, `EXISTS (SELECT 1 FROM "users" WHERE "business_unit_id" = $2)`)
	assert.NotContains(t, shared, `FROM "organizations"`)

	notExclusive := HoldingQuery(tables, false)
	assert.NotContains(t, notExclusive, `"users"`)
	assert.Contains(t, notExclusive, "$2::text", "the unit is bound even when no table asks for it")

	held := holdingTables(tables, []string{"users"})
	require.Len(t, held, 1)
	assert.Equal(t, "users", held[0].Name)
}

func TestOnlyKeysThatCanNameAnotherTenantAreCheckedForInboundReferences(t *testing.T) {
	t.Parallel()

	tables := []tenantTable{
		{Name: "shipments", HasOrg: true, HasBU: true},
		{Name: "users", HasBU: true},
		{Name: "comments", HasOrg: true},
		{Name: "memberships", HasOrg: true, HasBU: true},
	}
	plan := buildPlan(&planInput{
		Tables:    tables,
		AllTables: tables,
		ForeignKeys: []*foreignKey{
			{Name: "fk_shipments_created_by", ChildTable: "shipments", ParentTable: "users",
				ChildColumns: []string{"created_by_id"}, ParentColumns: []string{"id"}},
			{Name: "fk_comments_user", ChildTable: "comments", ParentTable: "users",
				ChildColumns: []string{"user_id"}, ParentColumns: []string{"id"}},
			{Name: "fk_memberships_user", ChildTable: "memberships", ParentTable: "users",
				ChildColumns:  []string{"user_id", "business_unit_id"},
				ParentColumns: []string{"id", "business_unit_id"}},
			{Name: "fk_shipments_org", ChildTable: "shipments", ParentTable: "organizations",
				ChildColumns: []string{"organization_id"}, ParentColumns: []string{"id"}},
			{Name: "fk_users_current_org", ChildTable: "users", ParentTable: "organizations",
				ChildColumns: []string{"current_organization_id"}, ParentColumns: []string{"id"}},
		},
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		ExclusiveBU:    true,
	})

	names := make([]string, 0)
	for _, ref := range plan.inboundReferences() {
		names = append(names, ref.fk.Name)
	}
	assert.Equal(t, []string{"fk_comments_user", "fk_shipments_created_by", "fk_users_current_org"}, names)

	queries := plan.CrossTenantQueries()
	require.Len(t, queries, 1)
	assert.Contains(t, queries[0], `c."business_unit_id" = $2`)
	assert.Contains(t, queries[0], `c."organization_id" IN (SELECT o."id" FROM "organizations" o WHERE o."business_unit_id" = $2)`)
	assert.Contains(t, queries[0], `p."id" = $1`, "an organization parent is the removed one")
	assert.Contains(t, queries[0], `p."business_unit_id" = $2`, "a unit table's parent rows are the tenant's")
	assert.True(t, strings.Contains(queries[0], "$1::text IS NULL"), "both parameters are always used")
}
