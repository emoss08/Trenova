package watchtower

import (
	"strings"

	"github.com/emoss08/trenova/shared/recordids"

	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	MaxFacts          = 6
	MaxImpact         = 6
	MaxTrail          = 8
	maxFactLabel      = 60
	maxFactValue      = 120
	maxImpactLength   = 60
	maxTrailText      = 160
	maxDueLabelLength = 120
	maxRecordLabel    = 200
)

type DueSource string

const (
	DueSourcePolicy = DueSource("Policy")
	DueSourceSource = DueSource("Source")
	DueSourcePerson = DueSource("Person")
)

func (s DueSource) String() string { return string(s) }

func AllDueSources() []DueSource {
	return []DueSource{DueSourcePolicy, DueSourceSource, DueSourcePerson}
}

func (s DueSource) IsValid() bool {
	switch s {
	case DueSourcePolicy, DueSourceSource, DueSourcePerson:
		return true
	default:
		return false
	}
}

type Fact struct {
	Label string `json:"label"`
	Value string `json:"value,omitempty"`
	At    int64  `json:"at,omitempty"`
}

type TrailEvent struct {
	At   int64  `json:"at,omitempty"`
	Text string `json:"text"`
}

type Deadline struct {
	At    int64
	Label string
}

type RecordRef struct {
	Kind  recordkind.Kind
	Label string
}

func NormalizeFacts(facts []Fact) []Fact {
	out := make([]Fact, 0, min(len(facts), MaxFacts))
	for _, fact := range facts {
		label := stringutils.Ellipsize(strings.TrimSpace(fact.Label), maxFactLabel)
		value := stringutils.Ellipsize(Plain(fact.Value), maxFactValue)
		at := max(fact.At, 0)
		if label == "" || (value == "" && at == 0) {
			continue
		}
		out = append(out, Fact{Label: label, Value: value, At: at})
		if len(out) == MaxFacts {
			break
		}
	}
	return out
}

func NormalizeImpact(impact []string) []string {
	out := make([]string, 0, min(len(impact), MaxImpact))
	for _, entry := range impact {
		entry = stringutils.Ellipsize(Plain(entry), maxImpactLength)
		if entry == "" {
			continue
		}
		out = append(out, entry)
		if len(out) == MaxImpact {
			break
		}
	}
	return out
}

func NormalizeTrail(trail []TrailEvent) []TrailEvent {
	start := 0
	if len(trail) > MaxTrail {
		start = len(trail) - MaxTrail
	}
	out := make([]TrailEvent, 0, len(trail)-start)
	for _, event := range trail[start:] {
		text := stringutils.Ellipsize(Plain(event.Text), maxTrailText)
		if text == "" {
			continue
		}
		out = append(out, TrailEvent{At: max(event.At, 0), Text: text})
	}
	return out
}

func NormalizeDueLabel(label string) string {
	return stringutils.Ellipsize(strings.TrimSpace(label), maxDueLabelLength)
}

func NormalizeRecordLabel(label string) string {
	return stringutils.Ellipsize(strings.TrimSpace(label), maxRecordLabel)
}

// Plain is text as a person reads it on the Watchtower: with no internal
// record id in it. Every item passes its title, summary, facts, impact and
// trail through here as it is stored, so no source can put an id on the
// feed whatever it was handed: a fact that held nothing but an id is left
// with no value and is dropped, since the item's link already opens that
// record.
func Plain(text string) string {
	return strings.TrimSpace(recordids.Strip(text))
}
