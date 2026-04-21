package invoice

type Status string

const (
	StatusDraft  = Status("Draft")
	StatusPosted = Status("Posted")
)

type InvoiceLineType string

const (
	InvoiceLineTypeFreight     = InvoiceLineType("Freight")
	InvoiceLineTypeAccessorial = InvoiceLineType("Accessorial")
)

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
	case StatusDraft, StatusPosted:
		return true
	default:
		return false
	}
}

func (t InvoiceLineType) IsValid() bool {
	switch t {
	case InvoiceLineTypeFreight, InvoiceLineTypeAccessorial:
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
