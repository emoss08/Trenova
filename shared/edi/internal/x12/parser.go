package x12

import (
	"bytes"
	"strings"
)

// ParseSegments splits raw X12 payload into segments using provided delimiters.
// It performs minimal normalization: trims whitespace around segments and ignores empties.
func ParseSegments(raw []byte, d Delimiters) ([]Segment, error) {
	// Normalize line endings: users sometimes have CR/LF around segments.
	r := bytes.ReplaceAll(raw, []byte{'\r', '\n'}, []byte{'\n'})

	// Split by segment terminator. Some files include trailing terminator; ignore empty tails.
	parts := bytes.Split(r, []byte{d.Segment})
	segs := make([]Segment, 0, len(parts))
	idx := 0
	for _, p := range parts {
		p = bytes.TrimSpace(p)
		if len(p) == 0 {
			continue
		}
		// Split elements by element separator; the first token contains tag.
		elems := splitKeepEmpty(p, d.Element)
		if len(elems) == 0 {
			continue
		}
		tag := string(elems[0])
		// For each subsequent element, split into components (composite elements)
		elements := make([][]string, 0, len(elems)-1)
		for _, e := range elems[1:] {
			if d.Component != 0 && bytes.ContainsRune(e, rune(d.Component)) {
				comps := splitKeepEmpty(e, d.Component)
				arr := make([]string, len(comps))
				for i := range comps {
					arr[i] = string(comps[i])
				}
				elements = append(elements, arr)
			} else {
				elements = append(elements, []string{string(e)})
			}
		}
		segs = append(segs, Segment{Tag: tag, Elements: elements, Index: idx})
		idx++
	}
	return segs, nil
}

// splitKeepEmpty splits b by sep, keeping empty tokens between consecutive seps.
func splitKeepEmpty(b []byte, sep byte) [][]byte {
	// bytes.Split will keep empties by design when separator appears back-to-back,
	// but we use a manual scan to avoid allocations from converting to string first.
	out := make([][]byte, 0, 8)
	start := 0
	for i := 0; i < len(b); i++ {
		if b[i] == sep {
			out = append(out, b[start:i])
			start = i + 1
		}
	}
	out = append(out, b[start:])
	return out
}

// FindSegments returns all segments matching a tag (e.g., "N1").
func FindSegments(segs []Segment, tag string) []Segment {
	res := make([]Segment, 0)
	for _, s := range segs {
		if strings.EqualFold(s.Tag, tag) {
			res = append(res, s)
		}
	}
	return res
}
