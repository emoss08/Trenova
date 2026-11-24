package replycheck

import (
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/shared/inflect"

	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	minMetricFigures     = 2
	minFigureRunes       = 2
	metricCorrectionLead = "Your reply states in prose figures a metric keeps:"
)

type metricStated struct {
	metric  string
	figures []string
	others  []string
}

func statedMetricFigures(t Turn, reply string) metricStated {
	read := make([]metricStated, 0, 2)
	for idx := len(t.Messages()) - 1; idx >= 0; idx-- {
		message := &t.Messages()[idx]
		if message.Role != llmports.RoleTool || message.IsError ||
			message.ToolName != agenttoolports.MetricReadTool {
			continue
		}
		metric := metricReadOf(message.Content)
		if metric.metric == "" || !drawsMoreThanFigures(metric.metric) ||
			slices.ContainsFunc(read, func(seen metricStated) bool {
				return seen.metric == metric.metric
			}) {
			continue
		}
		stated := make([]string, 0, len(metric.figures))
		for _, value := range metric.figures {
			if utf8.RuneCountInString(value) >= minFigureRunes &&
				stringutils.ContainsWords(reply, value) {
				stated = append(stated, value)
			}
		}
		read = append(read, metricStated{metric: metric.metric, figures: stated})
	}
	lead := slices.IndexFunc(read, func(found metricStated) bool {
		return len(found.figures) >= minMetricFigures
	})
	if lead < 0 {
		return metricStated{}
	}
	out := read[lead]
	for idx := len(read) - 1; idx >= 0; idx-- {
		if idx != lead && len(read[idx].figures) > 0 {
			out.others = append(out.others, read[idx].metric)
			out.figures = append(out.figures, read[idx].figures...)
		}
	}

	return out
}

func drawsMoreThanFigures(name string) bool {
	metric, known := boardregistry.Catalog().MetricNamed(name)

	return known && (metric.Form != boardcatalog.FormFigures || metric.Records != nil)
}

func metricReadOf(content string) metricStated {
	payload, ok := toolresult.Payload(content)
	if !ok {
		return metricStated{}
	}
	fields, _ := payload.(map[string]any)
	metric, _ := fields["metric"].(string)
	listed, _ := fields["figures"].([]any)
	out := metricStated{metric: metric, figures: make([]string, 0, len(listed))}
	for _, entry := range listed {
		figure, _ := entry.(map[string]any)
		if value, isText := figure["value"].(string); isText && strings.TrimSpace(value) != "" {
			out.figures = append(out.figures, strings.TrimSpace(value))
		}
	}

	return out
}

func unreadRecordsOf(t Turn, name string) string {
	metric, known := boardregistry.Catalog().MetricNamed(name)
	if !known || metric.Records == nil {
		return ""
	}
	for idx := range t.Messages() {
		message := &t.Messages()[idx]
		if message.Role != llmports.RoleTool || message.IsError {
			continue
		}
		for _, ref := range boardRecords(message.Content) {
			if recordkind.Kind(ref.Type) == metric.Records.Kind {
				return ""
			}
		}
	}

	return " Its board also lists " + metric.Records.Words + ", which this turn has not read: " +
		"read them with " + metric.Records.From + " first, then give their ids as the second " +
		"group."
}

func metricCorrection(stated *metricStated, unread string) string {
	metrics := append(slices.Clone(stated.others), stated.metric)
	groups := make([]string, 0, len(metrics))
	leads := make([]string, 0, len(metrics))
	share := board.MaxReadings / len(metrics)
	for _, name := range metrics {
		groups = append(groups, `{"metric": "`+name+`"}`)
		metric, _ := boardregistry.Catalog().MetricNamed(name)
		led, _ := metric.Led()
		marks := make([]string, 0, share)
		for _, figure := range led[:min(len(led), share)] {
			marks = append(marks, "{"+figure.Name+":"+name+"}")
		}
		if len(metrics) > 1 && len(marks) > 0 {
			leads = append(leads, strings.Join(marks, ", "))
		}
	}
	split := ""
	if len(leads) > 0 {
		split = " The board places at most " + strconv.Itoa(board.MaxReadings) + " figures in " +
			"all, so give each metric its share: " + strings.Join(leads, "; and ") + "."
	}

	return metricCorrectionLead + " " + strings.Join(stated.figures, ", ") + " are figures of " +
		"the " + strings.Join(metrics, " and ") + " " + inflect.Pluralize("metric", "metrics",
		len(metrics)) + ". An answer resting on several of them, or on totals over more records " +
		"than a list shows, belongs on a board in the conversation, not in prose and not as a " +
		"pointer to a table: call show_answer with " + inflect.Pluralize("a group ",
		"a group for each, ", len(groups)) + strings.Join(groups, ", ") + " and, when the " +
		"turn read them, a group of the records " +
		"that need a decision, writing each figure as {figure:metric}, never typed." + split +
		" It is drawn as your reply and ends your turn, so make any other call first. If the " +
		"person only asked to see a list to go through, or asked for one fact, reply again as " +
		"you did." + unread + UnseenCorrection
}
