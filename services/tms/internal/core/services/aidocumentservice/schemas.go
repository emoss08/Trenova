package aidocumentservice

import (
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
)

const (
	maxExtractFields       = 18
	maxExtractStops        = 8
	maxFieldValueRunes     = 256
	maxEvidenceRunes       = 200
	maxStopNameRunes       = 128
	maxStopAddressRunes    = 160
	maxStopCityRunes       = 80
	maxStopStateRunes      = 16
	maxStopPostalRunes     = 20
	maxStopDateRunes       = 40
	maxStopTimeWindowRunes = 64
)

var extractFieldKeys = aicorrection.PredictedFieldKeys

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
				"maxItems": maxExtractFields,
				"items":    extractFieldSchema(),
			},
			"stops": map[string]any{
				"type":     "array",
				"maxItems": maxExtractStops,
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
				"enum": slices.Clone(extractFieldKeys),
			},
			"value":          map[string]any{"type": "string", "maxLength": maxFieldValueRunes},
			"confidence":     map[string]any{"type": "number"},
			"pageNumber":     map[string]any{"type": "integer"},
			"reviewRequired": map[string]any{"type": "boolean"},
		},
		"required": []string{
			"key",
			"value",
			"confidence",
			"pageNumber",
			"reviewRequired",
		},
	}
}

func extractStopSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"role":                map[string]any{"type": "string"},
			"name":                map[string]any{"type": "string", "maxLength": maxStopNameRunes},
			"addressLine1":        map[string]any{"type": "string", "maxLength": maxStopAddressRunes},
			"addressLine2":        map[string]any{"type": "string", "maxLength": maxStopAddressRunes},
			"city":                map[string]any{"type": "string", "maxLength": maxStopCityRunes},
			"state":               map[string]any{"type": "string", "maxLength": maxStopStateRunes},
			"postalCode":          map[string]any{"type": "string", "maxLength": maxStopPostalRunes},
			"date":                map[string]any{"type": "string", "maxLength": maxStopDateRunes},
			"timeWindow":          map[string]any{"type": "string", "maxLength": maxStopTimeWindowRunes},
			"appointmentRequired": map[string]any{"type": "boolean"},
			"pageNumber":          map[string]any{"type": "integer"},
			"evidenceExcerpt":     map[string]any{"type": "string", "maxLength": maxEvidenceRunes},
			"confidence":          map[string]any{"type": "number"},
			"reviewRequired":      map[string]any{"type": "boolean"},
		},
		"required": []string{
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
		},
	}
}

func extractConflictSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"key": map[string]any{"type": "string", "maxLength": 64},
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
			"evidenceExcerpt": map[string]any{"type": "string", "maxLength": maxEvidenceRunes},
		},
		"required": []string{
			"key", "values", "pageNumbers", "evidenceExcerpt",
		},
	}
}
