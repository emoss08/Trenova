package runtimetest

func BatchSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"billingQueueItemIds": map[string]any{
				"type":       "array",
				"items":      map[string]any{"type": "string"},
				"x-subsetOf": "billing_queue_item",
			},
			"note": map[string]any{"type": "string"},
		},
	}
}

func IDs(n int) []any {
	out := make([]any, 0, n)
	for i := range n {
		out = append(out, "bqi_"+string(rune('a'+i%26))+string(rune('a'+i/26)))
	}

	return out
}
