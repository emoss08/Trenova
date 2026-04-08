package billingqueue

type Status string

const (
	StatusReadyForReview = Status("ReadyForReview")
	StatusInReview       = Status("InReview")
	StatusApproved       = Status("Approved")
	StatusPosted         = Status("Posted")
	StatusOnHold         = Status("OnHold")
	StatusSentBackToOps  = Status("SentBackToOps")
	StatusException      = Status("Exception")
	StatusCanceled       = Status("Canceled")
)

type BillType string

const (
	BillTypeInvoice    = BillType("Invoice")
	BillTypeCreditMemo = BillType("CreditMemo")
	BillTypeDebitMemo  = BillType("DebitMemo")
)

type ExceptionReasonCode string

const (
	ExceptionMissingDocumentation     = ExceptionReasonCode("MissingDocumentation")
	ExceptionIncorrectRates           = ExceptionReasonCode("IncorrectRates")
	ExceptionWeightDiscrepancy        = ExceptionReasonCode("WeightDiscrepancy")
	ExceptionAccessorialDispute       = ExceptionReasonCode("AccessorialDispute")
	ExceptionDuplicateCharge          = ExceptionReasonCode("DuplicateCharge")
	ExceptionMissingReferenceNumber   = ExceptionReasonCode("MissingReferenceNumber")
	ExceptionCustomerInformationError = ExceptionReasonCode("CustomerInformationError")
	ExceptionServiceFailure           = ExceptionReasonCode("ServiceFailure")
	ExceptionRateNotOnFile            = ExceptionReasonCode("RateNotOnFile")
	ExceptionOther                    = ExceptionReasonCode("Other")
)

func (c ExceptionReasonCode) IsValid() bool {
	switch c {
	case ExceptionMissingDocumentation,
		ExceptionIncorrectRates,
		ExceptionWeightDiscrepancy,
		ExceptionAccessorialDispute,
		ExceptionDuplicateCharge,
		ExceptionMissingReferenceNumber,
		ExceptionCustomerInformationError,
		ExceptionServiceFailure,
		ExceptionRateNotOnFile,
		ExceptionOther:
		return true
	default:
		return false
	}
}
