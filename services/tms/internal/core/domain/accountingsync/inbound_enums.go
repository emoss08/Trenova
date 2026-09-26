package accountingsync

type InboundPaymentPolicy string

const (
	InboundPaymentsOff     = InboundPaymentPolicy("Off")
	InboundPaymentsPropose = InboundPaymentPolicy("Propose")
	InboundPaymentsApply   = InboundPaymentPolicy("Apply")
)

func (p InboundPaymentPolicy) String() string { return string(p) }

func (p InboundPaymentPolicy) IsValid() bool {
	switch p {
	case InboundPaymentsOff, InboundPaymentsPropose, InboundPaymentsApply:
		return true
	default:
		return false
	}
}

func AllInboundPaymentPolicies() []InboundPaymentPolicy {
	return []InboundPaymentPolicy{InboundPaymentsOff, InboundPaymentsPropose, InboundPaymentsApply}
}

type InboundChangeKind string

const (
	InboundCustomerPayment = InboundChangeKind("CustomerPayment")
	InboundBillPayment     = InboundChangeKind("BillPayment")
)

func (k InboundChangeKind) String() string { return string(k) }

func (k InboundChangeKind) IsValid() bool {
	switch k {
	case InboundCustomerPayment, InboundBillPayment:
		return true
	default:
		return false
	}
}

func (k InboundChangeKind) EchoObjectTypes() []SyncObjectType {
	if k == InboundBillPayment {
		return []SyncObjectType{SyncObjectCarrierBillPay, SyncObjectDriverBillPay}
	}
	return []SyncObjectType{SyncObjectCustomerPayment, SyncObjectCreditApplication}
}

func (k InboundChangeKind) SyncObjectType() SyncObjectType {
	if k == InboundBillPayment {
		return SyncObjectCarrierBillPay
	}
	return SyncObjectCustomerPayment
}

func AllInboundChangeKinds() []InboundChangeKind {
	return []InboundChangeKind{InboundCustomerPayment, InboundBillPayment}
}

type InboundChangeStatus string

const (
	InboundStatusDetected   = InboundChangeStatus("Detected")
	InboundStatusProposed   = InboundChangeStatus("Proposed")
	InboundStatusApplied    = InboundChangeStatus("Applied")
	InboundStatusIgnored    = InboundChangeStatus("Ignored")
	InboundStatusSuperseded = InboundChangeStatus("Superseded")
)

func (s InboundChangeStatus) String() string { return string(s) }

func (s InboundChangeStatus) IsValid() bool {
	switch s {
	case InboundStatusDetected,
		InboundStatusProposed,
		InboundStatusApplied,
		InboundStatusIgnored,
		InboundStatusSuperseded:
		return true
	default:
		return false
	}
}

func (s InboundChangeStatus) IsOpen() bool {
	return s == InboundStatusDetected || s == InboundStatusProposed
}

func AllInboundChangeStatuses() []InboundChangeStatus {
	return []InboundChangeStatus{
		InboundStatusDetected,
		InboundStatusProposed,
		InboundStatusApplied,
		InboundStatusIgnored,
		InboundStatusSuperseded,
	}
}

type InboundChangeReason string

const (
	InboundReasonPolicyPropose      = InboundChangeReason("PolicyPropose")
	InboundReasonPeriodNotOpen      = InboundChangeReason("PeriodNotOpen")
	InboundReasonUnknownDocument    = InboundChangeReason("UnknownDocument")
	InboundReasonPartyMismatch      = InboundChangeReason("PartyMismatch")
	InboundReasonOverpayment        = InboundChangeReason("Overpayment")
	InboundReasonPartialBillPayment = InboundChangeReason("PartialBillPayment")
	InboundReasonAlreadyPaid        = InboundChangeReason("AlreadyPaid")
	InboundReasonCurrencyMismatch   = InboundChangeReason("CurrencyMismatch")
	InboundReasonVoided             = InboundChangeReason("Voided")
	InboundReasonNotTrenovaDocument = InboundChangeReason("NotTrenovaDocument")
	InboundReasonSentFromTrenova    = InboundChangeReason("SentFromTrenova")
	InboundReasonApplyFailed        = InboundChangeReason("ApplyFailed")
)

func (r InboundChangeReason) String() string { return string(r) }

func (r InboundChangeReason) IsValid() bool {
	switch r {
	case InboundReasonPolicyPropose,
		InboundReasonPeriodNotOpen,
		InboundReasonUnknownDocument,
		InboundReasonPartyMismatch,
		InboundReasonOverpayment,
		InboundReasonPartialBillPayment,
		InboundReasonAlreadyPaid,
		InboundReasonCurrencyMismatch,
		InboundReasonVoided,
		InboundReasonNotTrenovaDocument,
		InboundReasonSentFromTrenova,
		InboundReasonApplyFailed:
		return true
	default:
		return false
	}
}

func (r InboundChangeReason) Applicable() bool {
	return r == InboundReasonPolicyPropose || r == InboundReasonPeriodNotOpen ||
		r == InboundReasonApplyFailed
}

func AllInboundChangeReasons() []InboundChangeReason {
	return []InboundChangeReason{
		InboundReasonPolicyPropose,
		InboundReasonPeriodNotOpen,
		InboundReasonUnknownDocument,
		InboundReasonPartyMismatch,
		InboundReasonOverpayment,
		InboundReasonPartialBillPayment,
		InboundReasonAlreadyPaid,
		InboundReasonCurrencyMismatch,
		InboundReasonVoided,
		InboundReasonNotTrenovaDocument,
		InboundReasonSentFromTrenova,
		InboundReasonApplyFailed,
	}
}

type InboundDocumentKind string

const (
	InboundDocInvoice      = InboundDocumentKind("Invoice")
	InboundDocCreditMemo   = InboundDocumentKind("CreditMemo")
	InboundDocDebitMemo    = InboundDocumentKind("DebitMemo")
	InboundDocBill         = InboundDocumentKind("Bill")
	InboundDocVendorCredit = InboundDocumentKind("VendorCredit")
	InboundDocOther        = InboundDocumentKind("Other")
)

func (k InboundDocumentKind) SyncObjectTypes() []SyncObjectType {
	switch k { //nolint:exhaustive // other kinds never name a Trenova document
	case InboundDocInvoice:
		return []SyncObjectType{SyncObjectInvoice, SyncObjectDebitMemo}
	case InboundDocCreditMemo:
		return []SyncObjectType{SyncObjectCreditMemo}
	case InboundDocDebitMemo:
		return []SyncObjectType{SyncObjectDebitMemo}
	case InboundDocBill, InboundDocVendorCredit:
		return []SyncObjectType{SyncObjectCarrierBill, SyncObjectDriverBill}
	default:
		return nil
	}
}

type AppliedObjectType string

const (
	AppliedCustomerPayment       = AppliedObjectType("CustomerPayment")
	AppliedCreditMemoApplication = AppliedObjectType("CreditMemoApplication")
	AppliedCarrierSettlement     = AppliedObjectType("CarrierSettlement")
	AppliedDriverSettlement      = AppliedObjectType("DriverSettlement")
)
