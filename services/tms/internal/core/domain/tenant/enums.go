package tenant

type SequenceType string

const (
	SequenceTypeProNumber            = SequenceType("pro_number")
	SequenceTypeConsolidation        = SequenceType("consolidation")
	SequenceTypeInvoice              = SequenceType("invoice")
	SequenceTypeCreditMemo           = SequenceType("credit_memo")
	SequenceTypeDebitMemo            = SequenceType("debit_memo")
	SequenceTypeWorkOrder            = SequenceType("work_order")
	SequenceTypeJournalBatch         = SequenceType("journal_batch")
	SequenceTypeJournalEntry         = SequenceType("journal_entry")
	SequenceTypeManualJournalRequest = SequenceType("manual_journal_request")
)

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
	JournalSourceEventInvoicePosted           = JournalSourceEventType("InvoicePosted")
	JournalSourceEventCreditMemoPosted        = JournalSourceEventType("CreditMemoPosted")
	JournalSourceEventDebitMemoPosted         = JournalSourceEventType("DebitMemoPosted")
	JournalSourceEventCustomerPaymentPosted   = JournalSourceEventType("CustomerPaymentPosted")
	JournalSourceEventCustomerPaymentReversed = JournalSourceEventType("CustomerPaymentReversed")
	JournalSourceEventVendorBillPosted        = JournalSourceEventType("VendorBillPosted")
	JournalSourceEventVendorPaymentPosted     = JournalSourceEventType("VendorPaymentPosted")
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
		JournalSourceEventCustomerPaymentReversed,
		JournalSourceEventVendorBillPosted,
		JournalSourceEventVendorPaymentPosted:
		return true
	}
	return false
}

