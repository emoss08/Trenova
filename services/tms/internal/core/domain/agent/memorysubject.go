package agent

import (
	"strings"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const MaxMemorySubjectsPerTurn = 12

type MemoryRecordKind string

const (
	MemoryRecordCustomer         = MemoryRecordKind("customer")
	MemoryRecordLocation         = MemoryRecordKind("location")
	MemoryRecordWorker           = MemoryRecordKind("worker")
	MemoryRecordCarrier          = MemoryRecordKind("carrier")
	MemoryRecordShipment         = MemoryRecordKind("shipment")
	MemoryRecordShipmentMove     = MemoryRecordKind("shipment_move")
	MemoryRecordInvoice          = MemoryRecordKind("invoice")
	MemoryRecordBillingQueueItem = MemoryRecordKind("billing_queue_item")
	MemoryRecordInboundMessage   = MemoryRecordKind("inbound_message")
	MemoryRecordDocument         = MemoryRecordKind("document")
)

func AllMemoryRecordKinds() []MemoryRecordKind {
	return []MemoryRecordKind{
		MemoryRecordCustomer,
		MemoryRecordLocation,
		MemoryRecordWorker,
		MemoryRecordCarrier,
		MemoryRecordShipment,
		MemoryRecordShipmentMove,
		MemoryRecordInvoice,
		MemoryRecordBillingQueueItem,
		MemoryRecordInboundMessage,
		MemoryRecordDocument,
	}
}

func ParseMemoryRecordKind(raw string) (MemoryRecordKind, bool) {
	normalized := strings.ToLower(stringutils.ConvertCamelToSnake(strings.TrimSpace(raw)))
	for _, kind := range AllMemoryRecordKinds() {
		if string(kind) == normalized {
			return kind, true
		}
	}

	return "", false
}

func (k MemoryRecordKind) Subject() (MemorySubjectType, bool) {
	switch k {
	case MemoryRecordCustomer:
		return MemorySubjectCustomer, true
	case MemoryRecordLocation:
		return MemorySubjectLocation, true
	case MemoryRecordWorker:
		return MemorySubjectWorker, true
	case MemoryRecordCarrier:
		return MemorySubjectCarrier, true
	default:
		return "", false
	}
}

func (k MemoryRecordKind) NamesOthers() bool {
	switch k {
	case MemoryRecordShipment,
		MemoryRecordShipmentMove,
		MemoryRecordInvoice,
		MemoryRecordBillingQueueItem,
		MemoryRecordInboundMessage,
		MemoryRecordDocument:
		return true
	default:
		return false
	}
}

type MemoryRecordRef struct {
	Kind MemoryRecordKind
	ID   pulid.ID
}

func MemoryRecordRefOf(kind, id string) (MemoryRecordRef, bool) {
	parsedKind, ok := ParseMemoryRecordKind(kind)
	if !ok {
		return MemoryRecordRef{}, false
	}
	parsedID, err := pulid.Parse(strings.TrimSpace(id))
	if err != nil || parsedID.IsNil() {
		return MemoryRecordRef{}, false
	}

	return MemoryRecordRef{Kind: parsedKind, ID: parsedID}, true
}

type MemoryRelation string

const (
	MemoryRelationDirect  = MemoryRelation("Direct")
	MemoryRelationRelated = MemoryRelation("Related")
)

type MemorySubject struct {
	Type     MemorySubjectType `json:"type"`
	ID       pulid.ID          `json:"id"`
	Relation MemoryRelation    `json:"relation"`
}
