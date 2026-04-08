package tenant

type SequenceType string

const (
	SequenceTypeProNumber     = SequenceType("pro_number")
	SequenceTypeConsolidation = SequenceType("consolidation")
	SequenceTypeInvoice       = SequenceType("invoice")
	SequenceTypeCreditMemo    = SequenceType("credit_memo")
	SequenceTypeDebitMemo     = SequenceType("debit_memo")
	SequenceTypeWorkOrder     = SequenceType("work_order")
)

type AccountingMethodType string

const (
	AccountingMethodAccrual = AccountingMethodType("Accrual")
	AccountingMethodCash    = AccountingMethodType("Cash")
	AccountingMethodHybrid  = AccountingMethodType("Hybrid")
)

func (a AccountingMethodType) String() string {
	return string(a)
}

func (a AccountingMethodType) IsValid() bool {
	switch a {
	case AccountingMethodAccrual, AccountingMethodCash, AccountingMethodHybrid:
		return true
	}
	return false
}

func (a AccountingMethodType) GetDescription() string {
	switch a {
	case AccountingMethodAccrual:
		return "Record revenue when earned and expenses when incurred, regardless of when cash is exchanged (GAAP/ASC 606 compliant)"
	case AccountingMethodCash:
		return "Record revenue when payment is received and expenses when payment is made (not GAAP compliant)"
	case AccountingMethodHybrid:
		return "Record revenue on an accrual basis with expenses on a cash basis, commonly used for tax reporting under IRS guidelines"
	default:
		return "Unknown accounting method"
	}
}

// ValidRevenueRecognitionMethods returns the set of revenue recognition methods
// that are permitted under this accounting method.
func (a AccountingMethodType) ValidRevenueRecognitionMethods() []RevenueRecognitionType {
	switch a {
	case AccountingMethodCash:
		return []RevenueRecognitionType{RevenueRecognitionOnPayment}
	case AccountingMethodAccrual, AccountingMethodHybrid:
		return []RevenueRecognitionType{
			RevenueRecognitionOnDelivery,
			RevenueRecognitionOnBilling,
			RevenueRecognitionOnPickup,
		}
	default:
		return nil
	}
}

// ValidExpenseRecognitionMethods returns the set of expense recognition methods
// that are permitted under this accounting method.
func (a AccountingMethodType) ValidExpenseRecognitionMethods() []ExpenseRecognitionType {
	switch a {
	case AccountingMethodCash, AccountingMethodHybrid:
		return []ExpenseRecognitionType{ExpenseRecognitionOnPayment}
	case AccountingMethodAccrual:
		return []ExpenseRecognitionType{
			ExpenseRecognitionOnIncurrence,
			ExpenseRecognitionOnAccrual,
		}
	default:
		return nil
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
		return "Recognize revenue when goods are delivered (ASC 606 point-in-time)"
	case RevenueRecognitionOnBilling:
		return "Recognize revenue when invoice is created"
	case RevenueRecognitionOnPayment:
		return "Recognize revenue when payment is received (cash basis)"
	case RevenueRecognitionOnPickup:
		return "Recognize revenue when goods are picked up by the carrier"
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
	case ExpenseRecognitionOnIncurrence, ExpenseRecognitionOnPayment,
		ExpenseRecognitionOnAccrual:
		return true
	}
	return false
}

func (e ExpenseRecognitionType) GetDescription() string {
	switch e {
	case ExpenseRecognitionOnIncurrence:
		return "Recognize expense when service is performed or goods received (accrual basis)"
	case ExpenseRecognitionOnPayment:
		return "Recognize expense when payment is made (cash basis)"
	case ExpenseRecognitionOnAccrual:
		return "Recognize expense when vendor bill is received and accepted (accrual basis)"
	default:
		return "Unknown method"
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

type TransferSchedule string

const (
	TransferScheduleContinuous = TransferSchedule("Continuous")
	TransferScheduleHourly     = TransferSchedule("Hourly")
	TransferScheduleDaily      = TransferSchedule("Daily")
	TransferScheduleWeekly     = TransferSchedule("Weekly")
)

type ExceptionHandling string

const (
	BillingExceptionQueue       = ExceptionHandling("Queue")
	BillingExceptionNotify      = ExceptionHandling("Notify")
	BillingExceptionAutoResolve = ExceptionHandling("AutoResolve")
	BillingExceptionReject      = ExceptionHandling("Reject")
)

type PaymentTerm string

const (
	PaymentTermNet15        = PaymentTerm("Net15")
	PaymentTermNet30        = PaymentTerm("Net30")
	PaymentTermNet45        = PaymentTerm("Net45")
	PaymentTermNet60        = PaymentTerm("Net60")
	PaymentTermNet90        = PaymentTerm("Net90")
	PaymentTermDueOnReceipt = PaymentTerm("DueOnReceipt")
)
