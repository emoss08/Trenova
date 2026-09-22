package dispatchjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/watchtowersources"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// secondsPerHour turns the coverage window, which people set in hours, into
// the epoch seconds every timestamp here is in.
const secondsPerHour = int64(3600)

type ActivitiesParams struct {
	fx.In

	DispatchControlRepo repositories.DispatchControlRepository
	ProposalRepo        repositories.AgentProposalRepository
	AutoAssign          portservices.DispatchAutoAssignService
	// Watchtower puts a load about to leave with nobody on it on the feed.
	Watchtower portservices.WatchtowerProjector `optional:"true"`
	// Publisher wakes whichever agent covers dispatch for the same moves.
	Publisher portservices.AgentEventPublisher `optional:"true"`
	Logger    *zap.Logger
}

type Activities struct {
	dispatchControlRepo repositories.DispatchControlRepository
	proposalRepo        repositories.AgentProposalRepository
	autoAssign          portservices.DispatchAutoAssignService
	watchtower          portservices.WatchtowerProjector
	publisher           portservices.AgentEventPublisher
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
	tenants, err := a.dispatchControlRepo.ListHorizonPlanningTenants(ctx)
	if err != nil {
		return nil, err
	}

	result := &HorizonPlanSweepResult{
		TenantsScanned: len(tenants),
		TenantOutcomes: make([]*TenantHorizonPlan, 0, len(tenants)),
	}

	for _, tenant := range tenants {
		outcome := a.planTenant(ctx, tenant)
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

	plan, err := a.autoAssign.Plan(ctx, &portservices.DispatchPlanRequest{
		TenantInfo: tenant,
		Apply:      false,
	})
	if err != nil {
		a.logger.Error("horizon plan failed for tenant",
			zap.String("orgId", tenant.OrgID.String()),
			zap.Error(err),
		)
		outcome.Error = err.Error()
		return outcome
	}

	outcome.MovesPlanned = len(plan.Assignments)
	outcome.MovesUncovered = len(plan.Uncovered)
	outcome.ToursBuilt = len(plan.Tours)
	outcome.TotalScore = plan.TotalScore
	outcome.ShadowMode = plan.ShadowMode
	outcome.RunID = plan.RunID.String()

	a.retireProposals(ctx, tenant, plan, outcome)
	outcome.MovesAtRisk = a.raiseCoverageRisk(ctx, tenant, plan.Uncovered)

	for _, tour := range plan.Tours {
		outcome.TotalDeadheadMiles += tour.TotalDeadheadMiles
		if len(tour.MoveIDs) > 1 {
			outcome.ChainedMoves += len(tour.MoveIDs) - 1
		}
	}

	return outcome
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
	uncovered []*portservices.DispatchUncoveredMove,
) int {
	dated := make([]*portservices.DispatchUncoveredMove, 0, len(uncovered))
	for _, move := range uncovered {
		if move != nil && move.StartsAt > 0 {
			dated = append(dated, move)
		}
	}
	if len(dated) == 0 {
		return 0
	}

	control, err := a.dispatchControlRepo.GetByOrgID(
		ctx,
		repositories.GetDispatchControlRequest{TenantInfo: tenant},
	)
	if err != nil {
		a.logger.Warn("coverage window unavailable for tenant",
			zap.String("orgId", tenant.OrgID.String()),
			zap.Error(err),
		)

		return 0
	}

	now := a.now()
	cutoff := now + int64(control.CoverageWindowHours())*secondsPerHour

	raised := 0
	for _, move := range dated {
		if move.StartsAt > cutoff {
			continue
		}
		raised++

		item := watchtowersources.UncoveredMove{
			TenantInfo: tenant,
			MoveID:     move.MoveID,
			ProNumber:  move.ProNumber,
			Reason:     move.Reason,
			StartsAt:   move.StartsAt,
			HoursOut:   (move.StartsAt - now) / secondsPerHour,
		}
		if a.watchtower != nil {
			a.watchtower.Upsert(ctx, watchtowersources.DescribeMoveCoverageRisk(item))
		}
		portservices.PublishAgentEvent(ctx, a.publisher, portservices.AgentEvent{
			Kind:       agent.EventShipmentMoveCoverageAtRisk,
			SubjectID:  move.MoveID,
			TenantInfo: tenant,
		})
	}

	return raised
}

func (a *Activities) retireProposals(
	ctx context.Context,
	tenant pagination.TenantInfo,
	plan *portservices.DispatchPlan,
	outcome *TenantHorizonPlan,
) {
	if plan.RunID.IsNil() {
		return
	}

	expired, err := a.proposalRepo.ExpirePendingByRun(
		ctx,
		repositories.ExpireAgentProposalsByRunRequest{
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
