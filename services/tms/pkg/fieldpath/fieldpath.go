package fieldpath

import (
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
)

type Segment struct {
	key   string
	index int
}

func Key(name string) Segment {
	return Segment{key: name, index: -1}
}

func Index(i int) Segment {
	return Segment{index: max(i, 0)}
}

func (s Segment) IsIndex() bool {
	return s.index >= 0
}

func (s Segment) Key() string {
	return s.key
}

func (s Segment) Index() int {
	return s.index
}

type Path []Segment

func Of(segments ...Segment) Path {
	if len(segments) == 0 {
		return nil
	}

	return append(make(Path, 0, len(segments)), segments...)
}

func Parse(expr string) Path {
	if expr == "" {
		return nil
	}

	path := make(Path, 0, strings.Count(expr, ".")+strings.Count(expr, "[")+1)
	for part := range strings.SplitSeq(expr, ".") {
		path = appendPart(path, part)
	}
	if len(path) == 0 {
		return nil
	}

	return path
}

func appendPart(path Path, part string) Path {
	name, rest, bracketed := strings.Cut(part, "[")
	if name != "" {
		path = append(path, segmentOf(name))
	}
	for bracketed {
		var inner string
		var closed bool
		inner, rest, closed = strings.Cut(rest, "]")
		if !closed {
			return append(path, Key(inner))
		}
		if inner != "" {
			path = append(path, segmentOf(inner))
		}
		_, rest, bracketed = strings.Cut(rest, "[")
	}

	return path
}

func segmentOf(token string) Segment {
	if index, ok := parseIndex(token); ok {
		return Index(index)
	}

	return Key(token)
}

func parseIndex(token string) (int, bool) {
	if token == "" || len(token) > 9 {
		return 0, false
	}
	for i := range len(token) {
		if token[i] < '0' || token[i] > '9' {
			return 0, false
		}
	}
	index, err := strconv.Atoi(token)

	return index, err == nil
}

func (p Path) IsZero() bool {
	return len(p) == 0
}

func (p Path) Append(segments ...Segment) Path {
	if len(segments) == 0 {
		return p
	}
	out := make(Path, 0, len(p)+len(segments))
	out = append(out, p...)

	return append(out, segments...)
}

func (p Path) Join(child Path) Path {
	if len(p) == 0 {
		return child
	}

	return p.Append(child...)
}

func (p Path) HasPrefix(prefix Path) bool {
	return len(prefix) <= len(p) && p[:len(prefix)].Equal(prefix)
}

func (p Path) Equal(other Path) bool {
	if len(p) != len(other) {
		return false
	}
	for i := range p {
		if p[i] != other[i] {
			return false
		}
	}

	return true
}

func (p Path) String() string {
	if len(p) == 0 {
		return ""
	}

	var b strings.Builder
	b.Grow(len(p) * 8)
	for i, segment := range p {
		if segment.IsIndex() {
			b.WriteByte('[')
			b.WriteString(strconv.Itoa(segment.index))
			b.WriteByte(']')
			continue
		}
		if i > 0 {
			b.WriteByte('.')
		}
		b.WriteString(segment.key)
	}

	return b.String()
}

func (p Path) MarshalJSON() ([]byte, error) {
	out := make([]byte, 0, 2+len(p)*12)
	out = append(out, '[')
	for i, segment := range p {
		if i > 0 {
			out = append(out, ',')
		}
		if segment.IsIndex() {
			out = strconv.AppendInt(out, int64(segment.index), 10)
			continue
		}
		key, err := sonic.Marshal(segment.key)
		if err != nil {
			return nil, err
		}
		out = append(out, key...)
	}

	return append(out, ']'), nil
}
