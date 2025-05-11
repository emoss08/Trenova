package billinglog

type Status string

const (
	StatusDraft    = Status("Draft")
	StatusBilled   = Status("Billed")
	StatusCanceled = Status("Canceled")
)

func (s Status) String() string {
	return string(s)
}

func (s Status) Is(status Status) bool {
	return s == status
}

func BillingStatusFromString(s string) (Status, error) {
	switch s {
	case "Draft":
		return StatusDraft, nil
	case "Billed":
		return StatusBilled, nil
	case "Canceled":
		return StatusCanceled, nil
	default:
		return StatusDraft, nil
	}
}

type Type string

const (
	TypeInvoice    = Type("Invoice")
	TypeCreditMemo = Type("CreditMemo")
	TypeDebitMemo  = Type("DebitMemo")
)

func (t Type) String() string {
	return string(t)
}

func BillingTypeFromString(s string) (Type, error) {
	switch s {
	case "Invoice":
		return TypeInvoice, nil
	case "CreditMemo":
		return TypeCreditMemo, nil
	case "DebitMemo":
		return TypeDebitMemo, nil
	default:
		return TypeInvoice, nil
	}
}
