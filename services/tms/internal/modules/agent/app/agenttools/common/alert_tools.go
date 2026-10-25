package common

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
)

func ScheduleFormats(params map[string]any) []string {
	formats := StringSliceParam(params, "formats")
	normalized := make([]string, 0, len(formats))
	for _, format := range formats {
		if trimmed := strings.ToLower(strings.TrimSpace(format)); trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	if len(normalized) > 0 {
		return normalized
	}

	return []string{string(report.DefaultScheduleFormat())}
}

// stringSliceParam reads an array of strings, dropping anything that is not
// one rather than failing: a single bad entry should cost its own entry.
func StringSliceParam(params map[string]any, key string) []string {
	raw, _ := params[key].([]any)
	values := make([]string, 0, len(raw))
	for _, entry := range raw {
		if text, ok := entry.(string); ok && strings.TrimSpace(text) != "" {
			values = append(values, text)
		}
	}

	return values
}
