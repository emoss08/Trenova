package dispatchjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/watchtowersources"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/dispatchjobs/dispatchcoveragecontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// secondsPerHour turns the epoch seconds every timestamp here is in into the
// hours a move is out, which is how people read the coverage window.
const secondsPerHour = int64(3600)

type ActivitiesParams struct {
	fx.In

	DispatchControlRepo dispatchcontrolports.DispatchControlRepository
	ProposalRepo        agentproposalports.AgentProposalRepository
	AutoAssign          dispatchconsoleports.DispatchAutoAssignService
	// Watchtower puts a load about to leave with nobody on it on the feed.
	Watchtower watchtowerports.WatchtowerProjector
	// Publisher wakes whichever agent covers dispatch for the same moves.
	Publisher agenteventports.AgentEventPublisher
	Logger    *zap.Logger
}

type Activities struct {
	dispatchControlRepo dispatchcontrolports.DispatchControlRepository
	proposalRepo        agentproposalports.AgentProposalRepository
	autoAssign          dispatchconsoleports.DispatchAutoAssignService
	watchtower          watchtowerports.WatchtowerProjector
	publisher           agenteventports.AgentEventPublisher
	now                 func() int64
	logger              *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		dispatchControlRepo: p.DispatchControlRepo,
		proposalRepo:        p.ProposalRepo,
		autoAssign:          p.AutoAssign,
		watchtower:          p.Watchtower,
		publisher:           p.Publisher,
		now:                 timeutils.NowUnix,
		logger:              p.Logger.Named("dispatch-activities"),
	}
}

func (a *Activities) HorizonPlanSweepActivity(
	ctx context.Context,
) (*HorizonPlanSweepResult, error) {
	tenants, err := a.dispatchControlRepo.ListCoveragePlanningTenants(ctx)
	if err != nil {
		return nil, err
	}

	result := &HorizonPlanSweepResult{
		TenantsScanned: len(tenants),
		TenantOutcomes: make([]*TenantHorizonPlan, 0, len(tenants)),
	}

	for _, tenant := range tenants {
		outcome := a.planTenant(dbscope.WithTenant(ctx, tenant.DBTenant()), tenant)
		result.TenantOutcomes = append(result.TenantOutcomes, outcome)

		if outcome.Error != "" {
			result.TenantsFailed++
			continue
		}

		result.TenantsPlanned++
		result.MovesPlanned += outcome.MovesPlanned
		result.MovesUncovered += outcome.MovesUncovered
		result.MovesAtRisk += outcome.MovesAtRisk
		result.ToursBuilt += outcome.ToursBuilt
		result.ChainedMoves += outcome.ChainedMoves
	}

	return result, nil
}

func (a *Activities) planTenant(
	ctx context.Context,
	tenant pagination.TenantInfo,
) *TenantHorizonPlan {
	outcome := &TenantHorizonPlan{
		OrganizationID: tenant.OrgID.String(),
		BusinessUnitID: tenant.BuID.String(),
	}

	control, err := a.dispatchControlRepo.GetByOrgID(
		ctx,
		dispatchcontrolports.GetDispatchControlRequest{TenantInfo: tenant},
	)
	if err != nil {
		a.logger.Error("dispatch control unavailable for a planned tenant",
			zap.String("orgId", tenant.OrgID.String()),
			zap.Error(err),
		)
		outcome.Error = err.Error()
		return outcome
	}

	plan, err := a.autoAssign.Plan(ctx, &dispatchconsoleports.DispatchPlanRequest{
		TenantInfo: tenant,
		Apply:      control.AutoProposeCoverage,
		Trigger:    agent.RunTriggerScheduled,
	})
	if err != nil {
		a.logger.Error("coverage plan failed for tenant",
			zap.String("orgId", tenant.OrgID.String()),
			zap.Error(err),
		)
		outcome.Error = err.Error()
		return outcome
	}

	outcome.MovesPlanned = len(plan.Assignments)
	outcome.MovesUncovered = len(plan.Uncovered)
	outcome.ToursBuilt = len(plan.Tours)
	outcome.CostCents = plan.CostCents
	outcome.UncoveredCents = plan.UncoveredCents
	outcome.ShadowMode = plan.ShadowMode
	outcome.RunID = plan.RunID.String()

	// An organization that keeps coverage proposed leaves them waiting for a
	// dispatcher; one that plans the horizon only to spot risk does not leave
	// a plan nobody asked for in Decisions.
	if !control.AutoProposeCoverage {
		a.retireProposals(ctx, tenant, plan, outcome)
	}
	outcome.MovesAtRisk = a.raiseCoverageRisk(ctx, tenant, control, plan.Uncovered)

	for _, tour := range plan.Tours {
		outcome.TotalDeadheadMiles += tour.EmptyMiles
		if len(tour.MoveIDs) > 1 {
			outcome.ChainedMoves += len(tour.MoveIDs) - 1
		}
	}

	return outcome
}

