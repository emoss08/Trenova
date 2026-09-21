package narrator

import (
	"errors"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/insightservice/detector"
	"github.com/emoss08/trenova/shared/numberguard"
)

// systemPrompt tells the model what job it has, which is smaller than it might
// assume.
//
// It is not deciding what matters, how urgent anything is, or what a person
// should do about it beyond restating what the detector found. It is writing the
// sentence a dispatcher or controller reads at eight in the morning. The
// instruction against unstated numbers is repeated here because a clear prompt
// makes the guard's rejections rare, but the guard is what makes it true: this
// text is a request, and the check afterwards is the enforcement.
const systemPrompt = `You write short, plain explanations of findings from a transportation management system.

Each finding has already been computed from the company's own records. Your only job is to say what it means in language an operations manager understands.

Rules:
- Use ONLY the numbers given to you. Never state a figure that is not in the finding's metrics, not even an estimate, a total, an extrapolation, or a round number "for context". A number you were not given is wrong even if it seems reasonable.
- Do not say how urgent something is or rank findings against each other. That is already decided.
- Write about freight operations: customers, lanes, stops, drivers, billing. Do not discuss software, code, or this system's internals.
- Headline: one sentence, under 160 characters, naming the subject and what changed.
- Narrative: two or three sentences explaining what the numbers show and what plausibly drives it. Say "may" or "often" when you are inferring a cause; you cannot see the cause, only the numbers.
- Recommendation: one sentence naming a concrete next step a person can take today. If the numbers do not support a specific step, say what to look at rather than inventing an action.
- Return an entry for every finding you were given, keyed by its dedupeKey exactly as provided.`

// buildContext lays the findings out for the model.
//
// Values are rendered with FormatForPrompt so the figures the model is shown are
// exactly the figures the guard will accept back. The section is marked
// untrusted because subject names are customer- and location-supplied text that
// happens to be passing through a prompt.
func buildContext(findings []detector.Finding, windowDays int) services.DelimitedContext {
	var builder strings.Builder

	if windowDays > 0 {
		builder.WriteString("Reporting window: the last ")
		builder.WriteString(strconv.Itoa(windowDays))
		builder.WriteString(" days.\n\n")
	}

	for _, finding := range findings {
		writeFinding(&builder, finding)
	}

	return services.DelimitedContext{
		Sections: []services.ContextSection{{
			Title:   "Findings",
			Trusted: false,
			Content: builder.String(),
		}},
	}
}

func writeFinding(builder *strings.Builder, finding detector.Finding) {
	builder.WriteString("dedupeKey: ")
	builder.WriteString(finding.DedupeKey)
	builder.WriteString("\nsubject: ")
	builder.WriteString(finding.Subject)
	builder.WriteString("\nplain summary: ")
	builder.WriteString(finding.Headline)
	builder.WriteString("\nmetrics:\n")

	for _, metric := range finding.Metrics {
		builder.WriteString("  - ")
		builder.WriteString(metric.Label)
		builder.WriteString(": ")
		builder.WriteString(numberguard.FormatForPrompt(metric.Value))
		builder.WriteString(" ")
		builder.WriteString(metric.Unit.String())

		if metric.Baseline != nil {
			builder.WriteString(" (")
			builder.WriteString(metric.BaselineLabel)
			builder.WriteString(": ")
			builder.WriteString(numberguard.FormatForPrompt(*metric.Baseline))
			builder.WriteString(")")
		}

		builder.WriteString("\n")
	}

	builder.WriteString("\n")
}

// outputSchema constrains the model to the three fields that are wording and
// nothing else. There is deliberately no field for a number, a severity or a
// link: the schema is the first place the model is told it does not get to
// decide those, and the guard is the second.
func outputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"insights": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"dedupeKey": map[string]any{
							"type":        "string",
							"description": "The dedupeKey of the finding, copied exactly",
						},
						"headline": map[string]any{
							"type":        "string",
							"description": "One sentence naming the subject and what changed",
						},
						"narrative": map[string]any{
							"type":        "string",
							"description": "Two or three sentences explaining what the numbers show",
						},
						"recommendation": map[string]any{
							"type":        "string",
							"description": "One concrete next step",
						},
					},
					"required": []string{
						"dedupeKey",
						"headline",
						"narrative",
						"recommendation",
					},
					"additionalProperties": false,
				},
			},
		},
		"required":             []string{"insights"},
		"additionalProperties": false,
	}
}

// errorsIsNoProvider separates "this deployment has not configured AI" from "the
// call failed", because the first is a choice and should not be logged as a
// fault on every refresh.
func errorsIsNoProvider(err error) bool {
	return errors.Is(err, services.ErrNoProviderConfigured)
}
