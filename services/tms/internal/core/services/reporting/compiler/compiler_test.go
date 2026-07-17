package compiler

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	testOrgID  = pulid.ID("org_testorg")
	testBuID   = pulid.ID("bu_testbu")
	testUserID = pulid.ID("usr_testuser")

	// 2026-07-15T00:00:00-05:00 in America/Chicago is 2026-07-15T05:00:00Z.
	testNowUnix = int64(1784131200) // 2026-07-15T12:00:00Z
)

type resourceGrant struct {
	operations       []permission.Operation
	dataScope        permission.DataScope
	maxSensitivity   permission.FieldSensitivity
	accessibleFields []string
}

type fakePermissionEngine struct {
	defaultGrant resourceGrant
	overrides    map[string]resourceGrant
}

func allowAllEngine() *fakePermissionEngine {
	return &fakePermissionEngine{
		defaultGrant: resourceGrant{
			operations:     []permission.Operation{permission.OpRead, permission.OpExport},
			dataScope:      permission.DataScopeOrganization,
			maxSensitivity: permission.SensitivityConfidential,
		},
		overrides: make(map[string]resourceGrant),
	}
}

func (f *fakePermissionEngine) GetResourcePermissions(
	_ context.Context,
	_, _ pulid.ID,
	resource string,
) (*services.ResourcePermissionDetail, error) {
	grant := f.defaultGrant
	if override, ok := f.overrides[resource]; ok {
		grant = override
	}
	return &services.ResourcePermissionDetail{
		Resource:         resource,
		Operations:       grant.operations,
		DataScope:        grant.dataScope,
		MaxSensitivity:   grant.maxSensitivity,
		AccessibleFields: grant.accessibleFields,
	}, nil
}

func (f *fakePermissionEngine) Check(
	context.Context, *services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	panic("not used by compiler")
}

func (f *fakePermissionEngine) CheckBatch(
	context.Context, *services.BatchPermissionCheckRequest,
) (*services.BatchPermissionCheckResult, error) {
	panic("not used by compiler")
}

func (f *fakePermissionEngine) GetLightManifest(
	context.Context, pulid.ID, pulid.ID,
) (*services.LightPermissionManifest, error) {
	panic("not used by compiler")
}

func (f *fakePermissionEngine) InvalidateUser(context.Context, pulid.ID, pulid.ID) error {
	panic("not used by compiler")
}

func (f *fakePermissionEngine) GetEffectivePermissions(
	context.Context, pulid.ID, pulid.ID,
) (*services.EffectivePermissions, error) {
	panic("not used by compiler")
}

func (f *fakePermissionEngine) SimulatePermissions(
	context.Context, *services.SimulatePermissionsRequest,
) (*services.EffectivePermissions, error) {
	panic("not used by compiler")
}

func newTestCompiler(engine services.PermissionEngine) *Compiler {
	return NewWithCatalog(
		&reportcatalog.Default,
		engine,
		permission.NewRegistry(),
		&config.ReportingConfig{},
		zap.NewNop(),
	)
}

func newRequest(def *report.Definition) *services.ReportCompileRequest {
	return &services.ReportCompileRequest{
		Definition: def,
		Tenant: pagination.TenantInfo{
			OrgID:  testOrgID,
			BuID:   testBuID,
			UserID: testUserID,
		},
		OrgTimezone: "America/Chicago",
		NowUnix:     testNowUnix,
	}
}

func mustCompile(t *testing.T, c *Compiler, def *report.Definition) *services.CompiledReportQuery {
	t.Helper()
	compiled, err := c.Compile(context.Background(), newRequest(def))
	require.NoError(t, err)
	return compiled
}

func dim(id, field string, path ...string) report.ColumnSpec {
	return report.ColumnSpec{
		ID:   id,
		Ref:  report.FieldRef{Path: path, Field: field},
		Kind: report.ColumnKindDimension,
	}
}

func measure(
	id string,
	agg reportcatalog.Aggregation,
	field string,
	path ...string,
) report.ColumnSpec {
	return report.ColumnSpec{
		ID:   id,
		Ref:  report.FieldRef{Path: path, Field: field},
		Kind: report.ColumnKindMeasure,
		Agg:  agg,
	}
}

