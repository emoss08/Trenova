package modeladapter

import (
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

const (
	untrustedOpenTag  = "<untrusted_data>"
	untrustedCloseTag = "</untrusted_data>"
)

// BuildContextText renders a delimited context, fencing every untrusted section
// so prompt injection carried in customer data cannot be read as instruction.
func BuildContextText(deliminated serviceports.DelimitedContext) string {
	var builder strings.Builder

	for _, section := range deliminated.Sections {
		title := strings.TrimSpace(section.Title)
		if title != "" {
			builder.WriteString("## ")
			builder.WriteString(title)
			builder.WriteString("\n")
		}

		if section.Trusted {
			builder.WriteString(section.Content)
			builder.WriteString("\n\n")
			continue
		}

		builder.WriteString(untrustedOpenTag)
		builder.WriteString("\n")
		builder.WriteString(neutralizeUntrusted(section.Content))
		builder.WriteString("\n")
		builder.WriteString(untrustedCloseTag)
		builder.WriteString("\n\n")
	}

	return strings.TrimSpace(builder.String())
}

func neutralizeUntrusted(content string) string {
	return strings.ReplaceAll(content, untrustedCloseTag, "<\\/untrusted_data>")
}

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

	encoded, err := sonic.MarshalIndent(schema, "", "  ")
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
