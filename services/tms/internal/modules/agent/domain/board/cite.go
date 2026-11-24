package board

import (
	"regexp"
	"slices"
	"strconv"
)

var (
	citation     = regexp.MustCompile(`\[([^\[\]\n]+)\]\[(\d{1,2})\]`)
	artifactLink = regexp.MustCompile(`\[([^\[\]\n]+)\]\(artifact:([A-Za-z0-9_]+)\)`)
)

type Segment struct {
	Text     string `json:"text"`
	Cite     int    `json:"cite,omitempty"`
	Reading  string `json:"reading,omitempty"`
	Artifact string `json:"artifact,omitempty"`
}

func Segments(text string) []Segment {
	matches := citation.FindAllStringSubmatchIndex(text, -1)
	segments := make([]Segment, 0, len(matches)*2+1)
	last := 0
	for _, match := range matches {
		if match[0] > last {
			segments = appendWords(segments, text[last:match[0]])
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
		segments = appendWords(segments, text[last:])
	}

	return segments
}

func appendWords(segments []Segment, text string) []Segment {
	last := 0
	for _, match := range artifactLink.FindAllStringSubmatchIndex(text, -1) {
		if match[0] > last {
			segments = appendReadings(segments, text[last:match[0]])
		}
		segments = append(segments, Segment{
			Text:     text[match[2]:match[3]],
			Artifact: text[match[4]:match[5]],
		})
		last = match[1]
	}
	if last < len(text) {
		segments = appendReadings(segments, text[last:])
	}

	return segments
}

func appendReadings(segments []Segment, text string) []Segment {
	last := 0
	for _, match := range readingToken.FindAllStringSubmatchIndex(text, -1) {
		if match[0] > last {
			segments = append(segments, Segment{Text: text[last:match[0]]})
		}
		segments = append(segments, Segment{Reading: text[match[2]:match[3]]})
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

func Unlinked(text string) string {
	return artifactLink.ReplaceAllString(text, "$1")
}

func ArtifactLinks(text string) []string {
	matches := artifactLink.FindAllStringSubmatch(text, -1)
	ids := make([]string, 0, len(matches))
	for _, match := range matches {
		ids = append(ids, match[2])
	}

	return ids
}

func (b *Board) ArtifactLinks() []string {
	texts := []string{b.Verdict}
	for idx := range b.Blocks {
		texts = append(texts, b.Blocks[idx].Text, b.Blocks[idx].Caption)
	}
	ids := make([]string, 0, 1)
	for _, text := range texts {
		for _, id := range ArtifactLinks(text) {
			if !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
	}

	return ids
}
