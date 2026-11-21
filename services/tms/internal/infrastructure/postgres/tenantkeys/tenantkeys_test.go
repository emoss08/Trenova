package tenantkeys_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func table(name string, columns map[string]bool) *tenantkeys.Table {
	cols := map[string]bool{"id": true}
	for k, v := range columns {
		cols[k] = v
	}
	return &tenantkeys.Table{Name: name, Columns: cols}
}

func key(
	name, child, parent string,
	onDelete string,
	childCols, parentCols []string,
) *tenantkeys.ForeignKey {
	return &tenantkeys.ForeignKey{
		Name: name, Child: child, Parent: parent, ChildColumns: childCols, ParentColumns: parentCols,
		OnDelete: onDelete, OnUpdate: "a", Match: "s", Validated: true,
	}
}

func catalog(extra ...*tenantkeys.ForeignKey) *tenantkeys.Catalog {
	tenant := map[string]bool{"organization_id": true, "business_unit_id": true}
	tables := []*tenantkeys.Table{
		table("business_units", nil),
		table("organizations", map[string]bool{"business_unit_id": true}),
		table("users", map[string]bool{"business_unit_id": true}),
		table("customers", tenant),
		table("shipments", map[string]bool{
			"organization_id": true, "business_unit_id": true, "customer_id": true,
			"entered_by_id": false,
		}),
		table("mfa_codes", map[string]bool{"organization_id": true, "user_id": true}),
		table("roles", tenant),
		table("role_grants", map[string]bool{"organization_id": true, "role_id": true}),
		table("tender_transfers", map[string]bool{
			"source_organization_id": true, "source_business_unit_id": true,
			"target_organization_id": true, "target_business_unit_id": true,
		}),
		table(
			"shipment_links",
			map[string]bool{"business_unit_id": true, "tender_transfer_id": false},
		),
		table("us_states", nil),
	}
	tables[len(tables)-1].Global = true
	fks := append([]*tenantkeys.ForeignKey{
		key("fk_shipments_organization", "shipments", "organizations", "c",
			[]string{"organization_id"}, []string{"id"}),
		key("fk_shipments_business_unit", "shipments", "business_units", "c",
			[]string{"business_unit_id"}, []string{"id"}),
		key("fk_shipments_customer", "shipments", "customers", "r",
			[]string{"customer_id", "organization_id", "business_unit_id"},
			[]string{"id", "organization_id", "business_unit_id"}),
		key("fk_shipments_entered_by", "shipments", "users", "n",
			[]string{"entered_by_id"}, []string{"id"}),
		key("fk_mfa_codes_user", "mfa_codes", "users", "c", []string{"user_id"}, []string{"id"}),
		key(
			"fk_role_grants_role",
			"role_grants",
			"roles",
			"c",
			[]string{"role_id"},
			[]string{"id"},
		),
		key("fk_shipment_links_transfer", "shipment_links", "tender_transfers", "r",
			[]string{"tender_transfer_id"}, []string{"id"}),
		key(
			"fk_customers_state",
			"customers",
			"us_states",
			"a",
			[]string{"state_id"},
			[]string{"id"},
		),
	}, extra...)
	indexes := []*tenantkeys.Index{
		{Name: "users_pkey", Table: "users", Unique: true, Columns: []string{"id"}},
		{Name: "uq_users_id_business_unit", Table: "users", Unique: true,
			Columns: []string{"id", "business_unit_id"}},
		{Name: "pk_roles", Table: "roles", Unique: true,
			Columns: []string{"id", "business_unit_id", "organization_id"}},
		{
			Name:    "pk_tender_transfers",
			Table:   "tender_transfers",
			Unique:  true,
			Columns: []string{"id"},
		},
		{Name: "idx_shipments_tenant", Table: "shipments",
			Columns: []string{"organization_id", "business_unit_id", "customer_id"}},
		{Name: "idx_shipments_customer", Table: "shipments", Columns: []string{"customer_id"}},
		{Name: "idx_mfa_codes_user", Table: "mfa_codes", Columns: []string{"user_id"}},
		{Name: "idx_role_grants_role", Table: "role_grants", Columns: []string{"role_id"},
			Predicate: "(role_id IS NOT NULL)"},
		{Name: "idx_shipment_links_transfer", Table: "shipment_links",
			Columns: []string{"tender_transfer_id"}, Predicate: "(status = 'Open'::text)"},
	}
	return tenantkeys.NewCatalog(tables, fks, indexes, []string{"idx_shipments_tenant"})
}

func byName(inv *tenantkeys.Inventory) map[string]*tenantkeys.Key {
	out := make(map[string]*tenantkeys.Key, len(inv.Keys))
	for _, k := range inv.Keys {
		out[k.FK.Name] = k
	}
	return out
}