func TestCompileSimpleListing(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_pro", "proNumber"),
			dim("c_status", "status"),
		},
		Filters: &report.FilterGroup{
			Op: report.BoolOpAnd,
			Filters: []report.FieldFilter{
				{
					Ref:      report.FieldRef{Field: "status"},
					Operator: dbtype.OpEqual,
					Value:    "InTransit",
				},
			},
		},
		Limit: 500,
	})

	want := "SELECT t0.pro_number AS c0, t0.status AS c1 " +
		"FROM shipments AS t0 " +
		"WHERE t0.organization_id = ? AND t0.business_unit_id = ? AND (t0.status = ?) " +
		"LIMIT 500"
	assert.Equal(t, want, compiled.SQL)
	assert.Equal(t, []any{testOrgID, testBuID, "InTransit"}, compiled.Args)
	assert.Equal(t, []string{"shipment"}, compiled.ReferencedEntities)
	require.Len(t, compiled.Columns, 2)
	assert.Equal(t, reportcatalog.FieldEnum, compiled.Columns[1].Type)
}

func TestCompileToOneJoinDimension(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_customer", "name", "customer"),
			dim("c_pro", "proNumber"),
		},
	})

	want := "SELECT t1.name AS c0, t0.pro_number AS c1 " +
		"FROM shipments AS t0 " +
		"LEFT JOIN customers AS t1 ON t1.id = t0.customer_id" +
		" AND t1.organization_id = ? AND t1.business_unit_id = ? " +
		"WHERE t0.organization_id = ? AND t0.business_unit_id = ? " +
		"LIMIT 100000"
	assert.Equal(t, want, compiled.SQL)
	assert.Equal(t, []any{testOrgID, testBuID, testOrgID, testBuID}, compiled.Args)
	assert.ElementsMatch(t, []string{"shipment", "customer"}, compiled.ReferencedEntities)
}

func TestCompileGroupedMeasure(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_customer", "name", "customer"),
			measure("c_total", reportcatalog.AggSum, "totalChargeAmount"),
			measure("c_count", reportcatalog.AggCount, "id"),
		},
		Sort: []report.SortSpec{
			{ColumnID: "c_total", Direction: dbtype.SortDirectionDesc},
		},
	})

	want := "SELECT t1.name AS c0, SUM(t0.total_charge_amount) AS c1, COUNT(t0.id) AS c2 " +
		"FROM shipments AS t0 " +
		"LEFT JOIN customers AS t1 ON t1.id = t0.customer_id" +
		" AND t1.organization_id = ? AND t1.business_unit_id = ? " +
		"WHERE t0.organization_id = ? AND t0.business_unit_id = ? " +
		"GROUP BY t1.name " +
		"ORDER BY c1 DESC " +
		"LIMIT 100000"
	assert.Equal(t, want, compiled.SQL)
	assert.Equal(t, reportcatalog.FieldDecimal, compiled.Columns[1].Type)
	assert.Equal(t, reportcatalog.FieldInt, compiled.Columns[2].Type)
}

func TestCompileLateralMeasure(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "order",
		Columns: []report.ColumnSpec{
			dim("c_customer", "name", "customer"),
			measure("c_shipment_total", reportcatalog.AggSum, "totalChargeAmount", "shipments"),
		},
	})

	want := "SELECT t1.name AS c0, SUM(l0.agg_0) AS c1 " +
		"FROM orders AS t0 " +
		"LEFT JOIN customers AS t1 ON t1.id = t0.customer_id" +
		" AND t1.organization_id = ? AND t1.business_unit_id = ? " +
		"LEFT JOIN LATERAL (SELECT SUM(w0.total_charge_amount) AS agg_0" +
		" FROM shipments AS w0" +
		" WHERE w0.order_id = t0.id" +
		" AND w0.organization_id = ? AND w0.business_unit_id = ?) AS l0 ON TRUE " +
		"WHERE t0.organization_id = ? AND t0.business_unit_id = ? " +
		"GROUP BY t1.name " +
		"LIMIT 100000"
	assert.Equal(t, want, compiled.SQL)
	assert.Equal(t,
		[]any{testOrgID, testBuID, testOrgID, testBuID, testOrgID, testBuID},
		compiled.Args)
}

