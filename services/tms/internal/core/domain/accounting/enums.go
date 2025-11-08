package accounting

import "errors"

var (
	ErrInvalidFiscalYearStatus     = errors.New("invalid fiscal year status")
	ErrInvalidPeriodType           = errors.New("invalid period type")
	ErrInvalidPeriodStatus         = errors.New("invalid period status")
	ErrInvalidJournalEntryCriteria = errors.New("invalid journal entry criteria")
	ErrInvalidThresholdAction      = errors.New("invalid threshold action")
	ErrInvalidRevenueRecognition   = errors.New("invalid revenue recognition method")
	ErrInvalidExpenseRecognition   = errors.New("invalid expense recognition method")
	ErrInvalidJournalEntryStatus   = errors.New("invalid journal entry status")
	ErrInvalidJournalEntryType     = errors.New("invalid journal entry type")
)

type Category string

const (
	CategoryAsset         = Category("Asset")
	CategoryLiability     = Category("Liability")
	CategoryEquity        = Category("Equity")
	CategoryRevenue       = Category("Revenue")
	CategoryCostOfRevenue = Category("CostOfRevenue")
	CategoryExpense       = Category("Expense")
)

func (c Category) String() string {
	return string(c)
}

func (c Category) IsValid() bool {
	switch c {
	case CategoryAsset, CategoryLiability, CategoryEquity,
		CategoryRevenue, CategoryCostOfRevenue, CategoryExpense:
		return true
	}
	return false
}

func (c Category) GetDescription() string {
	switch c {
	case CategoryAsset:
		return "Resources owned by the company"
	case CategoryLiability:
		return "Obligations owed by the company"
	case CategoryEquity:
		return "Owner's stake in the company"
	case CategoryRevenue:
		return "Income from operations"
	case CategoryCostOfRevenue:
		return "Direct costs of providing service"
	case CategoryExpense:
		return "Operating expenses"
	default:
		return "Unknown category"
	}
}

type FiscalYearStatus string

const (
	FiscalYearStatusDraft  = FiscalYearStatus("Draft")
	FiscalYearStatusOpen   = FiscalYearStatus("Open")
	FiscalYearStatusClosed = FiscalYearStatus("Closed")
	FiscalYearStatusLocked = FiscalYearStatus("Locked")
)

func (s FiscalYearStatus) String() string {
	return string(s)
}

func (s FiscalYearStatus) IsValid() bool {
	switch s {
	case FiscalYearStatusDraft, FiscalYearStatusOpen,
		FiscalYearStatusClosed, FiscalYearStatusLocked:
		return true
	}
	return false
}

func (s FiscalYearStatus) GetDescription() string {
	switch s {
	case FiscalYearStatusDraft:
		return "Year is being set up, not yet active"
	case FiscalYearStatusOpen:
		return "Year is active and accepting transactions"
	case FiscalYearStatusClosed:
		return "Year-end closing complete, only adjusting entries allowed"
	case FiscalYearStatusLocked:
		return "Year is locked, no transactions allowed"
	default:
		return "Unknown status"
	}
}

func FiscalYearStatusFromString(status string) (FiscalYearStatus, error) {
	switch status {
	case "Draft":
		return FiscalYearStatusDraft, nil
	case "Open":
		return FiscalYearStatusOpen, nil
	case "Closed":
		return FiscalYearStatusClosed, nil
	case "Locked":
		return FiscalYearStatusLocked, nil
	default:
		return "", ErrInvalidFiscalYearStatus
	}
}

type PeriodType string

const (
	PeriodTypeMonth   = PeriodType("Month")
	PeriodTypeQuarter = PeriodType("Quarter")
	PeriodTypeYear    = PeriodType("Year")
)

func (p PeriodType) String() string {
	return string(p)
}

func (p PeriodType) IsValid() bool {
	switch p {
	case PeriodTypeMonth, PeriodTypeQuarter, PeriodTypeYear:
		return true
	}
	return false
}

func (p PeriodType) GetDescription() string {
	switch p {
	case PeriodTypeMonth:
		return "Month"
	case PeriodTypeQuarter:
		return "Quarter"
	case PeriodTypeYear:
		return "Year"
	}
	return "Unknown period type"
}

