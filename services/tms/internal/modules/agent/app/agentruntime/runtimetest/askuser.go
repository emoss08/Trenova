package runtimetest

func AskFieldArgs() []any {
	return []any{
		map[string]any{
			"key": "customerId", "label": "Customer", "type": "record",
			"recordKind": "customer",
		},
		map[string]any{
			"key": "pickupWindow", "label": "Pickup window", "type": "datetime",
			"hint": "When the truck can load",
		},
		map[string]any{
			"key": "weight", "label": "Weight", "type": "number", "optional": true,
		},
	}
}
