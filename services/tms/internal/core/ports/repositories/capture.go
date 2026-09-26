package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetCaptureDeviceByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListCaptureDevicesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	// UserID narrows the list to one person's devices. It is how "my devices"
	// is served, and it is set by the service from the caller, never from the
	// request.
	UserID pulid.ID             `json:"userId"`
	Status capture.DeviceStatus `json:"status"`
}

// TouchCaptureDeviceRequest records that a device was heard from. It is not an
// edit and does not move the version: a device heartbeating while an
// administrator revokes it must not make the revocation lose an optimistic
// lock race.
type TouchCaptureDeviceRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	SeenAt     int64                 `json:"seenAt"`
	IP         string                `json:"ip"`
}

type RevokeCaptureDevicesForUserRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	UserID     pulid.ID              `json:"userId"`
	RevokedAt  int64                 `json:"revokedAt"`
	Reason     string                `json:"reason"`
}

type CaptureDeviceRepository interface {
	Create(ctx context.Context, entity *capture.CaptureDevice) (*capture.CaptureDevice, error)
	Update(ctx context.Context, entity *capture.CaptureDevice) (*capture.CaptureDevice, error)
	GetByID(
		ctx context.Context,
		req GetCaptureDeviceByIDRequest,
	) (*capture.CaptureDevice, error)
	List(
		ctx context.Context,
		req *ListCaptureDevicesRequest,
	) (*pagination.ListResult[*capture.CaptureDevice], error)
	// The three token lookups are deliberately not tenant-scoped: the token is
	// what identifies the tenant, so there is nothing to scope by until it
	// resolves. Each hash is unique across every organization.
	GetByAccessTokenHash(ctx context.Context, hash string) (*capture.CaptureDevice, error)
	GetByRefreshTokenHash(ctx context.Context, hash string) (*capture.CaptureDevice, error)
	GetByPreviousRefreshHash(ctx context.Context, hash string) (*capture.CaptureDevice, error)
	Touch(ctx context.Context, req TouchCaptureDeviceRequest) error
}

type CapturePairingRepository interface {
	Create(ctx context.Context, entity *capture.CapturePairing) (*capture.CapturePairing, error)
	Update(ctx context.Context, entity *capture.CapturePairing) (*capture.CapturePairing, error)
	GetByDeviceCodeHash(ctx context.Context, hash string) (*capture.CapturePairing, error)
	// GetOpenByUserCode finds a code a person could still approve. A code
	// that expired or was used is not found, so it cannot be approved late.
	GetOpenByUserCode(ctx context.Context, userCode string) (*capture.CapturePairing, error)
	// ExpireStale closes every open grant whose time ran out, so its user code
	// can be issued again.
	ExpireStale(ctx context.Context, now int64) (int, error)
}

type GetCaptureProfileByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListCaptureProfilesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Status capture.ProfileStatus    `json:"status"`
}

type DeleteCaptureProfileRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type CaptureProfileRepository interface {
	List(
		ctx context.Context,
		req *ListCaptureProfilesRequest,
	) (*pagination.ListResult[*capture.CaptureProfile], error)
	GetByID(
		ctx context.Context,
		req GetCaptureProfileByIDRequest,
	) (*capture.CaptureProfile, error)
	// GetDefault returns the tenant's default profile, or a not-found error
	// when it has none.
	GetDefault(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*capture.CaptureProfile, error)
	// Create and Update clear the flag on any other default in the same
	// transaction when the entity is the default, so a tenant never has two.
	Create(ctx context.Context, entity *capture.CaptureProfile) (*capture.CaptureProfile, error)
	Update(ctx context.Context, entity *capture.CaptureProfile) (*capture.CaptureProfile, error)
	Delete(ctx context.Context, req DeleteCaptureProfileRequest) error
}

type GetCaptureRequestByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type ListOpenCaptureRequestsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	DeviceID   pulid.ID              `json:"deviceId"`
}

type ListCaptureRequestsForTargetRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	UserID     pulid.ID              `json:"userId"`
	TargetType string                `json:"targetType"`
	TargetID   pulid.ID              `json:"targetId"`
	Limit      int                   `json:"limit"`
}

