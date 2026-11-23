package boardrecords

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

var (
	groupFigure  = regexp.MustCompile(`^(total|largest)_([123])(?::(.+))?$`)
	clauseBreaks = []string{
		",",
		";",
		" and ",
		" against ",
		" while ",
		" but ",
		" whereas ",
		" versus ",
		" vs ",
		" than ",
	}
	sentenceBreaks  = []string{". ", "! ", "? ", "\n"}
	figureMeasureOf = "measure"
)

type labelAt struct {
	at int
	id string
}

func (r Read) GroupFigures(arguments map[string]any) (map[string]any, bool) {
	groups, _ := arguments["groups"].([]any)
	single := make(map[int]string, len(groups))
	board := make([]string, 0, len(groups))
	for idx, entry := range groups {
		fields, _ := entry.(map[string]any)
		listed, _ := fields[recordsKey].([]any)
		ids := make([]string, 0, len(listed))
		for _, element := range listed {
			if text, isText := element.(string); isText &&
				pulid.LooksLike(strings.TrimSpace(text)) {
				ids = append(ids, strings.TrimSpace(text))
			}
		}
		board = append(board, ids...)
		if len(ids) == 1 {
			single[idx+1] = ids[0]
		}
	}
	if len(groups) < 2 || len(single) == 0 {
		return arguments, false
	}
	measure, _ := arguments[figureMeasureOf].(string)
	refs := r.labelled(board)

	return rewriteFigures(arguments, func(text string, at int, mark string) string {
		return figureOfRecord(&figureSite{
			text: text, at: at, mark: mark, single: single,
			measure: strings.TrimSpace(measure), refs: refs,
		})
	})
}

type figureSite struct {
	text    string
	at      int
	mark    string
	single  map[int]string
	measure string
	refs    []agentpagecontext.EntityRef
}

func (r Read) labelled(ids []string) []agentpagecontext.EntityRef {
	refs := make([]agentpagecontext.EntityRef, 0, len(ids))
	for _, ref := range r.refs {
		if slices.Contains(ids, ref.ID) && utf8.RuneCountInString(ref.Label) >= minLabelRunes {
			refs = append(refs, ref)
		}
	}

	return refs
}

func figureOfRecord(site *figureSite) string {
	found := groupFigure.FindStringSubmatch(site.mark[1 : len(site.mark)-1])
	if found == nil {
		return site.mark
	}
	number, _ := strconv.Atoi(found[2])
	record, ok := site.single[number]
	if !ok {
		return site.mark
	}
	named := strings.TrimSpace(found[3])
	if named == "" {
		named = site.measure
	}
	if named == "" {
		return site.mark
	}
	if subject := subjectOf(site.text, site.at, site.refs); subject != "" {
		record = subject
	}

	return "{" + named + ":" + record + "}"
}

func subjectOf(text string, at int, refs []agentpagecontext.EntityRef) string {
	labels := labelsIn(text, refs)
	if id := nearest(labels, at, span(text, at, slices.Concat(clauseBreaks, sentenceBreaks))); id != "" {
		return id
	}
	sentence := span(text, at, sentenceBreaks)
	for idx := len(labels) - 1; idx >= 0; idx-- {
		if labels[idx].at >= sentence[0] && labels[idx].at < at {
			return labels[idx].id
		}
	}

	return ""
}

func labelsIn(text string, refs []agentpagecontext.EntityRef) []labelAt {
	found := make([]labelAt, 0, len(refs))
	for _, ref := range refs {
		for offset := 0; offset < len(text); {
			idx := stringutils.IndexWords(text[offset:], ref.Label)
			if idx < 0 {
				break
			}
			found = append(found, labelAt{at: offset + idx, id: ref.ID})
			offset += idx + len(ref.Label)
		}
	}
	slices.SortFunc(found, func(a, b labelAt) int { return a.at - b.at })

	return found
}

func nearest(labels []labelAt, at int, within [2]int) string {
	before, after := "", ""
	for _, label := range labels {
		if label.at < within[0] || label.at >= within[1] {
			continue
		}
		if label.at < at {
			before = label.id
		} else if after == "" {
			after = label.id
		}
	}
	if before != "" {
		return before
	}

	return after
}

func span(text string, at int, breaks []string) [2]int {
	start, end := 0, len(text)
	for _, mark := range breaks {
		for offset := 0; offset < len(text); {
			idx := strings.Index(text[offset:], mark)
			if idx < 0 {
				break
			}
			pos := offset + idx
			offset = pos + len(mark)
			if insideBraces(text, pos) {
				continue
			}
			if pos < at && pos+len(mark) > start {
				start = pos + len(mark)
			}
			if pos >= at && pos < end {
				end = pos
			}
		}
	}

	return [2]int{start, end}
}

func insideBraces(text string, pos int) bool {
	open := strings.LastIndex(text[:pos], "{")

	return open >= 0 && !strings.Contains(text[open:pos], "}")
}
