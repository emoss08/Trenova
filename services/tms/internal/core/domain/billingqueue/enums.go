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

// HoldReasonCode is why a biller set an item aside. The three are the reasons
// a biller gives in practice; the free-text review notes say the rest.
type HoldReasonCode string

const (
	HoldReasonWaitingOnPaperwork = HoldReasonCode("WaitingOnPaperwork")
	HoldReasonCustomerDispute    = HoldReasonCode("CustomerDispute")
	HoldReasonRateQuestion       = HoldReasonCode("RateQuestion")
)

func AllHoldReasonCodes() []HoldReasonCode {
	return []HoldReasonCode{
		HoldReasonWaitingOnPaperwork,
		HoldReasonCustomerDispute,
		HoldReasonRateQuestion,
	}
}

func (c HoldReasonCode) IsValid() bool {
	switch c {
	case HoldReasonWaitingOnPaperwork, HoldReasonCustomerDispute, HoldReasonRateQuestion:
		return true
	default:
		return false
	}
}

// Phrase is the reason as the activity log says it.
func (c HoldReasonCode) Phrase() string {
	switch c {
	case HoldReasonWaitingOnPaperwork:
		return "waiting on paperwork"
	case HoldReasonCustomerDispute:
		return "customer dispute"
	case HoldReasonRateQuestion:
		return "rate question"
	default:
		return "no reason given"
	}
}