func (j JournalSourceEventType) GetDescription() string {
	switch j {
	case JournalSourceEventInvoicePosted:
		return "Trigger on posted customer invoice"
	case JournalSourceEventCreditMemoPosted:
		return "Trigger on posted customer credit memo"
	case JournalSourceEventDebitMemoPosted:
		return "Trigger on posted customer debit memo"
	case JournalSourceEventCustomerPaymentPosted:
		return "Trigger on posted customer payment"
	case JournalSourceEventCustomerPaymentReversed:
		return "Trigger on reversed customer payment"
	case JournalSourceEventVendorBillPosted:
		return "Trigger on posted vendor bill"
	case JournalSourceEventVendorPaymentPosted:
		return "Trigger on posted vendor payment"
	default:
		return "Unknown journal source event"
	}
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
	LockedPeriodPostingPolicyBlockSubledgerAllowManualJe = LockedPeriodPostingPolicy("BlockSubledgerAllowManualJe")
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

type TransferSchedule string

const (
	TransferScheduleContinuous = TransferSchedule("Continuous")
	TransferScheduleHourly     = TransferSchedule("Hourly")
	TransferScheduleDaily      = TransferSchedule("Daily")
	TransferScheduleWeekly     = TransferSchedule("Weekly")
)

type EnforcementLevel string

const (
	EnforcementLevelIgnore        = EnforcementLevel("Ignore")
	EnforcementLevelWarn          = EnforcementLevel("Warn")
	EnforcementLevelRequireReview = EnforcementLevel("RequireReview")
	EnforcementLevelBlock         = EnforcementLevel("Block")
)

type BillingExceptionDisposition string

const (
	BillingExceptionDispositionRouteToBillingReview = BillingExceptionDisposition("RouteToBillingReview")
	BillingExceptionDispositionReturnToOperations   = BillingExceptionDisposition("ReturnToOperations")
)

type ReadyToBillAssignmentMode string

const (
	ReadyToBillAssignmentModeManualOnly            = ReadyToBillAssignmentMode("ManualOnly")
	ReadyToBillAssignmentModeAutomaticWhenEligible = ReadyToBillAssignmentMode("AutomaticWhenEligible")
)

type BillingQueueTransferMode string

const (
	BillingQueueTransferModeManualOnly         = BillingQueueTransferMode("ManualOnly")
	BillingQueueTransferModeAutomaticWhenReady = BillingQueueTransferMode("AutomaticWhenReady")
)

type InvoiceDraftCreationMode string

const (
	InvoiceDraftCreationModeManualOnly               = InvoiceDraftCreationMode("ManualOnly")
	InvoiceDraftCreationModeAutomaticWhenTransferred = InvoiceDraftCreationMode("AutomaticWhenTransferred")
)

type InvoicePostingMode string

const (
	InvoicePostingModeManualReviewRequired              = InvoicePostingMode("ManualReviewRequired")
	InvoicePostingModeAutomaticWhenNoBlockingExceptions = InvoicePostingMode("AutomaticWhenNoBlockingExceptions")
)

type RateVarianceAutoResolutionMode string

const (
	RateVarianceAutoResolutionModeDisabled                    = RateVarianceAutoResolutionMode("Disabled")
	RateVarianceAutoResolutionModeBypassReviewWithinTolerance = RateVarianceAutoResolutionMode("BypassReviewWithinTolerance")
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

type AdjustmentEligibilityPolicy string

const (
	AdjustmentEligibilityDisallow             = AdjustmentEligibilityPolicy("Disallow")
	AdjustmentEligibilityAllowWithApproval    = AdjustmentEligibilityPolicy("AllowWithApproval")
	AdjustmentEligibilityAllowWithoutApproval = AdjustmentEligibilityPolicy("AllowWithoutApproval")
)

type AdjustmentAccountingDatePolicy string

const (
	AdjustmentAccountingDateUseOriginalIfOpenElseNextOpen = AdjustmentAccountingDatePolicy("UseOriginalIfOpenElseNextOpen")
	AdjustmentAccountingDateAlwaysNextOpen                = AdjustmentAccountingDatePolicy("AlwaysNextOpen")
)

type ClosedPeriodAdjustmentPolicy string

const (
	ClosedPeriodAdjustmentPolicyDisallow                         = ClosedPeriodAdjustmentPolicy("Disallow")
	ClosedPeriodAdjustmentPolicyRequireReopen                    = ClosedPeriodAdjustmentPolicy("RequireReopen")
	ClosedPeriodAdjustmentPolicyPostInNextOpenPeriodWithApproval = ClosedPeriodAdjustmentPolicy("PostInNextOpenPeriodWithApproval")
)

type RequirementPolicy string

const (
	RequirementPolicyOptional = RequirementPolicy("Optional")
	RequirementPolicyRequired = RequirementPolicy("Required")
)

type AdjustmentAttachmentPolicy string

const (
	AdjustmentAttachmentPolicyOptional                    = AdjustmentAttachmentPolicy("Optional")
	AdjustmentAttachmentPolicyRequiredForCreditOrWriteOff = AdjustmentAttachmentPolicy("RequiredForCreditOrWriteOff")
	AdjustmentAttachmentPolicyRequiredForAll              = AdjustmentAttachmentPolicy("RequiredForAll")
)

type ApprovalPolicy string

const (
	ApprovalPolicyNone            = ApprovalPolicy("None")
	ApprovalPolicyAlways          = ApprovalPolicy("Always")
	ApprovalPolicyAmountThreshold = ApprovalPolicy("AmountThreshold")
)

type WriteOffApprovalPolicy string

const (
	WriteOffApprovalPolicyDisallow                      = WriteOffApprovalPolicy("Disallow")
	WriteOffApprovalPolicyAlwaysRequireApproval         = WriteOffApprovalPolicy("AlwaysRequireApproval")
	WriteOffApprovalPolicyRequireApprovalAboveThreshold = WriteOffApprovalPolicy("RequireApprovalAboveThreshold")
)

type ReplacementInvoiceReviewPolicy string

const (
	ReplacementInvoiceReviewPolicyNoAdditionalReview                   = ReplacementInvoiceReviewPolicy("NoAdditionalReview")
	ReplacementInvoiceReviewPolicyRequireReviewWhenEconomicTermsChange = ReplacementInvoiceReviewPolicy("RequireReviewWhenEconomicTermsChange")
	ReplacementInvoiceReviewPolicyAlwaysRequireReview                  = ReplacementInvoiceReviewPolicy("AlwaysRequireReview")
)

type CustomerCreditBalancePolicy string

const (
	CustomerCreditBalancePolicyDisallow             = CustomerCreditBalancePolicy("Disallow")
	CustomerCreditBalancePolicyAllowUnappliedCredit = CustomerCreditBalancePolicy("AllowUnappliedCredit")
)

type OverCreditPolicy string

const (
	OverCreditPolicyBlock             = OverCreditPolicy("Block")
	OverCreditPolicyAllowWithApproval = OverCreditPolicy("AllowWithApproval")
)

type SupersededInvoiceVisibilityPolicy string

const (
	SupersededInvoiceVisibilityPolicyShowCurrentOnlyExternally          = SupersededInvoiceVisibilityPolicy("ShowCurrentOnlyExternally")
	SupersededInvoiceVisibilityPolicyShowCurrentAndSupersededExternally = SupersededInvoiceVisibilityPolicy("ShowCurrentAndSupersededExternally")
)
