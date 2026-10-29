package runtimetest

func ShipmentBoard(ids ...string) map[string]any {
	records := make([]any, 0, len(ids))
	for _, id := range ids {
		records = append(records, id)
	}

	return map[string]any{
		"title":   "Late shipments on the Dallas lanes",
		"verdict": "Two are [held at the dock][1].",
		"blocks": []any{
			map[string]any{"type": "records", "records": records, "measure": "lateness"},
			map[string]any{
				"type":  "note",
				"tone":  "info",
				"title": "Re-tender the furthest behind",
				"text":  "A team keeps the appointment.",
			},
		},
	}
}
