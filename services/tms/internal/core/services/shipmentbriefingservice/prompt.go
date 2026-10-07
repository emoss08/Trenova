package shipmentbriefingservice

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/shared/jsonschemautils"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
)

const systemPrompt = `You write the brief at the top of a dispatcher's shipment board: the first thing they read when they sit down.

Every figure you are given was counted from the company's own records. Say what the day looks like in plain words a dispatcher takes in at a glance: what is moving, what is late and why, what still needs coverage, and what is waiting on money (detention accruing, freight ready to bill).

Rules:
- Use ONLY the numbers given to you. Never state a figure that is not in the facts: no totals, sums, percentages or estimates.
- Return the brief as an ordered list of segments. A segment that names a set of loads carries the filter for that set, copied exactly from the allowed filters; every other segment has an empty filter.
- Two sentences at most, under 420 characters. Lead with what needs a person first; leave out a fact that is zero.
- Write about freight: loads, drivers, carriers, customers. Never discuss software or how the numbers were gathered.
- No greetings, no sign-offs, no restating these instructions; the page greets the reader itself.`

func buildContext(
	facts *Facts,
	plain []services.ShipmentBriefingSegment,
) services.DelimitedContext {
	var builder strings.Builder

	writeFact(&builder, "Loads delivering today", facts.DeliveringToday,
		shipment.QuickFilterDeliveringToday)
	writeFact(&builder, "Loads moving on schedule", facts.Moving, shipment.QuickFilterMoving)
	writeFact(&builder, "Loads running late", facts.Late, shipment.QuickFilterLate)
	writeFact(&builder, "Loads still needing "+facts.OperationType.CoverageNoun(), facts.Uncovered,
		shipment.QuickFilterUncovered)
	writeFact(&builder, "Loads accruing detention", facts.Detention, shipment.QuickFilterDetention)
	writeFact(&builder, "Loads ready to bill", facts.ReadyToBill, shipment.QuickFilterReadyToBill)
	writeFact(&builder, "Loads running on a thin margin", facts.LowMargin,
		shipment.QuickFilterLowMargin)
	builder.WriteString("Suggested actions waiting for a dispatcher: ")
	builder.WriteString(strconv.Itoa(facts.OpenSuggestions))
	builder.WriteString("\n")
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

	segment := jsonschemautils.Object(map[string]any{
		"text": jsonschemautils.Text(
			"A run of the sentence, with its own spacing and punctuation",
		),
		"filter": jsonschemautils.Enum(
			"The filter for the loads this run names, or empty",
			filters...,
		),
	}, "text", "filter")

	return jsonschemautils.Object(map[string]any{
		"segments": jsonschemautils.Array(segment, 0),
	}, "segments")
}