func TestClassifySortsEveryKeyBetweenTenantTables(t *testing.T) {
	inv := tenantkeys.ClassifyWith(catalog(), map[string]string{
		"fk_mfa_codes_user": "the table has no business unit",
		"fk_gone":           "a key that no longer exists",
	})
	keys := byName(inv)

	require.NotContains(
		t,
		keys,
		"fk_shipments_business_unit",
		"business_units is not a tenant table",
	)
	require.NotContains(t, keys, "fk_customers_state", "a global parent is not a tenant table")
	assert.Equal(t, tenantkeys.ClassTenantColumn, keys["fk_shipments_organization"].Class)
	assert.Equal(t, tenantkeys.ClassHeld, keys["fk_shipments_customer"].Class)

	entered := keys["fk_shipments_entered_by"]
	assert.Equal(t, tenantkeys.ClassConvertible, entered.Class)
	assert.Equal(
		t,
		tenantkeys.Form{Child: "business_unit_id", Parent: "business_unit_id"},
		entered.Form,
	)
	assert.Equal(t, []string{"entered_by_id", "business_unit_id"}, entered.Columns())
	assert.Equal(t, []string{"id", "business_unit_id"}, entered.ParentColumns())

	transfer := keys["fk_shipment_links_transfer"]
	assert.Equal(t, tenantkeys.ClassConvertible, transfer.Class)
	assert.Equal(t, tenantkeys.Form{Child: "business_unit_id", Parent: "source_business_unit_id"},
		transfer.Form, "a two-party parent is held by its first party's business unit")

	role := keys["fk_role_grants_role"]
	assert.Equal(t, tenantkeys.ClassConvertible, role.Class)
	assert.Equal(t, tenantkeys.Form{Child: "organization_id", Parent: "organization_id"}, role.Form,
		"a table without a business unit is held by its organization")

	assert.Equal(t, tenantkeys.ClassDeclared, keys["fk_mfa_codes_user"].Class)
	assert.Equal(t, "the table has no business unit", keys["fk_mfa_codes_user"].Reason)
	assert.Equal(
		t,
		[]string{"fk_gone: no foreign key between tenant tables has this name"},
		inv.Stale,
	)
}

func TestAKeyThatCannotCarryTheTenantMustBeDeclared(t *testing.T) {
	inv := tenantkeys.ClassifyWith(catalog(), nil)
	assert.Equal(t, tenantkeys.ClassUndeclared, byName(inv)["fk_mfa_codes_user"].Class)
}

func TestADeclarationThatNoLongerAppliesIsStale(t *testing.T) {
	inv := tenantkeys.ClassifyWith(catalog(), map[string]string{
		"fk_shipments_entered_by": "said it could not carry the tenant",
		"fk_shipments_customer":   "said it could not carry the tenant",
	})
	assert.Equal(t, []string{
		"fk_shipments_customer: it already carries the tenant",
		"fk_shipments_entered_by: it can carry business_unit_id now",
	}, inv.Stale)
}

func TestEveryKeyNeedsAnIndexLedByItsReferenceColumns(t *testing.T) {
	inv := tenantkeys.ClassifyWith(catalog(), nil)
	needs := make(map[string]*tenantkeys.IndexNeed)
	for _, idx := range inv.Indexes {
		needs[idx.Table+"("+strings.Join(idx.Columns, ",")+")"] = idx
	}

	assert.NotContains(t, needs, "shipments(customer_id)", "its own column leads an index")
	assert.NotContains(t, needs, "shipments(organization_id)",
		"the tenant column leads idx_shipments_tenant")
	assert.NotContains(t, needs, "mfa_codes(user_id)")
	assert.NotContains(t, needs, "role_grants(role_id)", "a NOT NULL partial index covers the key")

	entered := needs["shipments(entered_by_id)"]
	require.NotNil(t, entered)
	assert.Equal(t, "(entered_by_id IS NOT NULL)", entered.Predicate)
	assert.Equal(t, "idx_shipments_fk_entered_by_id", entered.Name)

	transfer := needs["shipment_links(tender_transfer_id)"]
	require.NotNil(t, transfer, "a partial index on another condition does not serve the key")

	uniques := make([]string, 0, len(inv.Uniques))
	for _, u := range inv.Uniques {
		uniques = append(uniques, u.Name+"("+strings.Join(u.Columns, ",")+")")
	}
	assert.ElementsMatch(t, []string{
		"uq_tender_transfers_id_source_business_unit(id,source_business_unit_id)",
		"uq_roles_id_organization(id,organization_id)",
	}, uniques, "users already has (id, business_unit_id)")
}

