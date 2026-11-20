package journal

import (
	"slices"
	"strings"
	"time"
)

const SetTA1 = "TA1"

type Document struct {
	Number       uint64     `json:"number"`
	Set          string     `json:"set"`
	Control      string     `json:"control"`
	GroupID      string     `json:"groupId,omitempty"`
	GroupControl string     `json:"groupControl,omitempty"`
	Interchange  string     `json:"interchange"`
	Keys         []string   `json:"keys,omitempty"`
	Title        string     `json:"title"`
	ChargeCents  int64      `json:"chargeCents,omitempty"`
	Separator    string     `json:"separator"`
	Terminator   string     `json:"terminator"`
	Header       [][]string `json:"header"`
	Segments     [][]string `json:"segments"`
	Trailer      [][]string `json:"trailer"`
	Findings     []Note     `json:"findings,omitempty"`
	Guide        string     `json:"guide,omitempty"`
	GuideFile    string     `json:"guideFile,omitempty"`
	Judged       bool       `json:"judged"`
	Accepted     *bool      `json:"accepted,omitempty"`
	Acknowledges []Acked    `json:"acknowledges,omitempty"`
	AckCode      string     `json:"ackCode,omitempty"`
	Answers      []DocRef   `json:"answers,omitempty"`
	AnsweredBy   *DocRef    `json:"answeredBy,omitempty"`
}

type Note struct {
	Severity  string `json:"severity"`
	Level     string `json:"level"`
	Segment   int    `json:"segment,omitempty"`
	SegmentID string `json:"segmentId,omitempty"`
	Element   int    `json:"element,omitempty"`
	Component int    `json:"component,omitempty"`
	Value     string `json:"value,omitempty"`
	Code      string `json:"code,omitempty"`
	Rule      string `json:"rule,omitempty"`
	Message   string `json:"message"`
	File      string `json:"file,omitempty"`
}

type Acked struct {
	Set          string `json:"set"`
	GroupID      string `json:"groupId,omitempty"`
	GroupControl string `json:"groupControl,omitempty"`
	Control      string `json:"control,omitempty"`
	Interchange  string `json:"interchange,omitempty"`
	Code         string `json:"code,omitempty"`
}

type DocRef struct {
	MessageID   string    `json:"messageId"`
	Number      uint64    `json:"number"`
	Set         string    `json:"set"`
	Control     string    `json:"control"`
	Interchange string    `json:"interchange"`
	Title       string    `json:"title"`
	Code        string    `json:"code,omitempty"`
	At          time.Time `json:"at"`
}

func (d *Document) clone() Document {
	out := *d
	out.Keys = slices.Clone(d.Keys)
	out.Header = cloneRows(d.Header)
	out.Segments = cloneRows(d.Segments)
	out.Trailer = cloneRows(d.Trailer)
	out.Findings = slices.Clone(d.Findings)
	out.Acknowledges = slices.Clone(d.Acknowledges)
	out.Answers = slices.Clone(d.Answers)
	out.Accepted = clonePtr(d.Accepted)
	out.AnsweredBy = clonePtr(d.AnsweredBy)
	return out
}

func cloneRows(rows [][]string) [][]string {
	if rows == nil {
		return nil
	}
	out := make([][]string, len(rows))
	for index := range rows {
		out[index] = slices.Clone(rows[index])
	}
	return out
}

func (m *Message) WireAt() time.Time {
	if m.Direction == ToTrenova {
		if m.DeliveredAt != nil {
			return *m.DeliveredAt
		}
		if m.DueAt != nil {
			return *m.DueAt
		}
	}
	return m.At
}

func (m *Message) ref(d *Document, code string) DocRef {
	return DocRef{
		MessageID:   m.ID,
		Number:      d.Number,
		Set:         d.Set,
		Control:     d.Control,
		Interchange: d.Interchange,
		Title:       d.Title,
		Code:        code,
		At:          m.WireAt(),
	}
}

func (a *Acked) covers(d *Document) bool {
	if a.Set == SetTA1 {
		return SameControl(a.Interchange, d.Interchange)
	}
	if a.GroupID != "" && d.GroupID != "" && a.GroupID != d.GroupID {
		return false
	}
	if !SameControl(a.GroupControl, d.GroupControl) {
		return false
	}
	return a.Control == "" || SameControl(a.Control, d.Control)
}

func (j *Journal) LinkAck(ackID string, targetIDs ...string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	ack, ok := j.byID[ackID]
	if !ok {
		return
	}
	for index := range ack.Documents {
		answer := &ack.Documents[index]
		for _, acked := range answer.Acknowledges {
			for _, id := range targetIDs {
				target, found := j.byID[id]
				if !found || target == ack {
					continue
				}
				link(ack, answer, target, &acked)
			}
		}
	}
	j.version++
}

func link(ack *Message, answer *Document, target *Message, acked *Acked) {
	for index := range target.Documents {
		d := &target.Documents[index]
		if !acked.covers(d) {
			continue
		}
		if acked.Set != SetTA1 {
			ref := ack.ref(answer, acked.Code)
			d.AnsweredBy = &ref
		}
		if !slices.ContainsFunc(answer.Answers, func(r DocRef) bool {
			return r.Number == d.Number
		}) {
			answer.Answers = append(answer.Answers, target.ref(d, ""))
		}
		for _, key := range d.Keys {
			if !slices.Contains(answer.Keys, key) {
				answer.Keys = append(answer.Keys, key)
			}
		}
	}
}

func SameControl(a, b string) bool {
	return strings.TrimLeft(strings.TrimSpace(a), "0") ==
		strings.TrimLeft(strings.TrimSpace(b), "0")
}
