package briefing

import (
	"cmp"
	"slices"
	"strings"

	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	MaxHeadlineLinks = 3
	quietHeadline    = "Nothing needs you this morning."
)

type HeadlineLink struct {
	Text    string     `json:"text"`
	Section SectionKey `json:"section"`
	Path    string     `json:"path"`
}

type Phrase struct {
	Text    string
	Section SectionKey
}

func (s Section) LinkPath() string {
	if s.Path != "" && stringutils.IsSafeAppPath(s.Path) {
		return s.Path
	}
	for _, item := range s.Items {
		if item.Path != "" && item.IsSafePath() {
			return item.Path
		}
	}

	return ""
}

func ComputedHeadline(sections []Section) (string, []HeadlineLink) {
	for _, section := range sections {
		summary := strings.TrimSpace(section.Summary)
		if summary == "" {
			continue
		}

		return summary, LinkPhrases(summary, sections, []Phrase{{
			Text:    summary,
			Section: section.Key,
		}})
	}

	return quietHeadline, []HeadlineLink{}
}

func LinkPhrases(headline string, sections []Section, phrases []Phrase) []HeadlineLink {
	paths := make(map[SectionKey]string, len(sections))
	for _, section := range sections {
		if path := section.LinkPath(); path != "" {
			paths[section.Key] = path
		}
	}

	links := make([]HeadlineLink, 0, len(phrases))
	for _, phrase := range phrases {
		path, ok := paths[phrase.Section]
		if !ok {
			continue
		}
		links = append(links, HeadlineLink{
			Text:    strings.TrimSpace(phrase.Text),
			Section: phrase.Section,
			Path:    path,
		})
	}

	return FitLinks(headline, links)
}

func FitLinks(headline string, links []HeadlineLink) []HeadlineLink {
	type placed struct {
		link  HeadlineLink
		start int
		end   int
	}

	kept := make([]placed, 0, len(links))
	for _, link := range links {
		if len(kept) == MaxHeadlineLinks {
			break
		}
		if link.Text == "" || !link.Section.IsValid() || len(link.Path) > maxPathLength ||
			!stringutils.IsSafeAppPath(link.Path) {
			continue
		}
		start := strings.Index(headline, link.Text)
		if start < 0 {
			continue
		}
		end := start + len(link.Text)
		overlaps := slices.ContainsFunc(kept, func(other placed) bool {
			return start < other.end && other.start < end
		})
		if overlaps {
			continue
		}
		kept = append(kept, placed{link: link, start: start, end: end})
	}

	slices.SortFunc(kept, func(a, b placed) int { return cmp.Compare(a.start, b.start) })

	out := make([]HeadlineLink, len(kept))
	for index, entry := range kept {
		out[index] = entry.link
	}

	return out
}
