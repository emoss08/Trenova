package invoice

type Status string

const (
	StatusDraft  = Status("Draft")
	StatusPosted = Status("Posted")
	StatusVoided = Status("Voided")
)

// VoidDisposition says what happens to the freight behind a voided invoice:
// back to the queue to be billed again, or canceled so it never is.
type VoidDisposition string

const (
	VoidDispositionRebill      = VoidDisposition("Rebill")
	VoidDispositionDoNotRebill = VoidDisposition("DoNotRebill")
)

func (d VoidDisposition) IsValid() bool {
	switch d {
	case VoidDispositionRebill, VoidDispositionDoNotRebill:
		return true
	default:
		return false
	}
}

type Scope string

const (
	ScopeShipment     = Scope("Shipment")
	ScopeOrder        = Scope("Order")
	ScopeConsolidated = Scope("Consolidated")
	ScopeAdjustment   = Scope("Adjustment")
	// ScopeMemo is a credit or debit memo raised against a customer on its own,
	// with no shipment behind it.
	ScopeMemo = Scope("Memo")
)

type InvoiceLineType string

const (
	InvoiceLineTypeFreight     = InvoiceLineType("Freight")
	InvoiceLineTypeAccessorial = InvoiceLineType("Accessorial")
	// InvoiceLineTypeMemo is a free-form memo line; it counts toward the total
	// but neither the freight subtotal nor the accessorial figure.
	InvoiceLineTypeMemo = InvoiceLineType("Memo")
)

// MemoKind records why a standalone memo exists.
type MemoKind string

const (
	MemoKindManual     = MemoKind("Manual")
	MemoKindLateCharge = MemoKind("LateCharge")
)

func (k MemoKind) IsValid() bool {
	switch k {
	case MemoKindManual, MemoKindLateCharge:
		return true
	default:
		return false
	}
}

// EDISendStatus tracks the outbound 210 for an invoice separately from email.
type EDISendStatus string

const (
	EDISendStatusNotSent       = EDISendStatus("NotSent")
	EDISendStatusNotConfigured = EDISendStatus("NotConfigured")
	EDISendStatusQueued        = EDISendStatus("Queued")
	EDISendStatusGenerated     = EDISendStatus("Generated")
	EDISendStatusSending       = EDISendStatus("Sending")
	EDISendStatusSent          = EDISendStatus("Sent")
	EDISendStatusFailed        = EDISendStatus("Failed")
	EDISendStatusDeadLettered  = EDISendStatus("DeadLettered")
)

func (s EDISendStatus) IsValid() bool {
	switch s {
	case EDISendStatusNotSent, EDISendStatusNotConfigured, EDISendStatusQueued,
		EDISendStatusGenerated, EDISendStatusSending, EDISendStatusSent,
		EDISendStatusFailed, EDISendStatusDeadLettered:
		return true
	default:
		return false
	}
}

type PaymentTerm string

type SettlementStatus string

const (
	SettlementStatusUnpaid        = SettlementStatus("Unpaid")
	SettlementStatusPartiallyPaid = SettlementStatus("PartiallyPaid")
	SettlementStatusPaid          = SettlementStatus("Paid")
)

type DisputeStatus string

const (
	DisputeStatusNone     = DisputeStatus("None")
	DisputeStatusDisputed = DisputeStatus("Disputed")
)

type SendStatus string

const (
	SendStatusNotSent       = SendStatus("NotSent")
	SendStatusSending       = SendStatus("Sending")
	SendStatusSent          = SendStatus("Sent")
	SendStatusPartiallySent = SendStatus("PartiallySent")
	SendStatusFailed        = SendStatus("Failed")
)

type AttachmentDeliveryMethod string

const (
	AttachmentDeliveryMethodAttached = AttachmentDeliveryMethod("Attached")
	AttachmentDeliveryMethodLink     = AttachmentDeliveryMethod("Link")
	AttachmentDeliveryMethodSkipped  = AttachmentDeliveryMethod("Skipped")
	AttachmentDeliveryMethodFailed   = AttachmentDeliveryMethod("Failed")
)

const (
	PaymentTermNet10        = PaymentTerm("Net10")
	PaymentTermNet15        = PaymentTerm("Net15")
	PaymentTermNet30        = PaymentTerm("Net30")
	PaymentTermNet45        = PaymentTerm("Net45")
	PaymentTermNet60        = PaymentTerm("Net60")
	PaymentTermNet90        = PaymentTerm("Net90")
	PaymentTermDueOnReceipt = PaymentTerm("DueOnReceipt")
)

func (s Status) IsValid() bool {
	switch s {
	case StatusDraft, StatusPosted, StatusVoided:
		return true
	default:
		return false
	}
}

func (s Scope) IsValid() bool {
	switch s {
	case ScopeShipment, ScopeOrder, ScopeConsolidated, ScopeAdjustment, ScopeMemo:
		return true
	default:
		return false
	}
}

func (t InvoiceLineType) IsValid() bool {
	switch t {
	case InvoiceLineTypeFreight, InvoiceLineTypeAccessorial, InvoiceLineTypeMemo:
		return true
	default:
		return false
	}
}

func (t PaymentTerm) IsValid() bool {
	switch t {
	case PaymentTermNet10,
		PaymentTermNet15,
		PaymentTermNet30,
		PaymentTermNet45,
		PaymentTermNet60,
		PaymentTermNet90,
		PaymentTermDueOnReceipt:
		return true
	default:
		return false
	}
}

func (s SettlementStatus) IsValid() bool {
	switch s {
	case SettlementStatusUnpaid, SettlementStatusPartiallyPaid, SettlementStatusPaid:
		return true
	default:
		return false
	}
}

func (s DisputeStatus) IsValid() bool {
	switch s {
	case DisputeStatusNone, DisputeStatusDisputed:
		return true
	default:
		return false
	}
}

func (s SendStatus) IsValid() bool {
	switch s {
	case SendStatusNotSent,
		SendStatusSending,
		SendStatusSent,
		SendStatusPartiallySent,
		SendStatusFailed:
		return true
	default:
		return false
	}
}

func (m AttachmentDeliveryMethod) IsValid() bool {
	switch m {
	case AttachmentDeliveryMethodAttached,
		AttachmentDeliveryMethodLink,
		AttachmentDeliveryMethodSkipped,
		AttachmentDeliveryMethodFailed:
		return true
	default:
		return false
	}
}