type CaptureRequestRepository interface {
	Create(ctx context.Context, entity *capture.CaptureRequest) (*capture.CaptureRequest, error)
	Update(ctx context.Context, entity *capture.CaptureRequest) (*capture.CaptureRequest, error)
	GetByID(
		ctx context.Context,
		req GetCaptureRequestByIDRequest,
	) (*capture.CaptureRequest, error)
	// ListOpen returns what a device still has to do, oldest first.
	ListOpen(
		ctx context.Context,
		req ListOpenCaptureRequestsRequest,
	) ([]*capture.CaptureRequest, error)
	ListForTarget(
		ctx context.Context,
		req ListCaptureRequestsForTargetRequest,
	) ([]*capture.CaptureRequest, error)
	// ListExpired returns requests still waiting on a device whose time ran
	// out, across every tenant, a bounded batch at a time.
	ListExpired(ctx context.Context, now int64, limit int) ([]*capture.CaptureRequest, error)
}

type GetCaptureBatchByIDRequest struct {
	ID           pulid.ID              `json:"id"`
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	IncludePages bool                  `json:"includePages"`
	IncludeItems bool                  `json:"includeItems"`
	// IncludeDevice loads the companion the batch came from.
	IncludeDevice bool `json:"includeDevice"`
}

type GetCaptureBatchByClientKeyRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	DeviceID   pulid.ID              `json:"deviceId"`
	ClientKey  string                `json:"clientKey"`
}

type ListCaptureBatchesRequest struct {
	Filter   *pagination.QueryOptions `json:"filter"`
	Cursor   pagination.CursorInfo    `json:"-"`
	Statuses []capture.BatchStatus    `json:"statuses"`
	Source   capture.Source           `json:"source"`
	// UserID limits the queue to one person's batches. The service sets it
	// whenever the caller's data scope is their own records.
	UserID     pulid.ID `json:"userId"`
	TargetType string   `json:"targetType"`
	TargetID   pulid.ID `json:"targetId"`
}

// ListStaleCaptureBatchesRequest finds batches stuck in a status longer than
// their work could take, so a lost workflow is noticed and restarted.
type ListStaleCaptureBatchesRequest struct {
	Statuses      []capture.BatchStatus `json:"statuses"`
	UpdatedBefore int64                 `json:"updatedBefore"`
	Limit         int                   `json:"limit"`
}

// ListRetentionDueCaptureBatchesRequest finds batches past their retention,
// settled or not, excluding those still receiving or being read.
type ListRetentionDueCaptureBatchesRequest struct {
	Now   int64 `json:"now"`
	Limit int   `json:"limit"`
}

type DeleteCaptureBatchRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type IncrementCaptureBatchPagesRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type CaptureBatchRepository interface {
	Create(ctx context.Context, entity *capture.CaptureBatch) (*capture.CaptureBatch, error)
	Update(ctx context.Context, entity *capture.CaptureBatch) (*capture.CaptureBatch, error)
	GetByID(
		ctx context.Context,
		req *GetCaptureBatchByIDRequest,
	) (*capture.CaptureBatch, error)
	GetByClientKey(
		ctx context.Context,
		req GetCaptureBatchByClientKeyRequest,
	) (*capture.CaptureBatch, error)
	ListCursor(
		ctx context.Context,
		req *ListCaptureBatchesRequest,
	) (*pagination.CursorListResult[*capture.CaptureBatch], error)
	ListStale(
		ctx context.Context,
		req ListStaleCaptureBatchesRequest,
	) ([]*capture.CaptureBatch, error)
	ListRetentionDue(
		ctx context.Context,
		req ListRetentionDueCaptureBatchesRequest,
	) ([]*capture.CaptureBatch, error)
	// IncrementReceived counts one newly stored page without touching the
	// version, because pages land concurrently and each one is not an edit.
	IncrementReceived(ctx context.Context, req IncrementCaptureBatchPagesRequest) error
	Delete(ctx context.Context, req DeleteCaptureBatchRequest) error
}

