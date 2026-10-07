package aicontrolsummaryservice

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/aicontrolsummary"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/shipmentnarration"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/numberguard"
)

const (
	schemaName         = "ai_control_summary"
	maxOutputTokens    = 600
	maxSegments        = 16
	maxNarrationChars  = 320
	maxSegmentTextRune = 200
)

const systemPrompt = `You write the one sentence that heads a page where an administrator runs the company's AI agents and model providers.

You are given the facts and a plain version of the sentence. Rewrite it so it reads the way a calm, experienced operations lead would say it: what is true now, and what needs a person.

Rules:
- Use ONLY the numbers and names in the facts. Never add a figure, a percentage, an estimate or a name.
- Keep every link of the plain version, on words that say the same thing, with the same target and providerId. Add no other link.
- At most two short sentences, at most 260 characters in all.
- No greetings, no exclamation marks, no advice beyond what the facts call for, no talk about software or how the facts were gathered.
- Return the sentence as segments: plain text segments, and linked segments carrying their target.`

type draft struct {
	Segments []struct {
		Text       string `json:"text"`
		Strong     bool   `json:"strong"`
		Target     string `json:"target"`
		ProviderID string `json:"providerId"`
	} `json:"segments"`
}

func (s *Service) narrate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	tab aicontrolsummary.Tab,
	facts *aicontrolsummary.Facts,
	plain []aicontrolsummary.Segment,
) ([]aicontrolsummary.Segment, bool) {
	narration, ok := shipmentnarration.Narrate[draft](
		ctx,
		s.completion,
		s.l,
		&services.StructuredCompletionRequest{
			TenantInfo:   tenantInfo,
			Task:         aiprovider.TaskOperationalInsights,
			System:       systemPrompt,
			Context:      promptContext(tab, facts, plain),
			OutputSchema: outputSchema(),
			SchemaName:   schemaName,
			MaxTokens:    maxOutputTokens,
			Attribution:  services.AIUsageAttribution{UserID: tenantInfo.UserID},
		},
	)
	if !ok {
		return nil, false
	}

	return Accept(narration.Draft.segments(plain), facts, plain)
}

func (d *draft) segments(plain []aicontrolsummary.Segment) []aicontrolsummary.Segment {
	tones := make(map[string]aicontrolsummary.Tone, len(plain))
	for _, segment := range plain {
		if segment.Target != "" {
			tones[linkKey(segment.Target, segment.ProviderID)] = segment.Tone
		}
	}

	out := make([]aicontrolsummary.Segment, 0, len(d.Segments))
	for _, segment := range d.Segments {
		target := aicontrolsummary.Target(segment.Target)
		out = append(out, aicontrolsummary.Segment{
			Text:       segment.Text,
			Strong:     segment.Strong,
			Target:     target,
			ProviderID: segment.ProviderID,
			Tone:       tones[linkKey(target, segment.ProviderID)],
		})
	}
	return out
}

// Accept keeps a model's sentence only when it says nothing the facts do not
// and links exactly where the plain sentence links.
func Accept(
	segments []aicontrolsummary.Segment,
	facts *aicontrolsummary.Facts,
	plain []aicontrolsummary.Segment,
) ([]aicontrolsummary.Segment, bool) {
	if len(segments) == 0 || len(segments) > maxSegments {
		return nil, false
	}

	for _, segment := range segments {
		if utf8.RuneCountInString(segment.Text) > maxSegmentTextRune {
			return nil, false
		}
	}
	if !slices.Equal(links(segments), links(plain)) {
		return nil, false
	}

	prose := aicontrolsummary.Text(segments)
	if prose == "" || utf8.RuneCountInString(prose) > maxNarrationChars {
		return nil, false
	}
	if !numberguard.OnlyCounts(prose, facts.Numbers()) {
		return nil, false
	}
	// A single failing provider is named in the plain sentence; a rewording
	// that drops the name has lost what the reader needs to act.
	if len(facts.Failing) == 1 && !strings.Contains(prose, facts.Failing[0].Name) {
		return nil, false
	}

	return segments, true
}

// links are where a sentence leads, sorted, so two sentences can be compared
// whatever order they put them in.
func links(segments []aicontrolsummary.Segment) []string {
	out := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment.Target != "" {
			out = append(out, linkKey(segment.Target, segment.ProviderID))
		}
	}
	slices.Sort(out)
	return out
}

func linkKey(target aicontrolsummary.Target, providerID string) string {
	return string(target) + "#" + providerID
}

func promptContext(
	tab aicontrolsummary.Tab,
	facts *aicontrolsummary.Facts,
	plain []aicontrolsummary.Segment,
) services.DelimitedContext {
	var builder strings.Builder
	write := func(label string, value int) {
		builder.WriteString(label)
		builder.WriteString(": ")
		builder.WriteString(strconv.Itoa(value))
		builder.WriteByte('\n')
	}

	builder.WriteString("Page: ")
	builder.WriteString(string(tab))
	builder.WriteByte('\n')
	write("Agents in all", facts.Agents.Total)
	write("Agents on", facts.Agents.On)
	write("Agents working right now", facts.Agents.Working)
	write("Proposals waiting on a person", facts.Agents.Waiting)
	write("Agents in shadow", facts.Agents.Shadow)
	write("Proposals shadow agents recorded", facts.Agents.ShadowRecorded)
	write("Model providers on", facts.ProvidersOn)
	write("AI tasks no provider can take", facts.Uncovered)
	if facts.Paused {
		builder.WriteString("Every agent is paused and runs in shadow\n")
	}
	for _, failure := range facts.Failing {
		builder.WriteString("Provider failing: ")
		builder.WriteString(failure.Name)
		builder.WriteString(" (")
		builder.WriteString(strconv.Itoa(failure.FailedCalls))
		builder.WriteString(" failed calls)\n")
	}

	builder.WriteString("\nPlain sentence, as segments (text | target | providerId):\n")
	for _, segment := range plain {
		builder.WriteString(segment.Text)
		builder.WriteString(" | ")
		builder.WriteString(string(segment.Target))
		builder.WriteString(" | ")
		builder.WriteString(segment.ProviderID)
		builder.WriteByte('\n')
	}

	return services.DelimitedContext{Sections: []services.ContextSection{{
		Title:   "AI control",
		Trusted: false,
		Content: builder.String(),
	}}}
}

func outputSchema() map[string]any {
	targets := []any{
		"",
		string(aicontrolsummary.TargetWatchtower),
		string(aicontrolsummary.TargetProviders),
		string(aicontrolsummary.TargetRouting),
		string(aicontrolsummary.TargetAgentsShadow),
		string(aicontrolsummary.TargetAgentsWait),
		string(aicontrolsummary.TargetProvider),
	}
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"segments"},
		"properties": map[string]any{
			"segments": map[string]any{
				"type":     "array",
				"maxItems": maxSegments,
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []any{"text", "strong", "target", "providerId"},
					"properties": map[string]any{
						"text":       map[string]any{"type": "string", "maxLength": maxSegmentTextRune},
						"strong":     map[string]any{"type": "boolean"},
						"target":     map[string]any{"type": "string", "enum": targets},
						"providerId": map[string]any{"type": "string"},
					},
				},
			},
		},
	}
}
