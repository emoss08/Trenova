package scenario

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"
)

func (s *Step) Describe(ackSet string) string {
	switch s.Kind() {
	case KindSend:
		return describeSend(s.Send)
	case KindExpect:
		return describeExpect(s.Expect, ackSet)
	case KindTrenova:
		verb, about := s.Trenova.Verb()
		text := "trenova " + verb + " " + about
		switch verb {
		case "decide":
			text += " " + s.Trenova.Decision
		case "arrive", "depart":
			text += fmt.Sprintf(" stop %d", s.Trenova.Stop)
		case "tender":
			if s.Trenova.Rate != "" {
				text += " at " + s.Trenova.Rate
			}
		}
		return text
	case KindAdvance:
		return fmt.Sprintf("advance %s by %s", s.Advance.About, ShortDuration(s.Advance.By))
	case KindScript:
		return fmt.Sprintf("script %d answers", len(s.Script.Answers))
	case KindAcks:
		return fmt.Sprintf("acks %s %s %s", s.Acks.Action, s.Acks.Set, s.Acks.About)
	}
	return ""
}

func describeSend(s *Send) string {
	text := "send " + s.Set + " " + cmp.Or(s.Variation, "original")
	if s.As != "" {
		text += " as " + s.As
	}
	if s.Of != "" {
		text += " of " + s.Of
	}
	if s.Count > 1 {
		text += fmt.Sprintf(" ×%d", s.Count)
	}
	var deltas []string
	if s.Delta.RatePercent != 0 {
		deltas = append(deltas, fmt.Sprintf("total %+g%%", s.Delta.RatePercent))
	}
	if s.Delta.AddStop {
		deltas = append(deltas, "a stop added")
	}
	if s.Delta.Move != nil {
		deltas = append(deltas, fmt.Sprintf("stop %d moved %s", s.Delta.Move.Stop,
			ShortDuration(s.Delta.Move.By)))
	}
	if s.Load != nil {
		deltas = append(deltas, "a load of "+s.Load.String())
	}
	if len(deltas) > 0 {
		text += ", " + strings.Join(deltas, ", ")
	}
	return text
}

func describeExpect(e *Expect, ackSet string) string {
	kind, about := e.Subject()
	var text string
	switch kind {
	case "wire":
		text = "expect wire " + e.Wire + " about " + about
	case "ack":
		text = "expect " + ackSet + " for " + about
		if e.Set != "" {
			text += "'s " + e.Set
		}
	case "flow":
		text = "expect flow " + about
		if e.Status != "" {
			text += " " + strings.ToLower(e.Status)
		}
		if e.Why != "" {
			text += fmt.Sprintf(" %q", e.Why)
		}
	case "reconciled":
		text = "expect every document about " + about + " reconciled"
	case "trenova":
		text = describeState(e.Trenova)
	default:
		text = "expect " + kind + " about " + about
	}
	var details []string
	if e.Within > 0 {
		details = append(details, "within "+ShortDuration(e.Within))
	}
	if e.None {
		details = append(details, "none")
	}
	if e.Resent {
		details = append(details, "resent")
	}
	if e.Count > 1 {
		details = append(details, fmt.Sprintf("%d of them", e.Count))
	}
	if len(e.Statuses) > 0 {
		details = append(details, "statuses ["+strings.Join(e.Statuses, ", ")+"]")
	}
	for _, spec := range slices.Sorted(maps.Keys(e.Elements)) {
		details = append(details, spec+": "+e.Elements[spec])
	}
	if e.Total != "" {
		details = append(details, "total "+e.Total)
	}
	if e.Accepted != nil {
		switch {
		case kind == "wire" && *e.Accepted:
			details = append(details, "oracle accepted")
		case kind == "wire":
			details = append(details, "oracle rejects it")
		default:
			details = append(details, acceptedWord(*e.Accepted))
		}
	}
	if len(details) > 0 {
		text += ", " + strings.Join(details, ", ")
	}
	return text
}

func describeState(state *State) string {
	if state.Exists != nil && !*state.Exists {
		return "expect no shipment for " + state.Shipment
	}
	text := "expect shipment " + state.Shipment
	if state.Status != "" {
		text += " " + strings.ToLower(state.Status)
	}
	if state.Stops > 0 {
		text += fmt.Sprintf(", %d stops", state.Stops)
	}
	if state.Total != "" {
		text += ", total " + state.Total
	}
	return text
}

func ShortDuration(d time.Duration) string {
	text := d.String()
	if strings.HasSuffix(text, "m0s") {
		text = text[:len(text)-2]
	}
	if strings.HasSuffix(text, "h0m") {
		text = text[:len(text)-2]
	}
	return text
}

func acceptedWord(accepted bool) string {
	if accepted {
		return "accepted"
	}
	return "rejected"
}
