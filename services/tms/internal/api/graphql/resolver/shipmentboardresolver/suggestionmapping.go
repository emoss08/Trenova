package shipmentboardresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentsuggestionports"
)

func suggestionQueueToModel(
	q *shipmentsuggestionports.ShipmentSuggestionQueue,
) *gqlmodel.ShipmentSuggestionQueue {
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

func suggestionToModel(
	item *shipmentsuggestionports.ShipmentSuggestion,
) *gqlmodel.ShipmentSuggestion {
	impact := item.Impact
	if impact == nil {
		impact = []string{}
	}

	return &gqlmodel.ShipmentSuggestion{
		Key:        item.Key,
		Kind:       gqlmodel.ShipmentSuggestionKind(item.Kind),
		Tone:       gqlmodel.ShipmentSuggestionTone(item.Tone),
		ShipmentID: gqlid.IDPtr(item.ShipmentID),
		ProNumber:  gqlscalar.EmptyToNil(item.ProNumber),
		Title:      item.Title,
		Reason:     item.Reason,
		Impact:     impact,
		Primary: &gqlmodel.ShipmentSuggestionAction{
			Type:                  gqlmodel.ShipmentSuggestionActionType(item.Primary.Type),
			Label:                 item.Primary.Label,
			MoveID:                gqlid.IDPtr(item.Primary.MoveID),
			WorkerID:              gqlid.IDPtr(item.Primary.WorkerID),
			TractorID:             gqlid.IDPtr(item.Primary.TractorID),
			CarrierID:             gqlid.IDPtr(item.Primary.CarrierID),
			DetentionOccurrenceID: gqlid.IDPtr(item.Primary.DetentionOccurrenceID),
			Message:               gqlscalar.EmptyToNil(item.Primary.Message),
		},
		ManualLabel: item.ManualLabel,
		DueAt:       gqlscalar.IntPtr(item.DueAt),
		Deferred:    item.Deferred,
	}
}
