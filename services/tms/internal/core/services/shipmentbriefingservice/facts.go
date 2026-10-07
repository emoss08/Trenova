package shipmentbriefingservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentbrief"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/stringutils"
)

type Facts struct {
	shipmentbrief.Facts

	OperationType tenant.OperationType
}

func (f Facts) Counts() []int {
	return []int{
		f.DeliveringToday,
		f.Moving,
		f.Late,
		f.Uncovered,
		f.Detention,
		f.ReadyToBill,
		f.LowMargin,
		f.OpenSuggestions,
	}
}

func (f Facts) Filters() []shipment.QuickFilter {
	candidates := []struct {
		count  int
		filter shipment.QuickFilter
	}{
		{f.DeliveringToday, shipment.QuickFilterDeliveringToday},
		{f.Moving, shipment.QuickFilterMoving},
		{f.Late, shipment.QuickFilterLate},
		{f.Uncovered, shipment.QuickFilterUncovered},
		{f.Detention, shipment.QuickFilterDetention},
		{f.ReadyToBill, shipment.QuickFilterReadyToBill},
		{f.LowMargin, shipment.QuickFilterLowMargin},
	}

	out := make([]shipment.QuickFilter, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.count > 0 {
			out = append(out, candidate.filter)
		}
	}
	return out
}

func text(value string) services.ShipmentBriefingSegment {
	return services.ShipmentBriefingSegment{Text: value}
}

func linked(value string, filter shipment.QuickFilter) services.ShipmentBriefingSegment {
	return services.ShipmentBriefingSegment{Text: value, Filter: &filter}
}

func loads(count int) string {
	return stringutils.CountNoun(count, "load", "loads")
}

func joinClauses(clauses [][]services.ShipmentBriefingSegment) []services.ShipmentBriefingSegment {
	segments := make([]services.ShipmentBriefingSegment, 0, len(clauses)*2+1)
	for i, clause := range clauses {
		switch {
		case i == 0:
		case i == len(clauses)-1:
			segments = append(segments, text(" and "))
		default:
			segments = append(segments, text(", "))
		}
		segments = append(segments, clause...)
	}
	if len(clauses) > 0 {
		segments = append(segments, text(". "))
	}
	return segments
}

func Deterministic(f Facts) []services.ShipmentBriefingSegment {
	if f.DeliveringToday == 0 && f.Moving == 0 && f.Late == 0 && f.Uncovered == 0 &&
		f.Detention == 0 && f.ReadyToBill == 0 {
		return []services.ShipmentBriefingSegment{text("Nothing on the board needs you right now.")}
	}

	board := make([][]services.ShipmentBriefingSegment, 0, 3)
	if f.DeliveringToday > 0 {
		board = append(board, []services.ShipmentBriefingSegment{linked(
			loads(f.DeliveringToday)+" "+
				stringutils.Pluralize("delivers", "deliver", f.DeliveringToday)+" today",
			shipment.QuickFilterDeliveringToday,
		)})
	}
	if f.Moving > 0 {
		board = append(board, []services.ShipmentBriefingSegment{linked(
			loads(f.Moving)+" "+stringutils.Pluralize("is", "are", f.Moving)+" moving on schedule",
			shipment.QuickFilterMoving,
		)})
	}
	if f.Late > 0 {
		clause := []services.ShipmentBriefingSegment{linked(
			loads(f.Late)+" "+stringutils.Pluralize("is", "are", f.Late)+" late",
			shipment.QuickFilterLate,
		)}
		if reason := strings.TrimSpace(f.LateReason); reason != "" {
			clause = append(clause, text(", mostly "+strings.ToLower(reason)))
		}
		board = append(board, clause)
	}

	followUp := make([][]services.ShipmentBriefingSegment, 0, 3)
	if f.Uncovered > 0 {
		followUp = append(followUp, []services.ShipmentBriefingSegment{linked(
			loads(f.Uncovered)+" still "+stringutils.Pluralize("needs", "need", f.Uncovered)+" "+
				f.OperationType.CoverageNoun(),
			shipment.QuickFilterUncovered,
		)})
	}
	if f.Detention > 0 {
		followUp = append(followUp, []services.ShipmentBriefingSegment{linked(
			loads(f.Detention)+" "+stringutils.Pluralize("is", "are", f.Detention)+
				" accruing detention",
			shipment.QuickFilterDetention,
		)})
	}
	if f.ReadyToBill > 0 {
		followUp = append(followUp, []services.ShipmentBriefingSegment{linked(
			loads(f.ReadyToBill)+" "+stringutils.Pluralize("is", "are", f.ReadyToBill)+
				" ready to bill",
			shipment.QuickFilterReadyToBill,
		)})
	}

	segments := append(joinClauses(board), joinClauses(followUp)...)
	if last := len(segments) - 1; last >= 0 && segments[last].Text == ". " {
		segments[last].Text = "."
	}

	return segments
}
