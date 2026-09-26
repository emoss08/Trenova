package accountingsync

type DriftKind string

const (
	DriftAmountMismatch          = DriftKind("AmountMismatch")
	DriftStatusMismatch          = DriftKind("StatusMismatch")
	DriftDeletedInProvider       = DriftKind("DeletedInProvider")
	DriftVoidedInProvider        = DriftKind("VoidedInProvider")
	DriftCustomerBalanceMismatch = DriftKind("CustomerBalanceMismatch")
)

func (k DriftKind) String() string { return string(k) }

func (k DriftKind) IsValid() bool {
	switch k {
	case DriftAmountMismatch,
		DriftStatusMismatch,
		DriftDeletedInProvider,
		DriftVoidedInProvider,
		DriftCustomerBalanceMismatch:
		return true
	default:
		return false
	}
}

func (k DriftKind) IsMoney() bool {
	return k == DriftAmountMismatch || k == DriftCustomerBalanceMismatch
}

func (k DriftKind) Gone() bool {
	return k == DriftDeletedInProvider || k == DriftVoidedInProvider
}

func AllDriftKinds() []DriftKind {
	return []DriftKind{
		DriftAmountMismatch,
		DriftStatusMismatch,
		DriftDeletedInProvider,
		DriftVoidedInProvider,
		DriftCustomerBalanceMismatch,
	}
}

type DriftStatus string

const (
	DriftStatusOpen      = DriftStatus("Open")
	DriftStatusResolved  = DriftStatus("Resolved")
	DriftStatusDismissed = DriftStatus("Dismissed")
)

func (s DriftStatus) String() string { return string(s) }

func (s DriftStatus) IsValid() bool {
	switch s {
	case DriftStatusOpen, DriftStatusResolved, DriftStatusDismissed:
		return true
	default:
		return false
	}
}

func AllDriftStatuses() []DriftStatus {
	return []DriftStatus{DriftStatusOpen, DriftStatusResolved, DriftStatusDismissed}
}

type DriftResolution string

const (
	DriftPushedTrenovaValue = DriftResolution("PushedTrenovaValue")
	DriftAdjustedTrenova    = DriftResolution("AdjustedTrenova")
	DriftNoLongerDiffers    = DriftResolution("NoLongerDiffers")
	DriftDismissed          = DriftResolution("Dismissed")
)

func (r DriftResolution) String() string { return string(r) }

func (r DriftResolution) IsValid() bool {
	switch r {
	case DriftPushedTrenovaValue, DriftAdjustedTrenova, DriftNoLongerDiffers, DriftDismissed:
		return true
	default:
		return false
	}
}

func AllDriftResolutions() []DriftResolution {
	return []DriftResolution{
		DriftPushedTrenovaValue,
		DriftAdjustedTrenova,
		DriftNoLongerDiffers,
		DriftDismissed,
	}
}

type DriftDirection string

const (
	DriftPushTrenovaValue = DriftDirection("PushTrenovaValue")
	DriftAdjustTrenova    = DriftDirection("AdjustTrenova")
)

func (d DriftDirection) String() string { return string(d) }

func (d DriftDirection) IsValid() bool {
	return d == DriftPushTrenovaValue || d == DriftAdjustTrenova
}

func AllDriftDirections() []DriftDirection {
	return []DriftDirection{DriftPushTrenovaValue, DriftAdjustTrenova}
}

type DriftFixObject string

const (
	DriftFixSyncRecord      = DriftFixObject("SyncRecord")
	DriftFixCreditMemo      = DriftFixObject("CreditMemo")
	DriftFixDebitMemo       = DriftFixObject("DebitMemo")
	DriftFixInvoiceVoid     = DriftFixObject("InvoiceVoid")
	DriftFixPaymentReversal = DriftFixObject("PaymentReversal")
)

func (o DriftFixObject) String() string { return string(o) }

func DriftObjectTypes() []SyncObjectType {
	return []SyncObjectType{
		SyncObjectInvoice,
		SyncObjectCreditMemo,
		SyncObjectDebitMemo,
		SyncObjectCustomerPayment,
		SyncObjectCreditApplication,
		SyncObjectCarrierBill,
		SyncObjectCarrierBillPay,
		SyncObjectDriverBill,
		SyncObjectDriverBillPay,
	}
}

func DriftBalanceObjectTypes() []SyncObjectType {
	return []SyncObjectType{SyncObjectInvoice, SyncObjectDebitMemo}
}

func DriftCreateOperations() []SyncOperation {
	return []SyncOperation{SyncOperationCreate, SyncOperationRecreate}
}
