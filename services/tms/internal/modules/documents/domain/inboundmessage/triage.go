package inboundmessage

import "github.com/emoss08/trenova/shared/pulid"

type Parking string

const (
	ParkingSnoozed       = Parking("Snoozed")
	ParkingAwaitingReply = Parking("AwaitingReply")
)

func (p Parking) IsValid() bool {
	switch p {
	case ParkingSnoozed, ParkingAwaitingReply:
		return true
	default:
		return false
	}
}

type SnoozeEvent string

const (
	SnoozeEventShipmentDelivered = SnoozeEvent("ShipmentDelivered")
	SnoozeEventDocumentAttached  = SnoozeEvent("DocumentAttached")
	SnoozeEventInvoicePaid       = SnoozeEvent("InvoicePaid")
)

func (e SnoozeEvent) IsValid() bool {
	switch e {
	case SnoozeEventShipmentDelivered, SnoozeEventDocumentAttached, SnoozeEventInvoicePaid:
		return true
	default:
		return false
	}
}

func AllSnoozeEvents() []SnoozeEvent {
	return []SnoozeEvent{
		SnoozeEventShipmentDelivered,
		SnoozeEventDocumentAttached,
		SnoozeEventInvoicePaid,
	}
}

type NovaAction string

const (
	NovaActionFileOnShipment  = NovaAction("FileOnShipment")
	NovaActionFileAndQueuePay = NovaAction("FileAndQueuePay")
	NovaActionDraftReply      = NovaAction("DraftReply")
	NovaActionDraftLoad       = NovaAction("DraftLoad")
	NovaActionLinkBySender    = NovaAction("LinkBySender")
	NovaActionIgnore          = NovaAction("Ignore")
	NovaActionAssign          = NovaAction("Assign")
)

func (a NovaAction) IsValid() bool {
	switch a {
	case NovaActionFileOnShipment, NovaActionFileAndQueuePay, NovaActionDraftReply,
		NovaActionDraftLoad, NovaActionLinkBySender, NovaActionIgnore, NovaActionAssign:
		return true
	default:
		return false
	}
}

func AllNovaActions() []NovaAction {
	return []NovaAction{
		NovaActionFileOnShipment,
		NovaActionFileAndQueuePay,
		NovaActionDraftReply,
		NovaActionDraftLoad,
		NovaActionLinkBySender,
		NovaActionIgnore,
		NovaActionAssign,
	}
}

func (a NovaAction) Settles() bool {
	switch a {
	case NovaActionFileOnShipment, NovaActionFileAndQueuePay, NovaActionIgnore:
		return true
	case NovaActionDraftReply, NovaActionDraftLoad, NovaActionLinkBySender, NovaActionAssign:
		return false
	default:
		return false
	}
}

type FactTone string

const (
	FactToneNeutral = FactTone("Neutral")
	FactToneWarning = FactTone("Warning")
)

type Fact struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Value   string   `json:"value"`
	Quote   string   `json:"quote"`
	Tone    FactTone `json:"tone"`
	Flagged bool     `json:"flagged"`
}

type ConflictKind string

const (
	ConflictDuplicateDocument = ConflictKind("DuplicateDocument")
	ConflictOverlappingLoad   = ConflictKind("OverlappingLoad")
)

type ConflictResolution string

const (
	ConflictReplaceExisting = ConflictResolution("ReplaceExisting")
	ConflictKeepBoth        = ConflictResolution("KeepBoth")
	ConflictSkipDuplicate   = ConflictResolution("SkipDuplicate")
	ConflictConfirmNew      = ConflictResolution("ConfirmNew")
)

func (r ConflictResolution) FitsKind(kind ConflictKind) bool {
	switch kind {
	case ConflictDuplicateDocument:
		return r == ConflictReplaceExisting || r == ConflictKeepBoth
	case ConflictOverlappingLoad:
		return r == ConflictSkipDuplicate || r == ConflictConfirmNew
	default:
		return false
	}
}

type Conflict struct {
	ID                 string             `json:"id"`
	Kind               ConflictKind       `json:"kind"`
	AttachmentID       pulid.ID           `json:"attachmentId,omitempty"`
	ExistingDocumentID pulid.ID           `json:"existingDocumentId,omitempty"`
	ExistingShipmentID pulid.ID           `json:"existingShipmentId,omitempty"`
	ExistingFiledAt    int64              `json:"existingFiledAt,omitempty"`
	ExistingAmount     string             `json:"existingAmount,omitempty"`
	NewAmount          string             `json:"newAmount,omitempty"`
	LoadIndex          int                `json:"loadIndex,omitempty"`
	Resolution         ConflictResolution `json:"resolution,omitempty"`
	ResolvedAt         int64              `json:"resolvedAt,omitempty"`
	ResolvedBy         pulid.ID           `json:"resolvedBy,omitempty"`
}

