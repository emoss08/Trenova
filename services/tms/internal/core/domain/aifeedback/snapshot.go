package aifeedback

import (
	"cmp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	MaxSnapshotTextRunes   = 4000
	MaxSnapshotToolLines   = 20
	MaxSnapshotToolRunes   = 200
	MinRedactableRunes     = 3
	RedactedPlaceholder    = "[redacted]"
	patternKeySeparator    = "\x1f"
	patternToolSeparator   = ">"
	patternMaxToolSequence = 12
)

type ToolLine struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Failed  bool   `json:"failed"`
}

type TurnSnapshot struct {
	Question     string     `json:"question"`
	Answer       string     `json:"answer"`
	Tools        []ToolLine `json:"tools"`
	OmittedTools int        `json:"omittedTools"`
	Redacted     bool       `json:"redacted"`
}

type SnapshotParams struct {
	Question string
	Answer   string
	Tools    []ToolLine
	Redact   []string
}

func NewTurnSnapshot(p SnapshotParams) *TurnSnapshot {
	kept := min(len(p.Tools), MaxSnapshotToolLines)
	snapshot := &TurnSnapshot{
		Question:     p.Question,
		Answer:       p.Answer,
		Tools:        make([]ToolLine, 0, kept),
		OmittedTools: len(p.Tools) - kept,
	}
	for _, line := range p.Tools[:kept] {
		snapshot.Tools = append(snapshot.Tools, ToolLine{
			Name:    strings.TrimSpace(line.Name),
			Summary: line.Summary,
			Failed:  line.Failed,
		})
	}

	snapshot.Redacted = snapshot.redact(p.Redact)
	snapshot.bound()

	return snapshot
}

func (s *TurnSnapshot) bound() {
	s.Question = stringutils.Ellipsize(s.Question, MaxSnapshotTextRunes)
	s.Answer = stringutils.Ellipsize(s.Answer, MaxSnapshotTextRunes)
	for index := range s.Tools {
		s.Tools[index].Name = stringutils.Ellipsize(s.Tools[index].Name, MaxSnapshotToolRunes)
		s.Tools[index].Summary = stringutils.Ellipsize(
			singleLine(s.Tools[index].Summary),
			MaxSnapshotToolRunes,
		)
	}
}

func (s *TurnSnapshot) redact(values []string) bool {
	replacer := redactor(values)
	if replacer == nil {
		return false
	}

	redacted := false
	apply := func(text string) string {
		next := replacer.Replace(text)
		if next != text {
			redacted = true
		}

		return next
	}

	s.Question = apply(s.Question)
	s.Answer = apply(s.Answer)
	for index := range s.Tools {
		s.Tools[index].Summary = apply(s.Tools[index].Summary)
	}

	return redacted
}

func redactor(values []string) *strings.Replacer {
	unique := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if utf8.RuneCountInString(value) < MinRedactableRunes {
			continue
		}
		if _, dup := seen[value]; dup {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	if len(unique) == 0 {
		return nil
	}

	slices.SortFunc(unique, func(a, b string) int {
		return cmp.Compare(len(b), len(a))
	})

	pairs := make([]string, 0, len(unique)*2)
	for _, value := range unique {
		pairs = append(pairs, value, RedactedPlaceholder)
	}

	return strings.NewReplacer(pairs...)
}

func singleLine(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func ToolSequence(names []string) []string {
	sequence := make([]string, 0, min(len(names), patternMaxToolSequence))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if len(sequence) > 0 && sequence[len(sequence)-1] == name {
			continue
		}
		if len(sequence) == patternMaxToolSequence {
			break
		}
		sequence = append(sequence, name)
	}

	return sequence
}

func PatternKey(toolNames []string, firstReason Reason, subjectType string) string {
	var b strings.Builder
	b.WriteString(strings.Join(ToolSequence(toolNames), patternToolSeparator))
	b.WriteString(patternKeySeparator)
	b.WriteString(string(firstReason))
	b.WriteString(patternKeySeparator)
	b.WriteString(strings.TrimSpace(subjectType))

	return hashutils.SHA256Hex(b.String())
}
