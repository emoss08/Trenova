//go:build integration

package agentquerytoolservice_test

import (
	"context"
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report/reportdb"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/agentread"
	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/reportengine"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/reportrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

// reportTenant is one tenant's report rows: a saved report, the runs the
// reader and a colleague made, and the schedules each set up.
type reportTenant struct {
	ctx        context.Context
	definition string
	// runs are the reader's own, newest first: a stored success, a success
	// from before rows were stored, a failure and a built-in report's run.
	runs           []string
	colleaguesRun  string
	schedule       string
	colleaguesPlan string
}

func writeReportRows(
	t *testing.T,
	admin *keel.DB,
	tenant *agentreadtest.Tenant,
	label string,
) *reportTenant {
	t.Helper()

	ctx := t.Context()
	org, bu := tenant.Actor.OrganizationID, tenant.Actor.BusinessUnitID
	reader := tenant.Actor.UserID

	colleague := seedtest.NewUser(org, bu).
		WithUsername("colleague_"+strings.ToLower(label)+org.String()[4:12]).
		WithEmail("colleague_"+strings.ToLower(label)+org.String()[4:12]+"@example.com").
		Build(t, ctx, admin).ID

	definition := pulid.MustNew("rdef_")
	_, err := keel.Exec(ctx, admin, `INSERT INTO report_definitions
		(id, business_unit_id, organization_id, name, owner_id, catalog_version, definition)
		VALUES ($1, $2, $3, $4, $5, 'test', '{}'::jsonb)`,
		definition.String(), bu.String(), org.String(), "Volume "+label, reader.String())
	require.NoError(t, err)
	writeCtx := tenant.Ctx(ctx)

	now := timeutils.NowUnix()
	run := func(by pulid.ID, status report.RunStatus, rowsKey string, defined bool, age int64) string {
		entity := &report.ReportRun{
			ID:             pulid.MustNew("rrun_"),
			BusinessUnitID: bu,
			OrganizationID: org,
			RequestedByID:  by,
			Trigger:        report.RunTriggerManual,
			Format:         report.FormatCSV,
			Status:         status,
			RowsKey:        rowsKey,
		}
		if defined {
			entity.DefinitionID = definition
		} else {
			entity.CannedKey = "shipment-volume-by-status"
			entity.CannedVersion = "1.0.0"
		}
		require.NoError(t, reportdb.ReportRuns.Create(writeCtx, admin, entity))
		_, insertErr := keel.Exec(ctx, admin,
			"UPDATE report_runs SET created_at = $1 WHERE id = $2", now-age, entity.ID.String())
		require.NoError(t, insertErr)

		return entity.ID.String()
	}
	rows := &reportTenant{
		ctx: readscope.With(dbscope.WithTenant(ctx, tenant.Scope()), nil),
		runs: []string{
			run(reader, report.RunStatusSucceeded, "reports/"+label+"/rows.json", true, 10),
			run(reader, report.RunStatusSucceeded, "", true, 20),
			run(reader, report.RunStatusFailed, "", true, 30),
			run(reader, report.RunStatusSucceeded, "reports/"+label+"/canned.json", false, 40),
		},
		colleaguesRun: run(colleague, report.RunStatusSucceeded, "reports/x.json", true, 5),
		definition:    definition.String(),
	}

	schedule := func(runAs pulid.ID, enabled bool) string {
		entity := &report.ReportSchedule{
			ID:             pulid.MustNew("rsch_"),
			BusinessUnitID: bu,
			OrganizationID: org,
			DefinitionID:   definition,
			CronExpression: "0 7 * * 1",
			Timezone:       "America/Chicago",
			Formats:        []string{string(report.FormatCSV)},
			Delivery:       &report.ScheduleDelivery{EmailRecipients: []string{"ops@" + label + ".test"}},
			Enabled:        enabled,
			RunAsID:        runAs,
			NextRunAt:      now + 3600,
		}
		require.NoError(t, reportdb.ReportSchedules.Create(writeCtx, admin, entity))

		return entity.ID.String()
	}
	rows.schedule = schedule(reader, true)
	rows.colleaguesPlan = schedule(colleague, false)

	return rows
}

func readAt(
	t *testing.T,
	ctx context.Context,
	tool agentports.AgentQueryTool,
	actor *permissionports.RequestActor,
	args map[string]any,
	ceiling permission.FieldSensitivity,
) map[string]any {
	t.Helper()

	data, err := tool.Query(ctx, &agentports.QueryToolParams{
		OrganizationID: actor.OrganizationID, BusinessUnitID: actor.BusinessUnitID,
		Actor: actor, Timezone: "America/Chicago", Params: args,
		DataAccessCeiling: ceiling,
		RunID:             pulid.MustNew("ar_"),
	})
	require.NoError(t, err, tool.Name())

	return asMap(t, data)
}

func rowsByID(page map[string]any) map[string]map[string]any {
	items, _ := page["items"].([]any)
	rows := make(map[string]map[string]any, len(items))
	for _, item := range items {
		row := item.(map[string]any)
		rows[row["id"].(string)] = row
	}

	return rows
}

// The report lists read, through the screen's resolvers under row-level
// security, only the tenant's rows. A person's runs are their own, never a
// colleague's, newest first, and a stored success is the run they can
// compare; schedules are the organization's, as the screen and the
// repository tool read them, and mine names the reader's own. Nothing is
// withheld at Internal, and everything but the id at Public.
func TestConvertedLists_ReportReadsUnderRowLevelSecurity(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	logger := zap.NewNop()
	engine := &mocks.DenyResourcesPermissionEngine{}
	catalog, err := reportengine.NewCatalog()
	require.NoError(t, err)
	reportingService, err := reporting.New(reporting.Params{
		Logger: logger,
		RunRepo: reportrepository.NewRunRepository(
			reportrepository.RunParams{DB: conn, Logger: logger}),
		ScheduleRepo: reportrepository.NewScheduleRepository(
			reportrepository.ScheduleParams{DB: conn, Logger: logger}),
		Engine: reportengine.New(reportengine.Params{
			LC:      fxtest.NewLifecycle(t),
			Catalog: catalog,
			Config:  &config.Config{},
			Logger:  logger,
		}),
		PermissionEngine: engine,
		Config:           &config.Config{},
	})
	require.NoError(t, err)
	resolvers := agentread.ResolverParams{ReportingService: reportingService}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine, Resolvers: resolvers})
	public := agentreadtest.NewReader(t, agentreadtest.Options{
		Engine: &publicRole{}, Resolvers: resolvers,
	})

	mine := agentreadtest.SeedTenant(t, admin, "A")
	theirs := agentreadtest.SeedTenant(t, admin, "B")
	ours := writeReportRows(t, admin, mine, "A")
	other := writeReportRows(t, admin, theirs, "B")
	actor := mine.Actor

	t.Run("report_list_runs", func(t *testing.T) {
		runs, err := agentquerytoolservice.ConvertedListTool("report_list_runs", reader, engine)
		require.NoError(t, err)

		ids, pages := walkByKey(t, ours.ctx, runs, actor, nil, "id")
		assert.Greater(t, pages, 1, "the walk must cross a page")
		assert.Equal(t, ours.runs, ids, "the reader's own runs, newest first")
		assert.NotContains(t, ids, ours.colleaguesRun, "a colleague's runs are never listed")
		for _, id := range append(other.runs, other.colleaguesRun) {
			assert.NotContains(t, ids, id)
		}

		page := readAt(t, ours.ctx, runs, actor, map[string]any{}, permission.SensitivityInternal)
		assert.Empty(t, page["withheldByAccess"], "every run field is Internal")
		rows := rowsByID(page)
		require.Len(t, rows, len(ours.runs))
		stored := rows[ours.runs[0]]
		assert.Equal(t, string(report.RunStatusSucceeded), stored["status"])
		assert.Equal(t, true, stored["rowsStored"])
		assert.Equal(t, true, stored["comparable"], "a stored success can be compared")
		assert.NotContains(t, stored, "reason")
		assert.Equal(t, false, rows[ours.runs[1]]["comparable"])
		assert.Contains(t, rows[ours.runs[1]]["reason"], "before its rows were stored")
		assert.Equal(t, false, rows[ours.runs[2]]["comparable"])
		assert.Contains(t, rows[ours.runs[2]]["reason"], "ended as failed")
		assert.Equal(t, true, rows[ours.runs[3]]["comparable"])

		narrowed, _ := walkByKey(t, ours.ctx, runs, actor,
			map[string]any{"definitionId": ours.definition}, "id")
		assert.Equal(t, ours.runs[:3], narrowed)

		hidden, err := agentquerytoolservice.ConvertedListTool("report_list_runs", public, engine)
		require.NoError(t, err)
		withheld := callTool(t, ours.ctx, hidden, actor, map[string]any{})
		assert.NotEmpty(t, withheld["withheldByAccess"])
		for _, item := range withheld["items"].([]any) {
			row := item.(map[string]any)
			assert.NotContains(t, row, "status")
			assert.NotContains(t, row, "comparable", "a withheld run is not judged")
		}
	})

	t.Run("report_list_schedules", func(t *testing.T) {
		schedules, err := agentquerytoolservice.ConvertedListTool(
			"report_list_schedules", reader, engine)
		require.NoError(t, err)

		page := readAt(t, ours.ctx, schedules, actor, map[string]any{},
			permission.SensitivityInternal)
		assert.Empty(t, page["withheldByAccess"], "every schedule field is Internal")
		rows := rowsByID(page)
		require.Len(t, rows, 2)
		assert.Equal(t, true, rows[ours.schedule]["mine"])
		assert.Equal(t, []any{"ops@A.test"}, rows[ours.schedule]["emailRecipients"])
		assert.Equal(t, false, rows[ours.colleaguesPlan]["mine"],
			"a colleague's schedule is listed, as before, and is not the reader's")
		for _, id := range []string{other.schedule, other.colleaguesPlan} {
			assert.NotContains(t, rows, id)
		}

		enabled := rowsByID(callTool(t, ours.ctx, schedules, actor,
			map[string]any{"enabledOnly": true}))
		assert.Len(t, enabled, 1)
		assert.Contains(t, enabled, ours.schedule)

		theirReport := callTool(t, ours.ctx, schedules, actor,
			map[string]any{"definitionId": other.definition})
		assert.Empty(t, theirReport["items"])

		hidden, err := agentquerytoolservice.ConvertedListTool(
			"report_list_schedules", public, engine)
		require.NoError(t, err)
		withheld := callTool(t, ours.ctx, hidden, actor, map[string]any{})
		assert.NotEmpty(t, withheld["withheldByAccess"])
		for _, item := range withheld["items"].([]any) {
			row := item.(map[string]any)
			assert.NotContains(t, row, "emailRecipients")
			assert.NotContains(t, row, "cronExpression")
		}
	})
}
