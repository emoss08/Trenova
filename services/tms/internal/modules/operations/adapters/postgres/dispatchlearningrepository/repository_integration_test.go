//go:build integration

package dispatchlearningrepository_test

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/dispatchlearningrepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol/dispatchcontroldb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchlearning"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchplanprice/dispatchplanpricedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent/shipmenteventdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchlearningports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const decidedAt = int64(1_793_577_600)

type tenantScope struct {
	ctx     context.Context
	tenant  pagination.TenantInfo
	graph   *testutil.ShipmentGraph
	workers []*worker.Worker
}

type harness struct {
	db    *keel.DB
	repo  dispatchlearningports.Repository
	own   tenantScope
	other tenantScope
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	seed := func(data *seedtest.TestData, code string) tenantScope {
		tenant := data.Info()
		scoped := data.Ctx(ctx)
		fixture := testutil.SeedShipmentIntegrationFixture(t, ctx, db, data, tenant)
		graph := testutil.CreateShipmentGraph(
			t,
			ctx,
			db,
			fixture,
			tenant,
			testutil.ShipmentGraphParams{
				BOL:        "BOL-" + code,
				ProNumber:  "PRO-" + code,
				ShipmentID: pulid.MustNew("shp_"),
				MoveStatuses: []shipmentvocab.MoveStatus{
					shipmentvocab.MoveStatusNew,
					shipmentvocab.MoveStatusNew,
					shipmentvocab.MoveStatusNew,
					shipmentvocab.MoveStatusNew,
				},
			},
		)

		workers := make([]*worker.Worker, 0, 3)
		for i, name := range []string{"Planned", "Alternative", "Other"} {
			w := &worker.Worker{
				ID:                   pulid.MustNew("wrk_"),
				BusinessUnitID:       tenant.BuID,
				OrganizationID:       tenant.OrgID,
				StateID:              data.State.ID,
				Status:               domaintypes.StatusActive,
				Type:                 worker.WorkerTypeEmployee,
				DriverType:           worker.DriverTypeOTR,
				FirstName:            name,
				LastName:             code,
				AddressLine1:         "1 Depot Road",
				City:                 "Springfield",
				PostalCode:           "62701",
				Email:                strings.ToLower(name+"-"+code) + "@example.com",
				PhoneNumber:          "555-010" + string(rune('0'+i)),
				Gender:               worker.GenderMale,
				CanBeAssigned:        true,
				AvailableForDispatch: true,
			}
			require.NoError(t, workerdb.Workers.Create(scoped, db, w))
			workers = append(workers, w)
		}

		dc := dispatchcontroldb.DispatchControls
		_, err := dc.CreateOrSkip(
			scoped,
			db,
			[]*dispatchcontrol.DispatchControl{
				dispatchcontrol.NewDefaultDispatchControl(tenant.OrgID, tenant.BuID),
			},
			dc.OrganizationID,
			dc.BusinessUnitID,
		)
		require.NoError(t, err)
		_, err = dc.Where(dc.OrganizationID.Eq(tenant.OrgID)).
			UpdateReturning(scoped, db, dc.LearnDispatchStyle.To(code == "DLS"))
		require.NoError(t, err)

		return tenantScope{ctx: scoped, tenant: tenant, graph: graph, workers: workers}
	}

	return &harness{
		db: db,
		repo: dispatchlearningrepository.New(dispatchlearningrepository.Params{
			DB:     postgres.WrapDB(db),
			Logger: zap.NewNop(),
		}),
		own:   seed(seedtest.SeedFullTestData(t, ctx, db), "DLS"),
		other: seed(seedtest.SeedAdditionalTenant(t, ctx, db, "DL"), "DLS-OT"),
	}
}

func (h *harness) price(
	t *testing.T,
	s tenantScope,
	move int,
	generatedAt int64,
) {
	t.Helper()

	require.NoError(t, dispatchplanpricedb.DispatchPlanPrices.Create(s.ctx, h.db,
		&dispatchplanprice.DispatchPlanPrice{
			OrganizationID: s.tenant.OrgID,
			BusinessUnitID: s.tenant.BuID,
			MoveID:         s.graph.Moves[move].ID,
			WorkerID:       s.workers[0].ID,
			WorkerName:     "Planned",
			CostCents:      20_000,
			CostLines: []dispatcheconomics.Line{
				{Component: dispatcheconomics.ComponentHomeTime, Cents: 10_000},
			},
			CoveragePriceCents: 40_000,
			Alternatives: []dispatchplanprice.Alternative{{
				WorkerID:   s.workers[1].ID,
				WorkerName: "Alternative",
				ExtraCents: 1_500,
				CostCents:  21_000,
				Lines: []dispatcheconomics.Line{
					{Component: dispatcheconomics.ComponentHaulFit, Cents: 4_000},
				},
			}},
			GeneratedAt: generatedAt,
		}))
}

