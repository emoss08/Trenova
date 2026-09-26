package accountingsync

import "github.com/emoss08/trenova/internal/core/domain/billingqueue"

type SyncObjectType string

const (
	SyncObjectCustomer          = SyncObjectType("Customer")
	SyncObjectInvoice           = SyncObjectType("Invoice")
	SyncObjectCreditMemo        = SyncObjectType("CreditMemo")
	SyncObjectDebitMemo         = SyncObjectType("DebitMemo")
	SyncObjectCustomerPayment   = SyncObjectType("CustomerPayment")
	SyncObjectCreditApplication = SyncObjectType("CreditApplication")
	SyncObjectCarrierVendor     = SyncObjectType("CarrierVendor")
	SyncObjectDriverVendor      = SyncObjectType("DriverVendor")
	SyncObjectCarrierBill       = SyncObjectType("CarrierBill")
	SyncObjectCarrierBillPay    = SyncObjectType("CarrierBillPayment")
	SyncObjectDriverBill        = SyncObjectType("DriverBill")
	SyncObjectDriverBillPay     = SyncObjectType("DriverBillPayment")
)

func (t SyncObjectType) String() string { return string(t) }

func (t SyncObjectType) IsValid() bool {
	switch t {
	case SyncObjectCustomer,
		SyncObjectInvoice,
		SyncObjectCreditMemo,
		SyncObjectDebitMemo,
		SyncObjectCustomerPayment,
		SyncObjectCreditApplication,
		SyncObjectCarrierVendor,
		SyncObjectDriverVendor,
		SyncObjectCarrierBill,
		SyncObjectCarrierBillPay,
		SyncObjectDriverBill,
		SyncObjectDriverBillPay:
		return true
	default:
		return false
	}
}

func (t SyncObjectType) IsSalesDocument() bool {
	return t == SyncObjectInvoice || t == SyncObjectCreditMemo || t == SyncObjectDebitMemo
}

func (t SyncObjectType) IsVendor() bool {
	return t == SyncObjectCarrierVendor || t == SyncObjectDriverVendor
}

func (t SyncObjectType) IsBill() bool {
	return t == SyncObjectCarrierBill || t == SyncObjectDriverBill
}

func (t SyncObjectType) IsBillPayment() bool {
	return t == SyncObjectCarrierBillPay || t == SyncObjectDriverBillPay
}

func (t SyncObjectType) IsDriverSettlement() bool {
	return t == SyncObjectDriverBill || t == SyncObjectDriverBillPay
}

func (t SyncObjectType) IsPayable() bool {
	return t.IsVendor() || t.IsBill() || t.IsBillPayment()
}

func (t SyncObjectType) BillOf() SyncObjectType {
	switch t {
	case SyncObjectCarrierBillPay:
		return SyncObjectCarrierBill
	case SyncObjectDriverBillPay:
		return SyncObjectDriverBill
	default:
		return ""
	}
}

func (t SyncObjectType) VendorOf() SyncObjectType {
	switch t {
	case SyncObjectCarrierBill, SyncObjectCarrierBillPay:
		return SyncObjectCarrierVendor
	case SyncObjectDriverBill, SyncObjectDriverBillPay:
		return SyncObjectDriverVendor
	default:
		return ""
	}
}

func (t SyncObjectType) PartyTarget() (MappingTargetType, bool) {
	switch t {
	case SyncObjectCustomer:
		return TargetCustomer, true
	case SyncObjectCarrierVendor:
		return TargetCarrier, true
	case SyncObjectDriverVendor:
		return TargetDriver, true
	default:
		return "", false
	}
}

func (t SyncObjectType) NeedsDriverSettlements() bool {
	return t == SyncObjectDriverVendor || t.IsDriverSettlement()
}

func (t SyncObjectType) DispatchRank() int {
	switch t {
	case SyncObjectCustomer, SyncObjectCarrierVendor, SyncObjectDriverVendor:
		return 0
	case SyncObjectInvoice,
		SyncObjectCreditMemo,
		SyncObjectDebitMemo,
		SyncObjectCarrierBill,
		SyncObjectDriverBill:
		return 1
	case SyncObjectCustomerPayment,
		SyncObjectCreditApplication,
		SyncObjectCarrierBillPay,
		SyncObjectDriverBillPay:
		return 2
	default:
		return 3
	}
}

