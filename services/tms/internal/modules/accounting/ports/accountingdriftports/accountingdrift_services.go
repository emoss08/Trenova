package accountingdriftports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingdrift"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
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
}

type RecheckAccountingDriftRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	Documents    []accountingconnectorports.AccountingChangedDocument
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
	Direction  accountingdrift.DriftDirection
}

type DismissAccountingDriftRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	Note       string
}

type AccountingDriftFixPreview struct {
	Finding         *accountingdrift.AccountingDriftFinding
	Direction       accountingdrift.DriftDirection
	FixObject       accountingdrift.DriftFixObject
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
		actor *permissionports.RequestActor,
	) (*AccountingDriftFixPreview, error)
	Resolve(
		ctx context.Context,
		req *ResolveAccountingDriftRequest,
		actor *permissionports.RequestActor,
	) (*accountingdrift.AccountingDriftFinding, error)
	PreviewDismiss(
		ctx context.Context,
		req *DismissAccountingDriftRequest,
		actor *permissionports.RequestActor,
	) (*AccountingDriftFixPreview, error)
	Dismiss(
		ctx context.Context,
		req *DismissAccountingDriftRequest,
		actor *permissionports.RequestActor,
	) (*accountingdrift.AccountingDriftFinding, error)
}

type ListAccountingDriftFindingsRequest struct {
	TenantInfo      pagination.TenantInfo
	Filter          *pagination.QueryOptions
	Cursor          pagination.CursorInfo
	IntegrationType integration.Type
	Statuses        []accountingdrift.DriftStatus
	Kinds           []accountingdrift.DriftKind
	ObjectTypes     []accountingsync.SyncObjectType
	ObjectID        pulid.ID
	Search          string
}

type AccountingDriftOverviewRequest struct {
	TenantInfo      pagination.TenantInfo
	IntegrationType integration.Type
}

type AccountingDriftOverview struct {
	ConnectionID   pulid.ID
	ProviderName   string
	CheckedAt      *int64
	CheckError     string
	ToleranceMinor int64
	CurrencyCode   string
	Summary        *AccountingDriftSummary
}

type AccountingDriftService interface {
	AccountingDriftFixer
	Overview(
		ctx context.Context,
		req *AccountingDriftOverviewRequest,
	) (*AccountingDriftOverview, error)
	List(
		ctx context.Context,
		req *ListAccountingDriftFindingsRequest,
	) (*pagination.CursorListResult[*accountingdrift.AccountingDriftFinding], error)
	Get(
		ctx context.Context,
		req *GetAccountingDriftFindingRequest,
	) (*accountingdrift.AccountingDriftFinding, error)
	CheckNow(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		integrationType integration.Type,
	) (*AccountingDriftOverview, error)
}

type GetAccountingDriftFindingRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}