func TestRenderWritesTheFourMigrations(t *testing.T) {
	inv := tenantkeys.ClassifyWith(catalog(), map[string]string{
		"fk_mfa_codes_user": "the table has no business unit",
	})
	migrations, err := tenantkeys.Render(inv)
	require.NoError(t, err)
	require.Len(t, migrations, 4)

	indexes, added, validated, swapped := migrations[0], migrations[1], migrations[2], migrations[3]
	assert.Equal(t, "20261231000010_tenant_key_indexes.sql", indexes.File("20261231000010"))
	assert.True(t, strings.HasPrefix(indexes.SQL, "--keel:notx\n"))
	assert.Contains(
		t,
		indexes.SQL,
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipments_fk_entered_by_id"
    ON "shipments"("entered_by_id")
    WHERE "entered_by_id" IS NOT NULL;`,
	)
	assert.Contains(
		t,
		indexes.SQL,
		`CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS "uq_roles_id_organization"
    ON "roles"("id", "organization_id");`,
	)
	assert.Equal(t, len(inv.Uniques)+len(inv.Indexes)-1, strings.Count(indexes.SQL, "--keel:split"))

	temp := tenantkeys.TemporaryName("fk_shipments_entered_by")
	assert.Contains(
		t,
		added.SQL,
		`ADD CONSTRAINT "`+temp+`" FOREIGN KEY ("entered_by_id", "business_unit_id") `+
			`REFERENCES "users"("id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL ("entered_by_id") NOT VALID`,
	)
	assert.Contains(
		t,
		added.SQL,
		`FOREIGN KEY ("tender_transfer_id", "business_unit_id") REFERENCES `+
			`"tender_transfers"("id", "source_business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT NOT VALID`,
	)
	assert.Contains(t, added.SQL, `('shipments', ARRAY['entered_by_id'], 'users', ARRAY['id'], `+
		`'business_unit_id', 'business_unit_id')`)
	assert.Less(
		t,
		strings.Index(added.SQL, "DO $preflight$"),
		strings.Index(added.SQL, "ALTER TABLE"),
		"the data is checked before any key is added",
	)
	assert.NotContains(t, added.SQL, "fk_mfa_codes_user")

	assert.Contains(t, validated.SQL, `ALTER TABLE "shipments" VALIDATE CONSTRAINT "`+temp+`";`)
	assert.Contains(
		t,
		swapped.SQL,
		"ALTER TABLE \"shipments\"\n    DROP CONSTRAINT \"fk_shipments_entered_by\";",
	)
	assert.Contains(t, swapped.SQL, `ALTER TABLE "shipments" RENAME CONSTRAINT "`+temp+
		`" TO "fk_shipments_entered_by";`)
}

func TestRenderKeepsEveryDeleteBehaviour(t *testing.T) {
	cases := map[string]string{
		"a": "ON DELETE NO ACTION",
		"r": "ON DELETE RESTRICT",
		"c": "ON DELETE CASCADE",
		"n": `ON DELETE SET NULL ("entered_by_id")`,
		"d": `ON DELETE SET DEFAULT ("entered_by_id")`,
	}
	for code, want := range cases {
		cat := catalog()
		for _, fk := range cat.ForeignKeys {
			if fk.Name == "fk_shipments_entered_by" {
				fk.OnDelete = code
				fk.Deferrable, fk.Deferred = true, true
			}
		}
		migrations, err := tenantkeys.Render(tenantkeys.ClassifyWith(cat, tenantkeys.Exceptions()))
		require.NoError(t, err)
		assert.Contains(
			t,
			migrations[1].SQL,
			`REFERENCES "users"("id", "business_unit_id") ON UPDATE NO ACTION `+
				want+" DEFERRABLE INITIALLY DEFERRED NOT VALID",
			code,
		)
	}
}

func TestRenderRefusesWhatItCannotConvert(t *testing.T) {
	cat := catalog()
	cat.Tables["shipments"].Partitioned = true
	for _, fk := range cat.ForeignKeys {
		if fk.Name == "fk_role_grants_role" {
			fk.Match = "f"
		}
	}
	inv := tenantkeys.ClassifyWith(cat, map[string]string{"fk_mfa_codes_user": "no business unit"})
	_, err := tenantkeys.Render(inv)
	require.ErrorIs(t, err, tenantkeys.ErrUnsupported)
	assert.Contains(t, err.Error(), "fk_shipments_entered_by: a partitioned table")
	assert.Contains(t, err.Error(), "fk_role_grants_role: MATCH FULL")
}

func TestGeneratedNamesFitAndNeverCollide(t *testing.T) {
	long := strings.Repeat("a", 60)
	cat := catalog(key("fk_long", long, "users", "n", []string{"approved_by_id"}, []string{"id"}))
	cat.Tables[long] = table(
		long,
		map[string]bool{"business_unit_id": true, "approved_by_id": false},
	)
	inv := tenantkeys.ClassifyWith(cat, map[string]string{"fk_mfa_codes_user": "no business unit"})
	for _, idx := range inv.Indexes {
		assert.LessOrEqual(t, len(idx.Name), 63, idx.Name)
	}
	name := tenantkeys.TemporaryName("fk_long")
	assert.Equal(t, name, tenantkeys.TemporaryName("fk_long"))
	assert.NotEqual(t, name, tenantkeys.TemporaryName("fk_longer"))
	assert.LessOrEqual(t, len(name), 63)
}
