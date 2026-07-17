//go:build integration

package compiler

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/reporting/canned"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

type leakageOrg struct {
	tenant  pagination.TenantInfo
	marker  string
	ownerID pulid.ID
	otherID pulid.ID
	ownPro  string
}

func seedLeakageOrgs(t *testing.T, ctx context.Context, db *bun.DB) (orgA, orgB *leakageOrg) {
	t.Helper()

	dataA := seedtest.SeedFullTestData(t, ctx, db)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	stateB := seedtest.NewState().WithName("Texas").WithAbbreviation("TX").Build(t, ctx, tx)
	buB := seedtest.NewBusinessUnit().WithName("Leakage B BU").WithCode("LEAKB").Build(t, ctx, tx)
	orgEntityB := seedtest.NewOrganization(buB.ID, stateB.ID).
		WithName("Leakage Org B").
		WithScacCode("LKGB").
		WithBucketName("leakage-bucket-b").
		Build(t, ctx, tx)
	userB := seedtest.NewUser(orgEntityB.ID, buB.ID).
		WithUsername("leakage_b_user").
		WithEmail("leakage_b@example.com").
		WithPassword("password123").
		Build(t, ctx, tx)
	require.NoError(t, tx.Commit())

	dataB := &seedtest.TestData{
		BusinessUnit: buB,
		Organization: orgEntityB,
		User:         userB,
		State:        stateB,
	}

	orgA = seedLeakageData(t, ctx, db, dataA, "CANARY-A")
	orgB = seedLeakageData(t, ctx, db, dataB, "CANARY-B")
	return orgA, orgB
}

// seedLeakageData plants marker-carrying rows for one org: a customer, one
// shipment owned by the org's primary user, and one shipment owned by a
// second user (for the own-scope assertion).
func seedLeakageData(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	data *seedtest.TestData,
	marker string,
) *leakageOrg {
	t.Helper()

	tenantInfo := pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}

	fixture := testutil.SeedShipmentIntegrationFixture(t, ctx, db, data, tenantInfo)

	_, err := db.NewUpdate().
		Model((*customer.Customer)(nil)).
		Set("name = ?", marker+"-CUSTOMER").
		Where("id = ?", fixture.Customer.ID).
		Where("organization_id = ?", tenantInfo.OrgID).
		Exec(ctx)
	require.NoError(t, err)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	otherUser := seedtest.NewUser(tenantInfo.OrgID, tenantInfo.BuID).
		WithUsername(strings.ToLower(marker)+"_other").
		WithEmail(strings.ToLower(marker)+"_other@example.com").
		WithPassword("password123").
		Build(t, ctx, tx)
	require.NoError(t, tx.Commit())

	ownGraph := testutil.CreateShipmentGraph(t, ctx, db, fixture, tenantInfo,
		testutil.ShipmentGraphParams{
			BOL:          marker + "-BOL-OWN",
			ProNumber:    marker + "-PRO-OWN",
			ShipmentID:   pulid.MustNew("shp_"),
			MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusNew},
		})
	otherGraph := testutil.CreateShipmentGraph(t, ctx, db, fixture, tenantInfo,
		testutil.ShipmentGraphParams{
			BOL:          marker + "-BOL-OTHER",
			ProNumber:    marker + "-PRO-OTHER",
			ShipmentID:   pulid.MustNew("shp_"),
			MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusNew},
		})

	setShipmentOwnership(t, ctx, db, ownGraph.Shipment.ID, data.User.ID, "150.0000")
	setShipmentOwnership(t, ctx, db, otherGraph.Shipment.ID, otherUser.ID, "75.0000")

	return &leakageOrg{
		tenant:  tenantInfo,
		marker:  marker,
		ownerID: data.User.ID,
		otherID: otherUser.ID,
		ownPro:  marker + "-PRO-OWN",
	}
}

func setShipmentOwnership(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	shipmentID, ownerID pulid.ID,
	total string,
) {
	t.Helper()
	_, err := db.NewUpdate().
		Model((*shipment.Shipment)(nil)).
		Set("owner_id = ?", ownerID).
		Set("total_charge_amount = ?", total).
		Where("id = ?", shipmentID).
		Exec(ctx)
	require.NoError(t, err)
}

func leakageRequest(
	def *report.Definition,
	tenantInfo pagination.TenantInfo,
	params map[string]any,
) *services.ReportCompileRequest {
	return &services.ReportCompileRequest{
		Definition:  def,
		Tenant:      tenantInfo,
		Params:      params,
		OrgTimezone: "America/Chicago",
		NowUnix:     timeutils.NowUnix(),
	}
}

