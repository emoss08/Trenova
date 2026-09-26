package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var (
	ErrAccountingSyncLeaseLost = errors.New(
		"the accounting sync record was reclaimed before this push finished",
	)
	ErrAccountingBackfillActive = errors.New(
		"a backfill is already running for this connection",
	)
)

type EnqueueAccountingSyncRecordsResult struct {
	Inserted []*accountingsync.AccountingSyncRecord
	Existing int
}

type GetAccountingSyncRecordRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type GetAccountingSyncRecordsByIDsRequest struct {
	TenantInfo pagination.TenantInfo
	IDs        []pulid.ID
}

type ListAccountingSyncRecordsConnectionRequest struct {
	Filter          *pagination.QueryOptions
	Cursor          pagination.CursorInfo
	ConnectionID    pulid.ID
	Statuses        []accountingsync.SyncStatus
	ObjectTypes     []accountingsync.SyncObjectType
	ErrorCategories []accountingsync.SyncErrorCategory
	ObjectID        pulid.ID
	Search          string
}

type ListAccountingSyncRecordsByObjectsRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	ObjectTypes  []accountingsync.SyncObjectType
	ObjectIDs    []pulid.ID
}

type ListAccountingSyncRecordsByExternalIDsRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	ObjectTypes  []accountingsync.SyncObjectType
	ExternalIDs  []string
}

type CountAccountingSyncInFlightRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	ObjectTypes  []accountingsync.SyncObjectType
}

type ClaimAccountingSyncRecordsRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	Now          int64
	Lease        time.Duration
	Limit        int
}

type FinishAccountingSyncRecordRequest struct {
	Record  *accountingsync.AccountingSyncRecord
	Attempt *accountingsync.AccountingSyncAttempt
}

type AccountingSyncStatusCount struct {
	Status accountingsync.SyncStatus
	Count  int
}

type AccountingSyncConnectionRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
}

type ListAccountingSyncAttentionRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	Limit        int
}

type AccountingSyncAttentionGroup struct {
	Status         accountingsync.SyncStatus
	ErrorCategory  accountingsync.SyncErrorCategory
	Resolution     string
	Count          int
	OldestQueuedAt int64
	SampleRecordID pulid.ID
}

type AccountingSyncMappingUsageRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	MappingIDs   []pulid.ID
}

type RequeueAccountingSyncRecordsRequest struct {
	TenantInfo      pagination.TenantInfo
	ConnectionID    pulid.ID
	ErrorCategories []accountingsync.SyncErrorCategory
	IDs             []pulid.ID
	At              int64
}

type ReleaseAccountingSyncRecordsRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	IDs          []pulid.ID
	ActorID      pulid.ID
	At           int64
}

type SupersedeAccountingSyncRecordsRequest struct {
	TenantInfo     pagination.TenantInfo
	ConnectionID   pulid.ID
	ObjectType     accountingsync.SyncObjectType
	ObjectID       pulid.ID
	Operation      accountingsync.SyncOperation
	BeforeRevision int64
}

type ListAccountingSyncCandidatesRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	ObjectType   accountingsync.SyncObjectType
	Operation    accountingsync.SyncOperation
	PostedFrom   *int64
	PostedBefore *int64
	DatedFrom    int64
	UndoneBefore *int64
	AfterAt      int64
	AfterID      pulid.ID
	Limit        int
}

type AccountingSyncCandidate struct {
	ObjectType   accountingsync.SyncObjectType
	ObjectID     pulid.ID
	ObjectNumber string
	Operation    accountingsync.SyncOperation
	DocumentDate int64
	PostedAt     int64
}

type ListDueAccountingSyncConnectionsRequest struct {
	Now   int64
	Limit int
}

type AccountingSyncDueConnection struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	ConnectionID   pulid.ID
}

type PurgeAccountingSyncHistoryRequest struct {
	PayloadsSyncedBefore int64
	AttemptsBefore       int64
	Limit                int
}

type PurgeAccountingSyncHistoryResult struct {
	PayloadsCleared int64
	AttemptsDeleted int64
}

