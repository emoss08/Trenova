package oracle

import (
	"errors"
	"fmt"
	"strings"
)

var errNotInterchange = errors.New("the text does not start with an ISA segment")

const (
	isaElementCount      = 16
	repetitionSepVersion = "00402"
)

var (
	isaWidths = [isaElementCount]int{2, 10, 2, 10, 2, 15, 2, 15, 6, 4, 1, 5, 9, 1, 1, 1}
	isaCodes  = [isaElementCount]string{
		"010", "011", "012", "013", "005", "006", "007", "008",
		"014", "015", "016", "017", "018", "019", "020", "027",
	}
)

type separators struct {
	element       byte
	component     byte
	repetition    byte
	segment       byte
	hasRepetition bool
}

type isaHeader struct {
	seps     separators
	fields   [isaElementCount]string
	end      int
	squeezed bool
	findings []Finding
	fatal    bool
}

func readISA(raw []byte, start int) (*isaHeader, error) {
	if len(raw)-start < 4 || string(raw[start:start+3]) != "ISA" {
		return nil, errNotInterchange
	}
	h := &isaHeader{}
	h.seps.element = raw[start+3]
	pos := start + 4
	for i := range isaElementCount - 1 {
		next := indexByte(raw, pos, h.seps.element)
		if next < 0 {
			return nil, fmt.Errorf(
				"the ISA at byte %d ends before ISA%02d",
				start,
				i+2,
			)
		}
		h.fields[i] = string(raw[pos:next])
		pos = next + 1
	}
	if len(raw)-pos < 2 {
		return nil, fmt.Errorf("the ISA at byte %d has no segment terminator", start)
	}
	h.fields[15] = string(raw[pos])
	h.seps.component = raw[pos]
	h.seps.segment = raw[pos+1]
	h.end = pos + 2
	h.checkSeparators()
	if h.fatal {
		return h, nil
	}
	h.checkWidths()
	h.checkValues()
	return h, nil
}

func indexByte(raw []byte, from int, b byte) int {
	for i := from; i < len(raw); i++ {
		if raw[i] == b {
			return i
		}
	}
	return -1
}

func badSeparator(b byte) bool {
	return b == ' ' || isAlnum(b)
}

func isAlnum(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func (h *isaHeader) fail(code, rule, message string) {
	f := errorAt(LevelInterchange, code, rule, message)
	f.SegmentID = "ISA"
	h.findings = append(h.findings, f)
	h.fatal = true
}

func (h *isaHeader) checkSeparators() {
	s := &h.seps
	if badSeparator(s.element) || s.element == '\r' || s.element == '\n' {
		h.fail(
			codeInvalidElementSep,
			"ISA element separator",
			fmt.Sprintf("the element separator %q is a letter, digit or space", s.element),
		)
		return
	}
	if badSeparator(s.component) || s.component == s.element {
		h.fail(
			codeInvalidComponentSep,
			"ISA16 component separator",
			fmt.Sprintf("the component separator %q is not usable", s.component),
		)
		return
	}
	if badSeparator(s.segment) || s.segment == s.element || s.segment == s.component {
		h.fail(
			codeInvalidTerminator,
			"ISA segment terminator",
			fmt.Sprintf("the segment terminator %q is not usable", s.segment),
		)
		return
	}
	version := strings.TrimSpace(h.fields[11])
	rep := h.fields[10]
	if version < repetitionSepVersion || len(rep) != 1 || rep == "U" {
		return
	}
	r := rep[0]
	if badSeparator(r) || r == s.element || r == s.component || r == s.segment {
		h.fail(
			isaCodes[10],
			"ISA11 repetition separator",
			fmt.Sprintf("the repetition separator %q is not usable", r),
		)
		return
	}
	s.repetition = r
	s.hasRepetition = true
}

func (h *isaHeader) checkWidths() {
	total, want := 0, 0
	for i, f := range h.fields {
		total += len(f)
		want += isaWidths[i]
	}
	if total < want {
		h.squeezed = true
		h.findings = append(h.findings, Finding{
			Severity:  SeverityWarning,
			Level:     LevelInterchange,
			SegmentID: "ISA",
			Rule:      "ISA fixed widths",
			Message:   "the ISA padding was squeezed out; it was read by counting separators",
		})
	}
	for i, f := range h.fields {
		if len(f) == isaWidths[i] || (h.squeezed && len(f) < isaWidths[i]) {
			continue
		}
		h.widthFinding(i, f)
	}
}

func (h *isaHeader) widthFinding(i int, value string) {
	f := errorAt(
		LevelInterchange,
		isaCodes[i],
		fmt.Sprintf("ISA%02d width %d", i+1, isaWidths[i]),
		fmt.Sprintf("ISA%02d is %d characters, not %d", i+1, len(value), isaWidths[i]),
	)
	f.SegmentID = "ISA"
	f.ElementPos = i + 1
	f.Value = value
	h.findings = append(h.findings, f)
}

func (h *isaHeader) checkValues() {
	checks := []struct {
		pos  int
		ok   func(string) bool
		rule string
	}{
		{9, func(v string) bool { return len(v) == 6 && validDate(v) }, "YYMMDD"},
		{10, func(v string) bool { return len(v) == 4 && validTime(v) }, "HHMM"},
		{12, func(v string) bool { return len(v) == 5 && allDigits(v) }, "five digits"},
		{13, func(v string) bool { return len(v) > 0 && len(v) <= 9 && allDigits(v) }, "digits"},
		{14, func(v string) bool { return v == "0" || v == "1" }, "codes [0 1]"},
		{15, func(v string) bool { return v == "P" || v == "T" || v == "I" }, "codes [I P T]"},
	}
	for _, c := range checks {
		v := strings.TrimSpace(h.fields[c.pos-1])
		if c.ok(v) {
			continue
		}
		f := errorAt(
			LevelInterchange,
			isaCodes[c.pos-1],
			fmt.Sprintf("ISA%02d %s", c.pos, c.rule),
			fmt.Sprintf("ISA%02d %q is not %s", c.pos, v, c.rule),
		)
		f.SegmentID = "ISA"
		f.ElementPos = c.pos
		f.Value = v
		h.findings = append(h.findings, f)
	}
}

func (h *isaHeader) controlNumber() string {
	return strings.TrimSpace(h.fields[12])
}
