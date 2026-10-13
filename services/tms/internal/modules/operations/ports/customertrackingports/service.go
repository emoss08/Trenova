package customertrackingports

import (
	"context"
	"io"

	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type UpdateProfileRequest struct {
	TenantInfo      pagination.TenantInfo
	Profile         *customertracking.Profile
	ExpectedVersion *int64
	UpdatedByID     pulid.ID
}

type IssueLinkRequest struct {
	TenantInfo     pagination.TenantInfo
	ShipmentID     pulid.ID
	RecipientEmail string
	IssuedByID     pulid.ID
}

type IssuedLink struct {
	Link       *customertracking.Link
	URL        string
	Emailed    bool
	EmailError string
}

// LinkPreview is what issuing a link would do, worked out by the same checks
// IssueLink makes, with nothing written and nothing sent.
type LinkPreview struct {
	ShipmentID        pulid.ID
	ShipmentProNumber string
	CustomerID        pulid.ID
	CustomerName      string
	RecipientEmail    string
	ExpiresAt         int64
	VisibleFields     []customertracking.Field
}

type RevokeLinkRequest struct {
	TenantInfo  pagination.TenantInfo
	ShipmentID  pulid.ID
	LinkID      pulid.ID
	RevokedByID pulid.ID
}

// PublicStatus is the shipment's status as a customer reads it: four plain
// phases instead of the dispatcher's lifecycle.
type PublicStatus string

const (
	PublicStatusPlanned   = PublicStatus("Planned")
	PublicStatusInTransit = PublicStatus("InTransit")
	PublicStatusDelayed   = PublicStatus("Delayed")
	PublicStatusDelivered = PublicStatus("Delivered")
	PublicStatusCanceled  = PublicStatus("Canceled")
)

type PublicStop struct {
	Sequence         int    `json:"sequence"`
	Type             string `json:"type"`
	City             string `json:"city"`
	StateCode        string `json:"stateCode"`
	LocationName     string `json:"locationName,omitempty"`
	AddressLine      string `json:"addressLine,omitempty"`
	WindowStart      *int64 `json:"windowStart,omitempty"`
	WindowEnd        *int64 `json:"windowEnd,omitempty"`
	ArrivedAt        *int64 `json:"arrivedAt,omitempty"`
	DepartedAt       *int64 `json:"departedAt,omitempty"`
	EstimatedArrival *int64 `json:"estimatedArrival,omitempty"`
	Running          string `json:"running,omitempty"`
}

type PublicHold struct {
	Type      string `json:"type"`
	Label     string `json:"label,omitempty"`
	StartedAt int64  `json:"startedAt"`
}

type PublicNote struct {
	Text     string `json:"text"`
	PostedAt int64  `json:"postedAt"`
}

type PublicLocation struct {
	City       string `json:"city"`
	StateCode  string `json:"stateCode"`
	ReportedAt int64  `json:"reportedAt"`
}

type PublicPod struct {
	DocumentID pulid.ID `json:"documentId"`
	FileName   string   `json:"fileName"`
	FiledAt    int64    `json:"filedAt"`
}

// PublicTrackingView is the whole of what a customer tracking link reveals.
// Every optional part is filled only when the customer's profile shows that
// field; Fields lists them so the page draws only what it was given.
type PublicTrackingView struct {
	CompanyName       string                   `json:"companyName"`
	ShipmentProNumber string                   `json:"shipmentProNumber"`
	Fields            []customertracking.Field `json:"fields"`
	ExpiresAt         int64                    `json:"expiresAt"`
	Closed            bool                     `json:"closed"`
	Status            PublicStatus             `json:"status,omitempty"`
	BOL               string                   `json:"bol,omitempty"`
	CustomerReference string                   `json:"customerReference,omitempty"`
	CarrierName       string                   `json:"carrierName,omitempty"`
	Equipment         string                   `json:"equipment,omitempty"`
	Pieces            *int64                   `json:"pieces,omitempty"`
	Weight            *int64                   `json:"weight,omitempty"`
	LastLocation      *PublicLocation          `json:"lastLocation,omitempty"`
	Stops             []PublicStop             `json:"stops"`
	Holds             []PublicHold             `json:"holds,omitempty"`
	Notes             []PublicNote             `json:"notes,omitempty"`
	ProofOfDelivery   *PublicPod               `json:"proofOfDelivery,omitempty"`
}

type DownloadResult struct {
	FileName           string
	ContentType        string
	ContentLength      int64
	ContentDisposition string
	Body               io.ReadCloser
}

type EvaluateResult struct {
	Detected  int
	Recorded  int
	Delivered int
	Skipped   int
	Failed    int
}

type SweepResult struct {
	Shipments int
	EvaluateResult
}

type CustomerTrackingService interface {
	GetProfile(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		customerID pulid.ID,
	) (*customertracking.Profile, error)
	UpdateProfile(
		ctx context.Context,
		req *UpdateProfileRequest,
	) (*customertracking.Profile, error)
	IssueLink(ctx context.Context, req *IssueLinkRequest) (*IssuedLink, error)
	PreviewIssueLink(ctx context.Context, req *IssueLinkRequest) (*LinkPreview, error)
	RevokeLink(ctx context.Context, req *RevokeLinkRequest) (int, error)
	// PreviewRevokeLink lists the live links RevokeLink would revoke.
	PreviewRevokeLink(
		ctx context.Context,
		req *RevokeLinkRequest,
	) ([]*customertracking.Link, error)
	ListLinks(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
	) ([]*customertracking.Link, error)
	ListNotifications(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
	) ([]*customertracking.Notification, error)
	PreviewByToken(ctx context.Context, token string) (*PublicTrackingView, error)
	DownloadPodByToken(ctx context.Context, token string) (*DownloadResult, error)
}

// StatusNotifier owns the status-email ledger: it detects the events a
// shipment owes its customer, records each once, and delivers what is due.
type StatusNotifier interface {
	Evaluate(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
	) (*EvaluateResult, error)
	DeliverDue(ctx context.Context, tenantInfo pagination.TenantInfo) (*EvaluateResult, error)
	SweepTenant(ctx context.Context, tenantInfo pagination.TenantInfo) (*SweepResult, error)
	ListSweepTenants(ctx context.Context) ([]pagination.TenantInfo, error)
	PurgeDeadLinks(ctx context.Context) (int, error)
}

// EvaluationRequester asks for a shipment to be evaluated soon, coalescing a
// burst of changes into one evaluation. It never fails the caller.
type EvaluationRequester interface {
	RequestEvaluation(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
	)
}
