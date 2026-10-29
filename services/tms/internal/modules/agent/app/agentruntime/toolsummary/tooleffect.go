package toolsummary

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/stringutils"
)

const maxToolSummaryRunes = 80

var (
	recordNameKeys     = []string{"name", "displayName", "fullName", "title", "label", "subject"}
	recordNumberKeys   = []string{"proNumber", "code", "number"}
	argumentNameKeys   = []string{"name", "title", "subject"}
	collectionPrefixes = []string{"list_", "search_", "get_"}
)

func Outcome(call *llmports.ToolCall, outcome *toolcall.Result) string {
	if outcome.Failed {
		return ""
	}
	if outcome.Summary != "" {
		return outcome.Summary
	}
	if document, ok := outcome.Data.(map[string]any); ok {
		return Result(call.Name, document)
	}
	if outcome.Action != nil {
		return ArgumentSummary(call.Arguments)
	}

	return ""
}

func Result(toolName string, document any) string {
	switch typed := document.(type) {
	case []any:
		return countSummary(toolName, typed, len(typed), false)
	case map[string]any:
		if report := reportSummary(typed); report != "" {
			return report
		}
		if items, ok := typed["items"].([]any); ok {
			count := len(items)
			declared, counted := toolresult.WholeNumber(typed["count"])
			if counted && declared >= 0 {
				count = int(declared)
			}
			more, _ := typed["hasMore"].(bool)

			return countSummary(toolName, items, count, more)
		}

		return recordName(typed)
	default:
		return ""
	}
}

func countSummary(toolName string, items []any, count int, more bool) string {
	noun := collectionNoun(toolName)
	switch {
	case count == 0:
		return fmt.Sprintf("No %s", noun)
	case count == 1 && len(items) > 0:
		if record, ok := items[0].(map[string]any); ok {
			if name := recordName(record); name != "" {
				return name
			}
		}

		return "1 result"
	case more:
		return fmt.Sprintf("%d+ %s", count, noun)
	default:
		return fmt.Sprintf("%d %s", count, noun)
	}
}

func collectionNoun(toolName string) string {
	subject := agenttoolports.ReadingOrder(toolName)
	for _, prefix := range collectionPrefixes {
		if trimmed, found := strings.CutPrefix(subject, prefix); found {
			subject = trimmed
			break
		}
	}
	subject = strings.ReplaceAll(subject, "_", " ")
	if strings.HasSuffix(subject, "s") && !strings.HasSuffix(subject, "ss") {
		return subject
	}

	return "results"
}

func reportSummary(record map[string]any) string {
	name := summaryText(record["reportName"])
	if name == "" {
		return ""
	}

	rows, counted := toolresult.WholeNumber(record["rowCount"])
	finished, _ := record["finished"].(bool)
	if !finished && (!counted || rows <= 0) {
		return name
	}

	return Line(fmt.Sprintf("%s · %d %s",
		name, rows, inflect.Pluralize("row", "rows", int(rows))))
}

func recordName(record map[string]any) string {
	for _, key := range recordNameKeys {
		if name := summaryText(record[key]); name != "" {
			return name
		}
	}

	first := summaryText(record["firstName"])
	last := summaryText(record["lastName"])
	if first != "" || last != "" {
		return Line(strings.TrimSpace(first + " " + last))
	}

	for _, key := range recordNumberKeys {
		if number := summaryText(record[key]); number != "" {
			return number
		}
	}

	return ""
}

func ArgumentSummary(arguments map[string]any) string {
	for _, key := range argumentNameKeys {
		if name := summaryText(arguments[key]); name != "" {
			return name
		}
	}

	return ""
}

func summaryText(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}

	return Line(text)
}

func Line(text string) string {
	return stringutils.Ellipsize(stringutils.CollapseWhitespace(text), maxToolSummaryRunes)
}