func AllSyncObjectTypes() []SyncObjectType {
	return []SyncObjectType{
		SyncObjectCustomer,
		SyncObjectInvoice,
		SyncObjectCreditMemo,
		SyncObjectDebitMemo,
		SyncObjectCustomerPayment,
		SyncObjectCreditApplication,
		SyncObjectCarrierVendor,
		SyncObjectDriverVendor,
		SyncObjectCarrierBill,
		SyncObjectCarrierBillPay,
		SyncObjectDriverBill,
		SyncObjectDriverBillPay,
	}
}

type SyncOperation string

const (
	SyncOperationCreate   = SyncOperation("Create")
	SyncOperationUpdate   = SyncOperation("Update")
	SyncOperationVoid     = SyncOperation("Void")
	SyncOperationRecreate = SyncOperation("Recreate")
)

func (o SyncOperation) String() string { return string(o) }

func (o SyncOperation) IsValid() bool {
	switch o {
	case SyncOperationCreate, SyncOperationUpdate, SyncOperationVoid, SyncOperationRecreate:
		return true
	default:
		return false
	}
}

func AllSyncOperations() []SyncOperation {
	return []SyncOperation{
		SyncOperationCreate,
		SyncOperationUpdate,
		SyncOperationVoid,
		SyncOperationRecreate,
	}
}

type SyncSourceEvent string

const (
	SyncSourceInvoicePosted           = SyncSourceEvent("InvoicePosted")
	SyncSourceCreditMemoPosted        = SyncSourceEvent("CreditMemoPosted")
	SyncSourceDebitMemoPosted         = SyncSourceEvent("DebitMemoPosted")
	SyncSourceAdjustmentCreditMemo    = SyncSourceEvent("AdjustmentCreditMemo")
	SyncSourceCustomerPaymentPosted   = SyncSourceEvent("CustomerPaymentPosted")
	SyncSourceCustomerPaymentApplied  = SyncSourceEvent("CustomerPaymentApplied")
	SyncSourceCustomerPaymentReversed = SyncSourceEvent("CustomerPaymentReversed")
	SyncSourceCreditMemoApplied       = SyncSourceEvent("CreditMemoApplied")
	SyncSourceCreditMemoUnapplied     = SyncSourceEvent("CreditMemoUnapplied")
	SyncSourceCustomerUpdated         = SyncSourceEvent("CustomerUpdated")
	SyncSourceCarrierSettlementPosted = SyncSourceEvent("CarrierSettlementPosted")
	SyncSourceCarrierSettlementVoided = SyncSourceEvent("CarrierSettlementVoided")
	SyncSourceCarrierSettlementPaid   = SyncSourceEvent("CarrierSettlementPaid")
	SyncSourceDriverSettlementPosted  = SyncSourceEvent("DriverSettlementPosted")
	SyncSourceDriverSettlementVoided  = SyncSourceEvent("DriverSettlementVoided")
	SyncSourceDriverSettlementPaid    = SyncSourceEvent("DriverSettlementPaid")
	SyncSourceCarrierUpdated          = SyncSourceEvent("CarrierUpdated")
	SyncSourceDriverUpdated           = SyncSourceEvent("DriverUpdated")
	SyncSourceDependencyOf            = SyncSourceEvent("DependencyOf")
	SyncSourceSafetyNet               = SyncSourceEvent("SafetyNet")
	SyncSourceBackfill                = SyncSourceEvent("Backfill")
	SyncSourceDriftResolved           = SyncSourceEvent("DriftResolved")
)

func (e SyncSourceEvent) String() string { return string(e) }

func (e SyncSourceEvent) IsValid() bool {
	switch e {
	case SyncSourceInvoicePosted,
		SyncSourceCreditMemoPosted,
		SyncSourceDebitMemoPosted,
		SyncSourceAdjustmentCreditMemo,
		SyncSourceCustomerPaymentPosted,
		SyncSourceCustomerPaymentApplied,
		SyncSourceCustomerPaymentReversed,
		SyncSourceCreditMemoApplied,
		SyncSourceCreditMemoUnapplied,
		SyncSourceCustomerUpdated,
		SyncSourceCarrierSettlementPosted,
		SyncSourceCarrierSettlementVoided,
		SyncSourceCarrierSettlementPaid,
		SyncSourceDriverSettlementPosted,
		SyncSourceDriverSettlementVoided,
		SyncSourceDriverSettlementPaid,
		SyncSourceCarrierUpdated,
		SyncSourceDriverUpdated,
		SyncSourceDependencyOf,
		SyncSourceSafetyNet,
		SyncSourceBackfill,
		SyncSourceDriftResolved:
		return true
	default:
		return false
	}
}