func (h *harness) assign(
	t *testing.T,
	s tenantScope,
	move, workerIndex int,
	actor shipmentevent.ActorType,
	occurredAt int64,
) pulid.ID {
	t.Helper()

	workerID := s.workers[workerIndex].ID
	assignment := &shipment.Assignment{
		ID:              pulid.MustNew("asn_"),
		OrganizationID:  s.tenant.OrgID,
		BusinessUnitID:  s.tenant.BuID,
		ShipmentMoveID:  s.graph.Moves[move].ID,
		PrimaryWorkerID: &workerID,
	}
	require.NoError(t, shipmentdb.Assignments.Create(s.ctx, h.db, assignment))

	event := &shipmentevent.Event{
		ID:             pulid.MustNew("se_"),
		OrganizationID: s.tenant.OrgID,
		BusinessUnitID: s.tenant.BuID,
		ShipmentID:     s.graph.Shipment.ID,
		MoveID:         s.graph.Moves[move].ID,
		AssignmentID:   assignment.ID,
		Type:           shipmentevent.TypeDriverAssigned,
		Severity:       shipmentevent.SeverityMuted,
		ActorType:      actor,
		Summary:        "Driver assigned",
		OccurredAt:     occurredAt,
	}
	require.NoError(t, shipmenteventdb.Events.Create(s.ctx, h.db, event))

	return event.ID
}

func TestListDecisions_ReadsPeoplesAssignmentsAgainstTheirFreshPlan(t *testing.T) {
	h := newHarness(t)
	s := h.own

	h.price(t, s, 0, decidedAt-600)
	h.price(t, s, 1, decidedAt-600)
	h.price(t, s, 2, decidedAt-3*86_400)
	h.price(t, s, 3, decidedAt-600)

	h.assign(t, s, 0, 1, shipmentevent.ActorUser, decidedAt)
	h.assign(t, s, 1, 0, shipmentevent.ActorSystem, decidedAt+10)
	h.assign(t, s, 2, 1, shipmentevent.ActorUser, decidedAt+20)
	last := h.assign(t, s, 3, 2, shipmentevent.ActorUser, decidedAt+30)
	h.price(t, h.other, 0, decidedAt-600)
	h.assign(t, h.other, 0, 1, shipmentevent.ActorUser, decidedAt)

	page, err := h.repo.ListDecisions(s.ctx, &dispatchlearningports.ListDecisionsRequest{
		TenantInfo: s.tenant,
		Limit:      50,
	})
	require.NoError(t, err)

	assert.Equal(t, 3, page.Scanned, "only assignments people made, and only this tenant's")
	assert.Equal(t, decidedAt+30, page.Last.At)
	assert.Equal(t, last, page.Last.ID)
	require.Len(t, page.Decisions, 2, "a plan older than a day is not the context of a decision")

	override := page.Decisions[0]
	assert.Equal(t, dispatchlearning.OutcomeOverridden, override.Outcome())
	assert.Equal(t, s.workers[0].ID, override.Planned.WorkerID)
	assert.Equal(t, int64(20_000), override.Planned.CostCents)
	assert.Equal(t, []dispatcheconomics.Line{
		{Component: dispatcheconomics.ComponentHomeTime, Cents: 10_000},
	}, override.Planned.Lines)
	require.Len(t, override.Alternatives, 1)
	assert.Equal(t, int64(1_500), override.Alternatives[0].ExtraCents)
	assert.Equal(t, int64(21_000), override.Alternatives[0].CostCents)

	assert.Equal(t, dispatchlearning.OutcomeDeclined, page.Decisions[1].Outcome())

	next, err := h.repo.ListDecisions(s.ctx, &dispatchlearningports.ListDecisionsRequest{
		TenantInfo: s.tenant,
		After:      page.Last,
		Limit:      50,
	})
	require.NoError(t, err)
	assert.Zero(t, next.Scanned, "the cursor passes every scanned assignment")
	assert.Equal(t, page.Last, next.Last)
}

