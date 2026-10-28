package accountingdriftports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingdrift"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetAccountingDriftFindingQuery struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	ForUpdate  bool
}

type ListOpenAccountingDriftFindingsRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	ObjectIDs    []pulid.ID
}

type ListAccountingDriftFindingsConnectionRequest struct {
	Filter       *pagination.QueryOptions
	Cursor       pagination.CursorInfo
	ConnectionID pulid.ID
	Statuses     []accountingdrift.DriftStatus
	Kinds        []accountingdrift.DriftKind
	ObjectTypes  []accountingsync.SyncObjectType
	ObjectID     pulid.ID
	Search       string
}

type SummarizeAccountingDriftRequest struct {
	TenantInfo    pagination.TenantInfo
	ConnectionID  pulid.ID
	ResolvedSince int64
}

type AccountingDriftSummary struct {
	Open          int
	AmountOpen    int
	GoneOpen      int
	StatusOpen    int
	BalanceOpen   int
	ResolvedSince int
}

type ListAccountingDriftAttentionRequest struct {
	TenantInfo     pagination.TenantInfo
	ConnectionID   pulid.ID
	DetectedBefore int64
}

type AccountingDriftAttentionGroup struct {
	Kind             accountingdrift.DriftKind
	Count            int
	OldestDetectedAt int64
	SampleID         pulid.ID
}

type AccountingDriftFindingRepository interface {
	Create(
		ctx context.Context,
		entity *accountingdrift.AccountingDriftFinding,
	) (*accountingdrift.AccountingDriftFinding, error)
	Update(
		ctx context.Context,
		entity *accountingdrift.AccountingDriftFinding,
	) (*accountingdrift.AccountingDriftFinding, error)
	GetByID(
		ctx context.Context,
		req GetAccountingDriftFindingQuery,
	) (*accountingdrift.AccountingDriftFinding, error)
	ListOpen(
		ctx context.Context,
		req *ListOpenAccountingDriftFindingsRequest,
	) ([]*accountingdrift.AccountingDriftFinding, error)
	ListConnection(
		ctx context.Context,
		req *ListAccountingDriftFindingsConnectionRequest,
	) (*pagination.CursorListResult[*accountingdrift.AccountingDriftFinding], error)
	Summarize(
		ctx context.Context,
		req *SummarizeAccountingDriftRequest,
	) (*AccountingDriftSummary, error)
	ListAttention(
		ctx context.Context,
		req *ListAccountingDriftAttentionRequest,
	) ([]AccountingDriftAttentionGroup, error)
}

type ListAccountingDriftStatesRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	ObjectType   accountingsync.SyncObjectType
	ObjectIDs    []pulid.ID
}

type AccountingDriftState struct {
	ObjectID       pulid.ID
	Number         string
	PartyID        pulid.ID
	PartyName      string
	CurrencyCode   string
	AmountMinor    int64
	OpenMinor      int64
	AppliedMinor   int64
	ReflectedMinor int64
	Voided         bool
	State          string
}

type ListAccountingDriftBalancesRequest struct {
	TenantInfo      pagination.TenantInfo
	ConnectionID    pulid.ID
	DatedFrom       int64
	AfterCustomerID pulid.ID
	Customers       int
}

type AccountingDriftBalanceLine struct {
	CustomerID   pulid.ID
	CustomerName string
	CurrencyCode string
	ObjectType   accountingsync.SyncObjectType
	ObjectID     pulid.ID
	Number       string
	OpenMinor    int64
	ExternalID   string
}

type ListAccountingDriftRecordsRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	DatedFrom    int64
	AfterID      pulid.ID
	Limit        int
}

type ListAccountingDriftPendingCustomersRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	CustomerIDs  []pulid.ID
}

type AccountingDriftSource interface {
	ScopeStart(ctx context.Context, tenantInfo pagination.TenantInfo) (int64, error)
	ListRecords(
		ctx context.Context,
		req *ListAccountingDriftRecordsRequest,
	) ([]*accountingsync.AccountingSyncRecord, error)
	ListStates(
		ctx context.Context,
		req *ListAccountingDriftStatesRequest,
	) ([]*AccountingDriftState, error)
	ListBalances(
		ctx context.Context,
		req *ListAccountingDriftBalancesRequest,
	) ([]*AccountingDriftBalanceLine, error)
	ListPendingCustomers(
		ctx context.Context,
		req *ListAccountingDriftPendingCustomersRequest,
	) ([]pulid.ID, error)
}
