package archiveplan_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/cloud/archiveplan"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func col(name string, nullable bool) archiveplan.Column {
	return archiveplan.Column{Name: name, Type: "character varying(100)", Nullable: nullable}
}

func tenantTable(name string, extra ...archiveplan.Column) *archiveplan.Table {
	cols := append([]archiveplan.Column{
		col("id", false), col("organization_id", false), col("business_unit_id", false),
	}, extra...)
	return &archiveplan.Table{
		Name:       name,
		Columns:    cols,
		PrimaryKey: []string{"id", "business_unit_id", "organization_id"},
	}
}

func fk(name, child, parent string, columns ...string) *archiveplan.ForeignKey {
	parents := make([]string, len(columns))
	for i, c := range columns {
		switch c {
		case "organization_id", "business_unit_id":
			parents[i] = c
		default:
			parents[i] = "id"
		}
	}
	return &archiveplan.ForeignKey{
		Name: name, Child: child, Parent: parent, ChildColumns: columns, ParentColumns: parents,
	}
}

func catalog() *archiveplan.Catalog {
	generated := archiveplan.Column{Name: "search_vector", Type: "tsvector", Generated: true}
	tables := map[string]*archiveplan.Table{
		"business_units": {Name: "business_units", Columns: []archiveplan.Column{col("id", false)},
			PrimaryKey: []string{"id"}},
		"us_states": {Name: "us_states", Columns: []archiveplan.Column{col("id", false)},
			PrimaryKey: []string{"id"}},
		"customers": tenantTable("customers", col("state_id", true), generated),
		"invoices": tenantTable(
			"invoices",
			col("customer_id", false),
			col("queue_item_id", true),
		),
		"queue":      tenantTable("billing_queue_items", col("invoice_id", false)),
		"users":      tenantTable("users", col("password", false)),
		"projection": tenantTable("invoice_search_projections"),
		"work":       tenantTable("work_queue_items"),
		"marks":      tenantTable("detention_daily"),
	}
	byName := make(map[string]*archiveplan.Table, len(tables))
	for _, t := range tables {
		byName[t.Name] = t
	}
	return &archiveplan.Catalog{
		Tables: byName,
		ForeignKeys: []*archiveplan.ForeignKey{
			fk("fk_customers_state", "customers", "us_states", "state_id"),
			fk(
				"fk_invoices_customer",
				"invoices",
				"customers",
				"customer_id",
				"business_unit_id",
				"organization_id",
			),
			fk(
				"fk_invoices_queue",
				"invoices",
				"billing_queue_items",
				"queue_item_id",
				"business_unit_id",
				"organization_id",
			),
			fk(
				"fk_queue_invoice",
				"billing_queue_items",
				"invoices",
				"invoice_id",
				"business_unit_id",
				"organization_id",
			),
			{Name: "fk_customers_bu", Child: "customers", Parent: "business_units",
				ChildColumns: []string{"business_unit_id"}, ParentColumns: []string{"id"}},
		},
		Rollups: []string{"detention_daily"},
		Globals: map[string]string{"us_states": "us_states", "business_units": "business_units"},
	}
}

func TestThePlanCarriesTheTenantInLoadOrderAndBreaksCycles(t *testing.T) {
	t.Parallel()

	plan, err := archiveplan.Build(
		&archiveplan.Input{Catalog: catalog(), Purpose: tenantarchive.PurposeFinal},
	)
	require.NoError(t, err)

	position := map[string]int{}
	for _, table := range plan.Tables {
		position[table.Name] = table.Position
	}
	assert.NotContains(
		t,
		position,
		"invoice_search_projections",
		"a projection is rebuilt, not carried",
	)
	assert.NotContains(t, position, "work_queue_items", "in-flight work is not carried")
	assert.NotContains(t, position, "detention_daily", "a rollup is rebuilt, not carried")
	assert.NotContains(t, position, "us_states", "a global table is not carried")
	assert.Less(t, position["business_units"], position["customers"])
	assert.Less(t, position["customers"], position["invoices"])
	assert.Less(
		t,
		position["invoices"],
		position["billing_queue_items"],
		"the cycle is broken on the invoices' nullable reference: invoices load without it, then the queue that needs them",
	)

	invoices, ok := plan.Table("invoices")
	require.True(t, ok)
	assert.Equal(t, []string{"queue_item_id"}, invoices.Deferred)
	customers, ok := plan.Table("customers")
	require.True(t, ok)
	assert.Equal(t, "us_states", customers.Mapped["state_id"])
	assert.NotContains(t, customers.Columns, "search_vector", "generated columns are never written")
	users, ok := plan.Table("users")
	require.True(t, ok)
	assert.Contains(t, users.Columns, "password", "a move carries password hashes")
	assert.Empty(t, plan.Omitted)
	assert.NotEmpty(t, plan.Left)
}