func TestListDecisions_PagesInTheOrderAssignmentsHappened(t *testing.T) {
	h := newHarness(t)
	s := h.own
	for move := range 3 {
		h.price(t, s, move, decidedAt-600)
		h.assign(t, s, move, 1, shipmentevent.ActorUser, decidedAt+int64(move))
	}

	first, err := h.repo.ListDecisions(s.ctx, &dispatchlearningports.ListDecisionsRequest{
		TenantInfo: s.tenant,
		Limit:      2,
	})
	require.NoError(t, err)
	require.Equal(t, 2, first.Scanned)
	assert.Equal(t, decidedAt+1, first.Last.At)

	second, err := h.repo.ListDecisions(s.ctx, &dispatchlearningports.ListDecisionsRequest{
		TenantInfo: s.tenant,
		After:      first.Last,
		Limit:      2,
	})
	require.NoError(t, err)
	require.Equal(t, 1, second.Scanned)
	assert.Equal(t, decidedAt+2, second.Last.At)
}

func TestStyle_IsCreatedOnceSavedWithItsVersionAndHiddenFromOtherTenants(t *testing.T) {
	h := newHarness(t)
	s := h.own

	missing, err := h.repo.Get(s.ctx, s.tenant)
	require.NoError(t, err)
	assert.Nil(t, missing)

	style, err := h.repo.GetOrCreate(
		s.ctx,
		dispatchlearning.New(s.tenant.OrgID, s.tenant.BuID, decidedAt),
	)
	require.NoError(t, err)
	require.False(t, style.ID.IsNil())

	again, err := h.repo.GetOrCreate(
		s.ctx,
		dispatchlearning.New(s.tenant.OrgID, s.tenant.BuID, decidedAt+1),
	)
	require.NoError(t, err)
	assert.Equal(t, style.ID, again.ID)
	assert.Equal(t, decidedAt, again.LearningSince)

	style.Weights[dispatcheconomics.ComponentHomeTime] = dispatchlearning.Weight{
		Multiplier: 1.6,
		Samples:    12,
	}
	style.Decisions, style.Overrides = 12, 12
	style.Advance(decidedAt, pulid.ID("se_cursor"))
	saved, err := h.repo.Save(s.ctx, style)
	require.NoError(t, err)

	stale := *again
	stale.Decisions, stale.Approvals = 1, 1
	_, err = h.repo.Save(s.ctx, &stale)
	require.Error(t, err, "a save from an older read loses to the newer one")

	stored, err := h.repo.Get(s.ctx, s.tenant)
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, saved.Version, stored.Version)
	assert.InDelta(t, 1.6, stored.Weights[dispatcheconomics.ComponentHomeTime].Multiplier, 1e-9)
	assert.Equal(t, decidedAt, stored.ObservedThroughAt)
	assert.Equal(t, pulid.ID("se_cursor"), stored.ObservedThroughID)

	theirs, err := h.repo.Get(h.other.ctx, h.other.tenant)
	require.NoError(t, err)
	assert.Nil(t, theirs, "another organization never sees this style")
}

func TestListLearningTenants_ListsOnlyOrganizationsThatLearn(t *testing.T) {
	h := newHarness(t)

	tenants, err := h.repo.ListLearningTenants(
		t.Context(),
		&dispatchlearningports.ListTenantsRequest{Limit: 50},
	)
	require.NoError(t, err)

	assert.Contains(
		t,
		tenants,
		pagination.TenantInfo{OrgID: h.own.tenant.OrgID, BuID: h.own.tenant.BuID},
	)
	assert.NotContains(
		t,
		tenants,
		pagination.TenantInfo{OrgID: h.other.tenant.OrgID, BuID: h.other.tenant.BuID},
	)

	after, err := h.repo.ListLearningTenants(
		t.Context(),
		&dispatchlearningports.ListTenantsRequest{
			After: pagination.TenantInfo{OrgID: h.own.tenant.OrgID, BuID: h.own.tenant.BuID},
			Limit: 50,
		},
	)
	require.NoError(t, err)
	assert.NotContains(
		t,
		after,
		pagination.TenantInfo{OrgID: h.own.tenant.OrgID, BuID: h.own.tenant.BuID},
	)
}
