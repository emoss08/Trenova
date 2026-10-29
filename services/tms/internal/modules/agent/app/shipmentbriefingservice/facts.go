package shipmentbriefingservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/shipmentbrief"
	"github.com/emoss08/trenova/internal/modules/agent/ports/shipmentbriefports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/quickfilter"
	"github.com/emoss08/trenova/shared/inflect"
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
		f.OpenSuggestions,
	}
}

func (f Facts) Filters() []quickfilter.QuickFilter {
	candidates := []struct {
		count  int
		filter quickfilter.QuickFilter
	}{
		{f.DeliveringToday, quickfilter.QuickFilterDeliveringToday},
		{f.Moving, quickfilter.QuickFilterMoving},
		{f.Late, quickfilter.QuickFilterLate},
		{f.Uncovered, quickfilter.QuickFilterUncovered},
		{f.Detention, quickfilter.QuickFilterDetention},
		{f.ReadyToBill, quickfilter.QuickFilterReadyToBill},
	}

	out := make([]quickfilter.QuickFilter, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.count > 0 {
			out = append(out, candidate.filter)
		}
	}
	return out
}

func text(value string) shipmentbriefports.ShipmentBriefingSegment {
	return shipmentbriefports.ShipmentBriefingSegment{Text: value}
}

func linked(
	value string,
	filter quickfilter.QuickFilter,
) shipmentbriefports.ShipmentBriefingSegment {
	return shipmentbriefports.ShipmentBriefingSegment{Text: value, Filter: &filter}
}

func loads(count int) string {
	return inflect.CountNoun(count, "load", "loads")
}

func joinClauses(
	clauses [][]shipmentbriefports.ShipmentBriefingSegment,
) []shipmentbriefports.ShipmentBriefingSegment {
	segments := make([]shipmentbriefports.ShipmentBriefingSegment, 0, len(clauses)*2+1)
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

func Deterministic(f Facts) []shipmentbriefports.ShipmentBriefingSegment {
	if f.DeliveringToday == 0 && f.Moving == 0 && f.Late == 0 && f.Uncovered == 0 &&
		f.Detention == 0 && f.ReadyToBill == 0 {
		return []shipmentbriefports.ShipmentBriefingSegment{
			text("Nothing on the board needs you right now."),
		}
	}

	board := make([][]shipmentbriefports.ShipmentBriefingSegment, 0, 3)
	if f.DeliveringToday > 0 {
		board = append(board, []shipmentbriefports.ShipmentBriefingSegment{linked(
			loads(f.DeliveringToday)+" "+
				inflect.Pluralize("delivers", "deliver", f.DeliveringToday)+" today",
			quickfilter.QuickFilterDeliveringToday,
		)})
	}
	if f.Moving > 0 {
		board = append(board, []shipmentbriefports.ShipmentBriefingSegment{linked(
			loads(f.Moving)+" "+inflect.Pluralize("is", "are", f.Moving)+" moving on schedule",
			quickfilter.QuickFilterMoving,
		)})
	}
	if f.Late > 0 {
		clause := []shipmentbriefports.ShipmentBriefingSegment{linked(
			loads(f.Late)+" "+inflect.Pluralize("is", "are", f.Late)+" late",
			quickfilter.QuickFilterLate,
		)}
		if reason := strings.TrimSpace(f.LateReason); reason != "" {
			clause = append(clause, text(", mostly "+strings.ToLower(reason)))
		}
		board = append(board, clause)
	}

	followUp := make([][]shipmentbriefports.ShipmentBriefingSegment, 0, 3)
	if f.Uncovered > 0 {
		followUp = append(followUp, []shipmentbriefports.ShipmentBriefingSegment{linked(
			loads(f.Uncovered)+" still "+inflect.Pluralize("needs", "need", f.Uncovered)+" "+
				f.OperationType.CoverageNoun(),
			quickfilter.QuickFilterUncovered,
		)})
	}
	if f.Detention > 0 {
		followUp = append(followUp, []shipmentbriefports.ShipmentBriefingSegment{linked(
			loads(f.Detention)+" "+inflect.Pluralize("is", "are", f.Detention)+
				" accruing detention",
			quickfilter.QuickFilterDetention,
		)})
	}
	if f.ReadyToBill > 0 {
		followUp = append(followUp, []shipmentbriefports.ShipmentBriefingSegment{linked(
			loads(f.ReadyToBill)+" "+inflect.Pluralize("is", "are", f.ReadyToBill)+
				" ready to bill",
			quickfilter.QuickFilterReadyToBill,
		)})
	}

	segments := append(joinClauses(board), joinClauses(followUp)...)
	if last := len(segments) - 1; last >= 0 && segments[last].Text == ". " {
		segments[last].Text = "."
	}

	return segments
}
