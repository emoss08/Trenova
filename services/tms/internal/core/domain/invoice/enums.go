package invoice

type Status string

const (
	StatusDraft  = Status("Draft")
	StatusPosted = Status("Posted")
)

type LineType string

const (
	LineTypeFreight     = LineType("Freight")
	LineTypeAccessorial = LineType("Accessorial")
)

type PaymentTerm string

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

func (t LineType) IsValid() bool {
	switch t {
	case LineTypeFreight, LineTypeAccessorial:
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
