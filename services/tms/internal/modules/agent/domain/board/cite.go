package board

import (
	"regexp"
	"strconv"
)

var citation = regexp.MustCompile(`\[([^\[\]\n]+)\]\[(\d{1,2})\]`)

type Segment struct {
	Text string `json:"text"`
	Cite int    `json:"cite,omitempty"`
}

func Segments(text string) []Segment {
	matches := citation.FindAllStringSubmatchIndex(text, -1)
	segments := make([]Segment, 0, len(matches)*2+1)
	last := 0
	for _, match := range matches {
		if match[0] > last {
			segments = append(segments, Segment{Text: text[last:match[0]]})
		}
		position, err := strconv.Atoi(text[match[4]:match[5]])
		if err != nil || position < 1 {
			segments = append(segments, Segment{Text: text[match[2]:match[3]]})
		} else {
			segments = append(segments, Segment{Text: text[match[2]:match[3]], Cite: position})
		}
		last = match[1]
	}
	if last < len(text) {
		segments = append(segments, Segment{Text: text[last:]})
	}

	return segments
}

func PlainText(text string) string {
	return citation.ReplaceAllString(text, "$1")
}
