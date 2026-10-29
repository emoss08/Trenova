package formulastudioservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/formulastudioports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplatestudioservice"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxSaysRunes       = 400
	maxExplainedEvents = 200
	explainMaxTokens   = 3000
)

const explainSystem = "You are the formula assistant in Trenova's Formula Studio. You explain a " +
	"Tariff formula to a billing analyst from its rate trace, one step at a time. For every " +
	"step listed under Engine trace, write one plain sentence of at most 30 words saying what " +
	"it worked out and what it means for the bill, citing the step by its index. Write " +
	"identifiers in backticks. " + amountsRule + " " + untrustedNote

type traceEvent struct {
	Kind    string             `json:"kind"`
	Name    string             `json:"name"`
	Label   string             `json:"label"`
	Code    *string            `json:"code"`
	Display string             `json:"display"`
	Text    string             `json:"text"`
	Range   *rules.TariffRange `json:"range"`
}

func decodeTrace(raw rules.RawJSON) []traceEvent {
	if len(raw) == 0 {
		return nil
	}
	var events []traceEvent
	if err := sonic.Unmarshal(raw, &events); err != nil {
		return nil
	}
	if len(events) > maxExplainedEvents {
		events = events[:maxExplainedEvents]
	}
	return events
}

func (e *traceEvent) describe() string {
	parts := []string{e.Kind}
	for _, part := range []string{e.Name, e.Label, e.Text} {
		if part != "" {
			parts = append(parts, stringutils.OneLine(part, maxRenderedValueRunes))
		}
	}
	if e.Code != nil && *e.Code != "" {
		parts = append(parts, "code "+*e.Code)
	}
	line := strings.Join(parts, " ")
	if e.Display != "" {
		line += " => " + stringutils.OneLine(e.Display, maxRenderedValueRunes)
	}
	return line
}

func (s *Service) ExplainFormula(
	ctx context.Context,
	req *formulastudioports.ExplainFormulaRequest,
	actor *permissionports.RequestActor,
) (*formulastudioports.ExplainFormulaResponse, error) {
	definition, err := s.assistant(ctx, req.TenantInfo, actor)
	if err != nil {
		return nil, err
	}

	previewReq := &formulatemplatestudioservice.PreviewRequest{
		TenantInfo: req.TenantInfo,
		Source:     req.Draft.Source,
		Inputs:     req.Inputs,
	}
	if req.ShipmentID.IsNotNil() {
		shipmentID := req.ShipmentID
		previewReq.ShipmentID = &shipmentID
	}
	preview, err := s.studio.Preview(ctx, previewReq)
	if err != nil {
		return nil, err
	}
	locale := i18n.FromContext(ctx)
	if !preview.Valid {
		return nil, errortypes.NewValidationError(
			"source",
			errortypes.ErrInvalid,
			"The source must check before it can be explained: {0}",
			diagnosticsText(locale, &rules.TariffCheck{Diagnostics: preview.Diagnostics}),
		)
	}

	events := decodeTrace(preview.Trace)
	resp := &formulastudioports.ExplainFormulaResponse{
		Preview: preview,
		Items:   []formulastudioports.FormulaTraceNote{},
	}
	if len(events) == 0 {
		return resp, nil
	}

	var reply explainReply
	err = s.complete(ctx, &completion{
		tenant:     req.TenantInfo,
		definition: definition,
		feature:    aiusage.FeatureFormulaExplain,
		system:     explainSystem,
		sections: append(draftSections(&req.Draft, preview.Inputs), llmports.ContextSection{
			Title:   "Engine trace",
			Content: traceText(locale, events, preview),
		}),
		schema:     explainSchema(),
		schemaName: "formula_explanation",
		maxTokens:  explainMaxTokens,
	}, &reply)
	if err != nil {
		return nil, modelFailure(definition, err)
	}

	resp.Items = boundExplainItems(reply.Items, events)
	return resp, nil
}

type explainItem struct {
	Index int    `json:"index"`
	Says  string `json:"says"`
}

type explainReply struct {
	Items []explainItem `json:"items"`
}

func explainSchema() map[string]any {
	return objectSchema(map[string]any{
		"items": arraySchema(objectSchema(map[string]any{
			"index": typedField("integer", "The step's index from Engine trace"),
			"says":  typedField("string", "One sentence of at most 30 words"),
		}, "index", "says"), ""),
	}, "items")
}

func traceText(
	locale i18n.Locale,
	events []traceEvent,
	preview *formulatemplateports.PreviewResult,
) string {
	var builder strings.Builder
	for index := range events {
		fmt.Fprintf(&builder, "[%d] %s\n", index, events[index].describe())
	}
	switch {
	case preview.RatedBill != nil:
		builder.WriteString("Bill total: " + preview.RatedBill.Total.String())
		for i := range preview.RatedBill.Lines {
			line := &preview.RatedBill.Lines[i]
			fmt.Fprintf(&builder, "\n- %s %s %s: %s",
				line.Kind, line.Label, line.Code, line.Amount.String())
		}
	case preview.Refusal != nil:
		builder.WriteString("The formula refused to rate: " + preview.Refusal.Localize(locale))
	case preview.Error != nil:
		builder.WriteString(
			"The formula failed (" + preview.Error.Code + "): " + preview.Error.Message,
		)
	}
	return builder.String()
}

func boundExplainItems(
	items []explainItem,
	events []traceEvent,
) []formulastudioports.FormulaTraceNote {
	says := make(map[int]string, len(events))
	for _, item := range items {
		if item.Index < 0 || item.Index >= len(events) {
			continue
		}
		if _, taken := says[item.Index]; taken {
			continue
		}
		if text := stringutils.CollapseWhitespace(item.Says); text != "" {
			says[item.Index] = stringutils.Ellipsize(text, maxSaysRunes)
		}
	}

	notes := make([]formulastudioports.FormulaTraceNote, 0, len(events))
	for index := range events {
		text, ok := says[index]
		if !ok {
			text = fallbackSays(&events[index])
		}
		notes = append(notes, formulastudioports.FormulaTraceNote{
			Index: index,
			Range: events[index].Range,
			Says:  text,
		})
	}
	return notes
}

func fallbackSays(event *traceEvent) string {
	switch {
	case event.Name != "" && event.Display != "":
		return fmt.Sprintf("%#q comes to %s on this load.", event.Name, event.Display)
	case event.Label != "" && event.Display != "":
		return fmt.Sprintf("%q bills %s on this load.", event.Label, event.Display)
	case event.Text != "":
		return event.Text
	default:
		return fmt.Sprintf("The formula took a %s step here.", event.Kind)
	}
}
