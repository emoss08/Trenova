package x12

import (
	"bytes"
	"strings"
	"sync"
)

// Pool for reusing byte slices in parsing operations
var bufferPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 4096)
		return &b
	},
}

// Pool for reusing segment slices
var segmentPool = sync.Pool{
	New: func() any {
		s := make([]Segment, 0, 128)
		return &s
	},
}

// ParseSegments splits raw X12 payload into segments using provided delimiters.
// It performs minimal normalization: trims whitespace around segments and ignores empties.
func ParseSegments(raw []byte, d Delimiters) ([]Segment, error) {
	r := bytes.ReplaceAll(raw, []byte{'\r', '\n'}, []byte{'\n'})

	parts := bytes.Split(r, []byte{d.Segment})
	segs := make([]Segment, 0, len(parts))
	idx := 0
	for _, p := range parts {
		p = bytes.TrimSpace(p)
		if len(p) == 0 {
			continue
		}
		elems := splitKeepEmpty(p, d.Element)
		if len(elems) == 0 {
			continue
		}

		var tag string
		var elemStart int

		segStr := string(p)
		// ! Order matters - check longer tags first (B2A before B2)
		knownTags := []string{
			"ISA", "IEA", "GS", "GE", "ST", "SE",
			"B2A", "B2", "N1", "N3", "N4", "S5",
			"L11", "L3", "L5", "G61", "G62", "NTE", "AT5", "AT8", "LAD", "DTM", "N7", "RTT", "C3",
			"AK1", "AK2", "AK3", "AK4", "AK5", "AK9", "IK1", "IK3", "IK4", "IK5",
		}

		tagFound := false
		for _, kt := range knownTags {
			if strings.HasPrefix(segStr, kt) {
				tag = kt

				// ! Special handling for segments where tag is followed by delimiter
				// ! Examples: ISA`00`... or B2A*00*...
				afterTag := p[len(tag):]

				// ! If there's a delimiter immediately after the tag, skip it
				if len(afterTag) > 0 && afterTag[0] == d.Element {
					afterTag = afterTag[1:]
				}

				// ! Now split the remaining part by the delimiter
				elems = splitKeepEmpty(afterTag, d.Element)
				elemStart = 0 // Elements start from index 0 in our new array
				tagFound = true
				break
			}
		}

		// ! Fallback: if no known tag found, use first element as tag
		if !tagFound {
			tag = string(elems[0])
			elemStart = 1
		}

		elements := make([][]string, 0, len(elems)-elemStart)
		for _, e := range elems[elemStart:] {
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
	// ! bytes.Split will keep empties by design when separator appears back-to-back,
	// ! but we use a manual scan to avoid allocations from converting to string first.
	// ! Pre-allocate with a reasonable capacity to reduce allocations
	out := make([][]byte, 0, 16)
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
