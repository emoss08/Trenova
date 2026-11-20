package play

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
	"github.com/emoss08/trenova/edi-partner-sim/internal/transcript"
)

var reader = new(oracle.GuideSet)

type transaction struct {
	MessageID string
	Set       string
	Group     string
	Control   string
	Key       string
	At        int64
	Accepted  bool
	Findings  []oracle.Finding
	Segments  []oracle.Segment
	body      string
}

func (t *transaction) id() string {
	return t.MessageID + "/" + t.Group + "/" + t.Control
}

func (t *transaction) element(spec string) (string, bool) {
	segmentID, position, component, ok := parseElement(spec)
	if !ok {
		return "", false
	}
	for index := range t.Segments {
		segment := &t.Segments[index]
		if segment.ID != segmentID {
			continue
		}
		if component == 0 {
			return segment.Element(position), true
		}
		components := segment.Components(position)
		if component > len(components) {
			return "", true
		}
		return components[component-1], true
	}
	return "", false
}

func (t *transaction) statusCodes() []string {
	var codes []string
	for index := range t.Segments {
		if t.Segments[index].ID == "AT7" {
			if code := strings.TrimSpace(t.Segments[index].Element(1)); code != "" {
				codes = append(codes, code)
			}
		}
	}
	return codes
}

func (t *transaction) amountCents(spec string) (int64, error) {
	value, found := t.element(spec)
	if !found || strings.TrimSpace(value) == "" {
		return 0, fmt.Errorf("the %s has no %s", t.Set, spec)
	}
	value = strings.TrimSpace(value)
	if strings.Contains(value, ".") {
		whole, fraction, _ := strings.Cut(value, ".")
		fraction = (fraction + "00")[:2]
		value = whole + fraction
	}
	cents, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s %q is not an amount", spec, value)
	}
	return cents, nil
}

func parseElement(spec string) (string, int, int, bool) {
	parts := strings.Split(spec, "-")
	if len(parts) < 2 || len(parts) > 3 {
		return "", 0, 0, false
	}
	position, err := strconv.Atoi(parts[1])
	if err != nil || position < 1 {
		return "", 0, 0, false
	}
	component := 0
	if len(parts) == 3 {
		if component, err = strconv.Atoi(parts[2]); err != nil || component < 1 {
			return "", 0, 0, false
		}
	}
	return parts[0], position, component, true
}

func transactionsOf(m *journal.Message, key string) []transaction {
	read, err := reader.Judge([]byte(m.Raw))
	if err != nil {
		return nil
	}
	judged := map[string]*oracle.Report{}
	envelopeErrors := map[string]bool{}
	if m.Judgement != nil {
		for _, ic := range m.Judgement.Interchanges {
			for _, g := range ic.Groups {
				broken := hasErrors(ic.Findings) || hasErrors(g.Findings)
				for _, rep := range g.Reports {
					judged[g.ControlNumber+"/"+rep.ControlNumber] = rep
					envelopeErrors[g.ControlNumber+"/"+rep.ControlNumber] = broken
				}
			}
		}
	}
	ack := m.Purpose == journal.PurposeAck
	var out []transaction
	for _, ic := range read.Interchanges {
		for _, g := range ic.Groups {
			for _, rep := range g.Reports {
				repKey := transcript.KeyOf(rep)
				if !ack && key != "" && repKey != key {
					continue
				}
				id := g.ControlNumber + "/" + rep.ControlNumber
				t := transaction{
					MessageID: m.ID,
					Set:       rep.Set,
					Group:     g.ControlNumber,
					Control:   rep.ControlNumber,
					Key:       repKey,
					At:        m.At.UnixMilli(),
					Segments:  rep.Segments,
					body:      bodyOf(rep.Segments),
				}
				if judgement, ok := judged[id]; ok {
					t.Findings = judgement.Findings
					t.Accepted = judgement.Accepted() && !envelopeErrors[id]
				} else if m.Accepted != nil {
					t.Accepted = *m.Accepted
				}
				out = append(out, t)
			}
		}
	}
	return out
}

func bodyOf(segments []oracle.Segment) string {
	parts := make([]string, 0, len(segments))
	for index := range segments {
		if segments[index].ID == "ST" || segments[index].ID == "SE" {
			continue
		}
		parts = append(parts, segments[index].Text)
	}
	return strings.Join(parts, "~")
}

func hasErrors(findings []oracle.Finding) bool {
	return slices.ContainsFunc(findings, func(f oracle.Finding) bool {
		return f.Severity == oracle.SeverityError
	})
}

func fromTrenova(set string) func(*journal.Message) bool {
	return func(m *journal.Message) bool {
		return m.Direction == journal.FromTrenova && slices.Contains(m.Sets, set)
	}
}

func received(messages []journal.Message, set, key string) []transaction {
	var out []transaction
	for index := range messages {
		m := &messages[index]
		if !fromTrenova(set)(m) {
			continue
		}
		for _, t := range transactionsOf(m, key) {
			if t.Set == set {
				out = append(out, t)
			}
		}
	}
	return out
}

func describeFindings(findings []oracle.Finding) string {
	var parts []string
	for index := range findings {
		f := &findings[index]
		if f.Severity != oracle.SeverityError {
			continue
		}
		where := f.SegmentID
		if f.ElementPos > 0 {
			where = fmt.Sprintf("%s-%02d", f.SegmentID, f.ElementPos)
		}
		text := f.Message
		if f.Rule != "" {
			text = f.Rule + ": " + text
		}
		if where != "" {
			text = where + " " + text
		}
		parts = append(parts, text)
		if len(parts) == 3 {
			break
		}
	}
	return strings.Join(parts, "; ")
}

func BodyOf(m *journal.Message, set, key string) string {
	var parts []string
	for _, t := range transactionsOf(m, key) {
		if t.Set == set {
			parts = append(parts, t.body)
		}
	}
	return strings.Join(parts, "\n")
}
