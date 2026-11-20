package oracle

import (
	"slices"
	"strings"
)

type Segment struct {
	ID       string `json:"id"`
	Index    int    `json:"index,omitempty"`
	Offset   int    `json:"offset"`
	LoopID   string `json:"loopId,omitempty"`
	Text     string `json:"text"`
	elements []string
	seps     separators
}

func (s *Segment) ElementCount() int {
	return len(s.elements) - 1
}

func (s *Segment) Element(pos int) string {
	if pos < 1 || pos >= len(s.elements) {
		return ""
	}
	return s.elements[pos]
}

func (s *Segment) Repetitions(pos int) []string {
	v := s.Element(pos)
	if v == "" {
		return nil
	}
	return s.splitRepetitions(v)
}

func (s *Segment) Components(pos int) []string {
	reps := s.Repetitions(pos)
	if len(reps) == 0 {
		return nil
	}
	return s.splitComponents(reps[0])
}

func (s *Segment) splitComponents(value string) []string {
	return strings.Split(value, string(s.seps.component))
}

func (s *Segment) splitRepetitions(value string) []string {
	if !s.seps.hasRepetition {
		return []string{value}
	}
	return strings.Split(value, string(s.seps.repetition))
}

func (s *Segment) Elements() []string {
	if len(s.elements) == 0 && s.Text != "" {
		return []string{s.Text}
	}
	return slices.Clone(s.elements)
}

func (s *Segment) Separator() string {
	if s.seps.element == 0 {
		return ""
	}
	return string(s.seps.element)
}

func (s *Segment) Terminator() string {
	switch s.seps.segment {
	case 0, '\r', '\n':
		return ""
	}
	return string(s.seps.segment)
}