func TestAnExportLeavesOutSecrets(t *testing.T) {
	t.Parallel()

	plan, err := archiveplan.Build(&archiveplan.Input{
		Catalog:  catalog(),
		Purpose:  tenantarchive.PurposeExport,
		Envelope: map[string][]string{"customers": {"state_id"}},
	})
	require.NoError(t, err)
	users, ok := plan.Table("users")
	require.True(t, ok)
	assert.NotContains(t, users.Columns, "password")
	assert.Contains(t, plan.Omitted, tenantarchive.Omission{
		Table: "public.users", Column: "password", Reason: "password hashes",
	})
}

func TestACycleThatCannotBeBrokenIsRefused(t *testing.T) {
	t.Parallel()

	cat := catalog()
	cat.Tables["invoices"].Columns[4].Nullable = false
	cat.Tables["billing_queue_items"].Columns[3].Nullable = false
	_, err := archiveplan.Build(
		&archiveplan.Input{Catalog: cat, Purpose: tenantarchive.PurposeFinal},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "billing_queue_items")
}

func TestOnlyAKeyByIDAloneIsCheckedForRowsOutsideTheTenant(t *testing.T) {
	t.Parallel()

	cat := catalog()
	cat.Tables["invoices"].Columns = append(cat.Tables["invoices"].Columns,
		col("approved_by_id", true), col("posted_by_id", true))
	cat.ForeignKeys = append(cat.ForeignKeys,
		fk("fk_invoices_approved_by", "invoices", "users", "approved_by_id", "business_unit_id"),
		fk("fk_invoices_posted_by", "invoices", "users", "posted_by_id"),
	)
	plan, err := archiveplan.Build(
		&archiveplan.Input{Catalog: cat, Purpose: tenantarchive.PurposeFinal},
	)
	require.NoError(t, err)

	outside := make([]string, 0, len(plan.Outside))
	for _, k := range plan.Outside {
		outside = append(outside, k.Name)
	}
	assert.Contains(t, outside, "fk_invoices_posted_by")
	assert.NotContains(t, outside, "fk_invoices_approved_by",
		"a key that carries the business unit cannot name another tenant's row")

	cat.Tables["invoices"].Columns[2].Nullable = true
	plan, err = archiveplan.Build(
		&archiveplan.Input{Catalog: cat, Purpose: tenantarchive.PurposeFinal},
	)
	require.NoError(t, err)
	outside = outside[:0]
	for _, k := range plan.Outside {
		outside = append(outside, k.Name)
	}
	assert.Contains(t, outside, "fk_invoices_approved_by",
		"a business unit that may be empty leaves the key unchecked for that row")
}

func TestAGlobalReferenceWithoutANaturalKeyIsRefused(t *testing.T) {
	t.Parallel()

	cat := catalog()
	cat.Tables["hazmat"] = &archiveplan.Table{
		Name:       "hazmat",
		Columns:    []archiveplan.Column{col("id", false)},
		PrimaryKey: []string{"id"},
	}
	cat.Globals["hazmat"] = "hazmat"
	cat.Tables["customers"].Columns = append(
		cat.Tables["customers"].Columns,
		col("hazmat_id", true),
	)
	cat.ForeignKeys = append(
		cat.ForeignKeys,
		fk("fk_customers_hazmat", "customers", "hazmat", "hazmat_id"),
	)
	_, err := archiveplan.Build(
		&archiveplan.Input{Catalog: cat, Purpose: tenantarchive.PurposeFinal},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no natural key")
}
