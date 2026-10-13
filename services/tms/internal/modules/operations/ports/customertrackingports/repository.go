package customertrackingports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ProfileRepository interface {
	GetByCustomer(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		customerID pulid.ID,
	) (*customertracking.Profile, error)
	ListByCustomers(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		customerIDs []pulid.ID,
	) ([]*customertracking.Profile, error)
	Create(ctx context.Context, entity *customertracking.Profile) error
	Update(ctx context.Context, entity *customertracking.Profile) error
}

type RevokeLinksRequest struct {
	TenantInfo pagination.TenantInfo
	ShipmentID pulid.ID
	LinkID     pulid.ID
	At         int64
}

type LinkRepository interface {
	Create(ctx context.Context, entity *customertracking.Link) error
	GetByHash(ctx context.Context, tokenHash string) (*customertracking.Link, error)
	GetByID(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*customertracking.Link, error)
	ListForShipment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
	) ([]*customertracking.Link, error)
	Revoke(ctx context.Context, req *RevokeLinksRequest) (int, error)
	RecordView(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		at int64,
	) error
	PurgeDead(ctx context.Context, before int64, limit int) (int, error)
}

type DeliveryOutcome struct {
	TenantInfo     pagination.TenantInfo
	ID             pulid.ID
	Status         customertracking.NotificationStatus
	SkipReason     customertracking.SkipReason
	Recipients     []string
	LinkID         pulid.ID
	EmailMessageID pulid.ID
	Attempts       int16
	LastError      string
	NextAttemptAt  int64
	SentAt         *int64
	At             int64
}

type NotificationRepository interface {
	// InsertOnce writes the row unless one with the same shipment, event and
	// subject already exists, and reports whether it wrote it.
	InsertOnce(ctx context.Context, entity *customertracking.Notification) (bool, error)
	GetByID(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*customertracking.Notification, error)
	ListForShipment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
	) ([]*customertracking.Notification, error)
	ListDue(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		now int64,
		limit int,
	) ([]*customertracking.Notification, error)
	ListTenantsWithDue(ctx context.Context, now int64) ([]pagination.TenantInfo, error)
	// Record settles a pending row and reports whether it did: false means
	// another worker settled it first.
	Record(ctx context.Context, outcome *DeliveryOutcome) (bool, error)
}

type CandidateShipmentsRequest struct {
	TenantInfo pagination.TenantInfo
	Since      int64
	Limit      int
}

type PodDocument struct {
	ShipmentID pulid.ID
	DocumentID pulid.ID
	FileName   string
	FiledAt    int64
}

type ShipmentDescriptors struct {
	CarrierName string
	Equipment   string
}

// FactRepository reads what status detection and the public page need that
// no other port serves: which shipments could owe an email, and the proof of
// delivery on file for a shipment.
type FactRepository interface {
	ListNotifyingTenants(ctx context.Context) ([]pagination.TenantInfo, error)
	ListCandidateShipments(
		ctx context.Context,
		req *CandidateShipmentsRequest,
	) ([]pulid.ID, error)
	LatestPods(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentIDs []pulid.ID,
	) (map[pulid.ID]*PodDocument, error)
	CustomerVisibleHolds(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
	) ([]PublicHold, error)
	CustomerVisibleNotes(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
		limit int,
	) ([]PublicNote, error)
	Descriptors(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
	) (*ShipmentDescriptors, error)
	LastReportedLocation(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
	) (*PublicLocation, error)
}
