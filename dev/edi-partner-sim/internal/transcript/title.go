package transcript

import (
	"fmt"
	"math"
	"strings"

	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
)

var tenderPurposes = map[string]string{
	"00": "Original tender",
	"01": "Cancellation",
	"04": "Change",
	"05": "Replacement",
}

var answerCodes = map[string]string{
	"A": "Accept",
	"D": "Decline",
}

var statusWords = map[string]string{
	"AF": "departed pickup",
	"X3": "arrived pickup",
	"X1": "arrived delivery",
	"CD": "departed delivery",
	"D1": "delivered",
	"X6": "en route",
	"AG": "estimated delivery",
	"SD": "shipment delayed",
	"AA": "pickup appointment",
	"AB": "delivery appointment",
	"A3": "returned",
	"A7": "refused",
	"P1": "departed terminal",
	"X4": "arrived terminal",
	"J1": "delivered to connecting line",
	"S1": "trailer spotted",
	"OA": "out for delivery",
}

var ackWords = map[string]string{
	"A": "Accepted",
	"E": "Accepted with errors",
	"P": "Partly accepted",
	"X": "Rejected",
	"M": "Rejected",
	"W": "Rejected",
	"R": "Rejected",
}

func title(rep *oracle.Report, d *journal.Document, o *Options) string {
	switch rep.Set {
	case "204":
		return tenderTitle(rep, d, o)
	case "990":
		return answerTitle(rep)
	case "214":
		return statusTitle(rep)
	case "210":
		return invoiceTitle(rep)
	case "997", "999":
		return ackTitle(d)
	}
	return "Transaction " + rep.Set
}

func tenderTitle(rep *oracle.Report, d *journal.Document, o *Options) string {
	purpose := strings.TrimSpace(first(rep.Segments, "B2A").Element(1))
	text, ok := tenderPurposes[purpose]
	if !ok {
		text = "Tender"
		if purpose != "" {
			text += " " + purpose
		}
	}
	switch {
	case o.Resend:
		return "Resent: " + strings.ToLower(text[:1]) + text[1:]
	case o.Variation == "duplicate":
		return "Duplicate: " + strings.ToLower(text[:1]) + text[1:]
	}
	if (purpose != "04" && purpose != "05") || o.Earlier == nil || len(d.Keys) == 0 ||
		d.ChargeCents == 0 {
		return text
	}
	earlier, found := o.Earlier(d.Keys[0])
	if !found || earlier == 0 || earlier == d.ChargeCents {
		return text
	}
	percent := math.Round(float64(d.ChargeCents-earlier) * 100 / float64(earlier))
	if percent == 0 {
		return fmt.Sprintf("%s: rate changed", text)
	}
	return fmt.Sprintf("%s: rate %+.0f%%", text, percent)
}

func answerTitle(rep *oracle.Report) string {
	code := strings.ToUpper(strings.TrimSpace(first(rep.Segments, "B1").Element(4)))
	if text, ok := answerCodes[code]; ok {
		return text
	}
	if code == "" {
		return "Answer"
	}
	return "Answer " + code
}

func statusTitle(rep *oracle.Report) string {
	var parts []string
	for index := range rep.Segments {
		segment := &rep.Segments[index]
		if segment.ID != "AT7" {
			continue
		}
		code := strings.ToUpper(strings.TrimSpace(segment.Element(1)))
		if code == "" {
			code = strings.ToUpper(strings.TrimSpace(segment.Element(3)))
		}
		if code == "" {
			continue
		}
		if words, ok := statusWords[code]; ok && len(parts) == 0 {
			parts = append(parts, code+" "+words)
			continue
		}
		parts = append(parts, code)
	}
	if len(parts) == 0 {
		return "Status"
	}
	return strings.Join(parts, ", ")
}

func invoiceTitle(rep *oracle.Report) string {
	number := strings.TrimSpace(first(rep.Segments, "B3").Element(2))
	if number == "" {
		return "Invoice"
	}
	return "Invoice " + number
}

func ackTitle(d *journal.Document) string {
	word, ok := ackWords[d.AckCode]
	if !ok {
		word = "Acknowledged"
	}
	if len(d.Acknowledges) == 0 {
		return word
	}
	acked := d.Acknowledges[0]
	what := acked.Set
	if len(d.Acknowledges) > 1 {
		what = fmt.Sprintf("%d × %s", len(d.Acknowledges), acked.Set)
	}
	if what == "" {
		what = acked.GroupID
	}
	return strings.TrimSpace(fmt.Sprintf("%s %s %s", word, what, trimControl(acked.GroupControl)))
}

func interchangeWord(code string) string {
	switch strings.ToUpper(code) {
	case "A":
		return "Interchange accepted"
	case "E":
		return "Interchange accepted with errors"
	case "R":
		return "Interchange rejected"
	}
	return "Interchange answered"
}
