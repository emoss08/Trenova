//go:build integration

package agentquerytoolservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/bootstrap/crossref"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/schedulingrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerdrugalcoholrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/adapters/postgres/workerrepository"
	"github.com/emoss08/trenova/internal/modules/workforce/app/schedulingservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdrugalcoholservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// programmeTenant is what one tenant holds for the scheduling and random
// testing reads: a worker on two patterns with a stated weekday and an open
// and a closed swap, an active and a retired pattern, and a round drawn from
// an active pool beside a retired one.
type programmeTenant struct {
	ctx         context.Context
	workerID    pulid.ID
	assignments []string
	preference  string
	openSwap    string
	closedSwap  string
	active      string
	retired     string
	pool        string
	retiredPool string
	draw        string
}

func writeProgramme(
	t *testing.T,
	tenant *agentreadtest.Tenant,
	workers workerports.WorkerRepository,
	schedules workerports.SchedulingRepository,
	draws workerports.WorkerDrugAlcoholRepository,
	label string,
) *programmeTenant {
	t.Helper()

	ctx := readscope.With(dbscope.WithTenant(t.Context(), tenant.Scope()), nil)
	bu, org := tenant.Actor.BusinessUnitID, tenant.Actor.OrganizationID
	now := timeutils.NowUnix()
	out := &programmeTenant{ctx: ctx}

	driver, err := workers.Create(ctx, testWorker(tenant, "S"+label, false, pulid.Nil))
	require.NoError(t, err)
	out.workerID = driver.ID
	template := func(code string, status domaintypes.Status) *worker.ShiftTemplate {
		row, templateErr := schedules.CreateTemplate(ctx, &worker.ShiftTemplate{
			BusinessUnitID: bu, OrganizationID: org, Status: status, Code: code + label,
			Name: "Pattern " + code + label, DaysOfWeek: "0111110", StartMinute: 360,
			DurationMinutes: 600, CycleWeeks: 1,
		})
		require.NoError(t, templateErr)
		return row
	}
	early, late := template("E", domaintypes.StatusActive), template("R", domaintypes.StatusInactive)
	out.active, out.retired = early.ID.String(), late.ID.String()
	ended := now - 86400*10
	for _, assignment := range []*worker.WorkerShiftAssignment{
		{ShiftTemplateID: late.ID, EffectiveFrom: now - 86400*100, EffectiveTo: &ended},
		{ShiftTemplateID: early.ID, EffectiveFrom: now - 86400*9},
	} {
		assignment.BusinessUnitID, assignment.OrganizationID = bu, org
		assignment.WorkerID = driver.ID
		row, assignErr := schedules.CreateAssignment(ctx, assignment)
		require.NoError(t, assignErr)
		out.assignments = append(out.assignments, row.ID.String())
	}
	preference, err := schedules.UpsertPreference(ctx, &worker.WorkerAvailabilityPreference{
		BusinessUnitID: bu, OrganizationID: org, WorkerID: driver.ID, DayOfWeek: 6,
		Preference: worker.AvailabilityUnavailable, Note: "Family day " + label,
	})
	require.NoError(t, err)
	preference.Note = "Family day, every week " + label
	_, err = schedules.UpsertPreference(ctx, preference)
	require.NoError(t, err, "a second statement for the weekday replaces the first")
	out.preference = preference.ID.String()
	for _, status := range []worker.ShiftSwapStatus{worker.SwapProposed, worker.SwapDeclined} {
		swap, swapErr := schedules.CreateSwap(ctx, &worker.ShiftSwapRequest{
			BusinessUnitID: bu, OrganizationID: org, RequestingWorkerID: driver.ID,
			Status: status, ShiftDate: now + 86400*7, Reason: "Appointment " + label,
		})
		require.NoError(t, swapErr)
		if status == worker.SwapProposed {
			out.openSwap = swap.ID.String()
		} else {
			out.closedSwap = swap.ID.String()
		}
	}

	pool := func(code string, status domaintypes.Status) *worker.DOTRandomPool {
		row, poolErr := draws.CreatePool(ctx, &worker.DOTRandomPool{
			BusinessUnitID: bu, OrganizationID: org, Code: code + label, Name: "Pool " + code,
			Status: status, Period: worker.RandomPeriodQuarterly, DrugRatePercent: 50,
			AlcoholRatePercent: 10, IncludedDriverTypes: []string{},
		})
		require.NoError(t, poolErr)
		return row
	}
	active, retired := pool("A", domaintypes.StatusActive), pool("I", domaintypes.StatusInactive)
	out.pool, out.retiredPool = active.ID.String(), retired.ID.String()
	draw, err := draws.CreateDrawWithEntries(ctx, &worker.DOTRandomDraw{
		ID: pulid.MustNew("drdraw_"), BusinessUnitID: bu, OrganizationID: org,
		PoolID: active.ID, PeriodKey: "2026-Q4", PeriodStart: now - 86400*30,
		PeriodEnd: now + 86400*60, Status: worker.RandomDrawStatusFinal, PoolSize: 1,
		DrugTarget: 1, DrugSelected: 1, Seed: "seed" + label, Method: "sha256-rank",
		DrawnAt: now, FinalizedAt: &now,
	}, []*worker.DOTRandomDrawEntry{{
		ID: pulid.MustNew("drde_"), BusinessUnitID: bu, OrganizationID: org,
		WorkerID: driver.ID, Substance: worker.DOTSubstanceDrug, Rank: 1,
		Status: worker.RandomEntrySelected,
	}})
	require.NoError(t, err)
	out.draw = draw.ID.String()

	return out
}

