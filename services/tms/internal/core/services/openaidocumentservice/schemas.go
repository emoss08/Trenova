package openaidocumentservice

func buildRouteSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"shouldExtract": map[string]any{"type": "boolean"},
			"documentKind":  map[string]any{"type": "string"},
			"confidence":    map[string]any{"type": "number"},
			"signals": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
			},
			"reviewStatus":        map[string]any{"type": "string"},
			"classifierSource":    map[string]any{"type": "string"},
			"providerFingerprint": map[string]any{"type": "string"},
			"reason":              map[string]any{"type": "string"},
		},
		"required": []string{
			"shouldExtract",
			"documentKind",
			"confidence",
			"signals",
			"reviewStatus",
			"classifierSource",
			"providerFingerprint",
			"reason",
		},
	}
}

func buildExtractSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"documentKind":      map[string]any{"type": "string"},
			"overallConfidence": map[string]any{"type": "number"},
			"reviewStatus":      map[string]any{"type": "string"},
			"missingFields": map[string]any{
				"type":     "array",
				"maxItems": 12,
				"items":    map[string]any{"type": "string", "maxLength": 64},
			},
			"signals": map[string]any{
				"type":     "array",
				"maxItems": 8,
				"items":    map[string]any{"type": "string", "maxLength": 120},
			},
			"fields": map[string]any{
				"type":     "array",
				"maxItems": 18,
				"items":    extractFieldSchema(),
			},
			"stops": map[string]any{
				"type":     "array",
				"maxItems": 8,
				"items":    extractStopSchema(),
			},
			"conflicts": map[string]any{
				"type":     "array",
				"maxItems": 6,
				"items":    extractConflictSchema(),
			},
		},
		"required": []string{
			"documentKind",
			"overallConfidence",
			"reviewStatus",
			"missingFields",
			"signals",
			"fields",
			"stops",
			"conflicts",
		},
	}
}

func extractFieldSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"key": map[string]any{
				"type": "string",
				"enum": []string{
					"loadNumber", "referenceNumber", "shipper", "consignee",
					"rate", "equipmentType", "commodity",
					"pickupDate", "deliveryDate", "pickupWindow", "deliveryWindow",
					"pickupNumber", "deliveryNumber",
					"appointmentNumber", "bol", "poNumber", "scac", "proNumber",
					"paymentTerms", "billTo",
					"carrierName", "carrierContact", "containerNumber",
					"trailerNumber", "tractorNumber",
					"fuelSurcharge", "serviceType",
				},
			},
			"label":           map[string]any{"type": "string", "maxLength": 64},
			"value":           map[string]any{"type": "string", "maxLength": 256},
			"confidence":      map[string]any{"type": "number"},
			"evidenceExcerpt": map[string]any{"type": "string", "maxLength": 200},
			"pageNumber":      map[string]any{"type": "integer"},
			"reviewRequired":  map[string]any{"type": "boolean"},
			"conflict":        map[string]any{"type": "boolean"},
			"source":          map[string]any{"type": "string", "maxLength": 32},
			"alternativeValues": map[string]any{
				"type":     "array",
				"maxItems": 4,
				"items":    map[string]any{"type": "string", "maxLength": 128},
			},
		},
		"required": []string{
			"key",
			"label",
			"value",
			"confidence",
			"evidenceExcerpt",
			"pageNumber",
			"reviewRequired",
			"conflict",
			"source",
			"alternativeValues",
		},
	}
}

func extractStopSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"sequence":            map[string]any{"type": "integer"},
			"role":                map[string]any{"type": "string"},
			"name":                map[string]any{"type": "string", "maxLength": 128},
			"addressLine1":        map[string]any{"type": "string", "maxLength": 160},
			"addressLine2":        map[string]any{"type": "string", "maxLength": 160},
			"city":                map[string]any{"type": "string", "maxLength": 80},
			"state":               map[string]any{"type": "string", "maxLength": 16},
			"postalCode":          map[string]any{"type": "string", "maxLength": 20},
			"date":                map[string]any{"type": "string", "maxLength": 40},
			"timeWindow":          map[string]any{"type": "string", "maxLength": 64},
			"appointmentRequired": map[string]any{"type": "boolean"},
			"pageNumber":          map[string]any{"type": "integer"},
			"evidenceExcerpt":     map[string]any{"type": "string", "maxLength": 200},
			"confidence":          map[string]any{"type": "number"},
			"reviewRequired":      map[string]any{"type": "boolean"},
			"source":              map[string]any{"type": "string", "maxLength": 32},
		},
		"required": []string{
			"sequence",
			"role",
			"name",
			"addressLine1",
			"addressLine2",
			"city",
			"state",
			"postalCode",
			"date",
			"timeWindow",
			"appointmentRequired",
			"pageNumber",
			"evidenceExcerpt",
			"confidence",
			"reviewRequired",
			"source",
		},
	}
}

func extractConflictSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"key":   map[string]any{"type": "string", "maxLength": 64},
			"label": map[string]any{"type": "string", "maxLength": 64},
			"values": map[string]any{
				"type":     "array",
				"maxItems": 4,
				"items":    map[string]any{"type": "string", "maxLength": 128},
			},
			"pageNumbers": map[string]any{
				"type":     "array",
				"maxItems": 6,
				"items":    map[string]any{"type": "integer"},
			},
			"evidenceExcerpt": map[string]any{"type": "string", "maxLength": 200},
			"source":          map[string]any{"type": "string", "maxLength": 32},
		},
		"required": []string{
			"key", "label", "values", "pageNumbers", "evidenceExcerpt", "source",
		},
	}
}
