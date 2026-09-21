package agent

type EventKind string

const (
	EventBillingQueueItemException = EventKind("billing_queue.item_exception")
	EventBillingQueueItemOnHold    = EventKind("billing_queue.item_on_hold")
	EventShipmentMoveUnassigned    = EventKind("shipment_move.unassigned")
	EventShipmentCreated           = EventKind("shipment.created")
	EventDocumentExtracted         = EventKind("document.extracted")
	EventServiceFailureDetected    = EventKind("service_failure.detected")
	EventShipmentMoveArrived       = EventKind("shipment_move.arrived")
	EventShipmentMoveDeparted      = EventKind("shipment_move.departed")
)

type EventDescriptor struct {
	Kind        EventKind   `json:"kind"`
	SubjectType SubjectType `json:"subjectType"`
	Label       string      `json:"label"`
	Description string      `json:"description"`
}

var knownEvents = []EventDescriptor{
	{
		Kind:        EventBillingQueueItemException,
		SubjectType: SubjectBillingQueueItem,
		Label:       "Billing item blocked",
		Description: "A billing queue item entered the exception state and cannot be invoiced as it stands.",
	},
	{
		Kind:        EventBillingQueueItemOnHold,
		SubjectType: SubjectBillingQueueItem,
		Label:       "Billing item placed on hold",
		Description: "A billing queue item was put on hold by a person or a rule.",
	},
	{
		Kind:        EventShipmentMoveUnassigned,
		SubjectType: SubjectShipmentMove,
		Label:       "Move needs a driver",
		Description: "A shipment move was created or lost its assignment and has nobody to run it.",
	},
	{
		Kind:        EventShipmentCreated,
		SubjectType: SubjectShipment,
		Label:       "Shipment created",
		Description: "A new shipment was entered, whether by hand, by import, or by an integration.",
	},
	{
		Kind:        EventDocumentExtracted,
		SubjectType: SubjectDocument,
		Label:       "Document read",
		Description: "Document intelligence finished extracting a document's contents.",
	},
	{
		Kind:        EventServiceFailureDetected,
		SubjectType: SubjectShipment,
		Label:       "Service failure detected",
		Description: "A stop was reached late, or missed, and a service failure was opened on the shipment.",
	},
	{
		Kind:        EventShipmentMoveArrived,
		SubjectType: SubjectShipmentMove,
		Label:       "Truck arrived at a stop",
		Description: "A stop's arrival was recorded, by the driver, by a geofence, or by a dispatcher.",
	},
	{
		Kind:        EventShipmentMoveDeparted,
		SubjectType: SubjectShipmentMove,
		Label:       "Truck departed a stop",
		Description: "A stop's departure was recorded, which is when detention and lateness settle.",
	},
}

func KnownEvents() []EventDescriptor {
	out := make([]EventDescriptor, len(knownEvents))
	copy(out, knownEvents)

	return out
}

func (k EventKind) IsValid() bool {
	for _, event := range knownEvents {
		if event.Kind == k {
			return true
		}
	}

	return false
}

func (k EventKind) SubjectType() SubjectType {
	for _, event := range knownEvents {
		if event.Kind == k {
			return event.SubjectType
		}
	}

	return ""
}
