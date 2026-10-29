package accountingcontrol

type AccountingBasisType string

const (
	AccountingBasisAccrual = AccountingBasisType("Accrual")
	AccountingBasisCash    = AccountingBasisType("Cash")
)

func (a AccountingBasisType) String() string {
	return string(a)
}

func (a AccountingBasisType) IsValid() bool {
	switch a {
	case AccountingBasisAccrual, AccountingBasisCash:
		return true
	}
	return false
}

func (a AccountingBasisType) GetDescription() string {
	switch a {
	case AccountingBasisAccrual:
		return "Recognize revenue and expense from non-cash posting events"
	case AccountingBasisCash:
		return "Recognize revenue and expense only from cash settlement events"
	default:
		return "Unknown accounting basis"
	}
}

func (a AccountingBasisType) ValidRevenueRecognitionPolicies() []RevenueRecognitionPolicyType {
	switch a {
	case AccountingBasisCash:
		return []RevenueRecognitionPolicyType{RevenueRecognitionOnCashReceipt}
	case AccountingBasisAccrual:
		return []RevenueRecognitionPolicyType{
			RevenueRecognitionOnInvoicePost,
		}
	default:
		return nil
	}
}

func (a AccountingBasisType) ValidExpenseRecognitionPolicies() []ExpenseRecognitionPolicyType {
	switch a {
	case AccountingBasisCash:
		return []ExpenseRecognitionPolicyType{ExpenseRecognitionOnCashDisbursement}
	case AccountingBasisAccrual:
		return []ExpenseRecognitionPolicyType{
			ExpenseRecognitionOnVendorBillPost,
		}
	default:
		return nil
	}
}

type RevenueRecognitionPolicyType string

const (
	RevenueRecognitionOnInvoicePost = RevenueRecognitionPolicyType("OnInvoicePost")
	RevenueRecognitionOnCashReceipt = RevenueRecognitionPolicyType("OnCashReceipt")
)

func (r RevenueRecognitionPolicyType) String() string {
	return string(r)
}

func (r RevenueRecognitionPolicyType) IsValid() bool {
	switch r {
	case RevenueRecognitionOnInvoicePost, RevenueRecognitionOnCashReceipt:
		return true
	}
	return false
}

func (r RevenueRecognitionPolicyType) GetDescription() string {
	switch r {
	case RevenueRecognitionOnInvoicePost:
		return "Recognize revenue when a customer invoice is posted"
	case RevenueRecognitionOnCashReceipt:
		return "Recognize revenue when a customer payment is posted"
	default:
		return "Unknown method"
	}
}

type ExpenseRecognitionPolicyType string

const (
	ExpenseRecognitionOnVendorBillPost   = ExpenseRecognitionPolicyType("OnVendorBillPost")
	ExpenseRecognitionOnCashDisbursement = ExpenseRecognitionPolicyType("OnCashDisbursement")
)

func (e ExpenseRecognitionPolicyType) String() string {
	return string(e)
}

func (e ExpenseRecognitionPolicyType) IsValid() bool {
	switch e {
	case ExpenseRecognitionOnVendorBillPost, ExpenseRecognitionOnCashDisbursement:
		return true
	}
	return false
}

func (e ExpenseRecognitionPolicyType) GetDescription() string {
	switch e {
	case ExpenseRecognitionOnVendorBillPost:
		return "Recognize expense when a vendor bill is posted"
	case ExpenseRecognitionOnCashDisbursement:
		return "Recognize expense when a vendor payment is posted"
	default:
		return "Unknown method"
	}
}

type JournalPostingModeType string

const (
	JournalPostingModeManual    = JournalPostingModeType("Manual")
	JournalPostingModeAutomatic = JournalPostingModeType("Automatic")
)

type JournalSourceEventType string

