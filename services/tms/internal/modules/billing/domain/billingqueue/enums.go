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

type Lane string

const (
	LaneChecking = Lane("Checking")
	LaneNeedsYou = Lane("NeedsYou")
	LaneWaiting  = Lane("Waiting")
	LaneReady    = Lane("Ready")
	LaneHeld     = Lane("Held")
	LaneApproved = Lane("Approved")
	LaneClosed   = Lane("Closed")
)

func (l Lane) IsValid() bool {
	switch l {
	case LaneChecking, LaneNeedsYou, LaneWaiting, LaneReady, LaneHeld, LaneApproved, LaneClosed:
		return true
	default:
		return false
	}
}

func OpenLanes() []Lane {
	return []Lane{LaneChecking, LaneNeedsYou, LaneWaiting, LaneReady, LaneHeld, LaneApproved}
}

func ReviewLanes() []Lane {
	return []Lane{LaneChecking, LaneNeedsYou, LaneWaiting, LaneReady}
}

func HeldStatuses() []Status {
	return []Status{StatusOnHold, StatusException, StatusSentBackToOps}
}

func ClosedStatuses() []Status {
	return []Status{StatusPosted, StatusCanceled}
}

type AgentVerdict string

const (
	AgentVerdictAgrees  = AgentVerdict("Agrees")
	AgentVerdictConcern = AgentVerdict("Concern")
)

func (v AgentVerdict) IsValid() bool {
	return v == AgentVerdictAgrees || v == AgentVerdictConcern
}