func TestCompileExistsFilter(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns:   []report.ColumnSpec{dim("c_pro", "proNumber")},
		Filters: &report.FilterGroup{
			Op: report.BoolOpAnd,
			Filters: []report.FieldFilter{
				{
					Ref:      report.FieldRef{Path: []string{"moves"}, Field: "status"},
					Operator: dbtype.OpEqual,
					Value:    "InTransit",
				},
			},
		},
	})

	want := "SELECT t0.pro_number AS c0 " +
		"FROM shipments AS t0 " +
		"WHERE t0.organization_id = ? AND t0.business_unit_id = ? AND " +
		"(EXISTS (SELECT 1 FROM shipment_moves AS e0" +
		" WHERE e0.shipment_id = t0.id" +
		" AND e0.organization_id = ? AND e0.business_unit_id = ?" +
		" AND e0.status = ?)) " +
		"LIMIT 100000"
	assert.Equal(t, want, compiled.SQL)
	assert.Equal(t, []any{testOrgID, testBuID, testOrgID, testBuID, "InTransit"}, compiled.Args)
}

func TestCompileDateBucketAndRelativeFilter(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			{
				ID:     "c_month",
				Ref:    report.FieldRef{Field: "createdAt"},
				Kind:   report.ColumnKindDimension,
				Bucket: report.DateBucketMonth,
			},
			measure("c_count", reportcatalog.AggCount, "id"),
		},
		Filters: &report.FilterGroup{
			Op: report.BoolOpAnd,
			Filters: []report.FieldFilter{
				{
					Ref:      report.FieldRef{Field: "createdAt"},
					Operator: dbtype.OpLastNDays,
					Value:    float64(30),
				},
			},
		},
	})

	bucketExpr := "EXTRACT(EPOCH FROM date_trunc(?, to_timestamp(t0.created_at) AT TIME ZONE ?))::bigint"
	want := "SELECT " + bucketExpr + " AS c0, COUNT(t0.id) AS c1 " +
		"FROM shipments AS t0 " +
		"WHERE t0.organization_id = ? AND t0.business_unit_id = ? AND " +
		"(t0.created_at >= ? AND t0.created_at < ?) " +
		"GROUP BY " + bucketExpr + " " +
		"LIMIT 100000"
	assert.Equal(t, want, compiled.SQL)

	// 2026-07-15 in America/Chicago; last 30 days = [2026-06-15T00:00:00-05:00, 2026-07-16T00:00:00-05:00).
	require.Len(t, compiled.Args, 8)
	assert.Equal(t, "month", compiled.Args[0])
	assert.Equal(t, "America/Chicago", compiled.Args[1])
	assert.Equal(t, int64(1781499600), compiled.Args[4])
	assert.Equal(t, int64(1784178000), compiled.Args[5])
	assert.Equal(t, "month", compiled.Args[6])
}

func TestCompilePivot(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_customer", "name", "customer"),
			measure("c_total", reportcatalog.AggSum, "totalChargeAmount"),
		},
		Pivot: &report.PivotSpec{
			Ref:          report.FieldRef{Field: "status"},
			Values:       []string{"New", "InTransit"},
			MeasureIDs:   []string{"c_total"},
			IncludeOther: true,
		},
	})

	want := "SELECT t1.name AS c0, " +
		"SUM(t0.total_charge_amount) FILTER (WHERE t0.status = ?) AS c1, " +
		"SUM(t0.total_charge_amount) FILTER (WHERE t0.status = ?) AS c2, " +
		"SUM(t0.total_charge_amount) FILTER (WHERE (t0.status IS NULL OR t0.status NOT IN (?))) AS c3 " +
		"FROM shipments AS t0 " +
		"LEFT JOIN customers AS t1 ON t1.id = t0.customer_id" +
		" AND t1.organization_id = ? AND t1.business_unit_id = ? " +
		"WHERE t0.organization_id = ? AND t0.business_unit_id = ? " +
		"GROUP BY t1.name " +
		"LIMIT 100000"
	assert.Equal(t, want, compiled.SQL)

	require.Len(t, compiled.Columns, 4)
	assert.Equal(t, "c_total:New", compiled.Columns[1].ID)
	assert.Equal(t, "c_total:InTransit", compiled.Columns[2].ID)
	assert.Equal(t, "c_total:__other__", compiled.Columns[3].ID)
	assert.Contains(t, compiled.Columns[2].Label, "In Transit")
}

