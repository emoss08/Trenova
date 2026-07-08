package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/shared/sliceutils"
)

func shipmentEventsToModel(events []*shipmentevent.Event) ([]*gqlmodel.ShipmentEvent, error) {
	items := make([]*gqlmodel.ShipmentEvent, 0, len(events))
	for _, event := range events {
		item, err := shipmentEventToModel(event)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func shipmentEventTypesFromGraphQL(
	values []gqlmodel.ShipmentEventType,
) []shipmentevent.Type {
	if len(values) == 0 {
		return nil
	}
	types := make([]shipmentevent.Type, 0, len(values))
	for _, value := range values {
		types = append(types, shipmentevent.Type(value))
	}
	return types
}

func shipmentEventToModel(event *shipmentevent.Event) (*gqlmodel.ShipmentEvent, error) {
	metadata, err := optionalJSON(event.Metadata)
	if err != nil {
		return nil, err
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	return &gqlmodel.ShipmentEvent{
		ID:             event.ID.String(),
		OrganizationID: event.OrganizationID.String(),
		BusinessUnitID: event.BusinessUnitID.String(),
		ShipmentID:     event.ShipmentID.String(),
		MoveID:         idPtr(event.MoveID),
		StopID:         idPtr(event.StopID),
		AssignmentID:   idPtr(event.AssignmentID),
		CommentID:      idPtr(event.CommentID),
		HoldID:         idPtr(event.HoldID),
		Type:           gqlmodel.ShipmentEventType(event.Type),
		Severity:       gqlmodel.ShipmentEventSeverity(event.Severity),
		ActorType:      gqlmodel.ShipmentEventActorType(event.ActorType),
		ActorID:        idPtr(event.ActorID),
		ActorLabel:     event.ActorLabel,
		Summary:        event.Summary,
		ProNumber:      sliceutils.StringPtrValue(metadata["proNumber"]),
		PreviousStatus: sliceutils.StringPtrValue(metadata["previousStatus"]),
		NewStatus:      sliceutils.StringPtrValue(metadata["newStatus"]),
		Reason:         sliceutils.StringPtrValue(metadata["reason"]),
		PreviousOwnerID: sliceutils.StringPtrValue(
			metadata["previousOwnerId"],
		),
		NewOwnerID:        sliceutils.StringPtrValue(metadata["newOwnerId"]),
		PrimaryWorkerID:   sliceutils.StringPtrValue(metadata["primaryWorkerId"]),
		SecondaryWorkerID: sliceutils.StringPtrValue(metadata["secondaryWorkerId"]),
		TractorID:         sliceutils.StringPtrValue(metadata["tractorId"]),
		TrailerID:         sliceutils.StringPtrValue(metadata["trailerId"]),
		DriverName:        sliceutils.StringPtrValue(metadata["driverName"]),
		HoldType:          sliceutils.StringPtrValue(metadata["holdType"]),
		HoldSeverity:      sliceutils.StringPtrValue(metadata["holdSeverity"]),
		HoldSource:        sliceutils.StringPtrValue(metadata["holdSource"]),
		CommentBody:       sliceutils.StringPtrValue(metadata["commentBody"]),
		CommentType:       sliceutils.StringPtrValue(metadata["commentType"]),
		CommentVisibility: sliceutils.StringPtrValue(metadata["commentVisibility"]),
		CommentPriority:   sliceutils.StringPtrValue(metadata["commentPriority"]),
		MentionedUserIds:  sliceutils.StringSliceValue(metadata["mentionedUserIds"]),
		Metadata:          metadata,
		OccurredAt:        int(event.OccurredAt),
		CorrelationID:     stringPtrFromValue(event.CorrelationID),
		Actor:             event.Actor,
		Shipment:          shipmentEventShipmentReferenceToModel(event),
	}, nil
}

func shipmentEventShipmentReferenceToModel(
	event *shipmentevent.Event,
) *gqlmodel.ShipmentEventShipmentReference {
	if event.Shipment == nil {
		return nil
	}
	return &gqlmodel.ShipmentEventShipmentReference{
		ID:        idPtr(event.Shipment.ID),
		ProNumber: stringPtrFromValue(event.Shipment.ProNumber),
	}
}
