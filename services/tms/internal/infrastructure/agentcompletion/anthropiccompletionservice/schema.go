package anthropiccompletionservice

func buildDiagnosisSchema() map[string]any {
	evidenceItem := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"type": map[string]any{"type": "string"},
			"id":   map[string]any{"type": "string"},
			"note": map[string]any{"type": "string"},
		},
		"required":             []string{"type", "id"},
		"additionalProperties": false,
	}

	evidenceArray := map[string]any{
		"type":     "array",
		"minItems": 1,
		"items":    evidenceItem,
	}

	proposal := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"toolName":   map[string]any{"type": "string"},
			"toolParams": map[string]any{"type": "object"},
			"confidence": map[string]any{"type": "number"},
			"rationale":  map[string]any{"type": "string"},
			"evidence":   evidenceArray,
		},
		"required": []string{
			"toolName",
			"toolParams",
			"confidence",
			"rationale",
			"evidence",
		},
		"additionalProperties": false,
	}

	exception := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"category":       map[string]any{"type": "string"},
			"severity":       map[string]any{"type": "string"},
			"attemptSummary": map[string]any{"type": "string"},
			"blastRadius":    map[string]any{"type": "integer"},
			"evidence":       evidenceArray,
		},
		"required": []string{
			"category",
			"severity",
			"attemptSummary",
			"evidence",
		},
		"additionalProperties": false,
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"proposals":  map[string]any{"type": "array", "items": proposal},
			"exceptions": map[string]any{"type": "array", "items": exception},
		},
		"required":             []string{"proposals", "exceptions"},
		"additionalProperties": false,
	}
}