const (
	JournalSourceEventInvoicePosted              = JournalSourceEventType("InvoicePosted")
	JournalSourceEventCreditMemoPosted           = JournalSourceEventType("CreditMemoPosted")
	JournalSourceEventDebitMemoPosted            = JournalSourceEventType("DebitMemoPosted")
	JournalSourceEventCustomerPaymentPosted      = JournalSourceEventType("CustomerPaymentPosted")
	JournalSourceEventCustomerShortPayRecognized = JournalSourceEventType(
		"CustomerShortPayRecognized",
	)
	JournalSourceEventCustomerPaymentReversed  = JournalSourceEventType("CustomerPaymentReversed")
	JournalSourceEventVendorBillPosted         = JournalSourceEventType("VendorBillPosted")
	JournalSourceEventVendorPaymentPosted      = JournalSourceEventType("VendorPaymentPosted")
	JournalSourceEventDriverSettlementPosted   = JournalSourceEventType("DriverSettlementPosted")
	JournalSourceEventDriverSettlementVoided   = JournalSourceEventType("DriverSettlementVoided")
	JournalSourceEventEscrowInterestAccrued    = JournalSourceEventType("EscrowInterestAccrued")
	JournalSourceEventCarrierSettlementPosted  = JournalSourceEventType("CarrierSettlementPosted")
	JournalSourceEventCarrierSettlementVoided  = JournalSourceEventType("CarrierSettlementVoided")
	JournalSourceEventCarrierSettlementPaid    = JournalSourceEventType("CarrierSettlementPaid")
	JournalSourceEventDriverSettlementPaid     = JournalSourceEventType("DriverSettlementPaid")
	JournalSourceEventFactoringFunded          = JournalSourceEventType("FactoringFunded")
	JournalSourceEventFactoringReserveReleased = JournalSourceEventType(
		"FactoringReserveReleased",
	)
	JournalSourceEventFactoringChargedBack        = JournalSourceEventType("FactoringChargedBack")
	JournalSourceEventFactoringChargebackReversed = JournalSourceEventType(
		"FactoringChargebackReversed",
	)
)

func (j JournalSourceEventType) String() string {
	return string(j)
}

func (j JournalSourceEventType) IsValid() bool {
	switch j {
	case JournalSourceEventInvoicePosted,
		JournalSourceEventCreditMemoPosted,
		JournalSourceEventDebitMemoPosted,
		JournalSourceEventCustomerPaymentPosted,
		JournalSourceEventCustomerShortPayRecognized,
		JournalSourceEventCustomerPaymentReversed,
		JournalSourceEventVendorBillPosted,
		JournalSourceEventVendorPaymentPosted,
		JournalSourceEventDriverSettlementPosted,
		JournalSourceEventDriverSettlementVoided,
		JournalSourceEventEscrowInterestAccrued,
		JournalSourceEventCarrierSettlementPosted,
		JournalSourceEventCarrierSettlementVoided,
		JournalSourceEventCarrierSettlementPaid,
		JournalSourceEventDriverSettlementPaid,
		JournalSourceEventFactoringFunded,
		JournalSourceEventFactoringReserveReleased,
		JournalSourceEventFactoringChargedBack,
		JournalSourceEventFactoringChargebackReversed:
		return true
	}
	return false
}

var journalSourceEventDescriptions = map[JournalSourceEventType]string{
	JournalSourceEventInvoicePosted:               "Trigger on posted customer invoice",
	JournalSourceEventCreditMemoPosted:            "Trigger on posted customer credit memo",
	JournalSourceEventDebitMemoPosted:             "Trigger on posted customer debit memo",
	JournalSourceEventCustomerPaymentPosted:       "Trigger on posted customer payment",
	JournalSourceEventCustomerShortPayRecognized:  "Trigger on recognized customer short pay",
	JournalSourceEventCustomerPaymentReversed:     "Trigger on reversed customer payment",
	JournalSourceEventVendorBillPosted:            "Trigger on posted vendor bill",
	JournalSourceEventVendorPaymentPosted:         "Trigger on posted vendor payment",
	JournalSourceEventDriverSettlementPosted:      "Trigger on posted driver settlement",
	JournalSourceEventDriverSettlementVoided:      "Trigger on voided driver settlement",
	JournalSourceEventEscrowInterestAccrued:       "Trigger on accrued escrow interest",
	JournalSourceEventCarrierSettlementPosted:     "Trigger on posted carrier settlement",
	JournalSourceEventCarrierSettlementVoided:     "Trigger on voided carrier settlement",
	JournalSourceEventCarrierSettlementPaid:       "Trigger on paid carrier settlement",
	JournalSourceEventDriverSettlementPaid:        "Trigger on paid driver settlement",
	JournalSourceEventFactoringFunded:             "Trigger on recorded factor funding",
	JournalSourceEventFactoringReserveReleased:    "Trigger on released factoring reserve",
	JournalSourceEventFactoringChargedBack:        "Trigger on factoring chargeback",
	JournalSourceEventFactoringChargebackReversed: "Trigger on reversed factoring chargeback",
}

func (j JournalSourceEventType) GetDescription() string {
	if description, ok := journalSourceEventDescriptions[j]; ok {
		return description
	}
	return "Unknown journal source event"
}

type ManualJournalEntryPolicy string

