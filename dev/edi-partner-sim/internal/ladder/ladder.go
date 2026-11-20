package ladder

import (
	"cmp"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/runlog"
)

const (
	KindMessage = "message"
	KindEvent   = "event"
	KindGap     = "gap"
)

const (
	FromPartner = "partner"
	FromTrenova = "trenova"
)

const (
	ToneOK   = "ok"
	ToneBad  = "bad"
	ToneHold = "hold"
)

const (
	VerdictPasses    = "passes"
	VerdictBreaks    = "breaks"
	VerdictRejects   = "rejects"
	VerdictRead      = "read"
	VerdictRefused   = "refused"
	VerdictAwaiting  = "awaiting"
	VerdictDelivered = "delivered"
	VerdictScheduled = "scheduled"
	VerdictFailed    = "failed"
	VerdictPending   = "pending"
)

var rejectingCodes = []string{"R", "P", "X", "M", "W"}

type Row struct {
	Kind      string    `json:"kind"`
	At        time.Time `json:"at"`
	Number    uint64    `json:"number,omitempty"`
	MessageID string    `json:"messageId,omitempty"`
	Document  int       `json:"document"`
	From      string    `json:"from,omitempty"`
	Set       string    `json:"set,omitempty"`
	Title     string    `json:"title,omitempty"`
	Control   string    `json:"control,omitempty"`
	Ack       bool      `json:"ack,omitempty"`
	Verdict   string    `json:"verdict,omitempty"`
	Tone      string    `json:"tone,omitempty"`
	Note      string    `json:"note,omitempty"`
	Label     string    `json:"label,omitempty"`
	Detail    string    `json:"detail,omitempty"`
	Who       string    `json:"who,omitempty"`
	Expect    string    `json:"expect,omitempty"`
	Gap       string    `json:"gap,omitempty"`
}

func Compose(key string, messages []journal.Message, scenario *runlog.Scenario) []Row {
	var rows []Row
	index := map[uint64]int{}
	for mi := range messages {
		m := &messages[mi]
		for di := range m.Documents {
			d := &m.Documents[di]
			if !slices.Contains(d.Keys, key) {
				continue
			}
			index[d.Number] = len(rows)
			rows = append(rows, documentRow(m, di))
		}
	}
	if scenario != nil {
		rows = applyScenario(rows, index, key, messages, scenario)
	}
	slices.SortStableFunc(rows, func(a, b Row) int {
		return cmp.Or(a.At.Compare(b.At), cmp.Compare(order(a.Kind), order(b.Kind)))
	})
	return attachGaps(rows)
}

func order(kind string) int {
	switch kind {
	case KindMessage:
		return 0
	case KindEvent:
		return 1
	}
	return 2
}

func documentRow(m *journal.Message, at int) Row {
	d := &m.Documents[at]
	row := Row{
		Kind:      KindMessage,
		At:        m.WireAt(),
		Number:    d.Number,
		MessageID: m.ID,
		Document:  at,
		From:      FromPartner,
		Set:       d.Set,
		Title:     d.Title,
		Control:   d.Interchange,
		Ack:       isAck(d.Set),
	}
	if m.Direction == journal.FromTrenova {
		row.From = FromTrenova
	}
	row.Verdict, row.Tone, row.Note = Verdict(m, d)
	return row
}

func isAck(set string) bool {
	return set == "997" || set == "999" || set == journal.SetTA1
}

