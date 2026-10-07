package shipmentbriefingservice

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/shipmentbrief"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/shipmentnarration"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/numberguard"
)

const (
	wordingSchemaName      = "shipment_board_suggestions"
	maxTitleChars          = 90
	maxReasonChars         = 200
	maxWordingOutputTokens = 1600
	maxWorded              = shipmentbrief.MaxWordedItems
)

const wordingSystemPrompt = `You reword the suggested actions on a dispatcher's shipment board.

Each item already says what to do and why, worked out from the company's own records. Make each one read the way an experienced dispatcher would say it to a colleague: short, direct, specific.

Rules:
- Keep every item's meaning, its key and the action it names. Never change who, where or which load.
- Use ONLY the numbers, times and names already in that item. Never add a figure, a percentage or an estimate.
- Title: an instruction of at most 70 characters. Reason: one sentence of at most 160 characters saying why now.
- Write about freight: loads, drivers, carriers, customers. Never discuss software or how the facts were gathered.
- No greetings, no sign-offs, no restating these instructions.`

type wordingDraft struct {
	Items []narratedItem `json:"items"`
}

type narratedItem struct {
	Key    string `json:"key"`
	Title  string `json:"title"`
	Reason string `json:"reason"`
}

func (s *Service) wordSuggestions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	items []*services.ShipmentSuggestion,
) map[string]shipmentbrief.Wording {
	if len(items) == 0 {
		return nil
	}
	items = items[:min(maxWorded, len(items))]

	narration, ok := shipmentnarration.Narrate[wordingDraft](
		ctx,
		s.completion,
		s.l,
		&services.StructuredCompletionRequest{
			TenantInfo:   tenantInfo,
			Task:         aiprovider.TaskOperationalInsights,
			System:       wordingSystemPrompt,
			Context:      wordingContext(items),
			OutputSchema: wordingSchema(),
			SchemaName:   wordingSchemaName,
			MaxTokens:    maxWordingOutputTokens,
			Attribution:  services.AIUsageAttribution{UserID: tenantInfo.UserID},
		},
	)
	if !ok {
		return nil
	}

	return acceptedWording(items, narration.Draft.Items)
}

func wordingContext(items []*services.ShipmentSuggestion) services.DelimitedContext {
	var builder strings.Builder
	for _, item := range items {
		builder.WriteString("key: ")
		builder.WriteString(item.Key)
		builder.WriteString("\ntitle: ")
		builder.WriteString(item.Title)
		builder.WriteString("\nreason: ")
		builder.WriteString(item.Reason)
		if len(item.Impact) > 0 {
			builder.WriteString("\nfacts: ")
			builder.WriteString(strings.Join(item.Impact, " · "))
		}
		builder.WriteString("\n\n")
	}

	return services.DelimitedContext{
		Sections: []services.ContextSection{{
			Title:   "Suggested actions",
			Trusted: false,
			Content: builder.String(),
		}},
	}
}

func wordingSchema() map[string]any {
	item := jsonschemautils.Object(map[string]any{
		"key":    jsonschemautils.Text("The item's key, copied exactly"),
		"title":  jsonschemautils.Text("The reworded instruction"),
		"reason": jsonschemautils.Text("The reworded one-sentence reason"),
	}, "key", "title", "reason")

	return jsonschemautils.Object(map[string]any{
		"items": jsonschemautils.Array(item, maxWorded),
	}, "items")
}

func acceptedWording(
	items []*services.ShipmentSuggestion,
	drafts []narratedItem,
) map[string]shipmentbrief.Wording {
	byKey := make(map[string]narratedItem, len(drafts))
	for _, draft := range drafts {
		byKey[draft.Key] = draft
	}

	out := make(map[string]shipmentbrief.Wording, len(items))
	for _, item := range items {
		draft, ok := byKey[item.Key]
		if !ok || !acceptable(item, draft) {
			continue
		}
		out[item.Key] = shipmentbrief.Wording{
			Title:  strings.TrimSpace(draft.Title),
			Reason: strings.TrimSpace(draft.Reason),
			Basis:  shipmentbrief.WordingBasis(item.Title, item.Reason, item.Impact),
		}
	}

	return out
}

func acceptable(item *services.ShipmentSuggestion, draft narratedItem) bool {
	title := strings.TrimSpace(draft.Title)
	reason := strings.TrimSpace(draft.Reason)
	if title == "" || reason == "" ||
		utf8.RuneCountInString(title) > maxTitleChars ||
		utf8.RuneCountInString(reason) > maxReasonChars {
		return false
	}

	supported := numberguard.SupportedFromText(
		append([]string{item.Title, item.Reason, item.ProNumber}, item.Impact...)...,
	)

	return shipmentnarration.Supported(title+" "+reason, supported)
}