func rowIDs(t *testing.T, raw any) []string {
	t.Helper()

	rows, _ := raw.([]any)
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		object, ok := row.(map[string]any)
		require.True(t, ok)
		id, _ := object["id"].(string)
		out = append(out, id)
	}

	return out
}

func TestConvertedLists_SchedulingAndRandomTestingReadTheTenantsOwnUnderRowLevelSecurity(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	logger := zap.NewNop()
	_, resolvers := convertedListCases(conn, admin)
	workers := workerrepository.New(workerrepository.Params{DB: conn, Logger: logger})
	schedules := schedulingrepository.New(schedulingrepository.Params{
		DB: conn, Logger: logger, Assignments: crossref.RotaAssignments(),
	})
	draws := workerdrugalcoholrepository.New(
		workerdrugalcoholrepository.Params{DB: conn, Logger: logger})
	resolvers.SchedulingService = schedulingservice.New(schedulingservice.Params{
		Logger: logger, Repo: schedules,
	})
	resolvers.WorkerDrugAlcoholService = workerdrugalcoholservice.New(
		workerdrugalcoholservice.Params{Logger: logger, Repo: draws, WorkerRepo: workers})

	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine, Resolvers: resolvers})
	public := agentreadtest.NewReader(t, agentreadtest.Options{Engine: &publicRole{}, Resolvers: resolvers})
	mine := agentreadtest.SeedTenant(t, admin, "A")
	theirs := agentreadtest.SeedTenant(t, admin, "B")
	ours := writeProgramme(t, mine, workers, schedules, draws, "A")
	other := writeProgramme(t, theirs, workers, schedules, draws, "B")
	actor := mine.Actor
	tool := func(name string) agentports.AgentQueryTool {
		converted, err := agentquerytoolservice.ConvertedListTool(name, reader, engine)
		require.NoError(t, err)
		return converted
	}

	t.Run("a worker's schedule, their stated weekday and their open swaps", func(t *testing.T) {
		page := callTool(t, ours.ctx, tool("worker_get_schedule"), actor,
			map[string]any{"workerId": ours.workerID.String()})
		assert.ElementsMatch(t, ours.assignments, itemIDs(page))
		assert.Equal(t, []string{ours.preference}, rowIDs(t, page["preferences"]))
		assert.Equal(t, []string{ours.openSwap}, rowIDs(t, page["openSwaps"]),
			"only the open swaps, as the repository tool read them")
	})

	t.Run("another tenant's worker has no schedule here", func(t *testing.T) {
		page := callTool(t, ours.ctx, tool("worker_get_schedule"), actor,
			map[string]any{"workerId": other.workerID.String()})
		assert.Empty(t, itemIDs(page))
		assert.Empty(t, rowIDs(t, page["preferences"]))
		assert.Empty(t, rowIDs(t, page["openSwaps"]))
	})

	t.Run("the active patterns by default, every pattern on asking", func(t *testing.T) {
		templates := tool("worker_list_shift_templates")
		active := callTool(t, ours.ctx, templates, actor, map[string]any{})
		assert.Contains(t, itemIDs(active), ours.active)
		assert.NotContains(t, itemIDs(active), ours.retired)
		assert.NotContains(t, itemIDs(active), other.active)
		every := callTool(t, ours.ctx, templates, actor, map[string]any{"activeOnly": false})
		assert.Subset(t, itemIDs(every), []string{ours.active, ours.retired})
		assert.NotContains(t, itemIDs(every), other.retired)
	})

	t.Run("the rounds, with the active pools beside them", func(t *testing.T) {
		rounds := tool("worker_list_dot_random_draws")
		page := callTool(t, ours.ctx, rounds, actor, map[string]any{})
		assert.Equal(t, []string{ours.draw}, itemIDs(page))
		pools := rowIDs(t, page["pools"])
		assert.Contains(t, pools, ours.pool)
		assert.NotContains(t, pools, ours.retiredPool)
		assert.NotContains(t, pools, other.pool)

		narrowed := callTool(t, ours.ctx, rounds, actor, map[string]any{"poolId": ours.retiredPool})
		assert.Empty(t, itemIDs(narrowed))
	})

	t.Run("a field the role cannot see stays withheld", func(t *testing.T) {
		for name, args := range map[string]map[string]any{
			"worker_get_schedule":          {"workerId": ours.workerID.String()},
			"worker_list_shift_templates":  {},
			"worker_list_dot_random_draws": {},
		} {
			hidden, err := agentquerytoolservice.ConvertedListTool(name, public, &publicRole{})
			require.NoError(t, err)
			page := callTool(t, ours.ctx, hidden, actor, args)
			withheld, _ := page["withheldByAccess"].([]any)
			require.NotEmpty(t, withheld, name)
			items, _ := page["items"].([]any)
			require.NotEmpty(t, items, name)
			for _, item := range items {
				row, _ := item.(map[string]any)
				assert.Contains(t, row, "id", name)
				for _, field := range withheld {
					assert.NotContains(t, row, field, name)
				}
			}
		}
	})
}