func TestCompileHaving(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_customer", "name", "customer"),
			measure("c_total", reportcatalog.AggSum, "totalChargeAmount"),
		},
		Having: &report.FilterGroup{
			Op: report.BoolOpAnd,
			Filters: []report.FieldFilter{
				{
					Ref:      report.FieldRef{Field: "totalChargeAmount"},
					Operator: dbtype.OpGreaterThan,
					Value:    float64(10000),
					Agg:      reportcatalog.AggSum,
				},
			},
		},
	})

	assert.Contains(t, compiled.SQL, "HAVING SUM(t0.total_charge_amount) > ?")
}

func TestCompileParameterBinding(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	def := &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns:   []report.ColumnSpec{dim("c_pro", "proNumber")},
		Filters: &report.FilterGroup{
			Op: report.BoolOpAnd,
			Filters: []report.FieldFilter{
				{
					Ref:      report.FieldRef{Field: "status"},
					Operator: dbtype.OpEqual,
					Param:    "shipmentStatus",
				},
			},
		},
		Parameters: []report.ParameterDef{
			{Name: "shipmentStatus", Type: reportcatalog.FieldEnum, Required: true},
		},
	}

	req := newRequest(def)
	req.Params = map[string]any{"shipmentStatus": "Delayed"}
	compiled, err := c.Compile(context.Background(), req)
	require.NoError(t, err)
	assert.Contains(t, compiled.Args, "Delayed")

	req.Params = nil
	_, err = c.Compile(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "shipmentStatus")
}

func TestCompileOwnScopePredicate(t *testing.T) {
	engine := allowAllEngine()
	engine.overrides["shipment"] = resourceGrant{
		operations:     []permission.Operation{permission.OpRead, permission.OpExport},
		dataScope:      permission.DataScopeOwn,
		maxSensitivity: permission.SensitivityConfidential,
	}
	c := newTestCompiler(engine)

	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns:   []report.ColumnSpec{dim("c_pro", "proNumber")},
	})

	assert.Contains(t, compiled.SQL,
		"WHERE t0.organization_id = ? AND t0.business_unit_id = ? AND t0.owner_id = ?")
	assert.Contains(t, compiled.Args, testUserID)
}

func TestCompileOwnScopeUnsupportedDenies(t *testing.T) {
	engine := allowAllEngine()
	engine.overrides["customer"] = resourceGrant{
		operations:     []permission.Operation{permission.OpRead, permission.OpExport},
		dataScope:      permission.DataScopeOwn,
		maxSensitivity: permission.SensitivityConfidential,
	}
	c := newTestCompiler(engine)

	_, err := c.Compile(context.Background(), newRequest(&report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "customer",
		Columns:   []report.ColumnSpec{dim("c_name", "name")},
	}))
	require.Error(t, err)
	var authzErr *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authzErr)
	assert.Contains(t, err.Error(), "own records")
}

func TestAuthorizeDeniesMissingRead(t *testing.T) {
	engine := allowAllEngine()
	engine.overrides["customer"] = resourceGrant{
		operations:     []permission.Operation{},
		dataScope:      permission.DataScopeOrganization,
		maxSensitivity: permission.SensitivityConfidential,
	}
	c := newTestCompiler(engine)

	_, err := c.Compile(context.Background(), newRequest(&report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns:   []report.ColumnSpec{dim("c_customer", "name", "customer")},
	}))
	require.Error(t, err)
	var authzErr *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authzErr)
	assert.Contains(t, err.Error(), "Customers")
}

func TestAuthorizeDeniesMissingExportOnRunOnly(t *testing.T) {
	engine := allowAllEngine()
	engine.defaultGrant.operations = []permission.Operation{permission.OpRead}
	c := newTestCompiler(engine)

	def := &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns:   []report.ColumnSpec{dim("c_pro", "proNumber")},
	}

	_, err := c.ValidateAndAuthorize(context.Background(), newRequest(def))
	require.NoError(t, err, "save/preview requires read only")

	_, err = c.Compile(context.Background(), newRequest(def))
	require.Error(t, err, "runs require export")
	assert.Contains(t, err.Error(), "export")
}

