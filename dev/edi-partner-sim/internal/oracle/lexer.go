package oracle

import "strings"

type lexer struct {
	raw  []byte
	pos  int
	seps separators
}

func skipBlank(raw []byte, pos int, seps *separators) int {
	for pos < len(raw) {
		b := raw[pos]
		if seps != nil && (b == seps.element || b == seps.segment) {
			return pos
		}
		if b != ' ' && b != '\t' && b != '\r' && b != '\n' && b != 0 {
			return pos
		}
		pos++
	}
	return pos
}

func (l *lexer) next() (Segment, bool) {
	l.pos = skipBlank(l.raw, l.pos, &l.seps)
	for l.pos < len(l.raw) && l.raw[l.pos] == l.seps.segment {
		l.pos = skipBlank(l.raw, l.pos+1, &l.seps)
	}
	if l.pos >= len(l.raw) {
		return Segment{}, false
	}
	start := l.pos
	end := indexByte(l.raw, start, l.seps.segment)
	if end < 0 {
		end = len(l.raw)
		l.pos = end
	} else {
		l.pos = end + 1
	}
	text := strings.TrimRight(string(l.raw[start:end]), "\r\n")
	return newSegment(text, start, l.seps), true
}

func newSegment(text string, offset int, seps separators) Segment {
	elements := strings.Split(text, string(seps.element))
	return Segment{
		ID:       elements[0],
		Offset:   offset,
		Text:     text,
		elements: elements,
		seps:     seps,
	}
}
