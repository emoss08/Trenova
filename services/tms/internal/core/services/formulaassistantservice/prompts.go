package formulaassistantservice

import (
	"strings"

	"github.com/bytedance/sonic"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
)

const generateSystemPrompt = `You are a rating formula author for Trenova, a transportation management system. You write expressions in the expr expression language that compute a freight or accessorial charge in USD for a shipment.

Rules:
- Use only the variables and functions listed in the reference sections. Never invent a variable or function.
- The expression must evaluate to a single number.
- Ternary conditionals use the form: condition ? whenTrue : whenFalse.
- When an input the formula needs is not a shipment variable, declare it as a custom variable with a name, type, description, and a sensible default value.
- Only call lookup functions with a rate table code from the available rate tables section, and match the function to the table's axis count: lookup()/lookupOr() for single-axis tables, lookup2()/lookup2Or() for two-axis tables. When no table fits, do not use lookups.
- Prefer round(x, 2) on the final result when the computation can produce fractional cents.
- The explanation must describe the formula in plain English for a billing clerk, not a programmer.`

const explainSystemPrompt = `You explain Trenova rating formulas to billing staff who are not programmers. You are given an expression in the expr expression language, along with the variables and functions it can reference.

Rules:
- Describe, step by step, how the charge is computed and which shipment values change it.
- Call out conditions (surcharges that apply only sometimes), minimums, maximums, and rate table lookups explicitly.
- If the expression references a variable or function that is not in the reference sections, say so plainly.
- Write plain English. Never include code unless quoting a fragment of the expression to anchor a point.`

func buildGenerateContext(
	description *formulatemplatetypes.SchemaDescription,
	lookupTables []string,
	templateType, instruction string,
) serviceports.DelimitedContext {
	sections := []serviceports.ContextSection{
		{
			Title:   "Available Variables",
			Trusted: true,
			Content: marshalReference(description.Variables),
		},
		{
			Title:   "Available Functions",
			Trusted: true,
			Content: marshalReference(description.Functions),
		},
		{
			Title:   "Available Rate Tables",
			Trusted: true,
			Content: formatLookupTables(lookupTables),
		},
		{
			Title:   "Template Type",
			Trusted: true,
			Content: templateType,
		},
		{
			Title:   "User Request",
			Trusted: false,
			Content: instruction,
		},
	}

	return serviceports.DelimitedContext{Sections: sections}
}

func buildExplainContext(
	description *formulatemplatetypes.SchemaDescription,
	expression string,
) serviceports.DelimitedContext {
	sections := []serviceports.ContextSection{
		{
			Title:   "Available Variables",
			Trusted: true,
			Content: marshalReference(description.Variables),
		},
		{
			Title:   "Available Functions",
			Trusted: true,
			Content: marshalReference(description.Functions),
		},
		{
			Title:   "Expression To Explain",
			Trusted: false,
			Content: expression,
		},
	}

	return serviceports.DelimitedContext{Sections: sections}
}

func marshalReference(value any) string {
	encoded, err := sonic.MarshalIndent(value, "", "  ")
	if err != nil {
		return ""
	}

	return string(encoded)
}

func formatLookupTables(codes []string) string {
	if len(codes) == 0 {
		return "No rate tables are available. Do not use lookup(), lookupOr(), lookup2(), or lookup2Or()."
	}

	return strings.Join(codes, "\n")
}

func generateOutputSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"expression", "variables", "explanation"},
		"properties": map[string]any{
			"expression": map[string]any{
				"type":        "string",
				"description": "The expr expression that computes the charge",
			},
			"variables": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"name", "type", "description"},
					"properties": map[string]any{
						"name":         map[string]any{"type": "string"},
						"type":         map[string]any{"type": "string", "enum": []string{"Number", "String", "Boolean"}},
						"description":  map[string]any{"type": "string"},
						"defaultValue": map[string]any{"type": []string{"number", "string", "boolean", "null"}},
					},
				},
			},
			"explanation": map[string]any{
				"type":        "string",
				"description": "A plain-English explanation of how the formula computes the charge",
			},
		},
	}
}

func explainOutputSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"explanation"},
		"properties": map[string]any{
			"explanation": map[string]any{
				"type":        "string",
				"description": "A plain-English explanation of the expression",
			},
		},
	}
}
