package checkcallports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/trackingevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type CheckCallDetails struct {
	Kind             checkcall.Kind
	StopID           pulid.ID
	ReportedAt       *int64
	Latitude         *float64
	Longitude        *float64
	City             string
	StateCode        string
	Delayed          bool
	DelayReason      string
	EstimatedArrival *int64
	Note             string
	ContactName      string
}

type LogCheckCallRequest struct {
	TenantInfo   pagination.TenantInfo
	MoveID       pulid.ID
	Source       checkcall.Source
	SourceKey    string
	ReportedByID pulid.ID
	Details      CheckCallDetails
}

type CheckCallResult struct {
	CheckCall *checkcall.CheckCall
	Outcome   trackingevent.Outcome
	Reason    string
	Replayed  bool
}

func StopActualAction(kind checkcall.Kind) shipmentports.StopActualAction {
	if kind == checkcall.KindDeparture {
		return shipmentports.StopActualActionDepart
	}
	return shipmentports.StopActualActionArrive
}

type CheckCallPreview struct {
	CheckCall *checkcall.CheckCall
	Move      *shipment.ShipmentMove
}

type IssueTrackingLinkRequest struct {
	TenantInfo     pagination.TenantInfo
	MoveID         pulid.ID
	RecipientEmail string
	IssuedByID     pulid.ID
}

type IssuedTrackingLink struct {
	Link       *checkcall.TrackingLink
	URL        string
	Emailed    bool
	EmailError string
}

type RevokeTrackingLinkRequest struct {
	TenantInfo pagination.TenantInfo
	MoveID     pulid.ID
}

type PublicTrackingStop struct {
	ID           pulid.ID `json:"id"`
	Sequence     int64    `json:"sequence"`
	Type         string   `json:"type"`
	LocationName string   `json:"locationName"`
	City         string   `json:"city"`
	StateCode    string   `json:"stateCode"`
	WindowStart  int64    `json:"windowStart"`
	WindowEnd    *int64   `json:"windowEnd"`
	ArrivedAt    *int64   `json:"arrivedAt"`
	DepartedAt   *int64   `json:"departedAt"`
}

type PublicTrackingView struct {
	CompanyName       string               `json:"companyName"`
	CarrierName       string               `json:"carrierName"`
	ShipmentProNumber string               `json:"shipmentProNumber"`
	Closed            bool                 `json:"closed"`
	ExpiresAt         int64                `json:"expiresAt"`
	LastCheckCallAt   *int64               `json:"lastCheckCallAt"`
	Stops             []PublicTrackingStop `json:"stops"`
}

type PublicCheckCallInput struct {
	Details       CheckCallDetails
	ClientEventID string
}

type PublicCheckCallResult struct {
	Outcome  trackingevent.Outcome `json:"outcome,omitempty"`
	Reason   string                `json:"reason,omitempty"`
	Replayed bool                  `json:"replayed"`
}

type OverdueMove struct {
	MoveID        pulid.ID
	ShipmentID    pulid.ID
	ProNumber     string
	CarrierID     pulid.ID
	CarrierName   string
	LastContactAt int64
	IntervalHours int16
	OverdueBy     int64
}

type CheckCallService interface {
	Log(ctx context.Context, req *LogCheckCallRequest) (*CheckCallResult, error)
	PreviewLog(ctx context.Context, req *LogCheckCallRequest) (*CheckCallPreview, error)
	ListForShipment(
		ctx context.Context,
		req *ListCheckCallsByShipmentRequest,
	) ([]*checkcall.CheckCall, error)
	IssueLink(ctx context.Context, req *IssueTrackingLinkRequest) (*IssuedTrackingLink, error)
	RevokeLink(ctx context.Context, req *RevokeTrackingLinkRequest) error
	ActiveLink(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		moveID pulid.ID,
	) (*checkcall.TrackingLink, error)
	PreviewByToken(ctx context.Context, token string) (*PublicTrackingView, error)
	PostByToken(
		ctx context.Context,
		token string,
		input *PublicCheckCallInput,
	) (*PublicCheckCallResult, error)
	ListOverdue(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		now int64,
	) ([]*OverdueMove, error)
}

type CheckCallObserver interface {
	CheckCallRecorded(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID, moveID pulid.ID,
	)
}