const (
	ManualJournalEntryPolicyAllowAll       = ManualJournalEntryPolicy("AllowAll")
	ManualJournalEntryPolicyAdjustmentOnly = ManualJournalEntryPolicy("AdjustmentOnly")
	ManualJournalEntryPolicyDisallow       = ManualJournalEntryPolicy("Disallow")
)

type JournalReversalPolicyType string

const (
	JournalReversalPolicyDisallow       = JournalReversalPolicyType("Disallow")
	JournalReversalPolicyNextOpenPeriod = JournalReversalPolicyType("NextOpenPeriod")
)

type PeriodCloseModeType string

const (
	PeriodCloseModeManualOnly      = PeriodCloseModeType("ManualOnly")
	PeriodCloseModeSystemScheduled = PeriodCloseModeType("SystemScheduled")
)

type LockedPeriodPostingPolicy string

const (
	LockedPeriodPostingPolicyBlockSubledgerAllowManualJe = LockedPeriodPostingPolicy(
		"BlockSubledgerAllowManualJe",
	)
)

type ClosedPeriodPostingPolicy string

const (
	ClosedPeriodPostingPolicyRequireReopen  = ClosedPeriodPostingPolicy("RequireReopen")
	ClosedPeriodPostingPolicyPostToNextOpen = ClosedPeriodPostingPolicy("PostToNextOpen")
)

type ReconciliationModeType string

const (
	ReconciliationModeDisabled     = ReconciliationModeType("Disabled")
	ReconciliationModeWarnOnly     = ReconciliationModeType("WarnOnly")
	ReconciliationModeBlockPosting = ReconciliationModeType("BlockPosting")
)

type CurrencyModeType string

const (
	CurrencyModeSingleCurrency = CurrencyModeType("SingleCurrency")
	CurrencyModeMultiCurrency  = CurrencyModeType("MultiCurrency")
)

type ExchangeRateDatePolicy string

const (
	ExchangeRateDatePolicyDocumentDate   = ExchangeRateDatePolicy("DocumentDate")
	ExchangeRateDatePolicyAccountingDate = ExchangeRateDatePolicy("AccountingDate")
)

type ExchangeRateOverrideType string

const (
	ExchangeRateOverrideAllow           = ExchangeRateOverrideType("Allow")
	ExchangeRateOverrideRequireApproval = ExchangeRateOverrideType("RequireApproval")
	ExchangeRateOverrideDisallow        = ExchangeRateOverrideType("Disallow")
)

func (v JournalPostingModeType) IsValid() bool {
	switch v {
	case JournalPostingModeManual,
		JournalPostingModeAutomatic:
		return true
	default:
		return false
	}
}

func (v ManualJournalEntryPolicy) IsValid() bool {
	switch v {
	case ManualJournalEntryPolicyAllowAll,
		ManualJournalEntryPolicyAdjustmentOnly,
		ManualJournalEntryPolicyDisallow:
		return true
	default:
		return false
	}
}

func (v JournalReversalPolicyType) IsValid() bool {
	switch v {
	case JournalReversalPolicyDisallow,
		JournalReversalPolicyNextOpenPeriod:
		return true
	default:
		return false
	}
}

func (v PeriodCloseModeType) IsValid() bool {
	switch v {
	case PeriodCloseModeManualOnly,
		PeriodCloseModeSystemScheduled:
		return true
	default:
		return false
	}
}

func (v LockedPeriodPostingPolicy) IsValid() bool {
	switch v {
	case LockedPeriodPostingPolicyBlockSubledgerAllowManualJe:
		return true
	default:
		return false
	}
}

func (v ClosedPeriodPostingPolicy) IsValid() bool {
	switch v {
	case ClosedPeriodPostingPolicyRequireReopen,
		ClosedPeriodPostingPolicyPostToNextOpen:
		return true
	default:
		return false
	}
}

func (v ReconciliationModeType) IsValid() bool {
	switch v {
	case ReconciliationModeDisabled,
		ReconciliationModeWarnOnly,
		ReconciliationModeBlockPosting:
		return true
	default:
		return false
	}
}

func (v CurrencyModeType) IsValid() bool {
	switch v {
	case CurrencyModeSingleCurrency,
		CurrencyModeMultiCurrency:
		return true
	default:
		return false
	}
}

func (v ExchangeRateDatePolicy) IsValid() bool {
	switch v {
	case ExchangeRateDatePolicyDocumentDate,
		ExchangeRateDatePolicyAccountingDate:
		return true
	default:
		return false
	}
}

func (v ExchangeRateOverrideType) IsValid() bool {
	switch v {
	case ExchangeRateOverrideAllow,
		ExchangeRateOverrideRequireApproval,
		ExchangeRateOverrideDisallow:
		return true
	default:
		return false
	}
}
