//go:build integration

package keelcheck_test

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/keelcheck"
	vf "github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type checkConn struct {
	db *keel.DB
}

func (c checkConn) Keel(context.Context) *keel.DB { return c.db }

func (c checkConn) RunScoped(
	ctx context.Context,
	readOnly bool,
	fn func(context.Context) error,
) error {
	if readOnly {
		return keel.Read(ctx, c.db, fn)
	}

	return keel.Write(ctx, c.db, fn)
}

type checkTenant struct {
	ctx   context.Context
	orgID pulid.ID
	buID  pulid.ID
}

func newCheckTenant(ctx context.Context) checkTenant {
	tenant := dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
	}

	return checkTenant{
		ctx:   dbscope.WithTenant(ctx, tenant),
		orgID: tenant.OrganizationID,
		buID:  tenant.BusinessUnitID,
	}
}

func setupCheckDB(t *testing.T) (checkConn, checkTenant, checkTenant) {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	for _, stmt := range []string{
		`CREATE TABLE check_kinds (
			id VARCHAR(100) NOT NULL,
			organization_id VARCHAR(100) NOT NULL,
			business_unit_id VARCHAR(100) NOT NULL,
			PRIMARY KEY (id, organization_id, business_unit_id)
		)`,
		`CREATE TABLE check_states (id VARCHAR(100) PRIMARY KEY)`,
		`CREATE TABLE check_items (
			id VARCHAR(100) NOT NULL,
			organization_id VARCHAR(100) NOT NULL,
			business_unit_id VARCHAR(100) NOT NULL,
			code VARCHAR(50) NOT NULL,
			kind_id VARCHAR(100),
			PRIMARY KEY (id, organization_id, business_unit_id)
		)`,
	} {
		_, err := keel.Exec(ctx, db, stmt)
		require.NoError(t, err)
	}

	return checkConn{db: db}, newCheckTenant(ctx), newCheckTenant(ctx)
}

func insertCheckItem(t *testing.T, conn checkConn, tenant checkTenant, id pulid.ID, code string) {
	t.Helper()

	_, err := keel.Exec(
		t.Context(),
		conn.db,
		"INSERT INTO check_items (id, organization_id, business_unit_id, code) VALUES ($1, $2, $3, $4)",
		id.String(),
		tenant.orgID.String(),
		tenant.buID.String(),
		code,
	)
	require.NoError(t, err)
}

func TestKeelChecker_CheckUniqueness_Integration(t *testing.T) {
	conn, a, b := setupCheckDB(t)
	items, _, _ := integrationTables()
	checker := keelcheck.NewChecker(conn, items)

	existing := pulid.MustNew("ci_")
	insertCheckItem(t, conn, a, existing, "CODE001")

	unique := func(tenant checkTenant, req *vf.UniquenessRequest) bool {
		t.Helper()
		req.TableName = "check_items"
		req.OrganizationID = tenant.orgID
		req.BusinessUnitID = tenant.buID
		exists, err := checker.CheckUniqueness(tenant.ctx, req)
		require.NoError(t, err)
		return exists
	}

	assert.False(
		t,
		unique(
			a,
			&vf.UniquenessRequest{Fields: []vf.FieldCheck{{Column: "code", Value: "CODE002"}}},
		),
	)
	assert.True(
		t,
		unique(
			a,
			&vf.UniquenessRequest{Fields: []vf.FieldCheck{{Column: "code", Value: "CODE001"}}},
		),
	)
	assert.True(
		t,
		unique(
			a,
			&vf.UniquenessRequest{Fields: []vf.FieldCheck{{Column: "code", Value: "code001"}}},
		),
		"a case-insensitive check matches another case",
	)
	assert.False(t, unique(a, &vf.UniquenessRequest{
		Fields: []vf.FieldCheck{{Column: "code", Value: "code001", CaseSensitive: true}},
	}))
	assert.True(t, unique(a, &vf.UniquenessRequest{
		Fields: []vf.FieldCheck{{Column: "code", Value: "CODE001", CaseSensitive: true}},
	}))
	assert.False(t, unique(a, &vf.UniquenessRequest{
		ExcludeID: existing,
		Fields:    []vf.FieldCheck{{Column: "code", Value: "CODE001"}},
	}), "an update excludes the row itself")
	assert.True(t, unique(a, &vf.UniquenessRequest{
		ExcludeID: pulid.MustNew("ci_"),
		Fields:    []vf.FieldCheck{{Column: "code", Value: "CODE001"}},
	}))
	assert.False(
		t,
		unique(
			b,
			&vf.UniquenessRequest{Fields: []vf.FieldCheck{{Column: "code", Value: "CODE001"}}},
		),
		"another tenant's row is not a conflict",
	)
	assert.False(t, unique(a, &vf.UniquenessRequest{
		Fields:      []vf.FieldCheck{{Column: "code", Value: "CODE001"}},
		ScopeFields: []vf.FieldCheck{{Column: "id", Value: "ci_other", CaseSensitive: true}},
	}), "every field must match")

	system := dbscope.WithSystem(t.Context(), "uniqueness check test")
	exists, err := checker.CheckUniqueness(system, &vf.UniquenessRequest{
		TableName:      "check_items",
		OrganizationID: b.orgID,
		BusinessUnitID: b.buID,
		Fields:         []vf.FieldCheck{{Column: "code", Value: "CODE001"}},
	})
	require.NoError(t, err)
	assert.False(t, exists, "a system-scope check filters the named tenant")
}

