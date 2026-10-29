package detentionsync

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/operations/app/detentionengine"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionnoticerender"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
)

type chargeCommentKind string

const (
	chargeCommentAdded   = chargeCommentKind("Added")
	chargeCommentRevised = chargeCommentKind("Revised")
	chargeCommentRemoved = chargeCommentKind("Removed")
)

type chargeCommentParams struct {
	occurrence *detention.DetentionOccurrence
	previous   *detention.DetentionOccurrence
	stop       *shipment.Stop
}

// chargeCommentKindFor reports what the shipment thread has not been told yet.
// Only a crossing of the billable boundary or a movement in the money is worth
// a comment; a recalculation that lands on the same number is noise.
func chargeCommentKindFor(
	occurrence, previous *detention.DetentionOccurrence,
) (chargeCommentKind, bool) {
	current := occurrenceCarriesCharge(occurrence)
	prior := occurrenceCarriesCharge(previous)

	switch {
	case current && !prior:
		return chargeCommentAdded, true
	case !current && prior:
		return chargeCommentRemoved, true
	case current && prior && !occurrence.BillableAmount.Equal(previous.BillableAmount):
		return chargeCommentRevised, true
	default:
		return "", false
	}
}

func occurrenceCarriesCharge(occurrence *detention.DetentionOccurrence) bool {
	return occurrence != nil &&
		occurrence.Status.IsBillable() &&
		occurrence.BillableAmount.IsPositive()
}

func chargeCommentPriority(
	kind chargeCommentKind,
	occurrence *detention.DetentionOccurrence,
) shipment.CommentPriority {
	if kind != chargeCommentRemoved &&
		(occurrence.SuppressedByGate || occurrence.RequiresApproval) {
		return shipment.CommentPriorityHigh
	}

	return shipment.CommentPriorityNormal
}

func buildChargeComment(kind chargeCommentKind, p chargeCommentParams) string {
	occ := p.occurrence

	var b strings.Builder

	b.WriteString(chargeCommentHeadline(kind, p))
	b.WriteString("\n\n")
	b.WriteString(stopLabel(occ, p.stop))
	b.WriteString("\n")

	if kind == chargeCommentRemoved {
		b.WriteString("Recalculated from the recorded arrival and departure times; ")
		b.WriteString("this shipment no longer carries a detention charge.")
		return b.String()
	}

	fmt.Fprintf(&b, "%s on site · %s free · %s billable\n",
		detentionengine.FormatMinutes(occ.RawDwellMinutes),
		detentionengine.FormatMinutes(occ.FreeMinutesGranted),
		detentionengine.FormatMinutes(occ.RoundedMinutes))

	if snap := occ.PolicySnapshot; snap != nil {
		fmt.Fprintf(&b, "Rated at %s under %s (%s)\n",
			detentionnoticerender.NoticeRateLabel(snap), snap.PolicyName, snap.PolicyCode)
	}

	for _, note := range chargeCommentNotes(occ) {
		b.WriteString(note)
		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

func chargeCommentHeadline(kind chargeCommentKind, p chargeCommentParams) string {
	occ := p.occurrence
	amount := occ.BillableAmount.StringFixed(2) + " " + occ.Currency

	switch kind {
	case chargeCommentRevised:
		return fmt.Sprintf("Detention charge revised from %s to %s",
			p.previous.BillableAmount.StringFixed(2)+" "+p.previous.Currency, amount)
	case chargeCommentRemoved:
		return fmt.Sprintf("Detention charge removed (was %s)",
			p.previous.BillableAmount.StringFixed(2)+" "+p.previous.Currency)
	case chargeCommentAdded:
		return "Detention charge added — " + amount
	default:
		return "Detention charge updated — " + amount
	}
}

// chargeCommentNotes surfaces only the facts that change what someone should do
// next: money left on the table, a charge that cannot be billed yet, or one a
// customer has grounds to refuse.
func chargeCommentNotes(occ *detention.DetentionOccurrence) []string {
	notes := make([]string, 0, 4)

	if occ.ArrivedLate {
		notes = append(notes, fmt.Sprintf("Driver arrived %s late",
			detentionengine.FormatMinutes(occ.LateByMinutes)))
	}

	if occ.CapApplied != detention.CapKindNone {
		notes = append(notes, "Capped by policy: "+capKindLabel(occ.CapApplied))
	}

	if occ.ConvertedToLayover {
		notes = append(notes, "Dwell crossed the layover boundary and was converted")
	}

	if occ.SuppressedByGate {
		notes = append(notes,
			"No qualifying notice was sent in time — the customer can refuse this charge")
	}

	if occ.RequiresApproval {
		notes = append(notes, "Held for approval before it can be billed")
	}

	return notes
}

func capKindLabel(kind detention.CapKind) string {
	switch kind {
	case detention.CapKindMaxMinutes:
		return "billable minutes ceiling"
	case detention.CapKindPerStopAmount:
		return "per-stop ceiling"
	case detention.CapKindPerDayAmount:
		return "per-day ceiling"
	case detention.CapKindPerShipment:
		return "per-shipment ceiling"
	case detention.CapKindLayoverBoundary:
		return "layover boundary"
	case detention.CapKindNone:
		return "none"
	default:
		return string(kind)
	}
}

// stopLabel names the stop the way someone reading the shipment would, falling
// back through the detail that is actually loaded on the record.
func stopLabel(occ *detention.DetentionOccurrence, stop *shipment.Stop) string {
	label := string(occ.StopType)

	switch {
	case stop != nil && stop.Location != nil && stop.Location.Name != "":
		return label + " at " + stop.Location.Name
	case occ.LocationName != "":
		return label + " at " + occ.LocationName
	case stop != nil && stop.AddressLine != "":
		return label + " at " + stop.AddressLine
	default:
		return label + " stop"
	}
}
