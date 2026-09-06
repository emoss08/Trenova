package resolver

import (
	"fmt"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/sliceutils"
)

func shipmentEventsToModel(events []*shipmentevent.Event) ([]gqlmodel.ShipmentEvent, error) {
	items := make([]gqlmodel.ShipmentEvent, 0, len(events))
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

func shipmentEventToModel(event *shipmentevent.Event) (gqlmodel.ShipmentEvent, error) {
	metadata, err := optionalJSON(event.Metadata)
	if err != nil {
		return nil, err
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	envelope := shipmentEventEnvelope{event: event, metadata: metadata}

	switch event.Type {
	case shipmentevent.TypeShipmentCreated,
		shipmentevent.TypeShipmentUpdated,
		shipmentevent.TypeStatusChanged,
		shipmentevent.TypeShipmentCanceled,
		shipmentevent.TypeShipmentUncanceled:
		return envelope.lifecycle(), nil
	case shipmentevent.TypeOwnershipTransferred:
		return envelope.ownership(), nil
	case shipmentevent.TypeMoveStatusChanged,
		shipmentevent.TypeMoveDeparted,
		shipmentevent.TypeMoveArrived,
		shipmentevent.TypeStopCompleted:
		return envelope.move(), nil
	case shipmentevent.TypeDriverAssigned,
		shipmentevent.TypeDriverReassigned,
		shipmentevent.TypeDriverUnassigned:
		return envelope.assignment(), nil
	case shipmentevent.TypeCarrierAssigned,
		shipmentevent.TypeCarrierUnassigned:
		return envelope.carrier(), nil
	case shipmentevent.TypeTenderOffered,
		shipmentevent.TypeTenderAccepted,
		shipmentevent.TypeTenderDeclined,
		shipmentevent.TypeTenderExpired,
		shipmentevent.TypeTenderWithdrawn,
		shipmentevent.TypeTenderNeedsReview,
		shipmentevent.TypeRoutingGuideExhausted,
		shipmentevent.TypeTenderLateResponse,
		shipmentevent.TypeTenderDeliveryFailed,
		shipmentevent.TypeTenderEntrySkipped,
		shipmentevent.TypeTenderEntryWarned:
		return envelope.tender(), nil
	case shipmentevent.TypeHoldPlaced,
		shipmentevent.TypeHoldUpdated,
		shipmentevent.TypeHoldReleased:
		return envelope.hold(), nil
	case shipmentevent.TypeCommentPosted:
		return envelope.comment(), nil
	}

	return nil, fmt.Errorf(
		"shipment event %s has no GraphQL model for type %q",
		event.ID.String(),
		event.Type,
	)
}

type shipmentEventEnvelope struct {
	event    *shipmentevent.Event
	metadata map[string]any
}

func (e shipmentEventEnvelope) str(key string) *string {
	return sliceutils.StringPtrValue(e.metadata[key])
}

func (e shipmentEventEnvelope) strs(key string) []string {
	return sliceutils.StringSliceValue(e.metadata[key])
}

func (e shipmentEventEnvelope) moveID() *string {
	if id := e.str("moveId"); id != nil {
		return id
	}
	return idPtr(e.event.MoveID)
}

func (e shipmentEventEnvelope) lifecycle() *gqlmodel.ShipmentLifecycleEvent {
	return &gqlmodel.ShipmentLifecycleEvent{
		ID:             e.event.ID.String(),
		OrganizationID: e.event.OrganizationID.String(),
		BusinessUnitID: e.event.BusinessUnitID.String(),
		ShipmentID:     e.event.ShipmentID.String(),
		Type:           gqlmodel.ShipmentEventType(e.event.Type),
		Severity:       gqlmodel.ShipmentEventSeverity(e.event.Severity),
		ActorType:      gqlmodel.ShipmentEventActorType(e.event.ActorType),
		ActorID:        idPtr(e.event.ActorID),
		ActorLabel:     e.event.ActorLabel,
		Summary:        e.event.Summary,
		Metadata:       e.metadata,
		OccurredAt:     int(e.event.OccurredAt),
		CorrelationID:  stringPtrFromValue(e.event.CorrelationID),
		Actor:          e.event.Actor,
		Shipment:       shipmentEventShipmentReferenceToModel(e.event),
		ProNumber:      e.str("proNumber"),
		PreviousStatus: e.str("previousStatus"),
		NewStatus:      e.str("newStatus"),
		Reason:         e.str("reason"),
	}
}

func (e shipmentEventEnvelope) ownership() *gqlmodel.ShipmentOwnershipEvent {
	return &gqlmodel.ShipmentOwnershipEvent{
		ID:              e.event.ID.String(),
		OrganizationID:  e.event.OrganizationID.String(),
		BusinessUnitID:  e.event.BusinessUnitID.String(),
		ShipmentID:      e.event.ShipmentID.String(),
		Type:            gqlmodel.ShipmentEventType(e.event.Type),
		Severity:        gqlmodel.ShipmentEventSeverity(e.event.Severity),
		ActorType:       gqlmodel.ShipmentEventActorType(e.event.ActorType),
		ActorID:         idPtr(e.event.ActorID),
		ActorLabel:      e.event.ActorLabel,
		Summary:         e.event.Summary,
		Metadata:        e.metadata,
		OccurredAt:      int(e.event.OccurredAt),
		CorrelationID:   stringPtrFromValue(e.event.CorrelationID),
		Actor:           e.event.Actor,
		Shipment:        shipmentEventShipmentReferenceToModel(e.event),
		ProNumber:       e.str("proNumber"),
		PreviousOwnerID: e.str("previousOwnerId"),
		NewOwnerID:      e.str("newOwnerId"),
	}
}

func (e shipmentEventEnvelope) move() *gqlmodel.ShipmentMoveEvent {
	return &gqlmodel.ShipmentMoveEvent{
		ID:             e.event.ID.String(),
		OrganizationID: e.event.OrganizationID.String(),
		BusinessUnitID: e.event.BusinessUnitID.String(),
		ShipmentID:     e.event.ShipmentID.String(),
		Type:           gqlmodel.ShipmentEventType(e.event.Type),
		Severity:       gqlmodel.ShipmentEventSeverity(e.event.Severity),
		ActorType:      gqlmodel.ShipmentEventActorType(e.event.ActorType),
		ActorID:        idPtr(e.event.ActorID),
		ActorLabel:     e.event.ActorLabel,
		Summary:        e.event.Summary,
		Metadata:       e.metadata,
		OccurredAt:     int(e.event.OccurredAt),
		CorrelationID:  stringPtrFromValue(e.event.CorrelationID),
		Actor:          e.event.Actor,
		Shipment:       shipmentEventShipmentReferenceToModel(e.event),
		MoveID:         idPtr(e.event.MoveID),
		StopID:         idPtr(e.event.StopID),
		PreviousStatus: e.str("previousStatus"),
		NewStatus:      e.str("newStatus"),
	}
}

func (e shipmentEventEnvelope) assignment() *gqlmodel.ShipmentAssignmentEvent {
	return &gqlmodel.ShipmentAssignmentEvent{
		ID:                e.event.ID.String(),
		OrganizationID:    e.event.OrganizationID.String(),
		BusinessUnitID:    e.event.BusinessUnitID.String(),
		ShipmentID:        e.event.ShipmentID.String(),
		Type:              gqlmodel.ShipmentEventType(e.event.Type),
		Severity:          gqlmodel.ShipmentEventSeverity(e.event.Severity),
		ActorType:         gqlmodel.ShipmentEventActorType(e.event.ActorType),
		ActorID:           idPtr(e.event.ActorID),
		ActorLabel:        e.event.ActorLabel,
		Summary:           e.event.Summary,
		Metadata:          e.metadata,
		OccurredAt:        int(e.event.OccurredAt),
		CorrelationID:     stringPtrFromValue(e.event.CorrelationID),
		Actor:             e.event.Actor,
		Shipment:          shipmentEventShipmentReferenceToModel(e.event),
		MoveID:            idPtr(e.event.MoveID),
		AssignmentID:      idPtr(e.event.AssignmentID),
		PrimaryWorkerID:   e.str("primaryWorkerId"),
		SecondaryWorkerID: e.str("secondaryWorkerId"),
		TractorID:         e.str("tractorId"),
		TrailerID:         e.str("trailerId"),
		DriverName:        e.str("driverName"),
	}
}

func (e shipmentEventEnvelope) carrier() *gqlmodel.ShipmentCarrierEvent {
	return &gqlmodel.ShipmentCarrierEvent{
		ID:             e.event.ID.String(),
		OrganizationID: e.event.OrganizationID.String(),
		BusinessUnitID: e.event.BusinessUnitID.String(),
		ShipmentID:     e.event.ShipmentID.String(),
		Type:           gqlmodel.ShipmentEventType(e.event.Type),
		Severity:       gqlmodel.ShipmentEventSeverity(e.event.Severity),
		ActorType:      gqlmodel.ShipmentEventActorType(e.event.ActorType),
		ActorID:        idPtr(e.event.ActorID),
		ActorLabel:     e.event.ActorLabel,
		Summary:        e.event.Summary,
		Metadata:       e.metadata,
		OccurredAt:     int(e.event.OccurredAt),
		CorrelationID:  stringPtrFromValue(e.event.CorrelationID),
		Actor:          e.event.Actor,
		Shipment:       shipmentEventShipmentReferenceToModel(e.event),
		MoveID:         idPtr(e.event.MoveID),
		CarrierID:      e.str("carrierId"),
		CarrierName:    e.str("carrierName"),
		TotalCost:      e.str("totalCost"),
		Reason:         e.str("reason"),
		ProNumber:      e.str("proNumber"),
	}
}

func (e shipmentEventEnvelope) tender() *gqlmodel.ShipmentTenderEvent {
	return &gqlmodel.ShipmentTenderEvent{
		ID:             e.event.ID.String(),
		OrganizationID: e.event.OrganizationID.String(),
		BusinessUnitID: e.event.BusinessUnitID.String(),
		ShipmentID:     e.event.ShipmentID.String(),
		Type:           gqlmodel.ShipmentEventType(e.event.Type),
		Severity:       gqlmodel.ShipmentEventSeverity(e.event.Severity),
		ActorType:      gqlmodel.ShipmentEventActorType(e.event.ActorType),
		ActorID:        idPtr(e.event.ActorID),
		ActorLabel:     e.event.ActorLabel,
		Summary:        e.event.Summary,
		Metadata:       e.metadata,
		OccurredAt:     int(e.event.OccurredAt),
		CorrelationID:  stringPtrFromValue(e.event.CorrelationID),
		Actor:          e.event.Actor,
		Shipment:       shipmentEventShipmentReferenceToModel(e.event),
		TenderID:       e.str("tenderId"),
		OfferID:        e.str("offerId"),
		MoveID:         e.moveID(),
		CarrierName:    e.str("carrierName"),
		Rank:           intutils.IntPtrValue(e.metadata["rank"]),
		Channel:        e.str("channel"),
		Source:         e.str("source"),
		Reason:         e.str("reason"),
		Action:         e.str("action"),
		Mode:           e.str("mode"),
		Error:          e.str("error"),
		Reasons:        e.strs("reasons"),
		Warnings:       e.strs("warnings"),
	}
}

func (e shipmentEventEnvelope) hold() *gqlmodel.ShipmentHoldEvent {
	return &gqlmodel.ShipmentHoldEvent{
		ID:             e.event.ID.String(),
		OrganizationID: e.event.OrganizationID.String(),
		BusinessUnitID: e.event.BusinessUnitID.String(),
		ShipmentID:     e.event.ShipmentID.String(),
		Type:           gqlmodel.ShipmentEventType(e.event.Type),
		Severity:       gqlmodel.ShipmentEventSeverity(e.event.Severity),
		ActorType:      gqlmodel.ShipmentEventActorType(e.event.ActorType),
		ActorID:        idPtr(e.event.ActorID),
		ActorLabel:     e.event.ActorLabel,
		Summary:        e.event.Summary,
		Metadata:       e.metadata,
		OccurredAt:     int(e.event.OccurredAt),
		CorrelationID:  stringPtrFromValue(e.event.CorrelationID),
		Actor:          e.event.Actor,
		Shipment:       shipmentEventShipmentReferenceToModel(e.event),
		HoldID:         idPtr(e.event.HoldID),
		HoldType:       sliceutils.EnumPtrValue[holdreason.HoldType](e.metadata["holdType"]),
		HoldSeverity: sliceutils.EnumPtrValue[holdreason.HoldSeverity](
			e.metadata["holdSeverity"],
		),
		HoldSource: e.str("holdSource"),
	}
}

func (e shipmentEventEnvelope) comment() *gqlmodel.ShipmentCommentEvent {
	return &gqlmodel.ShipmentCommentEvent{
		ID:             e.event.ID.String(),
		OrganizationID: e.event.OrganizationID.String(),
		BusinessUnitID: e.event.BusinessUnitID.String(),
		ShipmentID:     e.event.ShipmentID.String(),
		Type:           gqlmodel.ShipmentEventType(e.event.Type),
		Severity:       gqlmodel.ShipmentEventSeverity(e.event.Severity),
		ActorType:      gqlmodel.ShipmentEventActorType(e.event.ActorType),
		ActorID:        idPtr(e.event.ActorID),
		ActorLabel:     e.event.ActorLabel,
		Summary:        e.event.Summary,
		Metadata:       e.metadata,
		OccurredAt:     int(e.event.OccurredAt),
		CorrelationID:  stringPtrFromValue(e.event.CorrelationID),
		Actor:          e.event.Actor,
		Shipment:       shipmentEventShipmentReferenceToModel(e.event),
		CommentID:      idPtr(e.event.CommentID),
		CommentBody:    e.str("commentBody"),
		CommentType: sliceutils.EnumPtrValue[gqlmodel.ShipmentCommentType](
			e.metadata["commentType"],
		),
		CommentVisibility: sliceutils.EnumPtrValue[gqlmodel.ShipmentCommentVisibility](
			e.metadata["commentVisibility"],
		),
		CommentPriority: sliceutils.EnumPtrValue[gqlmodel.ShipmentCommentPriority](
			e.metadata["commentPriority"],
		),
		MentionedUserIds: e.strs("mentionedUserIds"),
	}
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
