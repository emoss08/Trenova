package shipmentbriefingservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/stringutils"
)

type Facts struct {
	DeliveringToday int
	Moving          int
	Late            int
	Uncovered       int
	LateReason      string
	OperationType   tenant.OperationType
}

func (f Facts) Counts() []int {
	return []int{f.DeliveringToday, f.Moving, f.Late, f.Uncovered}
}

func (f Facts) Filters() []shipment.QuickFilter {
	out := make([]shipment.QuickFilter, 0, 4)
	if f.DeliveringToday > 0 {
		out = append(out, shipment.QuickFilterDeliveringToday)
	}
	if f.Moving > 0 {
		out = append(out, shipment.QuickFilterMoving)
	}
	if f.Late > 0 {
		out = append(out, shipment.QuickFilterLate)
	}
	if f.Uncovered > 0 {
		out = append(out, shipment.QuickFilterUncovered)
	}
	return out
}

func text(value string) services.ShipmentBriefingSegment {
	return services.ShipmentBriefingSegment{Text: value}
}

func linked(value string, filter shipment.QuickFilter) services.ShipmentBriefingSegment {
	return services.ShipmentBriefingSegment{Text: value, Filter: &filter}
}

func Deterministic(f Facts) []services.ShipmentBriefingSegment {
	if f.DeliveringToday == 0 && f.Moving == 0 && f.Late == 0 && f.Uncovered == 0 {
		return []services.ShipmentBriefingSegment{text("Nothing on the board needs you right now.")}
	}

	clauses := make([][]services.ShipmentBriefingSegment, 0, 3)
	if f.DeliveringToday > 0 {
		clauses = append(clauses, []services.ShipmentBriefingSegment{linked(
			stringutils.CountNoun(f.DeliveringToday, "load", "loads")+" "+
				stringutils.Pluralize("delivers", "deliver", f.DeliveringToday)+" today",
			shipment.QuickFilterDeliveringToday,
		)})
	}
	if f.Moving > 0 {
		clauses = append(clauses, []services.ShipmentBriefingSegment{linked(
			stringutils.CountNoun(f.Moving, "load", "loads")+" "+
				stringutils.Pluralize("is", "are", f.Moving)+" moving on schedule",
			shipment.QuickFilterMoving,
		)})
	}
	if f.Late > 0 {
		clause := []services.ShipmentBriefingSegment{linked(
			stringutils.CountNoun(f.Late, "load", "loads")+" "+
				stringutils.Pluralize("is", "are", f.Late)+" late",
			shipment.QuickFilterLate,
		)}
		if reason := strings.TrimSpace(f.LateReason); reason != "" {
			clause = append(clause, text(", mostly "+strings.ToLower(reason)))
		}
		clauses = append(clauses, clause)
	}

	segments := make([]services.ShipmentBriefingSegment, 0, 8)
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

	if f.Uncovered > 0 {
		segments = append(segments,
			linked(
				stringutils.CountNoun(f.Uncovered, "load", "loads")+" still "+
					stringutils.Pluralize("needs", "need", f.Uncovered)+" "+
					f.OperationType.CoverageNoun(),
				shipment.QuickFilterUncovered,
			),
			text("."),
		)
	}

	if last := len(segments) - 1; last >= 0 && segments[last].Text == ". " {
		segments[last].Text = "."
	}

	return segments
}
