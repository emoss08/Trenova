package etaprojection

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

type reasonInput struct {
	row     *stopeta.StopETA
	origin  origin
	cursor  *cursor
	now     int64
	stop    *shipment.Stop
	delayed bool
}

func describe(in *reasonInput) string {
	parts := make([]string, 0, 4)
	if text := basisText(in); text != "" {
		parts = append(parts, text)
	}
	if in.row.EstimatedArrival != nil && in.row.DriveMethod != stopeta.DriveMethodNone &&
		in.row.DriveMethod != stopeta.DriveMethodReported {
		parts = append(parts, driveText(in.cursor))
	}
	if text := restText(in.cursor); text != "" {
		parts = append(parts, text)
	}
	if text := hoursText(in.cursor.hosNote); text != "" {
		parts = append(parts, text)
	}
	if in.cursor.clamped && in.origin.basis == stopeta.BasisLastDeparture {
		parts = append(parts, "The truck should have arrived by now going by its last departure, "+
			"so the arrival is shown as now.")
	}
	if text := lateText(in); text != "" {
		parts = append(parts, text)
	}
	if in.delayed {
		parts = append(parts, "The shipment is marked delayed.")
	}

	return stringutils.TruncateRunes(strings.Join(parts, " "), stopeta.MaxReasonLength)
}

func basisText(in *reasonInput) string {
	age := timeutils.FormatLongDurationMs((in.now - in.origin.inputAt) * msPerSecond)
	switch in.origin.basis {
	case stopeta.BasisAtStop:
		return "The truck is at the stop."
	case stopeta.BasisPosition:
		if in.origin.stale {
			return fmt.Sprintf("From the truck's last position, which is %s old.", age)
		}
		return fmt.Sprintf("From the truck's position %s ago.", age)
	case stopeta.BasisCarrierReported:
		return fmt.Sprintf("The carrier gave this arrival on a check call %s ago.", age)
	case stopeta.BasisCheckCall:
		return fmt.Sprintf("From the position on a check call %s ago.", age)
	case stopeta.BasisLastDeparture:
		return fmt.Sprintf("From the last departure %s ago; there is no position since.", age)
	case stopeta.BasisNone:
		return "There is no position, check call or departure to estimate from."
	default:
		return ""
	}
}

func driveText(c *cursor) string {
	method := "planning speed"
	switch {
	case c.routed && c.planned:
		method = "partly routed, partly at planning speed"
	case c.routed:
		method = "routed"
	}

	return fmt.Sprintf("%.0f miles, %s of driving (%s).",
		c.miles, timeutils.FormatLongDurationMs(c.driveMs), method)
}

func restText(c *cursor) string {
	rest := c.rest
	if rest.RestMs <= 0 {
		return ""
	}
	pieces := make([]string, 0, 3)
	if rest.Breaks > 0 {
		pieces = append(pieces, plural(rest.Breaks, "30 minute break", "30 minute breaks"))
	}
	if rest.Resets > 0 {
		pieces = append(pieces, plural(rest.Resets, "10 hour reset", "10 hour resets"))
	}
	if rest.Restarts > 0 {
		pieces = append(pieces, plural(rest.Restarts, "34 hour restart", "34 hour restarts"))
	}

	return fmt.Sprintf("Includes %s for hours of service.", strings.Join(pieces, ", "))
}

func hoursText(note hosNote) string {
	switch note {
	case hosNoteMissing:
		return "The driver's hours are not on file, so rest is not counted."
	case hosNoteStale:
		return "The driver's hours are more than 12 hours old, so rest is not counted."
	case hosNoteExempt:
		return "The driver is exempt from logging hours, so rest is not counted."
	case hosNoteHandover:
		return "A different driver takes a later move, so rest is not counted past the handover."
	case hosNoteNone, hosNoteApplied, hosNoteCarrier:
		return ""
	default:
		return ""
	}
}

func lateText(in *reasonInput) string {
	cutoff := in.stop.EffectiveScheduledCutoff()
	if in.stop.ActualArrival != nil || cutoff <= 0 || in.now <= cutoff {
		return ""
	}

	return fmt.Sprintf("The window closed %s ago with no arrival recorded.",
		timeutils.FormatLongDurationMs((in.now-cutoff)*msPerSecond))
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "one " + one
	}

	return fmt.Sprintf("%d %s", n, many)
}
