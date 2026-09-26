package services

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AccountingSyncEnqueueRequest struct {
	TenantInfo   pagination.TenantInfo
	ObjectType   accountingsync.SyncObjectType
	ObjectID     pulid.ID
	ObjectNumber string
	Operation    accountingsync.SyncOperation
	Revision     int64
	SourceEvent  accountingsync.SyncSourceEvent
	DocumentDate int64
}

type AccountingSyncEnqueuer interface {
	Enqueue(ctx context.Context, req *AccountingSyncEnqueueRequest) error
}

// AccountingSyncPlanner says where a record would be queued for the
// accounting system, without queuing it, for a preview of the write that
// would.
type AccountingSyncPlanner interface {
	Destinations(
		ctx context.Context,
		req *AccountingSyncEnqueueRequest,
	) ([]AccountingSyncDestination, error)
}

// AccountingSyncDestination is one accounting connection a record would be
// queued for.
type AccountingSyncDestination struct {
	ConnectionID pulid.ID `json:"connectionId"`
	Integration  string   `json:"integration"`
	Company      string   `json:"company"`
	// AwaitsRelease is a connection that does not sync on its own, so the
	// record waits until a person releases it.
	AwaitsRelease bool `json:"awaitsRelease"`
}

func EnqueueAccountingSync(
	ctx context.Context,
	enqueuer AccountingSyncEnqueuer,
	req *AccountingSyncEnqueueRequest,
) error {
	if enqueuer == nil || req == nil {
		return nil
	}
	return enqueuer.Enqueue(ctx, req)
}

func InvoiceSyncRequest(
	inv *invoice.Invoice,
	source accountingsync.SyncSourceEvent,
) *AccountingSyncEnqueueRequest {
	if inv == nil {
		return nil
	}
	return &AccountingSyncEnqueueRequest{
		TenantInfo:   pagination.TenantInfo{OrgID: inv.OrganizationID, BuID: inv.BusinessUnitID},
		ObjectType:   accountingsync.SyncObjectTypeForBill(inv.BillType),
		ObjectID:     inv.ID,
		ObjectNumber: inv.Number,
		Operation:    accountingsync.SyncOperationCreate,
		Revision:     1,
		SourceEvent:  source,
		DocumentDate: inv.InvoiceDate,
	}
}

func PaymentSyncRequest(
	payment *customerpayment.Payment,
	operation accountingsync.SyncOperation,
	source accountingsync.SyncSourceEvent,
) *AccountingSyncEnqueueRequest {
	if payment == nil {
		return nil
	}
	revision := int64(1)
	if operation == accountingsync.SyncOperationUpdate {
		revision = max(payment.Version, 1)
	}
	return &AccountingSyncEnqueueRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: payment.OrganizationID,
			BuID:  payment.BusinessUnitID,
		},
		ObjectType:   accountingsync.SyncObjectCustomerPayment,
		ObjectID:     payment.ID,
		ObjectNumber: payment.ReferenceNumber,
		Operation:    operation,
		Revision:     revision,
		SourceEvent:  source,
		DocumentDate: payment.AccountingDate,
	}
}

func CreditApplicationSyncRequest(
	app *customerpayment.CreditMemoApplication,
	memoNumber string,
	operation accountingsync.SyncOperation,
	source accountingsync.SyncSourceEvent,
) *AccountingSyncEnqueueRequest {
	if app == nil {
		return nil
	}
	return &AccountingSyncEnqueueRequest{
		TenantInfo:   pagination.TenantInfo{OrgID: app.OrganizationID, BuID: app.BusinessUnitID},
		ObjectType:   accountingsync.SyncObjectCreditApplication,
		ObjectID:     app.ID,
		ObjectNumber: memoNumber,
		Operation:    operation,
		Revision:     1,
		SourceEvent:  source,
		DocumentDate: app.AccountingDate,
	}
}

func CustomerSyncRequest(
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
	name string,
	version int64,
) *AccountingSyncEnqueueRequest {
	return &AccountingSyncEnqueueRequest{
		TenantInfo:   tenantInfo,
		ObjectType:   accountingsync.SyncObjectCustomer,
		ObjectID:     customerID,
		ObjectNumber: name,
		Operation:    accountingsync.SyncOperationUpdate,
		Revision:     max(version, 1),
		SourceEvent:  accountingsync.SyncSourceCustomerUpdated,
	}
}