func AllSyncSourceEvents() []SyncSourceEvent {
	return []SyncSourceEvent{
		SyncSourceInvoicePosted,
		SyncSourceCreditMemoPosted,
		SyncSourceDebitMemoPosted,
		SyncSourceAdjustmentCreditMemo,
		SyncSourceCustomerPaymentPosted,
		SyncSourceCustomerPaymentApplied,
		SyncSourceCustomerPaymentReversed,
		SyncSourceCreditMemoApplied,
		SyncSourceCreditMemoUnapplied,
		SyncSourceCustomerUpdated,
		SyncSourceCarrierSettlementPosted,
		SyncSourceCarrierSettlementVoided,
		SyncSourceCarrierSettlementPaid,
		SyncSourceDriverSettlementPosted,
		SyncSourceDriverSettlementVoided,
		SyncSourceDriverSettlementPaid,
		SyncSourceCarrierUpdated,
		SyncSourceDriverUpdated,
		SyncSourceDependencyOf,
		SyncSourceSafetyNet,
		SyncSourceBackfill,
		SyncSourceDriftResolved,
	}
}

type SyncStatus string

const (
	SyncStatusQueued           = SyncStatus("Queued")
	SyncStatusAwaitingApproval = SyncStatus("AwaitingApproval")
	SyncStatusInFlight         = SyncStatus("InFlight")
	SyncStatusRetrying         = SyncStatus("Retrying")
	SyncStatusSynced           = SyncStatus("Synced")
	SyncStatusBlocked          = SyncStatus("Blocked")
	SyncStatusDeadLettered     = SyncStatus("DeadLettered")
	SyncStatusSkipped          = SyncStatus("Skipped")
	SyncStatusSuperseded       = SyncStatus("Superseded")
)

func (s SyncStatus) String() string { return string(s) }

func (s SyncStatus) IsValid() bool {
	switch s {
	case SyncStatusQueued,
		SyncStatusAwaitingApproval,
		SyncStatusInFlight,
		SyncStatusRetrying,
		SyncStatusSynced,
		SyncStatusBlocked,
		SyncStatusDeadLettered,
		SyncStatusSkipped,
		SyncStatusSuperseded:
		return true
	default:
		return false
	}
}

func (s SyncStatus) IsFinal() bool {
	return s == SyncStatusSynced || s == SyncStatusSkipped || s == SyncStatusSuperseded
}

func (s SyncStatus) NeedsAttention() bool {
	return s == SyncStatusBlocked || s == SyncStatusDeadLettered
}

func (s SyncStatus) Dispatchable() bool {
	return s == SyncStatusQueued || s == SyncStatusRetrying
}

func (s SyncStatus) Retryable() bool {
	return s == SyncStatusBlocked || s == SyncStatusDeadLettered || s == SyncStatusRetrying
}

func AllSyncStatuses() []SyncStatus {
	return []SyncStatus{
		SyncStatusQueued,
		SyncStatusAwaitingApproval,
		SyncStatusInFlight,
		SyncStatusRetrying,
		SyncStatusSynced,
		SyncStatusBlocked,
		SyncStatusDeadLettered,
		SyncStatusSkipped,
		SyncStatusSuperseded,
	}
}

type SyncErrorCategory string

const (
	SyncErrorTransient     = SyncErrorCategory("Transient")
	SyncErrorRateLimited   = SyncErrorCategory("RateLimited")
	SyncErrorAuth          = SyncErrorCategory("Auth")
	SyncErrorValidation    = SyncErrorCategory("Validation")
	SyncErrorMapping       = SyncErrorCategory("Mapping")
	SyncErrorClosedPeriod  = SyncErrorCategory("ClosedPeriod")
	SyncErrorCurrency      = SyncErrorCategory("Currency")
	SyncErrorDuplicate     = SyncErrorCategory("Duplicate")
	SyncErrorNotFound      = SyncErrorCategory("NotFound")
	SyncErrorConflict      = SyncErrorCategory("Conflict")
	SyncErrorConfiguration = SyncErrorCategory("Configuration")
)

func (c SyncErrorCategory) String() string { return string(c) }

func (c SyncErrorCategory) IsValid() bool {
	switch c {
	case SyncErrorTransient,
		SyncErrorRateLimited,
		SyncErrorAuth,
		SyncErrorValidation,
		SyncErrorMapping,
		SyncErrorClosedPeriod,
		SyncErrorCurrency,
		SyncErrorDuplicate,
		SyncErrorNotFound,
		SyncErrorConflict,
		SyncErrorConfiguration:
		return true
	default:
		return false
	}
}

func (c SyncErrorCategory) Retries() bool {
	return c == SyncErrorTransient || c == SyncErrorRateLimited
}

