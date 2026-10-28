package accountinginbound

import (
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
)

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

func (k InboundChangeKind) EchoObjectTypes() []accountingsync.SyncObjectType {
	if k == InboundBillPayment {
		return []accountingsync.SyncObjectType{
			accountingsync.SyncObjectCarrierBillPay,
			accountingsync.SyncObjectDriverBillPay,
		}
	}
	return []accountingsync.SyncObjectType{
		accountingsync.SyncObjectCustomerPayment,
		accountingsync.SyncObjectCreditApplication,
	}
}

func (k InboundChangeKind) SyncObjectType() accountingsync.SyncObjectType {
	if k == InboundBillPayment {
		return accountingsync.SyncObjectCarrierBillPay
	}
	return accountingsync.SyncObjectCustomerPayment
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

func (k InboundDocumentKind) SyncObjectTypes() []accountingsync.SyncObjectType {
	switch k { //nolint:exhaustive // other kinds never name a Trenova document
	case InboundDocInvoice:
		return []accountingsync.SyncObjectType{
			accountingsync.SyncObjectInvoice,
			accountingsync.SyncObjectDebitMemo,
		}
	case InboundDocCreditMemo:
		return []accountingsync.SyncObjectType{accountingsync.SyncObjectCreditMemo}
	case InboundDocDebitMemo:
		return []accountingsync.SyncObjectType{accountingsync.SyncObjectDebitMemo}
	case InboundDocBill, InboundDocVendorCredit:
		return []accountingsync.SyncObjectType{
			accountingsync.SyncObjectCarrierBill,
			accountingsync.SyncObjectDriverBill,
		}
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
