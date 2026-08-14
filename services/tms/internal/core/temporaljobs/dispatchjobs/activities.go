package dispatchjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	DispatchControlRepo repositories.DispatchControlRepository
	AutoAssign          portservices.DispatchAutoAssignService
	Logger              *zap.Logger
}

type Activities struct {
	dispatchControlRepo repositories.DispatchControlRepository
	autoAssign          portservices.DispatchAutoAssignService
	logger              *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		dispatchControlRepo: p.DispatchControlRepo,
		autoAssign:          p.AutoAssign,
		logger:              p.Logger.Named("dispatch-activities"),
	}
}

// HorizonPlanSweepActivity re-plans the horizon for every organization running
// horizon planning.
//
// It never applies. A scheduled pass exists to build evidence — the plan is recorded
// as an agent run with per-move proposals, so what the planner proposed can be read
// back against what dispatchers actually did. Whether any of it executes stays with
// the organization's autonomy tier, through the normal request path.
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

	for _, tour := range plan.Tours {
		outcome.TotalDeadheadMiles += tour.TotalDeadheadMiles
		// Only the moves past the first in a tour were chained; the first would have
		// been assigned by single-period planning too. This is the number that says
		// whether horizon planning is doing anything Immediate could not.
		if len(tour.MoveIDs) > 1 {
			outcome.ChainedMoves += len(tour.MoveIDs) - 1
		}
	}

	return outcome
}