func TestAuthorizeFailsClosedOnSensitiveField(t *testing.T) {
	engine := allowAllEngine()
	engine.overrides["worker"] = resourceGrant{
		operations:     []permission.Operation{permission.OpRead, permission.OpExport},
		dataScope:      permission.DataScopeOrganization,
		maxSensitivity: permission.SensitivityInternal,
	}
	c := newTestCompiler(engine)

	// dob is classified confidential in the permission registry.
	_, err := c.Compile(context.Background(), newRequest(&report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "worker_profile",
		Columns: []report.ColumnSpec{
			dim("c_dob", "dob"),
		},
	}))
	require.Error(t, err)
	var authzErr *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authzErr)
	assert.Contains(t, err.Error(), "Date of Birth")

	// firstName-level internal fields still compile.
	_, err = c.Compile(context.Background(), newRequest(&report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "worker",
		Columns:   []report.ColumnSpec{dim("c_first", "firstName")},
	}))
	require.NoError(t, err)
}

func TestAuthorizeAccessibleFieldsRestriction(t *testing.T) {
	engine := allowAllEngine()
	engine.overrides["shipment"] = resourceGrant{
		operations:       []permission.Operation{permission.OpRead, permission.OpExport},
		dataScope:        permission.DataScopeOrganization,
		maxSensitivity:   permission.SensitivityConfidential,
		accessibleFields: []string{"proNumber", "status"},
	}
	c := newTestCompiler(engine)

	_, err := c.Compile(context.Background(), newRequest(&report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns:   []report.ColumnSpec{dim("c_bol", "bol")},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accessible field set")
}

func TestValidationRejectsDimensionAcrossToMany(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	_, err := c.Compile(context.Background(), newRequest(&report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_move_status", "status", "moves"),
		},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "to-many")
}

func TestValidationRejectsCountDistinctAcrossToMany(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	_, err := c.Compile(context.Background(), newRequest(&report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			dim("c_pro", "proNumber"),
			measure("c_moves", reportcatalog.AggCountDistinct, "id", "moves"),
		},
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "count_distinct")
}

func TestValidationRejectsUnknownEntityAndField(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	_, err := c.Compile(context.Background(), newRequest(&report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "users; DROP TABLE users",
		Columns:   []report.ColumnSpec{dim("c", "id")},
	}))
	require.Error(t, err)

	_, err = c.Compile(context.Background(), newRequest(&report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns:   []report.ColumnSpec{dim("c", "pro_number; --")},
	}))
	require.Error(t, err)
}

func TestLimitClamping(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	def := &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns:   []report.ColumnSpec{dim("c_pro", "proNumber")},
		Limit:     10_000_000,
	}
	_, err := c.Compile(context.Background(), newRequest(def))
	require.Error(t, err, "limit above maximum is rejected at validation")

	def.Limit = 0
	compiled := mustCompile(t, c, def)
	assert.True(t, strings.HasSuffix(compiled.SQL, "LIMIT 100000"))
}

// --- Tenancy tripwire -------------------------------------------------------

var aliasPattern = regexp.MustCompile(`(?:AS|as)\s+((?:t|w|e)\d+)\b`)

// assertTenantBinds mechanically verifies that every table alias present in
// the emitted SQL carries organization_id AND business_unit_id predicates,
// unless the alias belongs to an untenanted reference entity.
func assertTenantBinds(t *testing.T, sql string) {
	t.Helper()

	aliases := map[string]bool{"t0": true}
	for _, match := range aliasPattern.FindAllStringSubmatch(sql, -1) {
		aliases[match[1]] = true
	}

	for alias := range aliases {
		orgPredicate := alias + ".organization_id = ?"
		buPredicate := alias + ".business_unit_id = ?"
		assert.Contains(t, sql, orgPredicate,
			"alias %s is missing its organization predicate:\n%s", alias, sql)
		assert.Contains(t, sql, buPredicate,
			"alias %s is missing its business unit predicate:\n%s", alias, sql)
	}
}