func Verdict(m *journal.Message, d *journal.Document) (string, string, string) {
	switch m.State {
	case journal.StateFailed:
		return VerdictFailed, ToneBad, "not delivered"
	case journal.StateScheduled:
		return VerdictScheduled, ToneHold, ""
	case journal.StateCancelled:
		return VerdictFailed, ToneBad, "cancelled"
	}
	if d.Judged && d.Accepted != nil && !*d.Accepted {
		return VerdictBreaks, ToneBad, fmt.Sprintf("oracle: %s", errorCount(d.Findings))
	}
	if isAck(d.Set) && slices.Contains(rejectingCodes, d.AckCode) {
		return VerdictRejects, ToneBad, ackElement(d.Set) + " " + d.AckCode
	}
	if m.Direction == journal.FromTrenova {
		if d.Judged {
			return VerdictPasses, ToneOK, ""
		}
		return VerdictPending, "", ""
	}
	if d.AnsweredBy != nil {
		if slices.Contains(rejectingCodes, d.AnsweredBy.Code) {
			return VerdictRefused, ToneBad, ackElement(d.AnsweredBy.Set) + " " +
				d.AnsweredBy.Code
		}
		return VerdictRead, ToneOK, ""
	}
	if !isAck(d.Set) && m.Ack != nil && m.Ack.Status == journal.AckAwaiting {
		return VerdictAwaiting, ToneHold, ""
	}
	return VerdictDelivered, ToneOK, ""
}

func ackElement(set string) string {
	switch set {
	case "999":
		return "IK5"
	case journal.SetTA1:
		return "TA104"
	}
	return "AK5"
}

func errorCount(notes []journal.Note) string {
	count := 0
	for _, note := range notes {
		if note.Severity == "error" {
			count++
		}
	}
	if count == 1 {
		return "1 error"
	}
	return fmt.Sprintf("%d errors", count)
}

func applyScenario(
	rows []Row,
	index map[uint64]int,
	key string,
	messages []journal.Message,
	scenario *runlog.Scenario,
) []Row {
	byID := map[string]*journal.Message{}
	for mi := range messages {
		byID[messages[mi].ID] = &messages[mi]
	}
	for _, step := range scenario.Steps {
		mark := markOf(step.Status)
		if mark == "" {
			continue
		}
		for _, match := range step.Matches {
			if match.Key != "" && match.Key != key {
				continue
			}
			m, ok := byID[match.MessageID]
			if !ok {
				continue
			}
			for di := range m.Documents {
				d := &m.Documents[di]
				if !matches(&match, d, key) {
					continue
				}
				if at, found := index[d.Number]; found {
					rows[at].Expect = worse(rows[at].Expect, mark)
				}
			}
		}
		expectation := step.Kind == "expect"
		for _, event := range step.Events {
			if event.Key != "" && event.Key != key {
				continue
			}
			row := Row{
				Kind:   KindEvent,
				At:     event.At,
				Label:  event.Label,
				Detail: event.Detail,
				Tone:   cmp.Or(event.Tone, runlog.ToneTrenova),
				Who:    event.Who,
			}
			if expectation {
				row.Expect = mark
			}
			rows = append(rows, row)
		}
		if step.Gap != "" && (len(step.Keys) == 0 || slices.Contains(step.Keys, key)) {
			rows = append(rows, Row{Kind: KindGap, At: step.Ended, Gap: step.Gap})
		}
	}
	return rows
}

func matches(match *runlog.Match, d *journal.Document, key string) bool {
	if match.Control != "" {
		return journal.SameControl(match.Control, d.Control) &&
			(match.GroupControl == "" || journal.SameControl(match.GroupControl, d.GroupControl))
	}
	return slices.Contains(d.Keys, key)
}

func markOf(status string) string {
	switch status {
	case runlog.StatusPassed:
		return ToneOK
	case runlog.StatusFailed:
		return ToneBad
	}
	return ""
}

func worse(a, b string) string {
	if a == ToneBad || b == ToneBad {
		return ToneBad
	}
	return b
}

func attachGaps(rows []Row) []Row {
	out := make([]Row, 0, len(rows))
	pending := ""
	for _, row := range rows {
		if row.Kind == KindGap {
			if pending != "" {
				pending += ", "
			}
			pending += row.Gap
			continue
		}
		if pending != "" {
			row.Gap, pending = pending, ""
		}
		out = append(out, row)
	}
	if pending != "" {
		out = append(out, Row{Kind: KindGap, Gap: pending})
	}
	return out
}