// RefreshCoverageActivity plans one organization's coverage again after its
// board changed, for an organization that keeps coverage proposed. The plan
// supersedes the proposals of the one before it, so a dispatcher always
// reads the newest.
func (a *Activities) RefreshCoverageActivity(
	ctx context.Context,
	payload *dispatchcoveragecontract.RefreshPayload,
) (*CoverageRefreshResult, error) {
	tenant := payload.TenantInfo()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	control, err := a.dispatchControlRepo.GetByOrgID(
		ctx,
		dispatchcontrolports.GetDispatchControlRequest{TenantInfo: tenant},
	)
	if err != nil {
		return nil, err
	}
	if !control.EnableAutoAssignment || !control.AutoProposeCoverage {
		return &CoverageRefreshResult{Skipped: true}, nil
	}

	plan, err := a.autoAssign.Plan(ctx, &dispatchconsoleports.DispatchPlanRequest{
		TenantInfo: tenant,
		Apply:      true,
		Trigger:    agent.RunTriggerEvent,
	})
	if err != nil {
		return nil, err
	}

	return &CoverageRefreshResult{
		RunID:          plan.RunID.String(),
		MovesProposed:  len(plan.Assignments),
		MovesUncovered: len(plan.Uncovered),
		MovesAtRisk:    a.raiseCoverageRisk(ctx, tenant, control, plan.Uncovered),
	}, nil
}

// raiseCoverageRisk puts the uncovered moves that are close enough to
// their start to matter on the feed, and wakes the dispatch desk for each.
// The planner looks further ahead than this on purpose: it plans a whole
// day, while the coverage window asks who is going to run a load that is
// about to leave. A move outside the window is still uncovered and still
// planned for; it is simply not yet news.
func (a *Activities) raiseCoverageRisk(
	ctx context.Context,
	tenant pagination.TenantInfo,
	control *dispatchcontrol.DispatchControl,
	uncovered []*dispatchplanports.Uncovered,
) int {
	now := a.now()

	raised := 0
	for _, move := range uncovered {
		if move == nil || move.StartsAt <= 0 ||
			!control.StartsInsideCoverageWindow(move.StartsAt, now) {
			continue
		}
		raised++

		item := watchtowersources.UncoveredMove{
			TenantInfo: tenant,
			MoveID:     move.MoveID,
			ShipmentID: move.ShipmentID,
			ProNumber:  move.ProNumber,
			Reason:     move.Explanation(),
			StartsAt:   move.StartsAt,
			HoursOut:   (move.StartsAt - now) / secondsPerHour,
		}
		a.watchtower.Upsert(ctx, watchtowersources.DescribeMoveCoverageRisk(item))

		agenteventports.PublishAgentEvent(ctx, a.publisher, agenteventports.AgentEvent{
			Kind:       agenttrigger.EventShipmentMoveCoverageAtRisk,
			SubjectID:  move.MoveID,
			TenantInfo: tenant,
		})
	}

	return raised
}

func (a *Activities) retireProposals(
	ctx context.Context,
	tenant pagination.TenantInfo,
	plan *dispatchconsoleports.DispatchPlan,
	outcome *TenantHorizonPlan,
) {
	if plan.RunID.IsNil() {
		return
	}

	expired, err := a.proposalRepo.ExpirePendingByRun(
		ctx,
		agentproposalports.ExpireAgentProposalsByRunRequest{
			RunID:      plan.RunID,
			TenantInfo: tenant,
		},
	)
	if err != nil {
		a.logger.Warn("failed to retire swept dispatch proposals",
			zap.String("orgId", tenant.OrgID.String()),
			zap.String("runId", plan.RunID.String()),
			zap.Error(err),
		)
		return
	}

	outcome.ProposalsRetired = expired
}
