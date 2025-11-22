package accounting

import "errors"

var (
	ErrInvalidFiscalYearStatus      = errors.New("invalid fiscal year status")
	ErrInvalidPeriodType            = errors.New("invalid period type")
	ErrInvalidPeriodStatus          = errors.New("invalid period status")
	ErrInvalidJournalEntryStatus    = errors.New("invalid journal entry status")
	ErrInvalidJournalEntryType      = errors.New("invalid journal entry type")
	ErrInvalidInvoiceDeliveryMethod = errors.New("invalid invoice delivery method")
	ErrInvalidInvoiceFormat         = errors.New("invalid invoice format")
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

// NormalBalance returns whether this category typically has a debit or credit balance
func (c Category) NormalBalance() string {
	switch c {
	case CategoryAsset, CategoryExpense, CategoryCostOfRevenue:
		return "Debit"
	case CategoryLiability, CategoryEquity, CategoryRevenue:
		return "Credit"
	default:
		return "Unknown"
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
	JournalEntryCriteriaInvoicePosted      = JournalEntryCriteriaType("InvoicePosted")
	JournalEntryCriteriaBillPosted         = JournalEntryCriteriaType("BillPosted")
	JournalEntryCriteriaPaymentReceived    = JournalEntryCriteriaType("PaymentReceived")
	JournalEntryCriteriaPaymentMade        = JournalEntryCriteriaType("PaymentMade")
	JournalEntryCriteriaDeliveryComplete   = JournalEntryCriteriaType("DeliveryComplete")
	JournalEntryCriteriaShipmentDispatched = JournalEntryCriteriaType("ShipmentDispatched")
)

func (j JournalEntryCriteriaType) String() string {
	return string(j)
}

func (j JournalEntryCriteriaType) IsValid() bool {
	switch j {
	case JournalEntryCriteriaInvoicePosted, JournalEntryCriteriaBillPosted,
		JournalEntryCriteriaPaymentReceived, JournalEntryCriteriaPaymentMade,
		JournalEntryCriteriaDeliveryComplete, JournalEntryCriteriaShipmentDispatched:
		return true
	}
	return false
}

func (j JournalEntryCriteriaType) GetDescription() string {
	switch j {
	case JournalEntryCriteriaInvoicePosted:
		return "Create journal entry when customer invoice is posted"
	case JournalEntryCriteriaBillPosted:
		return "Create journal entry when vendor bill is posted"
	case JournalEntryCriteriaPaymentReceived:
		return "Create journal entry when customer payment is received"
	case JournalEntryCriteriaPaymentMade:
		return "Create journal entry when vendor payment is made"
	case JournalEntryCriteriaDeliveryComplete:
		return "Create journal entry when delivery is complete"
	case JournalEntryCriteriaShipmentDispatched:
		return "Create journal entry when shipment is dispatched"
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
		return "Recognize revenue when payment is received (cash basis)"
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
	ExpenseRecognitionOnBilling    = ExpenseRecognitionType("OnBilling")
)

func (e ExpenseRecognitionType) String() string {
	return string(e)
}

func (e ExpenseRecognitionType) IsValid() bool {
	switch e {
	case ExpenseRecognitionOnIncurrence, ExpenseRecognitionOnPayment,
		ExpenseRecognitionOnAccrual, ExpenseRecognitionOnBilling:
		return true
	}
	return false
}

func (e ExpenseRecognitionType) GetDescription() string {
	switch e {
	case ExpenseRecognitionOnIncurrence:
		return "Recognize expense when service is performed or goods received"
	case ExpenseRecognitionOnPayment:
		return "Recognize expense when payment is made (cash basis)"
	case ExpenseRecognitionOnAccrual:
		return "Recognize expense when bill is received (accrual basis)"
	case ExpenseRecognitionOnBilling:
		return "Recognize expense when vendor bill is posted"
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
	JournalEntryStatusVoid     = JournalEntryStatus("Void")
)

func (j JournalEntryStatus) String() string {
	return string(j)
}

func (j JournalEntryStatus) IsValid() bool {
	switch j {
	case JournalEntryStatusDraft, JournalEntryStatusPending, JournalEntryStatusApproved,
		JournalEntryStatusPosted, JournalEntryStatusReversed, JournalEntryStatusRejected,
		JournalEntryStatusVoid:
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
		return "Entry has been reversed by another entry"
	case JournalEntryStatusRejected:
		return "Entry has been rejected during approval"
	case JournalEntryStatusVoid:
		return "Entry was cancelled before posting"
	default:
		return "Unknown status"
	}
}

func JournalEntryStatusFromString(status string) (JournalEntryStatus, error) {
	switch status {
	case "Draft":
		return JournalEntryStatusDraft, nil
	case "Pending":
		return JournalEntryStatusPending, nil
	case "Approved":
		return JournalEntryStatusApproved, nil
	case "Posted":
		return JournalEntryStatusPosted, nil
	case "Reversed":
		return JournalEntryStatusReversed, nil
	case "Rejected":
		return JournalEntryStatusRejected, nil
	case "Void":
		return JournalEntryStatusVoid, nil
	default:
		return "", ErrInvalidJournalEntryStatus
	}
}

type JournalEntryType string

const (
	JournalEntryTypeStandard         = JournalEntryType("Standard")
	JournalEntryTypeAdjusting        = JournalEntryType("Adjusting")
	JournalEntryTypeClosing          = JournalEntryType("Closing")
	JournalEntryTypeReversal         = JournalEntryType("Reversal")
	JournalEntryTypeReclassification = JournalEntryType("Reclassification")
	JournalEntryTypeAutoGenerated    = JournalEntryType("AutoGenerated")
	JournalEntryTypeReconciliation   = JournalEntryType("Reconciliation")
)

func (j JournalEntryType) String() string {
	return string(j)
}

func (j JournalEntryType) IsValid() bool {
	switch j {
	case JournalEntryTypeStandard, JournalEntryTypeAdjusting, JournalEntryTypeClosing,
		JournalEntryTypeReversal, JournalEntryTypeReclassification,
		JournalEntryTypeAutoGenerated, JournalEntryTypeReconciliation:
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
	case JournalEntryTypeAutoGenerated:
		return "Auto-generated from invoice, bill, or payment"
	case JournalEntryTypeReconciliation:
		return "Reconciliation adjustment entry"
	default:
		return "Unknown type"
	}
}

func JournalEntryTypeFromString(entryType string) (JournalEntryType, error) {
	switch entryType {
	case "Standard":
		return JournalEntryTypeStandard, nil
	case "Adjusting":
		return JournalEntryTypeAdjusting, nil
	case "Closing":
		return JournalEntryTypeClosing, nil
	case "Reversal":
		return JournalEntryTypeReversal, nil
	case "Reclassification":
		return JournalEntryTypeReclassification, nil
	case "AutoGenerated":
		return JournalEntryTypeAutoGenerated, nil
	case "Reconciliation":
		return JournalEntryTypeReconciliation, nil
	default:
		return "", ErrInvalidJournalEntryType
	}
}

type InvoiceDeliveryMethod string

const (
	InvoiceDeliveryEmail  = InvoiceDeliveryMethod("Email")
	InvoiceDeliveryPortal = InvoiceDeliveryMethod("Portal")
	InvoiceDeliveryEDI    = InvoiceDeliveryMethod("EDI")
	InvoiceDeliveryPrint  = InvoiceDeliveryMethod("Print")
	InvoiceDeliveryAPI    = InvoiceDeliveryMethod("API")
)

func (i InvoiceDeliveryMethod) String() string {
	return string(i)
}

func (i InvoiceDeliveryMethod) IsValid() bool {
	switch i {
	case InvoiceDeliveryEmail, InvoiceDeliveryPortal, InvoiceDeliveryEDI,
		InvoiceDeliveryPrint, InvoiceDeliveryAPI:
		return true
	}
	return false
}

func (i InvoiceDeliveryMethod) GetDescription() string {
	switch i {
	case InvoiceDeliveryEmail:
		return "Deliver invoice via email"
	case InvoiceDeliveryPortal:
		return "Customer accesses invoice through customer portal"
	case InvoiceDeliveryEDI:
		return "Deliver invoice via EDI (Electronic Data Interchange)"
	case InvoiceDeliveryPrint:
		return "Print and mail physical invoice"
	case InvoiceDeliveryAPI:
		return "Deliver invoice via API integration"
	default:
		return "Unknown delivery method"
	}
}

func InvoiceDeliveryMethodFromString(method string) (InvoiceDeliveryMethod, error) {
	switch method {
	case "Email":
		return InvoiceDeliveryEmail, nil
	case "Portal":
		return InvoiceDeliveryPortal, nil
	case "EDI":
		return InvoiceDeliveryEDI, nil
	case "Print":
		return InvoiceDeliveryPrint, nil
	case "API":
		return InvoiceDeliveryAPI, nil
	default:
		return "", ErrInvalidInvoiceDeliveryMethod
	}
}

type InvoiceFormat string

const (
	InvoiceFormatPDF  = InvoiceFormat("PDF")
	InvoiceFormatHTML = InvoiceFormat("HTML")
	InvoiceFormatEDI  = InvoiceFormat("EDI")
	InvoiceFormatXML  = InvoiceFormat("XML")
	InvoiceFormatJSON = InvoiceFormat("JSON")
)

func (i InvoiceFormat) String() string {
	return string(i)
}

func (i InvoiceFormat) IsValid() bool {
	switch i {
	case InvoiceFormatPDF, InvoiceFormatHTML, InvoiceFormatEDI,
		InvoiceFormatXML, InvoiceFormatJSON:
		return true
	}
	return false
}

func (i InvoiceFormat) GetDescription() string {
	switch i {
	case InvoiceFormatPDF:
		return "PDF document format"
	case InvoiceFormatHTML:
		return "HTML web format"
	case InvoiceFormatEDI:
		return "EDI format (X12, EDIFACT)"
	case InvoiceFormatXML:
		return "XML format"
	case InvoiceFormatJSON:
		return "JSON format for API integrations"
	default:
		return "Unknown format"
	}
}

func InvoiceFormatFromString(format string) (InvoiceFormat, error) {
	switch format {
	case "PDF":
		return InvoiceFormatPDF, nil
	case "HTML":
		return InvoiceFormatHTML, nil
	case "EDI":
		return InvoiceFormatEDI, nil
	case "XML":
		return InvoiceFormatXML, nil
	case "JSON":
		return InvoiceFormatJSON, nil
	default:
		return "", ErrInvalidInvoiceFormat
	}
}
