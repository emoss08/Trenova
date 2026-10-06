package shipmentboardresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/ports/services"
)

func suggestionQueueToModel(q *services.ShipmentSuggestionQueue) *gqlmodel.ShipmentSuggestionQueue {
	items := make([]*gqlmodel.ShipmentSuggestion, 0, len(q.Items))
	for _, item := range q.Items {
		items = append(items, suggestionToModel(item))
	}

	return &gqlmodel.ShipmentSuggestionQueue{
		Items:            items,
		HandledThisShift: q.HandledThisShift,
		Narrated:         q.Narrated,
	}
}

func suggestionToModel(item *services.ShipmentSuggestion) *gqlmodel.ShipmentSuggestion {
	impact := item.Impact
	if impact == nil {
		impact = []string{}
	}

	return &gqlmodel.ShipmentSuggestion{
		Key:        item.Key,
		Kind:       gqlmodel.ShipmentSuggestionKind(item.Kind),
		Tone:       gqlmodel.ShipmentSuggestionTone(item.Tone),
		ShipmentID: base.IDPtr(item.ShipmentID),
		ProNumber:  base.EmptyToNil(item.ProNumber),
		Title:      item.Title,
		Reason:     item.Reason,
		Impact:     impact,
		Primary: &gqlmodel.ShipmentSuggestionAction{
			Type:                  gqlmodel.ShipmentSuggestionActionType(item.Primary.Type),
			Label:                 item.Primary.Label,
			MoveID:                base.IDPtr(item.Primary.MoveID),
			WorkerID:              base.IDPtr(item.Primary.WorkerID),
			TractorID:             base.IDPtr(item.Primary.TractorID),
			CarrierID:             base.IDPtr(item.Primary.CarrierID),
			DetentionOccurrenceID: base.IDPtr(item.Primary.DetentionOccurrenceID),
			Message:               base.EmptyToNil(item.Primary.Message),
		},
		ManualLabel: item.ManualLabel,
		DueAt:       base.IntPtr(item.DueAt),
		Deferred:    item.Deferred,
	}
}
