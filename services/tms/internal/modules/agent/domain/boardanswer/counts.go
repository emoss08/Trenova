package boardanswer

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardtyped"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/shared/pulid"
)

var countOfCount = regexp.MustCompile(
	`\{count_(\d)\}\s+(?i:out\s+of|of)\s+(?i:the\s+|all\s+)?\{count_(\d)\}`,
)

func (f *figure) counts() bool {
	return f.base == NameCount || f.base == NameAll
}

func (t *translator) partOf(used []figure, groups []group, texts ...string) {
	if len(used) == 0 {
		return
	}
	for _, text := range texts {
		for _, match := range countOfCount.FindAllStringSubmatch(text, -1) {
			part, _ := strconv.Atoi(match[1])
			whole, _ := strconv.Atoi(match[2])
			if part == whole || part < 1 || whole < 1 || part > len(groups) ||
				whole > len(groups) ||
				groups[part-1].set == nil ||
				groups[whole-1].set == nil {
				continue
			}
			if !slices.ContainsFunc(groups[part-1].set.IDs, func(id pulid.ID) bool {
				return !slices.Contains(groups[whole-1].set.IDs, id)
			}) {
				continue
			}
			t.addf("{count_%d} of {count_%d} would read %d of %d, but group %d's records are "+
				"not among group %d's, so it would state a share that is not true; for how many "+
				"of every group's records, write {count_%d} of {count}, and for the whole list "+
				"they came from, {count_%d} of {%s}", part, whole,
				len(groups[part-1].set.IDs), len(groups[whole-1].set.IDs), part, whole, part,
				part, NameAll)
		}
	}
}

func (t *translator) oneGroupNumbered(used []figure, groups []group) {
	if len(groups) != 1 || groups[0].set == nil {
		return
	}
	bare := slices.ContainsFunc(used, func(found figure) bool {
		return found.number == 0 && found.metric == noMetric && found.record.IsNil()
	})
	for _, found := range used {
		if found.number != 1 || !bare {
			continue
		}
		t.addf("{%s} is the one group's figure, the same as {%s}, so beside it it would read "+
			"all %d records again; to give a figure for part of them, put that part in a group "+
			"of its own and write its {%s_2}", found.raw, found.base, len(groups[0].set.IDs),
			found.base)

		return
	}
}

func (t *translator) said(used []figure, measure board.Measure, texts ...string) []figure {
	for idx := range used {
		found := &used[idx]
		if (found.base != NameTotal && found.base != NameLargest) || found.record.IsNotNil() ||
			found.metric != noMetric {
			continue
		}
		read := found.measureOr(measure)
		if named := t.namedMeasure(read, found, texts); named != "" {
			found.measure = named
		}
	}

	return used
}

func (t *translator) namedMeasure(
	read board.Measure,
	found *figure,
	texts []string,
) board.Measure {
	mark := "{" + found.raw + "}"
	clauses := make([]string, 0, 1)
	for _, text := range texts {
		for offset := 0; ; {
			idx := strings.Index(text[offset:], mark)
			if idx < 0 {
				break
			}
			at := offset + idx
			clauses = append(clauses, boardtyped.Clause(text, at, at+len(mark)))
			offset = at + len(mark)
		}
	}
	if len(clauses) == 0 {
		return ""
	}
	words := strings.FieldsFunc(strings.ToLower(strings.Join(clauses, " ")), func(r rune) bool {
		return !unicode.IsLetter(r)
	})
	saying := func(spec *boardcatalog.Measure) bool {
		return slices.ContainsFunc(spec.Says, func(word string) bool {
			return slices.Contains(words, strings.ToLower(word))
		})
	}
	measures := boardregistry.Catalog().MeasuresOf(t.kind)
	named := board.Measure("")
	for idx := range measures {
		spec := &measures[idx]
		if !saying(spec) {
			continue
		}
		if board.Measure(spec.Name) == read || named != "" ||
			(found.base == NameTotal && !spec.Summable) {
			return ""
		}
		named = board.Measure(spec.Name)
	}

	return named
}

func leadsOf(groups []group) string {
	parts := make([]string, 0, len(groups))
	share := board.MaxReadings / max(1, len(slices.DeleteFunc(slices.Clone(groups),
		func(g group) bool { return g.metric == nil })))
	for idx := range groups {
		metric := groups[idx].metric
		if metric == nil || len(metric.Leads) == 0 {
			continue
		}
		leads, rest := metric.Led()
		marks := make([]string, 0, len(leads))
		for _, lead := range leads[:min(len(leads), share)] {
			marks = append(marks, fmt.Sprintf("{%s:%s}", lead.Name, metric.Name))
		}
		parts = append(parts, fmt.Sprintf("%s's answer leads with %s, and its %d other figures "+
			"are drawn on the board", metric.Name, strings.Join(marks, ", "),
			len(rest)+len(leads)-len(marks)))
	}
	if len(parts) == 0 {
		return ""
	}

	return " (" + strings.Join(parts, "; ") + ")"
}