type SettlementSyncRequest struct {
	TenantInfo  pagination.TenantInfo
	ObjectType  accountingsync.SyncObjectType
	ObjectID    pulid.ID
	Number      string
	Operation   accountingsync.SyncOperation
	SourceEvent accountingsync.SyncSourceEvent
	PostedAt    *int64
}

func SettlementSync(req *SettlementSyncRequest) *AccountingSyncEnqueueRequest {
	if req == nil || req.PostedAt == nil {
		return nil
	}
	return &AccountingSyncEnqueueRequest{
		TenantInfo:   req.TenantInfo,
		ObjectType:   req.ObjectType,
		ObjectID:     req.ObjectID,
		ObjectNumber: req.Number,
		Operation:    req.Operation,
		Revision:     1,
		SourceEvent:  req.SourceEvent,
		DocumentDate: *req.PostedAt,
	}
}

func VendorSyncRequest(
	tenantInfo pagination.TenantInfo,
	objectType accountingsync.SyncObjectType,
	objectID pulid.ID,
	name string,
	version int64,
) *AccountingSyncEnqueueRequest {
	source := accountingsync.SyncSourceCarrierUpdated
	if objectType == accountingsync.SyncObjectDriverVendor {
		source = accountingsync.SyncSourceDriverUpdated
	}
	return &AccountingSyncEnqueueRequest{
		TenantInfo:   tenantInfo,
		ObjectType:   objectType,
		ObjectID:     objectID,
		ObjectNumber: name,
		Operation:    accountingsync.SyncOperationUpdate,
		Revision:     max(version, 1),
		SourceEvent:  source,
	}
}

type AccountingSyncDispatcher interface {
	Kick(ctx context.Context, tenantInfo pagination.TenantInfo, connectionID pulid.ID) error
	StartBackfill(ctx context.Context, backfill *accountingsync.AccountingBackfill) error
}

type AccountingSyncConnectionRef struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
}

type DrainAccountingSyncRequest struct {
	TenantInfo   pagination.TenantInfo
	ConnectionID pulid.ID
	Limit        int
	Lease        time.Duration
	Heartbeat    func()
}

type AccountingSyncDrainResult struct {
	Claimed      int  `json:"claimed"`
	Synced       int  `json:"synced"`
	Retrying     int  `json:"retrying"`
	Waiting      int  `json:"waiting"`
	Blocked      int  `json:"blocked"`
	DeadLettered int  `json:"deadLettered"`
	LeaseLost    int  `json:"leaseLost"`
	Held         bool `json:"held"`
}

type AccountingSafetyNetResult struct {
	Found   int `json:"found"`
	Queued  int `json:"queued"`
	Checked int `json:"checked"`
}

type AccountingBackfillStepResult struct {
	Done     bool `json:"done"`
	Stopped  bool `json:"stopped"`
	Enqueued int  `json:"enqueued"`
	Existing int  `json:"existing"`
}

type EnableAccountingSyncRequest struct {
	TenantInfo        pagination.TenantInfo
	UserID            pulid.ID
	IntegrationType   integration.Type
	StartDate         int64
	AutoSync          bool
	DriverSettlements bool
	Backfill          bool
}

type UpdateAccountingSyncSettingsRequest struct {
	TenantInfo        pagination.TenantInfo
	UserID            pulid.ID
	IntegrationType   integration.Type
	AutoSync          bool
	DriverSettlements bool
}

type PauseAccountingSyncRequest struct {
	TenantInfo      pagination.TenantInfo
	UserID          pulid.ID
	IntegrationType integration.Type
	Reason          string
}

type RetryAccountingSyncRequest struct {
	TenantInfo      pagination.TenantInfo
	UserID          pulid.ID
	IntegrationType integration.Type
	IDs             []pulid.ID
	ErrorCategories []accountingsync.SyncErrorCategory
}

type ReleaseAccountingSyncRequest struct {
	TenantInfo      pagination.TenantInfo
	UserID          pulid.ID
	IntegrationType integration.Type
	IDs             []pulid.ID
}

type SkipAccountingSyncRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	ID         pulid.ID
	Reason     string
}

