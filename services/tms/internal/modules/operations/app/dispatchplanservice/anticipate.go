package dispatchplanservice

import (
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/shared/assignmentsolver"
	"github.com/emoss08/trenova/shared/pulid"
)

func reanticipate(
	graph *dispatchcandidateservice.Graph,
	snapshot *dispatchcandidateservice.FleetSnapshot,
	stab stability,
) {
	if !snapshot.Forecast.Ready() || !anticipating(graph) {
		return
	}

	first := assignmentsolver.Solve(stab.apply(problemOf(graph)))
	next := snapshot.Forecast.WithSupply(plannedSupply(graph, snapshot, first.ColAssignment))
	for i := range graph.Arcs {
		for j := range graph.Arcs[i] {
			arc := &graph.Arcs[i][j]
			candidate := arc.Candidate
			if candidate == nil || candidate.EndMarket == nil ||
				candidate.EndMarket.Source != dispatchplanports.EndMarketForecast {
				continue
			}
			before := candidate.EndMarket.Cents
			dispatchcandidateservice.Reprice(candidate.EndMarket, next, candidate.WorkerID)
			delta := candidate.EndMarket.Cents - before
			if delta == 0 {
				continue
			}
			arc.CostCents -= delta
			candidate.HardCents -= delta
			candidate.Cost = withEndMarket(candidate.Cost, -candidate.EndMarket.Cents)
		}
	}
}

func anticipating(graph *dispatchcandidateservice.Graph) bool {
	for _, arcs := range graph.Arcs {
		for _, arc := range arcs {
			if arc.Candidate != nil && arc.Candidate.EndMarket != nil &&
				arc.Candidate.EndMarket.Source == dispatchplanports.EndMarketForecast {
				return true
			}
		}
	}

	return false
}

func plannedSupply(
	graph *dispatchcandidateservice.Graph,
	snapshot *dispatchcandidateservice.FleetSnapshot,
	colAssignment []int,
) []demandforecast.Driver {
	ends := make(map[pulid.ID]demandforecast.Driver, len(colAssignment))
	for col, row := range colAssignment {
		if row == assignmentsolver.Unassigned || row >= len(graph.Moves) {
			continue
		}
		move := graph.Moves[row]
		candidate := arcCandidate(graph.Arcs[row], col)
		if candidate == nil || move.DestinationLatitude == nil || move.DestinationLongitude == nil {
			continue
		}
		cell, ok := dispatchmarket.CellOf(dispatchmarket.Point{
			Latitude:  *move.DestinationLatitude,
			Longitude: *move.DestinationLongitude,
		})
		if !ok {
			continue
		}
		ends[candidate.WorkerID] = demandforecast.Driver{
			WorkerID: candidate.WorkerID,
			Cell:     cell,
			FreeAt:   candidate.ProjectedFreeAt,
		}
	}

	supply := make([]demandforecast.Driver, 0, len(snapshot.Drivers)+len(ends))
	for _, driver := range snapshot.Supply() {
		if _, planned := ends[driver.WorkerID]; !planned {
			supply = append(supply, driver)
		}
	}
	for col := range colAssignment {
		if col >= len(graph.Drivers) {
			continue
		}
		if end, planned := ends[graph.Drivers[col].WorkerID]; planned {
			supply = append(supply, end)
		}
	}

	return supply
}

func arcCandidate(arcs []dispatchcandidateservice.Arc, col int) *dispatchplanports.CandidateScore {
	for _, arc := range arcs {
		if arc.Driver == col {
			return arc.Candidate
		}
	}

	return nil
}

func withEndMarket(cost dispatcheconomics.Cost, cents int64) dispatcheconomics.Cost {
	lines := make([]dispatcheconomics.Line, 0, len(cost.Lines)+1)
	total := int64(0)
	for _, line := range cost.Lines {
		if line.Component == dispatcheconomics.ComponentEndMarket {
			continue
		}
		lines = append(lines, line)
		total += line.Cents
	}
	if cents != 0 {
		lines = append(lines, dispatcheconomics.Line{
			Component: dispatcheconomics.ComponentEndMarket,
			Cents:     cents,
		})
		total += cents
	}

	return dispatcheconomics.Cost{TotalCents: total, Lines: lines}
}
