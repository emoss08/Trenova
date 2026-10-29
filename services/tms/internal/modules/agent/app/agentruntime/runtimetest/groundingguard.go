package runtimetest

func GroundedShipmentSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"customerId": map[string]any{"type": "string"},
			"baseRate":   map[string]any{"type": "number"},
			"commodities": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"commodityId": map[string]any{"type": "string"},
						"pieces":      map[string]any{"type": "integer"},
					},
				},
			},
			"additionalCharges": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "object"},
			},
		},
		"required": []string{"customerId"},
	}
}
