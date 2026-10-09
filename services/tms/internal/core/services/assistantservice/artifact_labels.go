package assistantservice

import (
	"strconv"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	// maxTableLabels bounds how many first-column values a shown table hands
	// the loop. They cross from the tool's activity to workflow code with the
	// outcome, and half of a reprint's rows are enough to recognise it.
	maxTableLabels    = 200
	maxTableLabelRune = 80
)

// tableLabels is a table artifact's first column, row by row, as far as its
// kept rows go: the values a reply that reprints the table writes down its
// left edge. A table built here keeps its columns as display columns; one
// read back, or a report's, keeps them as decoded JSON, and both are read.
func tableLabels(artifact *assistantartifact.Artifact) []string {
	if !tabularKind(artifact.Kind) {
		return nil
	}
	rows, _ := artifact.Payload["rows"].([]any)
	if len(rows) == 0 {
		return nil
	}
	key := firstColumnKey(artifact.Payload["columns"])

	labels := make([]string, 0, min(len(rows), maxTableLabels))
	for _, row := range rows {
		if len(labels) == maxTableLabels {
			break
		}
		if label := labelOf(firstCell(row, key)); label != "" {
			labels = append(labels, label)
		}
	}

	return labels
}

func firstColumnKey(columns any) string {
	switch typed := columns.(type) {
	case []assistantartifact.DisplayColumn:
		if len(typed) > 0 {
			return typed[0].Key
		}
	case []string:
		if len(typed) > 0 {
			return typed[0]
		}
	case []any:
		if len(typed) == 0 {
			return ""
		}
		switch first := typed[0].(type) {
		case string:
			return first
		case map[string]any:
			for _, field := range []string{"key", "field", "name"} {
				if key := typeutils.StringOfTrimmed(first[field]); key != "" {
					return key
				}
			}
		}
	}

	return ""
}

func firstCell(row any, key string) any {
	switch typed := row.(type) {
	case map[string]any:
		if key == "" {
			return nil
		}

		return typed[key]
	case []any:
		if len(typed) > 0 {
			return typed[0]
		}
	}

	return nil
}

func labelOf(value any) string {
	var text string
	switch typed := value.(type) {
	case string:
		text = typed
	case float64:
		text = strconv.FormatFloat(typed, 'f', -1, 64)
	case map[string]any:
		for _, field := range []string{"label", "text", "name"} {
			if text = typeutils.StringOfTrimmed(typed[field]); text != "" {
				break
			}
		}
	}
	if utf8.RuneCountInString(text) > maxTableLabelRune {
		return string([]rune(text)[:maxTableLabelRune])
	}

	return text
}
