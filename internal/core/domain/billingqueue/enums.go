// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package billingqueue

type Status string

const (
	StatusReadyForReview = Status("ReadyForReview")
	StatusInReview       = Status("InReview")
	StatusApproved       = Status("Approved")
	StatusCanceled       = Status("Canceled")
	StatusException      = Status("Exception")
)

func (s Status) String() string {
	return string(s)
}

func (s Status) Is(status Status) bool {
	return s == status
}

func QueueStatusFromString(s string) (Status, error) {
	switch s {
	case "ReadyForReview":
		return StatusReadyForReview, nil
	case "InReview":
		return StatusInReview, nil
	case "Approved":
		return StatusApproved, nil
	case "Canceled":
		return StatusCanceled, nil
	case "Exception":
		return StatusException, nil
	default:
		return StatusReadyForReview, nil
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

func QueueTypeFromString(s string) (Type, error) {
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
