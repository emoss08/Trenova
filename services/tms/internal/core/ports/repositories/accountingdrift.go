package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetAccountingDriftFindingRequest struct {
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
	Statuses     []accountingsync.DriftStatus
	Kinds        []accountingsync.DriftKind
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
	Open          int `bun:"open"`
	AmountOpen    int `bun:"amount_open"`
	GoneOpen      int `bun:"gone_open"`
	StatusOpen    int `bun:"status_open"`
	BalanceOpen   int `bun:"balance_open"`
	ResolvedSince int `bun:"resolved_since"`
}

type ListAccountingDriftAttentionRequest struct {
	TenantInfo     pagination.TenantInfo
	ConnectionID   pulid.ID
	DetectedBefore int64
}

type AccountingDriftAttentionGroup struct {
	Kind             accountingsync.DriftKind `bun:"kind"`
	Count            int                      `bun:"count"`
	OldestDetectedAt int64                    `bun:"oldest_detected_at"`
	SampleID         pulid.ID                 `bun:"sample_id"`
}

type AccountingDriftFindingRepository interface {
	Create(
		ctx context.Context,
		entity *accountingsync.AccountingDriftFinding,
	) (*accountingsync.AccountingDriftFinding, error)
	Update(
		ctx context.Context,
		entity *accountingsync.AccountingDriftFinding,
	) (*accountingsync.AccountingDriftFinding, error)
	GetByID(
		ctx context.Context,
		req GetAccountingDriftFindingRequest,
	) (*accountingsync.AccountingDriftFinding, error)
	ListOpen(
		ctx context.Context,
		req *ListOpenAccountingDriftFindingsRequest,
	) ([]*accountingsync.AccountingDriftFinding, error)
	ListConnection(
		ctx context.Context,
		req *ListAccountingDriftFindingsConnectionRequest,
	) (*pagination.CursorListResult[*accountingsync.AccountingDriftFinding], error)
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
	ObjectID       pulid.ID `bun:"object_id"`
	Number         string   `bun:"number"`
	PartyID        pulid.ID `bun:"party_id"`
	PartyName      string   `bun:"party_name"`
	CurrencyCode   string   `bun:"currency_code"`
	AmountMinor    int64    `bun:"amount_minor"`
	OpenMinor      int64    `bun:"open_minor"`
	AppliedMinor   int64    `bun:"applied_minor"`
	ReflectedMinor int64    `bun:"reflected_minor"`
	Voided         bool     `bun:"voided"`
	State          string   `bun:"state"`
}

type ListAccountingDriftBalancesRequest struct {
	TenantInfo      pagination.TenantInfo
	ConnectionID    pulid.ID
	DatedFrom       int64
	AfterCustomerID pulid.ID
	Customers       int
}

type AccountingDriftBalanceLine struct {
	CustomerID   pulid.ID                      `bun:"customer_id"`
	CustomerName string                        `bun:"customer_name"`
	CurrencyCode string                        `bun:"currency_code"`
	ObjectType   accountingsync.SyncObjectType `bun:"object_type"`
	ObjectID     pulid.ID                      `bun:"object_id"`
	Number       string                        `bun:"number"`
	OpenMinor    int64                         `bun:"open_minor"`
	ExternalID   string                        `bun:"external_id"`
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
