package boardrecords

import (
	"maps"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardanswer"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

var placedFigure = regexp.MustCompile(`\{([^{}:\s]{1,60}):([^{}:\s]{1,60})\}`)

func MetricGroups(arguments map[string]any) (grouped map[string]any, names []string) {
	groups, _ := arguments["groups"].([]any)
	shownMetrics := make([]string, 0, len(groups))
	for _, entry := range groups {
		fields, _ := entry.(map[string]any)
		if name, isText := fields["metric"].(string); isText {
			if metric, known := boardregistry.Catalog().MetricNamed(name); known {
				shownMetrics = append(shownMetrics, metric.Name)
			}
		}
	}
	wanted := make([]boardcatalog.Metric, 0, 1)
	for _, text := range figureTexts(arguments, groups) {
		for _, found := range placedFigure.FindAllStringSubmatch(text, -1) {
			metric, declares := figureMetric(found[1], found[2])
			if !declares || slices.Contains(shownMetrics, metric.Name) ||
				slices.ContainsFunc(wanted, func(m boardcatalog.Metric) bool {
					return m.Name == metric.Name
				}) {
				continue
			}
			wanted = append(wanted, metric)
		}
	}
	if len(wanted) == 0 || len(groups)+len(wanted) > boardanswer.MaxGroups {
		return arguments, nil
	}
	out := maps.Clone(arguments)
	added := slices.Clone(groups)
	names = make([]string, 0, len(wanted))
	for idx := range wanted {
		added = append(added, map[string]any{
			titleKey: metricTitle(&wanted[idx]),
			"metric": wanted[idx].Name,
		})
		names = append(names, wanted[idx].Name)
	}
	out["groups"] = added

	return out, names
}

func figureTexts(arguments map[string]any, groups []any) []string {
	texts := make([]string, 0, len(groups)+1)
	if answer, isText := arguments["answer"].(string); isText {
		texts = append(texts, answer)
	}
	for _, entry := range groups {
		fields, _ := entry.(map[string]any)
		if note, isText := fields["note"].(string); isText {
			texts = append(texts, note)
		}
	}

	return texts
}

func figureMetric(left, right string) (boardcatalog.Metric, bool) {
	for _, sides := range [][2]string{{left, right}, {right, left}} {
		metric, known := boardregistry.Catalog().MetricNamed(sides[1])
		if !known {
			continue
		}
		if _, declared := metric.Figure(sides[0]); declared {
			return metric, true
		}
	}

	return boardcatalog.Metric{}, false
}

func metricTitle(metric *boardcatalog.Metric) string {
	title := metric.Words
	if cut := strings.IndexAny(title, ":;"); cut > 0 {
		title = title[:cut]
	}
	title = strings.TrimSpace(title)
	if utf8.RuneCountInString(title) > board.MaxBlockTitleRunes {
		runes := []rune(title)
		title = string(runes[:board.MaxBlockTitleRunes])
		if cut := strings.LastIndexByte(title, ' '); cut > 0 {
			title = title[:cut]
		}
		title = strings.TrimSpace(title)
	}
	if title == "" {
		return metric.Name
	}
	first, size := utf8.DecodeRuneInString(title)

	return string(unicode.ToUpper(first)) + title[size:]
}