type ListAccountingSyncAttemptsRequest struct {
	TenantInfo   pagination.TenantInfo
	SyncRecordID pulid.ID
	Limit        int
}

type AccountingSyncRecordRepository interface {
	Enqueue(
		ctx context.Context,
		records []*accountingsync.AccountingSyncRecord,
	) (*EnqueueAccountingSyncRecordsResult, error)
	GetByID(
		ctx context.Context,
		req GetAccountingSyncRecordRequest,
	) (*accountingsync.AccountingSyncRecord, error)
	GetByIDs(
		ctx context.Context,
		req GetAccountingSyncRecordsByIDsRequest,
	) ([]*accountingsync.AccountingSyncRecord, error)
	ListConnection(
		ctx context.Context,
		req *ListAccountingSyncRecordsConnectionRequest,
	) (*pagination.CursorListResult[*accountingsync.AccountingSyncRecord], error)
	ListByObjects(
		ctx context.Context,
		req *ListAccountingSyncRecordsByObjectsRequest,
	) ([]*accountingsync.AccountingSyncRecord, error)
	ListByExternalIDs(
		ctx context.Context,
		req *ListAccountingSyncRecordsByExternalIDsRequest,
	) ([]*accountingsync.AccountingSyncRecord, error)
	CountInFlight(ctx context.Context, req *CountAccountingSyncInFlightRequest) (int, error)
	Claim(
		ctx context.Context,
		req *ClaimAccountingSyncRecordsRequest,
	) ([]*accountingsync.AccountingSyncRecord, error)
	Finish(ctx context.Context, req *FinishAccountingSyncRecordRequest) error
	Update(
		ctx context.Context,
		record *accountingsync.AccountingSyncRecord,
	) (*accountingsync.AccountingSyncRecord, error)
	CountByStatus(
		ctx context.Context,
		req AccountingSyncConnectionRequest,
	) ([]AccountingSyncStatusCount, error)
	ListAttention(
		ctx context.Context,
		req ListAccountingSyncAttentionRequest,
	) ([]AccountingSyncAttentionGroup, error)
	MappingUsage(
		ctx context.Context,
		req *AccountingSyncMappingUsageRequest,
	) (map[pulid.ID]int, error)
	Requeue(ctx context.Context, req *RequeueAccountingSyncRecordsRequest) (int64, error)
	Release(ctx context.Context, req *ReleaseAccountingSyncRecordsRequest) (int64, error)
	SupersedeOlder(ctx context.Context, req *SupersedeAccountingSyncRecordsRequest) (int64, error)
	ListCandidates(
		ctx context.Context,
		req *ListAccountingSyncCandidatesRequest,
	) ([]AccountingSyncCandidate, error)
	ListDueConnections(
		ctx context.Context,
		req ListDueAccountingSyncConnectionsRequest,
	) ([]AccountingSyncDueConnection, error)
	PurgeHistory(
		ctx context.Context,
		req PurgeAccountingSyncHistoryRequest,
	) (*PurgeAccountingSyncHistoryResult, error)
	ListAttempts(
		ctx context.Context,
		req ListAccountingSyncAttemptsRequest,
	) ([]*accountingsync.AccountingSyncAttempt, error)
}

type GetAccountingBackfillRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
}

type ListAccountingBackfillsRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	Limit        int
}

type AccountingBackfillRepository interface {
	Create(
		ctx context.Context,
		entity *accountingsync.AccountingBackfill,
	) (*accountingsync.AccountingBackfill, error)
	GetByID(
		ctx context.Context,
		req GetAccountingBackfillRequest,
	) (*accountingsync.AccountingBackfill, error)
	GetActive(
		ctx context.Context,
		req AccountingSyncConnectionRequest,
	) (*accountingsync.AccountingBackfill, error)
	ListByConnection(
		ctx context.Context,
		req ListAccountingBackfillsRequest,
	) ([]*accountingsync.AccountingBackfill, error)
	Update(
		ctx context.Context,
		entity *accountingsync.AccountingBackfill,
	) (*accountingsync.AccountingBackfill, error)
}