func PeriodTypeFromString(periodType string) (PeriodType, error) {
	switch periodType {
	case "Month":
		return PeriodTypeMonth, nil
	case "Quarter":
		return PeriodTypeQuarter, nil
	case "Year":
		return PeriodTypeYear, nil
	default:
		return "", ErrInvalidPeriodType
	}
}

type PeriodStatus string

const (
	PeriodStatusOpen   = PeriodStatus("Open")
	PeriodStatusClosed = PeriodStatus("Closed")
	PeriodStatusLocked = PeriodStatus("Locked")
)

func (p PeriodStatus) String() string {
	return string(p)
}

func (p PeriodStatus) IsValid() bool {
	switch p {
	case PeriodStatusOpen, PeriodStatusClosed, PeriodStatusLocked:
		return true
	}
	return false
}

func (p PeriodStatus) GetDescription() string {
	switch p {
	case PeriodStatusOpen:
		return "Open"
	case PeriodStatusClosed:
		return "Closed"
	case PeriodStatusLocked:
		return "Locked"
	default:
		return "Unknown period status"
	}
}

func PeriodStatusFromString(periodStatus string) (PeriodStatus, error) {
	switch periodStatus {
	case "Open":
		return PeriodStatusOpen, nil
	case "Closed":
		return PeriodStatusClosed, nil
	case "Locked":
		return PeriodStatusLocked, nil
	default:
		return "", ErrInvalidPeriodStatus
	}
}

type JournalEntryCriteriaType string

const (
	JournalEntryCriteriaShipmentBilled    = JournalEntryCriteriaType("ShipmentBilled")
	JournalEntryCriteriaPaymentReceived   = JournalEntryCriteriaType("PaymentReceived")
	JournalEntryCriteriaExpenseRecognized = JournalEntryCriteriaType("ExpenseRecognized")
	JournalEntryCriteriaDeliveryComplete  = JournalEntryCriteriaType("DeliveryComplete")
)

func (j JournalEntryCriteriaType) String() string {
	return string(j)
}

func (j JournalEntryCriteriaType) IsValid() bool {
	switch j {
	case JournalEntryCriteriaShipmentBilled, JournalEntryCriteriaPaymentReceived,
		JournalEntryCriteriaExpenseRecognized, JournalEntryCriteriaDeliveryComplete:
		return true
	}
	return false
}

func (j JournalEntryCriteriaType) GetDescription() string {
	switch j {
	case JournalEntryCriteriaShipmentBilled:
		return "Create journal entry when shipment is billed"
	case JournalEntryCriteriaPaymentReceived:
		return "Create journal entry when payment is received"
	case JournalEntryCriteriaExpenseRecognized:
		return "Create journal entry when expense is recognized"
	case JournalEntryCriteriaDeliveryComplete:
		return "Create journal entry when delivery is complete"
	default:
		return "Unknown criteria"
	}
}

type ThresholdActionType string

const (
	ThresholdActionWarn   = ThresholdActionType("Warn")
	ThresholdActionBlock  = ThresholdActionType("Block")
	ThresholdActionNotify = ThresholdActionType("Notify")
)

func (t ThresholdActionType) String() string {
	return string(t)
}

func (t ThresholdActionType) IsValid() bool {
	switch t {
	case ThresholdActionWarn, ThresholdActionBlock, ThresholdActionNotify:
		return true
	}
	return false
}

func (t ThresholdActionType) GetDescription() string {
	switch t {
	case ThresholdActionWarn:
		return "Display warning when threshold is exceeded"
	case ThresholdActionBlock:
		return "Block operations when threshold is exceeded"
	case ThresholdActionNotify:
		return "Send notifications when threshold is exceeded"
	default:
		return "Unknown action"
	}
}

type RevenueRecognitionType string

const (
	RevenueRecognitionOnDelivery = RevenueRecognitionType("OnDelivery")
	RevenueRecognitionOnBilling  = RevenueRecognitionType("OnBilling")
	RevenueRecognitionOnPayment  = RevenueRecognitionType("OnPayment")
	RevenueRecognitionOnPickup   = RevenueRecognitionType("OnPickup")
)

func (r RevenueRecognitionType) String() string {
	return string(r)
}

func (r RevenueRecognitionType) IsValid() bool {
	switch r {
	case RevenueRecognitionOnDelivery, RevenueRecognitionOnBilling,
		RevenueRecognitionOnPayment, RevenueRecognitionOnPickup:
		return true
	}
	return false
}

