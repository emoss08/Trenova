package assistanttest

import (
	"strings"
)

const PagedTool = "billing_list_queue_items"

func QueueItem(n int) map[string]any {
	return map[string]any{
		"id":     "bqi_" + strings.Repeat("0", 3) + string(rune('A'+n)),
		"number": "BQ-" + string(rune('A'+n)),
		"status": "ReadyForReview",
	}
}

func QueuePage(from, count int, more map[string]any) map[string]any {
	items := make([]any, 0, count)
	for n := from; n < from+count; n++ {
		items = append(items, QueueItem(n))
	}
	page := map[string]any{
		"items":   items,
		"columns": []any{"number", "status"},
		"count":   float64(count),
	}
	for key, value := range more {
		page[key] = value
	}

	return page
}