type ListCapturePagesRequest struct {
	BatchID    pulid.ID              `json:"batchId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetCapturePageBySequenceRequest struct {
	BatchID    pulid.ID              `json:"batchId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Sequence   int                   `json:"sequence"`
}

type GetCapturePageByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type CapturePageRepository interface {
	// Insert stores a page, or returns the page already stored at its
	// sequence and reports it was not new. A retried upload is therefore a
	// no-op rather than a duplicate or an error.
	Insert(ctx context.Context, entity *capture.CapturePage) (*capture.CapturePage, bool, error)
	Update(ctx context.Context, entity *capture.CapturePage) (*capture.CapturePage, error)
	GetByID(ctx context.Context, req GetCapturePageByIDRequest) (*capture.CapturePage, error)
	GetBySequence(
		ctx context.Context,
		req GetCapturePageBySequenceRequest,
	) (*capture.CapturePage, error)
	ListByBatch(ctx context.Context, req ListCapturePagesRequest) ([]*capture.CapturePage, error)
}

type ListCaptureItemsRequest struct {
	BatchID    pulid.ID              `json:"batchId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetCaptureItemByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

// ReplaceOpenCaptureItemsRequest swaps a batch's open items for a new split.
// Filed, filing and discarded items are never touched: a split can only
// rearrange what nobody has acted on yet.
type ReplaceOpenCaptureItemsRequest struct {
	BatchID    pulid.ID               `json:"batchId"`
	TenantInfo pagination.TenantInfo  `json:"tenantInfo"`
	Items      []*capture.CaptureItem `json:"items"`
}

type CaptureItemRepository interface {
	Create(ctx context.Context, entity *capture.CaptureItem) (*capture.CaptureItem, error)
	Update(ctx context.Context, entity *capture.CaptureItem) (*capture.CaptureItem, error)
	GetByID(ctx context.Context, req GetCaptureItemByIDRequest) (*capture.CaptureItem, error)
	ListByBatch(ctx context.Context, req ListCaptureItemsRequest) ([]*capture.CaptureItem, error)
	ReplaceOpen(ctx context.Context, req *ReplaceOpenCaptureItemsRequest) error
}

type GetCaptureCoverSheetByTokenRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	TokenHash  string                `json:"-"`
}

type MarkCaptureCoverSheetUsedRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	UsedAt     int64                 `json:"usedAt"`
}

type GetCaptureCoverSheetByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type CaptureCoverSheetRepository interface {
	CreateMany(ctx context.Context, entities []*capture.CaptureCoverSheet) error
	GetByID(
		ctx context.Context,
		req GetCaptureCoverSheetByIDRequest,
	) (*capture.CaptureCoverSheet, error)
	GetByTokenHash(
		ctx context.Context,
		req GetCaptureCoverSheetByTokenRequest,
	) (*capture.CaptureCoverSheet, error)
	MarkUsed(ctx context.Context, req MarkCaptureCoverSheetUsedRequest) error
}

// CaptureRecordFinder confirms a record a capture is to be filed onto exists in
// the tenant. It answers for every fileable kind in one place, so a new kind
// is added here once rather than in each caller.
type CaptureRecordFinder interface {
	Exists(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		resourceType string,
		id pulid.ID,
	) (bool, error)
	// Labels names records of one kind for display. A record that is gone or
	// outside the tenant is simply absent from the result.
	Labels(
		ctx context.Context,
		req *ListCaptureRecordLabelsRequest,
	) ([]*CaptureRecordLabel, error)
}

type ListCaptureRecordLabelsRequest struct {
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	ResourceType string                `json:"resourceType"`
	IDs          []pulid.ID            `json:"ids"`
}

// CaptureRecordLabel is how a person recognises a record a capture is filed
// onto: a shipment's PRO and BOL, a worker's name, a unit's number and plate.
type CaptureRecordLabel struct {
	ResourceType string   `json:"resourceType" bun:"-"`
	ID           pulid.ID `json:"id"           bun:"id"`
	Title        string   `json:"title"        bun:"title"`
	Subtitle     string   `json:"subtitle"     bun:"subtitle"`
}

// GetID lets a label be loaded by id.
func (l *CaptureRecordLabel) GetID() pulid.ID { return l.ID }
