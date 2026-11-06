package dispatchplanservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/assignmentsolver"
	"github.com/emoss08/trenova/shared/dispatchplanner"
	"github.com/emoss08/trenova/shared/pulid"
)

type stability struct {
	incumbents []int
	cents      int64
}

func (s *Service) standingOf(
	ctx context.Context,
	tenant pagination.TenantInfo,
	moves []*dispatchboardports.BoardMove,
) (map[pulid.ID]pulid.ID, error) {
	moveIDs := make([]pulid.ID, 0, len(moves))
	for _, move := range moves {
		moveIDs = append(moveIDs, move.MoveID)
	}

	prices, err := s.prices.ListByMoves(ctx, &dispatchplanports.ListPricesRequest{
		TenantInfo: tenant,
		MoveIDs:    moveIDs,
	})
	if err != nil {
		return nil, err
	}

	standing := make(map[pulid.ID]pulid.ID, len(prices))
	for _, price := range prices {
		if price.Covered() {
			standing[price.MoveID] = price.WorkerID
		}
	}

	return standing, nil
}

func stabilityOf(
	graph *dispatchcandidateservice.Graph,
	standing map[pulid.ID]pulid.ID,
	cents int64,
) stability {
	return stability{incumbents: incumbentsOf(graph, standing), cents: max(cents, 0)}
}

func (s stability) apply(problem assignmentsolver.Problem) assignmentsolver.Problem {
	if s.incumbents == nil || s.cents == 0 {
		return problem
	}

	uncovered := make([]int64, len(problem.Uncovered))
	copy(uncovered, problem.Uncovered)
	rows := make([][]assignmentsolver.Arc, len(problem.Rows))
	for i, arcs := range problem.Rows {
		incumbent := s.incumbents[i]
		if incumbent == dispatchplanner.Unassigned {
			rows[i] = arcs
			continue
		}
		uncovered[i] += s.cents
		rows[i] = make([]assignmentsolver.Arc, len(arcs))
		for j, arc := range arcs {
			if arc.Col != incumbent {
				arc.Cost += s.cents
			}
			rows[i][j] = arc
		}
	}

	return assignmentsolver.Problem{Cols: problem.Cols, Rows: rows, Uncovered: uncovered}
}