func (r RevenueRecognitionType) GetDescription() string {
	switch r {
	case RevenueRecognitionOnDelivery:
		return "Recognize revenue when goods are delivered"
	case RevenueRecognitionOnBilling:
		return "Recognize revenue when invoice is created"
	case RevenueRecognitionOnPayment:
		return "Recognize revenue when payment is received"
	case RevenueRecognitionOnPickup:
		return "Recognize revenue when goods are picked up"
	default:
		return "Unknown method"
	}
}

type ExpenseRecognitionType string

const (
	ExpenseRecognitionOnIncurrence = ExpenseRecognitionType("OnIncurrence")
	ExpenseRecognitionOnPayment    = ExpenseRecognitionType("OnPayment")
	ExpenseRecognitionOnAccrual    = ExpenseRecognitionType("OnAccrual")
)

func (e ExpenseRecognitionType) String() string {
	return string(e)
}

func (e ExpenseRecognitionType) IsValid() bool {
	switch e {
	case ExpenseRecognitionOnIncurrence, ExpenseRecognitionOnPayment, ExpenseRecognitionOnAccrual:
		return true
	}
	return false
}

func (e ExpenseRecognitionType) GetDescription() string {
	switch e {
	case ExpenseRecognitionOnIncurrence:
		return "Recognize expense when incurred"
	case ExpenseRecognitionOnPayment:
		return "Recognize expense when payment is made"
	case ExpenseRecognitionOnAccrual:
		return "Recognize expense on accrual basis"
	default:
		return "Unknown method"
	}
}

type JournalEntryStatus string

const (
	JournalEntryStatusDraft    = JournalEntryStatus("Draft")
	JournalEntryStatusPending  = JournalEntryStatus("Pending")
	JournalEntryStatusApproved = JournalEntryStatus("Approved")
	JournalEntryStatusPosted   = JournalEntryStatus("Posted")
	JournalEntryStatusReversed = JournalEntryStatus("Reversed")
	JournalEntryStatusRejected = JournalEntryStatus("Rejected")
)

func (j JournalEntryStatus) String() string {
	return string(j)
}

func (j JournalEntryStatus) IsValid() bool {
	switch j {
	case JournalEntryStatusDraft, JournalEntryStatusPending, JournalEntryStatusApproved,
		JournalEntryStatusPosted, JournalEntryStatusReversed, JournalEntryStatusRejected:
		return true
	}
	return false
}

func (j JournalEntryStatus) GetDescription() string {
	switch j {
	case JournalEntryStatusDraft:
		return "Entry is being created and can be edited"
	case JournalEntryStatusPending:
		return "Entry is pending approval"
	case JournalEntryStatusApproved:
		return "Entry has been approved and ready to post"
	case JournalEntryStatusPosted:
		return "Entry has been posted to the general ledger"
	case JournalEntryStatusReversed:
		return "Entry has been reversed"
	case JournalEntryStatusRejected:
		return "Entry has been rejected"
	default:
		return "Unknown status"
	}
}

type JournalEntryType string

const (
	JournalEntryTypeStandard         = JournalEntryType("Standard")
	JournalEntryTypeAdjusting        = JournalEntryType("Adjusting")
	JournalEntryTypeClosing          = JournalEntryType("Closing")
	JournalEntryTypeReversal         = JournalEntryType("Reversal")
	JournalEntryTypeReclassification = JournalEntryType("Reclassification")
)

func (j JournalEntryType) String() string {
	return string(j)
}

func (j JournalEntryType) IsValid() bool {
	switch j {
	case JournalEntryTypeStandard, JournalEntryTypeAdjusting, JournalEntryTypeClosing,
		JournalEntryTypeReversal, JournalEntryTypeReclassification:
		return true
	}
	return false
}

func (j JournalEntryType) GetDescription() string {
	switch j {
	case JournalEntryTypeStandard:
		return "Standard journal entry for normal transactions"
	case JournalEntryTypeAdjusting:
		return "Adjusting entry for period-end adjustments"
	case JournalEntryTypeClosing:
		return "Closing entry for year-end closing"
	case JournalEntryTypeReversal:
		return "Reversal entry to reverse a previous entry"
	case JournalEntryTypeReclassification:
		return "Reclassification entry to move amounts between accounts"
	default:
		return "Unknown type"
	}
}
