package boardassemble_test

import "github.com/emoss08/trenova/shared/pulid"

func ids(prefix string, n int) []any {
	out := make([]any, 0, n)
	for range n {
		out = append(out, pulid.MustNew(prefix).String())
	}

	return out
}

func lateBoard() map[string]any {
	shipments := ids("shp_", 6)

	return map[string]any{
		"title":   "Six late shipments on the Dallas lanes",
		"verdict": "Four of the six are held behind [one driver's hours][3], and two are [late at the dock][2].",
		"blocks": []any{
			map[string]any{
				"type":  "figures",
				"title": "Where they stand",
				"figures": []any{
					map[string]any{
						"label":   "Late now",
						"records": shipments,
						"measure": "lateness",
					},
					map[string]any{
						"label":     "Furthest behind",
						"records":   shipments,
						"measure":   "lateness",
						"aggregate": "max",
					},
					map[string]any{"label": "Active shipments", "metric": "activeShipments"},
				},
			},
			map[string]any{
				"type":    "records",
				"title":   "The six, furthest behind first",
				"records": shipments,
				"measure": "lateness",
			},
			map[string]any{"type": "timeline", "record": shipments[0], "half": true},
			map[string]any{
				"type":  "note",
				"tone":  "info",
				"title": "Re-tender the two furthest behind",
				"text":  "A team keeps the appointments.",
				"half":  true,
			},
		},
	}
}
