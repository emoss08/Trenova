package modeladapter

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
)

// WithSchemaInstruction appends the schema to a system prompt for providers that
// cannot enforce it on the wire. Without this a prompted-mode provider is given
// no indication of the shape it is expected to produce, since the schema would
// otherwise travel in a request field the endpoint ignores.
func WithSchemaInstruction(
	system string,
	schema map[string]any,
	mode aiprovider.StructuredOutputMode,
) string {
	if schema == nil || mode == aiprovider.StructuredOutputJSONSchema {
		return system
	}

	encoded, err := RequestJSON.MarshalIndent(schema, "", "  ")
	if err != nil {
		return system
	}

	var builder strings.Builder
	builder.WriteString(system)
	builder.WriteString("\n\n## Required Output Format\n")
	builder.WriteString(
		"Reply with a single JSON object and nothing else. Do not wrap it in markdown " +
			"fences, and do not add any commentary before or after it. The object must " +
			"validate against this JSON Schema:\n\n",
	)
	builder.Write(encoded)

	return builder.String()
}
