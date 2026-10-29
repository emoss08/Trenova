package keelcheck

import (
	"testing"

	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func checkTable(name, alias string, extra ...schema.Column) *schema.Table {
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

func checkTables() (*schema.Table, *schema.Table, *schema.Table) {
	kinds := checkTable("check_kinds", "ck")
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

	items := checkTable("check_items", "ci",
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

func TestKeelChecker_UniquenessQuery(t *testing.T) {
	t.Parallel()

	items, _, _ := checkTables()
	checker := NewChecker(nil, items)

	ctx := keeltest.TenantContext(t)
	q, err := checker.uniquenessQuery(ctx, &validationframework.UniquenessRequest{
		TableName:      "check_items",
		OrganizationID: pulid.ID("org_1"),
		BusinessUnitID: pulid.ID("bu_1"),
		ExcludeID:      pulid.ID("ci_1"),
		Fields:         []validationframework.FieldCheck{{Column: "code", Value: "ABC"}},
		ScopeFields: []validationframework.FieldCheck{
			{Column: "kind_id", Value: "ck_1", CaseSensitive: true},
		},
	})
	require.NoError(t, err)

	keeltest.SnapshotIn(t, ctx, q)
}

func TestKeelChecker_ReferenceQueryReachesRelatedTables(t *testing.T) {
	t.Parallel()

	items, _, _ := checkTables()
	checker := NewChecker(nil, items)

	ctx := keeltest.TenantContext(t)
	q, err := checker.referenceQuery(ctx, &validationframework.ReferenceRequest{
		TableName:      "check_kinds",
		OrganizationID: pulid.ID("org_1"),
		BusinessUnitID: pulid.ID("bu_1"),
		ID:             pulid.ID("ck_1"),
	})
	require.NoError(t, err)

	keeltest.SnapshotIn(t, ctx, q)
}

func TestKeelChecker_RefusesWhatItDoesNotKnow(t *testing.T) {
	t.Parallel()

	items, _, _ := checkTables()
	checker := NewChecker(nil, items)

	ctx := keeltest.TenantContext(t)
	_, err := checker.referenceQuery(
		ctx,
		&validationframework.ReferenceRequest{TableName: "elsewhere", ID: pulid.ID("x_1")},
	)
	require.Error(t, err)

	_, err = checker.referenceQuery(
		ctx,
		&validationframework.ReferenceRequest{TableName: "check_kinds"},
	)
	require.Error(t, err)

	_, err = checker.uniquenessQuery(
		ctx,
		&validationframework.UniquenessRequest{TableName: "check_items"},
	)
	require.Error(t, err)

	_, err = checker.uniquenessQuery(ctx, &validationframework.UniquenessRequest{
		TableName: "check_items",
		Fields:    []validationframework.FieldCheck{{Column: "code; DROP TABLE x", Value: "A"}},
	})
	require.Error(t, err)

	_, err = checker.uniquenessQuery(ctx, &validationframework.UniquenessRequest{
		Fields: []validationframework.FieldCheck{{Column: "code", Value: "A"}},
	})
	require.Error(t, err)
}

func TestKeelChecker_ASystemScopeReadsEveryTenant(t *testing.T) {
	t.Parallel()

	items, _, _ := checkTables()
	ctx := dbscope.WithSystem(t.Context(), "reference check test")
	q, err := NewChecker(nil, items).referenceQuery(ctx, &validationframework.ReferenceRequest{
		TableName:      "check_items",
		OrganizationID: pulid.ID("org_1"),
		BusinessUnitID: pulid.ID("bu_1"),
		ID:             pulid.ID("ci_1"),
	})
	require.NoError(t, err)

	keeltest.SnapshotIn(t, ctx, q)
}

func TestKeelReferenceCheck_AGlobalTableNeedsNoTenant(t *testing.T) {
	t.Parallel()

	_, _, states := checkTables()
	q := keelReferenceQuery(
		t.Context(),
		states,
		pulid.ID("org_1"),
		pulid.ID("bu_1"),
		pulid.ID("st_1"),
	)

	keeltest.SnapshotIn(t, t.Context(), q)

	exists, err := ReferenceCheck(nil, states)(t.Context(), pulid.Nil, pulid.Nil, pulid.Nil)
	require.NoError(t, err)
	assert.True(t, exists)
}
