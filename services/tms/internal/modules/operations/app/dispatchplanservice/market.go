package dispatchplanservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchmarketports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/assignmentsolver"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type marketParams struct {
	TenantInfo pagination.TenantInfo
	Plan       *dispatchplanports.Plan
	Graph      *dispatchcandidateservice.Graph
	Snapshot   *dispatchcandidateservice.FleetSnapshot
}

func (s *Service) recordMarket(ctx context.Context, p *marketParams) {
	if err := s.market.Record(ctx, &dispatchmarketports.RecordRequest{
		TenantInfo: p.TenantInfo,
		ObservedAt: p.Plan.GeneratedAt,
		Values:     driverValues(p),
	}); err != nil {
		s.l.Warn("dispatch market values were not recorded; the plan stands",
			zap.String("organizationId", p.TenantInfo.OrgID.String()),
			zap.Error(err))
	}
}

func driverValues(p *marketParams) []dispatchmarket.DriverValue {
	duals := driverDuals(p.Plan, p.Graph)
	values := make([]dispatchmarket.DriverValue, 0, len(p.Graph.Drivers))
	for _, driver := range p.Graph.Drivers {
		start, availableAt, ok := p.Snapshot.DriverStart(driver.WorkerID)
		if !ok {
			continue
		}
		values = append(values, dispatchmarket.DriverValue{
			Start:       start,
			AvailableAt: availableAt,
			ValueCents:  duals[driver.WorkerID],
		})
	}

	return values
}

func driverDuals(
	plan *dispatchplanports.Plan,
	graph *dispatchcandidateservice.Graph,
) map[pulid.ID]int64 {
	duals := make(map[pulid.ID]int64, len(plan.Pairings))
	if plan.Mode == dispatchcontrol.PlanningModeImmediate {
		for _, pairing := range plan.Pairings {
			if pairing.Candidate != nil && pairing.DriverValueCents > 0 {
				duals[pairing.Candidate.WorkerID] = pairing.DriverValueCents
			}
		}
		return duals
	}

	solution := assignmentsolver.SolveWithPrices(problemOf(graph))
	for col, driver := range graph.Drivers {
		if value := solution.ColumnValue(col); value > 0 {
			duals[driver.WorkerID] = value
		}
	}

	return duals
}
