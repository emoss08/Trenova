//go:build integration

package validationframework

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
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

	_, err := keel.Exec(t.Context(), conn.db,
		"INSERT INTO check_items (id, organization_id, business_unit_id, code) VALUES ($1, $2, $3, $4)",
		id.String(), tenant.orgID.String(), tenant.buID.String(), code)
	require.NoError(t, err)
}

func TestKeelChecker_CheckUniqueness_Integration(t *testing.T) {
	conn, a, b := setupCheckDB(t)
	items, _, _ := checkTables()
	checker := NewKeelChecker(conn, items)

	existing := pulid.MustNew("ci_")
	insertCheckItem(t, conn, a, existing, "CODE001")

	unique := func(tenant checkTenant, req *UniquenessRequest) bool {
		t.Helper()
		req.TableName = "check_items"
		req.OrganizationID = tenant.orgID
		req.BusinessUnitID = tenant.buID
		exists, err := checker.CheckUniqueness(tenant.ctx, req)
		require.NoError(t, err)
		return exists
	}

	assert.False(t, unique(a, &UniquenessRequest{Fields: []FieldCheck{{Column: "code", Value: "CODE002"}}}))
	assert.True(t, unique(a, &UniquenessRequest{Fields: []FieldCheck{{Column: "code", Value: "CODE001"}}}))
	assert.True(t, unique(a, &UniquenessRequest{Fields: []FieldCheck{{Column: "code", Value: "code001"}}}),
		"a case-insensitive check matches another case")
	assert.False(t, unique(a, &UniquenessRequest{
		Fields: []FieldCheck{{Column: "code", Value: "code001", CaseSensitive: true}},
	}))
	assert.True(t, unique(a, &UniquenessRequest{
		Fields: []FieldCheck{{Column: "code", Value: "CODE001", CaseSensitive: true}},
	}))
	assert.False(t, unique(a, &UniquenessRequest{
		ExcludeID: existing,
		Fields:    []FieldCheck{{Column: "code", Value: "CODE001"}},
	}), "an update excludes the row itself")
	assert.True(t, unique(a, &UniquenessRequest{
		ExcludeID: pulid.MustNew("ci_"),
		Fields:    []FieldCheck{{Column: "code", Value: "CODE001"}},
	}))
	assert.False(t, unique(b, &UniquenessRequest{Fields: []FieldCheck{{Column: "code", Value: "CODE001"}}}),
		"another tenant's row is not a conflict")
	assert.False(t, unique(a, &UniquenessRequest{
		Fields:      []FieldCheck{{Column: "code", Value: "CODE001"}},
		ScopeFields: []FieldCheck{{Column: "id", Value: "ci_other", CaseSensitive: true}},
	}), "every field must match")

	system := dbscope.WithSystem(t.Context(), "uniqueness check test")
	exists, err := checker.CheckUniqueness(system, &UniquenessRequest{
		TableName:      "check_items",
		OrganizationID: b.orgID,
		BusinessUnitID: b.buID,
		Fields:         []FieldCheck{{Column: "code", Value: "CODE001"}},
	})
	require.NoError(t, err)
	assert.False(t, exists, "a system-scope check filters the named tenant")
}

func TestKeelChecker_CheckReference_Integration(t *testing.T) {
	conn, a, b := setupCheckDB(t)
	items, _, states := checkTables()
	checker := NewKeelChecker(conn, items)

	kind := pulid.MustNew("ck_")
	_, err := keel.Exec(t.Context(), conn.db,
		"INSERT INTO check_kinds (id, organization_id, business_unit_id) VALUES ($1, $2, $3)",
		kind.String(), a.orgID.String(), a.buID.String())
	require.NoError(t, err)

	reference := func(tenant checkTenant, table string, id pulid.ID) bool {
		t.Helper()
		exists, refErr := checker.CheckReference(tenant.ctx, &ReferenceRequest{
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

	_, err = checker.CheckReference(a.ctx, &ReferenceRequest{ID: kind})
	require.Error(t, err)
	_, err = checker.CheckReference(a.ctx, &ReferenceRequest{TableName: "check_kinds"})
	require.Error(t, err)

	state := pulid.MustNew("st_")
	_, err = keel.Exec(t.Context(), conn.db, "INSERT INTO check_states (id) VALUES ($1)", state.String())
	require.NoError(t, err)

	exists, err := KeelReferenceCheck(conn, states)(b.ctx, b.orgID, b.buID, state)
	require.NoError(t, err)
	assert.True(t, exists, "a global table answers every tenant")
}
