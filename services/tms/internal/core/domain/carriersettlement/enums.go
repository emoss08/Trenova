package carriersettlement

type Status string

const (
	StatusDraft           = Status("Draft")
	StatusPendingApproval = Status("PendingApproval")
	StatusApproved        = Status("Approved")
	StatusPosted          = Status("Posted")
	StatusPaid            = Status("Paid")
	StatusVoided          = Status("Voided")
)

func (s Status) String() string { return string(s) }

func (s Status) IsValid() bool {
	switch s {
	case StatusDraft, StatusPendingApproval, StatusApproved, StatusPosted,
		StatusPaid, StatusVoided:
		return true
	default:
		return false
	}
}

func (s Status) IsTerminal() bool {
	return s == StatusPaid || s == StatusVoided
}

type CostEventType string

const (
	CostEventTypeLinehaulCost  = CostEventType("LinehaulCost")
	CostEventTypeFuelSurcharge = CostEventType("FuelSurcharge")
	CostEventTypeAccessorial   = CostEventType("Accessorial")
	CostEventTypeAdjustment    = CostEventType("Adjustment")
)

func (c CostEventType) String() string { return string(c) }

func (c CostEventType) IsValid() bool {
	switch c {
	case CostEventTypeLinehaulCost, CostEventTypeFuelSurcharge,
		CostEventTypeAccessorial, CostEventTypeAdjustment:
		return true
	default:
		return false
	}
}

type CostEventStatus string

const (
	CostEventStatusPending  = CostEventStatus("Pending")
	CostEventStatusAttached = CostEventStatus("Attached")
	CostEventStatusSettled  = CostEventStatus("Settled")
	CostEventStatusVoided   = CostEventStatus("Voided")
)

func (c CostEventStatus) String() string { return string(c) }

func (c CostEventStatus) IsValid() bool {
	switch c {
	case CostEventStatusPending, CostEventStatusAttached,
		CostEventStatusSettled, CostEventStatusVoided:
		return true
	default:
		return false
	}
}

type BatchStatus string

const (
	BatchStatusOpen      = BatchStatus("Open")
	BatchStatusCompleted = BatchStatus("Completed")
	BatchStatusCanceled  = BatchStatus("Canceled")
)

func (b BatchStatus) String() string { return string(b) }

func (b BatchStatus) IsValid() bool {
	switch b {
	case BatchStatusOpen, BatchStatusCompleted, BatchStatusCanceled:
		return true
	default:
		return false
	}
}

type LedgerEntryType string

const (
	LedgerEntryTypeBill       = LedgerEntryType("Bill")
	LedgerEntryTypePayment    = LedgerEntryType("Payment")
	LedgerEntryTypeAdjustment = LedgerEntryType("Adjustment")
)

func (l LedgerEntryType) String() string { return string(l) }

func (l LedgerEntryType) IsValid() bool {
	switch l {
	case LedgerEntryTypeBill, LedgerEntryTypePayment, LedgerEntryTypeAdjustment:
		return true
	default:
		return false
	}
}

type InvoiceMatchStatus string

const (
	InvoiceMatchStatusSuggested = InvoiceMatchStatus("Suggested")
	InvoiceMatchStatusMatched   = InvoiceMatchStatus("Matched")
	InvoiceMatchStatusVariance  = InvoiceMatchStatus("Variance")
	InvoiceMatchStatusResolved  = InvoiceMatchStatus("Resolved")
	InvoiceMatchStatusRejected  = InvoiceMatchStatus("Rejected")
)

func (i InvoiceMatchStatus) String() string { return string(i) }

func (i InvoiceMatchStatus) IsValid() bool {
	switch i {
	case InvoiceMatchStatusSuggested, InvoiceMatchStatusMatched,
		InvoiceMatchStatusVariance, InvoiceMatchStatusResolved,
		InvoiceMatchStatusRejected:
		return true
	default:
		return false
	}
}

func (i InvoiceMatchStatus) IsOpen() bool {
	switch i {
	case InvoiceMatchStatusSuggested, InvoiceMatchStatusMatched, InvoiceMatchStatusVariance:
		return true
	case InvoiceMatchStatusResolved, InvoiceMatchStatusRejected:
		return false
	default:
		return false
	}
}