func (c SyncErrorCategory) WaitsOnConnection() bool {
	return c == SyncErrorAuth
}

func AllSyncErrorCategories() []SyncErrorCategory {
	return []SyncErrorCategory{
		SyncErrorTransient,
		SyncErrorRateLimited,
		SyncErrorAuth,
		SyncErrorValidation,
		SyncErrorMapping,
		SyncErrorClosedPeriod,
		SyncErrorCurrency,
		SyncErrorDuplicate,
		SyncErrorNotFound,
		SyncErrorConflict,
		SyncErrorConfiguration,
	}
}

type SyncAttemptOutcome string

const (
	SyncAttemptSynced       = SyncAttemptOutcome("Synced")
	SyncAttemptRetrying     = SyncAttemptOutcome("Retrying")
	SyncAttemptBlocked      = SyncAttemptOutcome("Blocked")
	SyncAttemptDeadLettered = SyncAttemptOutcome("DeadLettered")
	SyncAttemptWaiting      = SyncAttemptOutcome("Waiting")
)

func (o SyncAttemptOutcome) String() string { return string(o) }

func (o SyncAttemptOutcome) IsValid() bool {
	switch o {
	case SyncAttemptSynced,
		SyncAttemptRetrying,
		SyncAttemptBlocked,
		SyncAttemptDeadLettered,
		SyncAttemptWaiting:
		return true
	default:
		return false
	}
}

func AllSyncAttemptOutcomes() []SyncAttemptOutcome {
	return []SyncAttemptOutcome{
		SyncAttemptSynced,
		SyncAttemptRetrying,
		SyncAttemptBlocked,
		SyncAttemptDeadLettered,
		SyncAttemptWaiting,
	}
}

type BackfillStatus string

const (
	BackfillStatusQueued    = BackfillStatus("Queued")
	BackfillStatusRunning   = BackfillStatus("Running")
	BackfillStatusPaused    = BackfillStatus("Paused")
	BackfillStatusCompleted = BackfillStatus("Completed")
	BackfillStatusFailed    = BackfillStatus("Failed")
	BackfillStatusCancelled = BackfillStatus("Cancelled")
)

func (s BackfillStatus) String() string { return string(s) }

func (s BackfillStatus) IsValid() bool {
	switch s {
	case BackfillStatusQueued,
		BackfillStatusRunning,
		BackfillStatusPaused,
		BackfillStatusCompleted,
		BackfillStatusFailed,
		BackfillStatusCancelled:
		return true
	default:
		return false
	}
}

func (s BackfillStatus) InProgress() bool {
	return s == BackfillStatusQueued || s == BackfillStatusRunning || s == BackfillStatusPaused
}

func AllBackfillStatuses() []BackfillStatus {
	return []BackfillStatus{
		BackfillStatusQueued,
		BackfillStatusRunning,
		BackfillStatusPaused,
		BackfillStatusCompleted,
		BackfillStatusFailed,
		BackfillStatusCancelled,
	}
}

func SyncObjectTypeForBill(billType billingqueue.BillType) SyncObjectType {
	switch billType {
	case billingqueue.BillTypeCreditMemo:
		return SyncObjectCreditMemo
	case billingqueue.BillTypeDebitMemo:
		return SyncObjectDebitMemo
	case billingqueue.BillTypeInvoice:
		return SyncObjectInvoice
	default:
		return SyncObjectInvoice
	}
}

func (t SyncObjectType) BillType() (billingqueue.BillType, bool) {
	switch t {
	case SyncObjectInvoice:
		return billingqueue.BillTypeInvoice, true
	case SyncObjectCreditMemo:
		return billingqueue.BillTypeCreditMemo, true
	case SyncObjectDebitMemo:
		return billingqueue.BillTypeDebitMemo, true
	case SyncObjectCustomer,
		SyncObjectCustomerPayment,
		SyncObjectCreditApplication,
		SyncObjectCarrierVendor,
		SyncObjectDriverVendor,
		SyncObjectCarrierBill,
		SyncObjectCarrierBillPay,
		SyncObjectDriverBill,
		SyncObjectDriverBillPay:
		return "", false
	default:
		return "", false
	}
}

func PostedSourceEvent(billType billingqueue.BillType) SyncSourceEvent {
	switch billType {
	case billingqueue.BillTypeCreditMemo:
		return SyncSourceCreditMemoPosted
	case billingqueue.BillTypeDebitMemo:
		return SyncSourceDebitMemoPosted
	case billingqueue.BillTypeInvoice:
		return SyncSourceInvoicePosted
	default:
		return SyncSourceInvoicePosted
	}
}
