package shipmentimportassistantservice

import (
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

// The tools the import assistant may call. They are declared in the port's own
// shape rather than a vendor SDK's, so the same list serves whichever provider
// the completion router picks for this organization.

func buildTools() []serviceports.ToolSpec { //nolint:funlen // one entry per tool the assistant may call
	return []serviceports.ToolSpec{
		{
			Name:        "accept_field",
			Description: "Accept an extracted field value as correct",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"field_key": map[string]any{"type": "string"},
				},
				"required":             []string{"field_key"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "accept_all_confident",
			Description: "Accept all high-confidence extracted fields at once",
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "set_field_value",
			Description: "Set or override an extracted field value",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"field_key": map[string]any{"type": "string"},
					"value":     map[string]any{"type": "string"},
				},
				"required":             []string{"field_key", "value"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "set_required_field",
			Description: "Set a required shipment field by entity ID after confirming with the user",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"field_key": map[string]any{
						"type": "string",
						"enum": []string{
							"customerId",
							"serviceTypeId",
							"shipmentTypeId",
							"formulaTemplateId",
						},
					},
					"entity_id": map[string]any{"type": "string"},
					"label":     map[string]any{"type": "string"},
				},
				"required":             []string{"field_key", "entity_id", "label"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "search_customers",
			Description: "Search the customer database by name",
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{"query": map[string]any{"type": "string"}},
				"required":             []string{"query"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "search_locations",
			Description: "Search the location database by name, city, or address",
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{"query": map[string]any{"type": "string"}},
				"required":             []string{"query"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "search_service_types",
			Description: "Search available service types",
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{"query": map[string]any{"type": "string"}},
				"required":             []string{"query"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "set_stop_location",
			Description: "Set a stop's location by matching to an existing location in the system",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"stop_index": map[string]any{
						"type":        "integer",
						"description": "0-based index of the stop",
					},
					"location_id": map[string]any{
						"type":        "string",
						"description": "ID of the location to assign",
					},
				},
				"required":             []string{"stop_index", "location_id"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "set_stop_schedule",
			Description: "Set a stop's scheduled pickup/delivery window. Provide ISO 8601 datetime strings.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"stop_index": map[string]any{
						"type":        "integer",
						"description": "0-based index of the stop",
					},
					"window_start": map[string]any{
						"type":        "string",
						"description": "Start time as ISO 8601 (e.g. 2025-03-15T08:00:00Z)",
					},
					"window_end": map[string]any{
						"type":        "string",
						"description": "End time as ISO 8601 (optional)",
					},
				},
				"required":             []string{"stop_index", "window_start"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "set_shipment_field",
			Description: "Set a top-level shipment field like bol, weight, pieces, freightChargeAmount",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"field": map[string]any{
						"type":        "string",
						"description": "Field name (bol, weight, pieces, freightChargeAmount, proNumber)",
					},
					"value": map[string]any{"type": "string", "description": "Value to set"},
				},
				"required":             []string{"field", "value"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "get_customer_requirements",
			Description: "Check if a customer requires BOL for invoicing. Call this after setting the customer.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"customer_id": map[string]any{"type": "string"},
				},
				"required":             []string{"customer_id"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "get_shipment_control",
			Description: "Get the organization's shipment control settings (weight limits, BOL checking, etc.)",
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "search_shipment_types",
			Description: "Search available shipment types",
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{"query": map[string]any{"type": "string"}},
				"required":             []string{"query"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "search_formula_templates",
			Description: "Search available rating methods / formula templates",
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{"query": map[string]any{"type": "string"}},
				"required":             []string{"query"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "add_location",
			Description: "Create a new location in the system from extracted address data. Use this when no matching location exists. The location will be created and its ID returned so you can assign it to a stop.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "Location name (e.g. company/facility name)",
					},
					"address_line1": map[string]any{
						"type":        "string",
						"description": "Street address",
					},
					"city": map[string]any{"type": "string", "description": "City name"},
					"state_abbrev": map[string]any{
						"type":        "string",
						"description": "Two-letter US state abbreviation (e.g. CA, TX, NY)",
					},
					"postal_code": map[string]any{"type": "string", "description": "ZIP code"},
				},
				"required": []string{
					"name",
					"address_line1",
					"city",
					"state_abbrev",
					"postal_code",
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "create_shipment",
			Description: "Create the shipment. Only call this when ALL required fields and stop locations are set. This triggers the actual shipment creation.",
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "suggest_quick_actions",
			Description: "Provide 2-3 action buttons. Call at the end of every response. type='prompt' for confirmations, type='input' when user needs to type a value, type='action' for triggering app actions like creating the shipment.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"suggestions": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"label": map[string]any{
									"type":        "string",
									"description": "Button label",
								},
								"prompt": map[string]any{
									"type":        "string",
									"description": "For type=prompt: message to send. For type=input: prefix before user's typed value (e.g. 'Search for customer ')",
								},
								"type": map[string]any{
									"type":        "string",
									"enum":        []string{"prompt", "input", "action", "date"},
									"description": "prompt = sends message. input = shows text field. action = triggers app action. date = shows a date+time picker.",
								},
								"action": map[string]any{
									"type":        "string",
									"description": "For type=action: the action ID (e.g. 'create_shipment')",
								},
								"placeholder": map[string]any{
									"type":        "string",
									"description": "For type=input: placeholder text in the input field",
								},
								"submitLabel": map[string]any{
									"type":        "string",
									"description": "For type=input: submit button label. Use 'Confirm' for values, 'Search' for queries. Default: 'Search'",
								},
							},
							"required":             []string{"label", "prompt", "type"},
							"additionalProperties": false,
						},
						"maxItems": 3,
					},
				},
				"required":             []string{"suggestions"},
				"additionalProperties": false,
			},
		},
	}
}