type StopsDecision string

const (
	StopsDecisionWaive   = StopsDecision("Waive")
	StopsDecisionStandBy = StopsDecision("StandBy")
)

func (d StopsDecision) IsValid() bool {
	return d == StopsDecisionWaive || d == StopsDecisionStandBy
}

type ExceptionKind string

const (
	ExceptionShortage = ExceptionKind("Shortage")
	ExceptionOverage  = ExceptionKind("Overage")
	ExceptionDamage   = ExceptionKind("Damage")
)

func (k ExceptionKind) IsValid() bool {
	switch k {
	case ExceptionShortage, ExceptionOverage, ExceptionDamage:
		return true
	default:
		return false
	}
}

type DetentionWaiver struct {
	Amount            string   `json:"amount,omitempty"`
	Currency          string   `json:"currency,omitempty"`
	InvoiceNumbers    []string `json:"invoiceNumbers,omitempty"`
	CreditMemoNumbers []string `json:"creditMemoNumbers,omitempty"`
	CreditPending     bool     `json:"creditPending,omitempty"`
}

func (w *DetentionWaiver) Credited() bool {
	return w != nil && len(w.InvoiceNumbers) > 0
}

type ActionState struct {
	StopsDecision     StopsDecision    `json:"stopsDecision,omitempty"`
	StopsDecidedAt    int64            `json:"stopsDecidedAt,omitempty"`
	DetentionWaiver   *DetentionWaiver `json:"detentionWaiver,omitempty"`
	ExceptionKind     ExceptionKind    `json:"exceptionKind,omitempty"`
	ExceptionLoggedAt int64            `json:"exceptionLoggedAt,omitempty"`
	SenderBlockedAt   int64            `json:"senderBlockedAt,omitempty"`
	LinkedAt          int64            `json:"linkedAt,omitempty"`
	CreatedShipmentID pulid.ID         `json:"createdShipmentId,omitempty"`
	CreatedShipmentAt int64            `json:"createdShipmentAt,omitempty"`
	HandedOffTo       pulid.ID         `json:"handedOffTo,omitempty"`
	HandedOffAt       int64            `json:"handedOffAt,omitempty"`
}

func (s *ActionState) CreditPending() bool {
	return s.StopsDecision == StopsDecisionWaive &&
		s.DetentionWaiver != nil && s.DetentionWaiver.CreditPending
}

type LoadFieldKey string

const (
	LoadFieldCustomer     = LoadFieldKey("Customer")
	LoadFieldOrigin       = LoadFieldKey("Origin")
	LoadFieldDestination  = LoadFieldKey("Destination")
	LoadFieldPickup       = LoadFieldKey("Pickup")
	LoadFieldDelivery     = LoadFieldKey("Delivery")
	LoadFieldEquipment    = LoadFieldKey("Equipment")
	LoadFieldWeight       = LoadFieldKey("Weight")
	LoadFieldRate         = LoadFieldKey("Rate")
	LoadFieldLoads        = LoadFieldKey("Loads")
	LoadFieldWeekOf       = LoadFieldKey("WeekOf")
	LoadFieldRespondBy    = LoadFieldKey("RespondBy")
	LoadFieldReference    = LoadFieldKey("Reference")
	LoadFieldCommodity    = LoadFieldKey("Commodity")
	LoadFieldInstructions = LoadFieldKey("Instructions")
)

func (k LoadFieldKey) IsValid() bool {
	switch k {
	case LoadFieldCustomer, LoadFieldOrigin, LoadFieldDestination, LoadFieldPickup,
		LoadFieldDelivery, LoadFieldEquipment, LoadFieldWeight, LoadFieldRate,
		LoadFieldLoads, LoadFieldWeekOf, LoadFieldRespondBy, LoadFieldReference,
		LoadFieldCommodity, LoadFieldInstructions:
		return true
	default:
		return false
	}
}

type LoadDraftField struct {
	Key   LoadFieldKey `json:"key"`
	Value string       `json:"value"`
}

type LoadDraft struct {
	Source        string           `json:"source"`
	Fields        []LoadDraftField `json:"fields"`
	DraftedAt     int64            `json:"draftedAt"`
	DraftedByNova bool             `json:"draftedByNova"`
}

type AttachmentPage struct {
	Number        int    `json:"number"`
	Label         string `json:"label"`
	SuggestedType string `json:"suggestedType"`
}

type SplitDocument struct {
	FromPage     int      `json:"fromPage"`
	ToPage       int      `json:"toPage"`
	DocumentID   pulid.ID `json:"documentId"`
	DocumentType string   `json:"documentType"`
}