func TestKeelChecker_CheckReference_Integration(t *testing.T) {
	conn, a, b := setupCheckDB(t)
	items, _, states := integrationTables()
	checker := keelcheck.NewChecker(conn, items)

	kind := pulid.MustNew("ck_")
	_, err := keel.Exec(t.Context(), conn.db,
		"INSERT INTO check_kinds (id, organization_id, business_unit_id) VALUES ($1, $2, $3)",
		kind.String(), a.orgID.String(), a.buID.String())
	require.NoError(t, err)

	reference := func(tenant checkTenant, table string, id pulid.ID) bool {
		t.Helper()
		exists, refErr := checker.CheckReference(tenant.ctx, &vf.ReferenceRequest{
			TableName:      table,
			OrganizationID: tenant.orgID,
			BusinessUnitID: tenant.buID,
			ID:             id,
		})
		require.NoError(t, refErr)
		return exists
	}

	assert.True(t, reference(a, "check_kinds", kind), "a related table is known to the checker")
	assert.False(t, reference(a, "check_kinds", pulid.MustNew("ck_")))
	assert.False(t, reference(b, "check_kinds", kind), "another tenant's row is not a reference")

	_, err = checker.CheckReference(a.ctx, &vf.ReferenceRequest{ID: kind})
	require.Error(t, err)
	_, err = checker.CheckReference(a.ctx, &vf.ReferenceRequest{TableName: "check_kinds"})
	require.Error(t, err)

	state := pulid.MustNew("st_")
	_, err = keel.Exec(
		t.Context(),
		conn.db,
		"INSERT INTO check_states (id) VALUES ($1)",
		state.String(),
	)
	require.NoError(t, err)

	exists, err := keelcheck.ReferenceCheck(conn, states)(b.ctx, b.orgID, b.buID, state)
	require.NoError(t, err)
	assert.True(t, exists, "a global table answers every tenant")
}

func integrationTable(name, alias string, extra ...schema.Column) *schema.Table {
	cols := append([]schema.Column{
		{Name: "id", Field: "ID", Kind: schema.KindString, Flags: schema.FlagPK},
		{
			Name:  "organization_id",
			Field: "OrganizationID",
			Kind:  schema.KindString,
			Flags: schema.FlagPK | schema.FlagTenant,
		},
		{
			Name:  "business_unit_id",
			Field: "BusinessUnitID",
			Kind:  schema.KindString,
			Flags: schema.FlagPK | schema.FlagTenant,
		},
	}, extra...)
	for i := range cols {
		cols[i].Quoted = schema.Quote(cols[i].Name)
	}

	t := &schema.Table{Name: name, Quoted: schema.Quote(name), Alias: alias, Columns: cols}
	schema.Derive(t)

	return t
}

func integrationTables() (*schema.Table, *schema.Table, *schema.Table) {
	kinds := integrationTable("check_kinds", "ck")
	states := &schema.Table{
		Name:   "check_states",
		Quoted: schema.Quote("check_states"),
		Alias:  "cs",
		Columns: []schema.Column{
			{
				Name:   "id",
				Quoted: schema.Quote("id"),
				Field:  "ID",
				Kind:   schema.KindString,
				Flags:  schema.FlagPK,
			},
		},
	}
	schema.Derive(states)

	items := integrationTable("check_items", "ci",
		schema.Column{Name: "code", Field: "Code", Kind: schema.KindString},
		schema.Column{Name: "kind_id", Field: "KindID", Kind: schema.KindString},
	)
	items.Relations = []schema.Relation{
		{
			Name:   "Kind",
			Alias:  "k",
			Kind:   schema.BelongsTo,
			Target: kinds,
			Local:  []int32{4},
			Remote: []int32{0},
		},
	}

	return items, kinds, states
}
