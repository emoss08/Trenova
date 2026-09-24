package agent

type EventKind string

const (
	EventBillingQueueItemException    = EventKind("billing_queue.item_exception")
	EventBillingQueueItemOnHold       = EventKind("billing_queue.item_on_hold")
	EventShipmentMoveUnassigned       = EventKind("shipment_move.unassigned")
	EventShipmentCreated              = EventKind("shipment.created")
	EventDocumentExtracted            = EventKind("document.extracted")
	EventServiceFailureDetected       = EventKind("service_failure.detected")
	EventShipmentMoveArrived          = EventKind("shipment_move.arrived")
	EventShipmentMoveDeparted         = EventKind("shipment_move.departed")
	EventInsightDetected              = EventKind("insight.detected")
	EventBankReceiptException         = EventKind("bank_receipt.exception")
	EventDetentionOccurrenceOpened    = EventKind("detention.occurrence_opened")
	EventDetentionNoticeDue           = EventKind("detention.notice_due")
	EventWorkerCredentialExpiring     = EventKind("worker_credential.expiring")
	EventCarrierIntelEventOpened      = EventKind("carrier_intel.event_opened")
	EventShipmentMoveCoverageAtRisk   = EventKind("shipment_move.coverage_at_risk")
	EventEDIFileQuarantined           = EventKind("edi.file_quarantined")
	EventInboundMessageClassified     = EventKind("inbound_message.classified")
	EventAccountingConnectionDegraded = EventKind("accounting.connection_degraded")
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
		Description: "A shipment move lost its assignment and has nobody to run it.",
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
	{
		Kind:        EventInsightDetected,
		SubjectType: SubjectInsight,
		Label:       "Insight found",
		Description: "A detector found something new: a finding that was not on the insights page before this refresh.",
	},
	{
		Kind:        EventBankReceiptException,
		SubjectType: SubjectBankReceipt,
		Label:       "Bank receipt needs matching",
		Description: "An imported bank receipt could not be matched to a customer payment on its own and is waiting in the reconciliation queue.",
	},
	{
		Kind:        EventDetentionOccurrenceOpened,
		SubjectType: SubjectDetentionOccurrence,
		Label:       "Detention clock started",
		Description: "A truck went past its free time at a stop and a detention occurrence opened against the shipment.",
	},
	{
		Kind:        EventDetentionNoticeDue,
		SubjectType: SubjectDetentionOccurrence,
		Label:       "Detention notice due",
		Description: "A detention notice's window opened on a policy that leaves sending to a person, so nothing has gone to the customer yet.",
	},
	{
		Kind:        EventWorkerCredentialExpiring,
		SubjectType: SubjectWorker,
		Label:       "Credentials coming due",
		Description: "A driver has one or more credentials expiring inside the compliance horizon, raised once a day however many papers are due.",
	},
	{
		Kind:        EventCarrierIntelEventOpened,
		SubjectType: SubjectCarrierIntelEvent,
		Label:       "Carrier changed",
		Description: "Carrier monitoring opened a finding worth a person's attention: authority, insurance, safety scores or a watch list.",
	},
	{
		Kind:        EventShipmentMoveCoverageAtRisk,
		SubjectType: SubjectShipmentMove,
		Label:       "Move may go uncovered",
		Description: "A move starting inside the coverage window still has nobody on it and the planner could not find a candidate.",
	},
	{
		Kind:        EventEDIFileQuarantined,
		SubjectType: SubjectEDIInboundFile,
		Label:       "EDI file held back",
		Description: "An inbound EDI file could not be processed and is holding in quarantine rather than becoming shipments or updates.",
	},
	{
		Kind:        EventInboundMessageClassified,
		SubjectType: SubjectInboundMessage,
		Label:       "Message classified",
		Description: "A message that arrived on a monitored address was read and turned out to need a decision.",
	},
	{
		Kind:        EventAccountingConnectionDegraded,
		SubjectType: SubjectAccountingConnection,
		Label:       "Accounting connection needs attention",
		Description: "The link to the accounting system stopped working normally: calls are failing, the authorization was revoked, or it expires soon.",
	}}

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