type RequestAccountingBackfillRequest struct {
	TenantInfo      pagination.TenantInfo
	UserID          pulid.ID
	IntegrationType integration.Type
	RangeStart      *int64
	RangeEnd        *int64
	ObjectTypes     []accountingsync.SyncObjectType
}

type AccountingBackfillAction string

const (
	AccountingBackfillPause  = AccountingBackfillAction("Pause")
	AccountingBackfillResume = AccountingBackfillAction("Resume")
	AccountingBackfillCancel = AccountingBackfillAction("Cancel")
)

type ChangeAccountingBackfillRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	ID         pulid.ID
	Action     AccountingBackfillAction
}

type ListAccountingSyncRecordsRequest struct {
	TenantInfo      pagination.TenantInfo
	Filter          *pagination.QueryOptions
	IntegrationType integration.Type
	Statuses        []accountingsync.SyncStatus
	ObjectTypes     []accountingsync.SyncObjectType
	ErrorCategories []accountingsync.SyncErrorCategory
	ObjectID        pulid.ID
	Search          string
	Cursor          pagination.CursorInfo
}

type AccountingSyncSummary struct {
	IntegrationType integration.Type
	ProviderName    string
	Connection      *accountingsync.AccountingConnection
	Counts          []repositories.AccountingSyncStatusCount
	Attention       []repositories.AccountingSyncAttentionGroup
	ActiveBackfill  *accountingsync.AccountingBackfill
}

type AccountingSyncObjectState struct {
	ObjectType   accountingsync.SyncObjectType
	ObjectID     pulid.ID
	ProviderName string
	Record       *accountingsync.AccountingSyncRecord
}

type AccountingSyncService interface {
	Summary(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		integrationType integration.Type,
	) (*AccountingSyncSummary, error)
	ListRecords(
		ctx context.Context,
		req *ListAccountingSyncRecordsRequest,
	) (*pagination.CursorListResult[*accountingsync.AccountingSyncRecord], error)
	GetRecord(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*accountingsync.AccountingSyncRecord, error)
	ListAttempts(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		recordID pulid.ID,
	) ([]*accountingsync.AccountingSyncAttempt, error)
	ObjectStates(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		objectIDs []pulid.ID,
	) (map[pulid.ID]*AccountingSyncObjectState, error)
	ListBackfills(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		integrationType integration.Type,
	) ([]*accountingsync.AccountingBackfill, error)
	EnableSync(
		ctx context.Context,
		req *EnableAccountingSyncRequest,
	) (*accountingsync.AccountingConnection, error)
	UpdateSettings(
		ctx context.Context,
		req *UpdateAccountingSyncSettingsRequest,
	) (*accountingsync.AccountingConnection, error)
	Pause(
		ctx context.Context,
		req *PauseAccountingSyncRequest,
	) (*accountingsync.AccountingConnection, error)
	Resume(
		ctx context.Context,
		req *PauseAccountingSyncRequest,
	) (*accountingsync.AccountingConnection, error)
	Retry(ctx context.Context, req *RetryAccountingSyncRequest) (int64, error)
	Release(ctx context.Context, req *ReleaseAccountingSyncRequest) (int64, error)
	Skip(
		ctx context.Context,
		req *SkipAccountingSyncRequest,
	) (*accountingsync.AccountingSyncRecord, error)
	RequestBackfill(
		ctx context.Context,
		req *RequestAccountingBackfillRequest,
	) (*accountingsync.AccountingBackfill, error)
	ChangeBackfill(
		ctx context.Context,
		req *ChangeAccountingBackfillRequest,
	) (*accountingsync.AccountingBackfill, error)
	Drain(
		ctx context.Context,
		req *DrainAccountingSyncRequest,
	) (*AccountingSyncDrainResult, error)
	ListDueConnections(
		ctx context.Context,
		limit int,
	) ([]repositories.AccountingSyncDueConnection, error)
	SafetyNet(
		ctx context.Context,
		ref AccountingSyncConnectionRef,
	) (*AccountingSafetyNetResult, error)
	BackfillStep(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		backfillID pulid.ID,
	) (*AccountingBackfillStepResult, error)
	PurgeHistory(ctx context.Context) (*repositories.PurgeAccountingSyncHistoryResult, error)
}
