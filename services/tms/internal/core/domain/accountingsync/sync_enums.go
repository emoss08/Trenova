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
)

func (t SyncObjectType) String() string { return string(t) }

func (t SyncObjectType) IsValid() bool {
	switch t {
	case SyncObjectCustomer,
		SyncObjectInvoice,
		SyncObjectCreditMemo,
		SyncObjectDebitMemo,
		SyncObjectCustomerPayment,
		SyncObjectCreditApplication:
		return true
	default:
		return false
	}
}

func (t SyncObjectType) IsSalesDocument() bool {
	return t == SyncObjectInvoice || t == SyncObjectCreditMemo || t == SyncObjectDebitMemo
}

func (t SyncObjectType) DispatchRank() int {
	switch t {
	case SyncObjectCustomer:
		return 0
	case SyncObjectInvoice, SyncObjectCreditMemo, SyncObjectDebitMemo:
		return 1
	case SyncObjectCustomerPayment, SyncObjectCreditApplication:
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
	}
}

type SyncOperation string

const (
	SyncOperationCreate = SyncOperation("Create")
	SyncOperationUpdate = SyncOperation("Update")
	SyncOperationVoid   = SyncOperation("Void")
)

func (o SyncOperation) String() string { return string(o) }

func (o SyncOperation) IsValid() bool {
	switch o {
	case SyncOperationCreate, SyncOperationUpdate, SyncOperationVoid:
		return true
	default:
		return false
	}
}

func AllSyncOperations() []SyncOperation {
	return []SyncOperation{SyncOperationCreate, SyncOperationUpdate, SyncOperationVoid}
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
	SyncSourceDependencyOf            = SyncSourceEvent("DependencyOf")
	SyncSourceSafetyNet               = SyncSourceEvent("SafetyNet")
	SyncSourceBackfill                = SyncSourceEvent("Backfill")
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
		SyncSourceDependencyOf,
		SyncSourceSafetyNet,
		SyncSourceBackfill:
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
		SyncSourceDependencyOf,
		SyncSourceSafetyNet,
		SyncSourceBackfill,
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
	case SyncObjectCustomer, SyncObjectCustomerPayment, SyncObjectCreditApplication:
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
