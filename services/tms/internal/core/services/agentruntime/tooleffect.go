package agentruntime

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/stringutils"
)

const maxToolSummaryRunes = 80

var (
	recordNameKeys     = []string{"name", "displayName", "fullName", "title", "label", "subject"}
	recordNumberKeys   = []string{"proNumber", "code", "number"}
	argumentNameKeys   = []string{"name", "title", "subject"}
	collectionPrefixes = []string{"list_", "search_", "get_"}
)

func (s *Service) ToolEffect(name string) agent.ToolEffect {
	switch name {
	case findToolsName:
		return agent.ToolEffectDiscover
	case askUserName:
		return agent.ToolEffectAsk
	case publishArtifactName:
		return agent.ToolEffectPresent
	}

	return serviceports.EffectOf(s.toolNamed(name))
}

func (s *Service) MarkToolEffects(messages []conversation.Message) {
	if s == nil {
		return
	}

	for idx := range messages {
		message := &messages[idx]
		if message.Role == conversation.RoleTool {
			message.ToolEffect = s.ToolEffect(message.ToolName)
		}
		if len(message.ToolCalls) > 0 {
			message.ToolCalls = s.callsWithEffects(message.ToolCalls)
		}
	}
}

func (s *Service) callsWithEffects(
	calls []conversation.ToolCallRecord,
) []conversation.ToolCallRecord {
	marked := slices.Clone(calls)
	for idx := range marked {
		marked[idx].Effect = s.ToolEffect(marked[idx].Name)
	}

	return marked
}

func summarizeOutcome(call serviceports.ToolCall, outcome toolOutcome) string {
	if outcome.failed {
		return ""
	}
	if outcome.summary != "" {
		return outcome.summary
	}
	if document, ok := outcome.data.(map[string]any); ok {
		return summarizeResult(call.Name, document)
	}
	if outcome.action != nil {
		return argumentSummary(call.Arguments)
	}

	return ""
}

func summarizeResult(toolName string, document any) string {
	switch typed := document.(type) {
	case []any:
		return countSummary(toolName, typed, len(typed), false)
	case map[string]any:
		if report := reportSummary(typed); report != "" {
			return report
		}
		if items, ok := typed["items"].([]any); ok {
			count := len(items)
			if declared, counted := wholeNumber(typed["count"]); counted && declared >= 0 {
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
	subject := toolName
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

	rows, counted := wholeNumber(record["rowCount"])
	finished, _ := record["finished"].(bool)
	if !finished && (!counted || rows <= 0) {
		return name
	}

	return summaryLine(fmt.Sprintf("%s · %d %s",
		name, rows, stringutils.Pluralize("row", "rows", int(rows))))
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
		return summaryLine(strings.TrimSpace(first + " " + last))
	}

	for _, key := range recordNumberKeys {
		if number := summaryText(record[key]); number != "" {
			return number
		}
	}

	return ""
}

func argumentSummary(arguments map[string]any) string {
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

	return summaryLine(text)
}

func summaryLine(text string) string {
	return stringutils.Ellipsize(stringutils.CollapseWhitespace(text), maxToolSummaryRunes)
}
