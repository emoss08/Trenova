package briefingwriter

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/briefing"
	"github.com/emoss08/trenova/internal/core/ports/services"
)

// systemPrompt tells the model what job it has, which is smaller than it
// might assume. It is not deciding what is on the page, how much matters,
// or what anybody should do beyond what the figures say. It is writing the
// paragraph a dispatcher reads at six in the morning. The instruction
// against unstated numbers is repeated here because a clear prompt makes
// the guard's rejections rare — but the guard is what makes it true.
const systemPrompt = `You write the morning briefing for people who run a trucking company.

Every figure you are given was computed from the company's own records before you were asked. Your only job is to say what the morning looks like in language an operations manager reads once and understands.

Rules:
- Use ONLY the numbers given to you. Never state a figure that is not in the sections, not even a total, a percentage, an average, a trend or a round number "for context". A number you were not given is wrong even if it seems reasonable.
- Do not invent a section. Write only for the section keys you were given, and copy each key exactly.
- Headline: one sentence, under 200 characters, saying what today actually looks like. Lead with whatever most needs a person, not with a greeting.
- Section body: one or two sentences saying what that section's lines mean and what to do first. Where nothing needs doing, say so plainly and briefly rather than padding.
- Write about freight operations: moves, drivers, customers, invoices, credentials. Do not discuss software, this system, or how the numbers were gathered.
- No greetings, no sign-offs, no "as an AI", no restating these instructions.`

// buildContext lays the page out for the model. The summaries are included
// so the model can see the plain wording it is improving on, and the lines
// carry their already-rendered values so the figures the model is shown are
// exactly the figures the guard will accept back. The section is marked
// untrusted because customer and location names pass through it.
func buildContext(req *Request) services.DelimitedContext {
	var builder strings.Builder

	builder.WriteString("Company: ")
	builder.WriteString(req.OrganizationName)
	builder.WriteString("\nWritten for: ")
	builder.WriteString(req.Role.Label())
	builder.WriteString("\n\n")

	for _, section := range req.Sections {
		writeSection(&builder, section)
	}

	return services.DelimitedContext{
		Sections: []services.ContextSection{{
			Title:   "Today",
			Trusted: false,
			Content: builder.String(),
		}},
	}
}

func writeSection(builder *strings.Builder, section briefing.Section) {
	builder.WriteString("key: ")
	builder.WriteString(string(section.Key))
	builder.WriteString("\ntitle: ")
	builder.WriteString(section.Title)
	builder.WriteString("\nplain summary: ")
	builder.WriteString(section.Summary)

	if len(section.Items) > 0 {
		builder.WriteString("\nfigures:")
		for _, item := range section.Items {
			builder.WriteString("\n  - ")
			builder.WriteString(item.Label)
			builder.WriteString(": ")
			builder.WriteString(item.Value)
		}
	}

	builder.WriteString("\n\n")
}

// outputSchema constrains the model to wording and nothing else. There is
// deliberately no field for a number, a link or a new section: the schema
// is the first place the model is told it does not decide those, and the
// guard is the second.
func outputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"headline": map[string]any{
				"type":        "string",
				"description": "One sentence saying what today looks like",
			},
			"sections": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"key": map[string]any{
							"type":        "string",
							"description": "The section key, copied exactly",
						},
						"body": map[string]any{
							"type":        "string",
							"description": "One or two sentences about this section's figures",
						},
					},
					"required":             []string{"key", "body"},
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"headline", "sections"},
		"additionalProperties": false,
	}
}