// leakageCorpus is every definition executed during the leakage sweep: the
// unit-test tripwire corpus, the integration corpus, flat marker-bearing
// selections, and every canned report shipped in the registry.
func leakageCorpus() []struct {
	name   string
	def    *report.Definition
	params map[string]any
} {
	type entry = struct {
		name   string
		def    *report.Definition
		params map[string]any
	}

	var corpus []entry
	for i, def := range integrationCorpus() {
		corpus = append(corpus, entry{name: fmt.Sprintf("integration-%d", i), def: def})
	}

	corpus = append(corpus,
		entry{
			name: "shipment-flat-markers",
			def: &report.Definition{
				IRVersion: report.CurrentIRVersion,
				Entity:    "shipment",
				Columns: []report.ColumnSpec{
					dim("c_pro", "proNumber"),
					dim("c_bol", "bol"),
					dim("c_customer", "name", "customer"),
				},
			},
		},
		entry{
			name: "customer-flat-markers",
			def: &report.Definition{
				IRVersion: report.CurrentIRVersion,
				Entity:    "customer",
				Columns: []report.ColumnSpec{
					dim("c_name", "name"),
					dim("c_code", "code"),
				},
			},
		},
		entry{
			name: "shipment-revenue-by-customer",
			def: &report.Definition{
				IRVersion: report.CurrentIRVersion,
				Entity:    "shipment",
				Columns: []report.ColumnSpec{
					dim("c_customer", "name", "customer"),
					measure("c_total", reportcatalog.AggSum, "totalChargeAmount"),
					measure("c_count", reportcatalog.AggCount, "id"),
				},
			},
		},
	)

	for _, cannedEntry := range canned.Default().All() {
		params := make(map[string]any, len(cannedEntry.Definition.Parameters))
		for _, param := range cannedEntry.Definition.Parameters {
			params[param.Name] = param.Default
		}
		corpus = append(corpus, entry{
			name:   "canned-" + cannedEntry.Key,
			def:    cannedEntry.Definition,
			params: params,
		})
	}

	return corpus
}

func collectResultCells(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	compiled *services.CompiledReportQuery,
) []string {
	t.Helper()

	rows, err := db.QueryContext(ctx, compiled.SQL, compiled.Args...)
	require.NoError(t, err, "query failed:\n%s", compiled.SQL)
	defer rows.Close()

	cols, err := rows.Columns()
	require.NoError(t, err)

	var cells []string
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		require.NoError(t, rows.Scan(ptrs...))
		for _, value := range values {
			switch v := value.(type) {
			case nil:
			case []byte:
				cells = append(cells, string(v))
			default:
				cells = append(cells, fmt.Sprint(v))
			}
		}
	}
	require.NoError(t, rows.Err())
	return cells
}

// TestNoCrossTenantLeakage seeds two organizations with marker-carrying data
// and executes the full corpus (tripwire + integration + flat + canned) as
// each org, asserting the other org's markers never appear in any result cell.
// Marker reachability is asserted so the sweep can never pass vacuously.
func TestNoCrossTenantLeakage(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	orgA, orgB := seedLeakageOrgs(t, ctx, db)
	c := newTestCompiler(allowAllEngine())

	directions := []struct {
		runner    *leakageOrg
		forbidden string
	}{
		{runner: orgA, forbidden: orgB.marker},
		{runner: orgB, forbidden: orgA.marker},
	}

	for _, direction := range directions {
		foundOwnMarker := false

		for _, item := range leakageCorpus() {
			compiled, err := c.Compile(
				context.Background(),
				leakageRequest(item.def, direction.runner.tenant, item.params),
			)
			require.NoError(t, err, "%s failed to compile", item.name)

			for _, cell := range collectResultCells(t, ctx, db, compiled) {
				require.NotContains(t, cell, direction.forbidden,
					"%s leaked cross-tenant data as %s:\n%s",
					item.name, direction.runner.marker, compiled.SQL)
				if strings.Contains(cell, direction.runner.marker) {
					foundOwnMarker = true
				}
			}
		}

		require.True(t, foundOwnMarker,
			"corpus never surfaced %s's own markers — leakage assertions are vacuous",
			direction.runner.marker)
	}
}

// TestOwnScopeRowFiltering asserts DataScopeOwn compiles to a live ownership
// predicate: the runner sees only shipments they own, and rows owned by other
// users in the SAME org never appear.
func TestOwnScopeRowFiltering(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	orgA, _ := seedLeakageOrgs(t, ctx, db)

	engine := allowAllEngine()
	engine.overrides["shipment"] = resourceGrant{
		operations:     []permission.Operation{permission.OpRead, permission.OpExport},
		dataScope:      permission.DataScopeOwn,
		maxSensitivity: permission.SensitivityConfidential,
	}
	c := newTestCompiler(engine)

	def := &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns:   []report.ColumnSpec{dim("c_pro", "proNumber")},
	}

	compiled, err := c.Compile(context.Background(), leakageRequest(def, orgA.tenant, nil))
	require.NoError(t, err)

	cells := collectResultCells(t, ctx, db, compiled)
	require.Contains(t, cells, orgA.ownPro, "own-scope runner must see their own shipment")
	for _, cell := range cells {
		require.NotContains(t, cell, "PRO-OTHER",
			"own-scope runner saw a shipment owned by another user")
	}
}
