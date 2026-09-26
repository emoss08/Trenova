package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ReconcileAccountingDriftRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	AfterID      pulid.ID
	Limit        int
	EventBudget  int
}

type AccountingDriftBatchResult struct {
	Held     bool
	More     bool
	LastID   pulid.ID
	Compared int
	Skipped  int
	Opened   int
	Updated  int
	Resolved int
	Events   int
}

type ReconcileAccountingDriftBalancesRequest struct {
	TenantInfo      pagination.TenantInfo
	ConnectionID    pulid.ID
	AfterCustomerID pulid.ID
	Customers       int
	EventBudget     int
}

type AccountingDriftBalanceResult struct {
	Held           bool
	More           bool
	LastCustomerID pulid.ID
	Customers      int
	Skipped        int
	Opened         int
	Updated        int
	Resolved       int
	Events         int
}

type FinishAccountingDriftCheckRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	Failure      string
}

type RecheckAccountingDriftRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	Documents    []AccountingChangedDocument
	EventBudget  int
}

type AccountingDriftReconciler interface {
	ReconcileBatch(
		ctx context.Context,
		req *ReconcileAccountingDriftRequest,
	) (*AccountingDriftBatchResult, error)
	ReconcileBalances(
		ctx context.Context,
		req *ReconcileAccountingDriftBalancesRequest,
	) (*AccountingDriftBalanceResult, error)
	FinishCheck(ctx context.Context, req *FinishAccountingDriftCheckRequest) error
}

type AccountingDriftRechecker interface {
	RecheckDocuments(
		ctx context.Context,
		req *RecheckAccountingDriftRequest,
	) (*AccountingDriftBatchResult, error)
}

type AccountingDriftChecker interface {
	CheckNow(ctx context.Context, tenantInfo pagination.TenantInfo, connectionID pulid.ID) error
}

type ResolveAccountingDriftRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	Direction  accountingsync.DriftDirection
}

type DismissAccountingDriftRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	Note       string
}

type AccountingDriftFixPreview struct {
	Finding         *accountingsync.AccountingDriftFinding
	Direction       accountingsync.DriftDirection
	FixObject       accountingsync.DriftFixObject
	Operation       accountingsync.SyncOperation
	AmountMinor     int64
	CurrencyCode    string
	ToleranceMinor  int64
	WithinTolerance bool
	Summary         string
}

type AccountingDriftFixer interface {
	PreviewResolve(
		ctx context.Context,
		req *ResolveAccountingDriftRequest,
		actor *RequestActor,
	) (*AccountingDriftFixPreview, error)
	Resolve(
		ctx context.Context,
		req *ResolveAccountingDriftRequest,
		actor *RequestActor,
	) (*accountingsync.AccountingDriftFinding, error)
	PreviewDismiss(
		ctx context.Context,
		req *DismissAccountingDriftRequest,
		actor *RequestActor,
	) (*AccountingDriftFixPreview, error)
	Dismiss(
		ctx context.Context,
		req *DismissAccountingDriftRequest,
		actor *RequestActor,
	) (*accountingsync.AccountingDriftFinding, error)
}
