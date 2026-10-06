package shipmentbriefingservice

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
)

const systemPrompt = `You write the one or two sentences at the top of a dispatcher's shipment board.

Every figure you are given was counted from the company's own records. Your job is to say what the board looks like right now in plain words a dispatcher reads in three seconds.

Rules:
- Use ONLY the numbers given to you. Never state a figure that is not in the facts: no totals, percentages or estimates.
- Return the sentence as an ordered list of segments. A segment that names a set of loads carries the filter for that set, copied exactly from the allowed filters; every other segment has an empty filter.
- Keep it under 300 characters. Lead with what needs a person first.
- Write about freight: loads, drivers, carriers, customers. Never discuss software or how the numbers were gathered.
- No greetings, no sign-offs, no restating these instructions.`

func buildContext(
	facts *Facts,
	plain []services.ShipmentBriefingSegment,
) services.DelimitedContext {
	var builder strings.Builder

	writeFact(
		&builder,
		"Loads delivering today",
		facts.DeliveringToday,
		shipment.QuickFilterDeliveringToday,
	)
	writeFact(&builder, "Loads moving on schedule", facts.Moving, shipment.QuickFilterMoving)
	writeFact(&builder, "Loads running late", facts.Late, shipment.QuickFilterLate)
	writeFact(&builder, "Loads still needing "+facts.OperationType.CoverageNoun(), facts.Uncovered,
		shipment.QuickFilterUncovered)
	if facts.LateReason != "" {
		builder.WriteString("Most common reason loads are late: ")
		builder.WriteString(facts.LateReason)
		builder.WriteString("\n")
	}

	builder.WriteString("\nPlain wording you are improving on: ")
	for _, segment := range plain {
		builder.WriteString(segment.Text)
	}
	builder.WriteString("\n")

	return services.DelimitedContext{
		Sections: []services.ContextSection{{
			Title:   "Shipment board",
			Trusted: false,
			Content: builder.String(),
		}},
	}
}

func writeFact(builder *strings.Builder, label string, count int, filter shipment.QuickFilter) {
	builder.WriteString(label)
	builder.WriteString(": ")
	builder.WriteString(strconv.Itoa(count))
	builder.WriteString(" (filter ")
	builder.WriteString(filter.String())
	builder.WriteString(")\n")
}

func outputSchema(allowed []shipment.QuickFilter) map[string]any {
	filters := make([]string, 0, len(allowed)+1)
	filters = append(filters, "")
	for _, filter := range allowed {
		filters = append(filters, filter.String())
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"segments": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"text": map[string]any{
							"type":        "string",
							"description": "A run of the sentence, with its own spacing and punctuation",
						},
						"filter": map[string]any{
							"type":        "string",
							"enum":        filters,
							"description": "The filter for the loads this run names, or empty",
						},
					},
					"required":             []string{"text", "filter"},
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"segments"},
		"additionalProperties": false,
	}
}