func tripwireCorpus() []*report.Definition {
	return []*report.Definition{
		{
			IRVersion: report.CurrentIRVersion,
			Entity:    "shipment",
			Columns: []report.ColumnSpec{
				dim("c1", "proNumber"),
				dim("c2", "name", "customer"),
				dim("c3", "code", "serviceType"),
			},
		},
		{
			IRVersion: report.CurrentIRVersion,
			Entity:    "order",
			Columns: []report.ColumnSpec{
				dim("c1", "name", "customer"),
				measure("c2", reportcatalog.AggSum, "totalChargeAmount", "shipments"),
				measure("c3", reportcatalog.AggAvg, "totalChargeAmount", "shipments"),
			},
		},
		{
			IRVersion: report.CurrentIRVersion,
			Entity:    "shipment",
			Columns:   []report.ColumnSpec{dim("c1", "proNumber")},
			Filters: &report.FilterGroup{
				Op: report.BoolOpAnd,
				Filters: []report.FieldFilter{
					{
						Ref:      report.FieldRef{Path: []string{"moves", "stops"}, Field: "type"},
						Operator: dbtype.OpIsNotNull,
					},
				},
			},
		},
		{
			IRVersion: report.CurrentIRVersion,
			Entity:    "worker",
			Columns: []report.ColumnSpec{
				dim("c1", "firstName"),
				dim("c2", "code", "fleetCode"),
				dim("c3", "licenseNumber", "profile"),
			},
		},
		{
			IRVersion: report.CurrentIRVersion,
			Entity:    "invoice",
			Columns: []report.ColumnSpec{
				dim("c1", "number"),
				dim("c2", "name", "customer"),
				measure("c3", reportcatalog.AggSum, "amount", "lines"),
			},
		},
	}
}

func TestTenancyTripwire(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	for i, def := range tripwireCorpus() {
		t.Run(fmt.Sprintf("corpus_%d", i), func(t *testing.T) {
			compiled := mustCompile(t, c, def)
			assertTenantBinds(t, compiled.SQL)
		})
	}
}

// --- Injection properties ---------------------------------------------------

var hostileStrings = []string{
	`'; DROP TABLE shipments; --`,
	`" OR 1=1 --`,
	`\'); DELETE FROM users; --`,
	`Robert'); DROP TABLE students;--`,
	`%' OR '1'='1`,
	"`; TRUNCATE audit_entries; `",
}

func TestHostileValuesNeverReachSQLText(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	for _, hostile := range hostileStrings {
		def := &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    "shipment",
			Columns:   []report.ColumnSpec{dim("c_pro", "proNumber")},
			Filters: &report.FilterGroup{
				Op: report.BoolOpAnd,
				Filters: []report.FieldFilter{
					{
						Ref:      report.FieldRef{Field: "proNumber"},
						Operator: dbtype.OpContains,
						Value:    hostile,
					},
					{
						Ref:      report.FieldRef{Field: "bol"},
						Operator: dbtype.OpIn,
						Value:    []any{hostile, "safe"},
					},
				},
			},
		}

		compiled, err := c.Compile(context.Background(), newRequest(def))
		require.NoError(t, err)
		assert.NotContains(t, compiled.SQL, "DROP")
		assert.NotContains(t, compiled.SQL, "DELETE")
		assert.NotContains(t, compiled.SQL, "TRUNCATE")
		assert.NotContains(t, compiled.SQL, hostile)
	}
}

func TestHostileIdentifiersRejected(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	hostileRefs := []report.FieldRef{
		{Field: "proNumber; DROP TABLE x"},
		{Path: []string{"customer; --"}, Field: "name"},
		{Path: []string{"customer"}, Field: `name" FROM users --`},
	}

	for _, ref := range hostileRefs {
		_, err := c.Compile(context.Background(), newRequest(&report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    "shipment",
			Columns: []report.ColumnSpec{
				{ID: "c1", Ref: ref, Kind: report.ColumnKindDimension},
			},
		}))
		require.Error(t, err, "hostile ref %v must be rejected", ref)
	}

	hostileSort := &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns:   []report.ColumnSpec{dim("c1", "proNumber")},
		Sort: []report.SortSpec{
			{ColumnID: "c1; DROP TABLE x", Direction: dbtype.SortDirectionAsc},
		},
	}
	_, err := c.Compile(context.Background(), newRequest(hostileSort))
	require.Error(t, err, "sort referencing unknown column ID must be rejected")
}

func TestUserLabelsNeverReachSQLText(t *testing.T) {
	c := newTestCompiler(allowAllEngine())

	hostileLabel := `"; DROP TABLE shipments; --`
	compiled := mustCompile(t, c, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			{
				ID:    "c1",
				Ref:   report.FieldRef{Field: "proNumber"},
				Kind:  report.ColumnKindDimension,
				Label: hostileLabel,
			},
		},
	})

	assert.NotContains(t, compiled.SQL, "DROP")
	assert.Equal(t, hostileLabel, compiled.Columns[0].Label)
}
