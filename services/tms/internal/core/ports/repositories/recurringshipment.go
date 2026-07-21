package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListRecurringShipmentsRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type ListRecurringShipmentConnectionRequest struct {
	Filter                   *pagination.QueryOptions `json:"filter"`
	Cursor                   pagination.CursorInfo    `json:"-"`
	RecurringShipmentColumns []string                 `json:"-"`
}

type GetRecurringShipmentByIDRequest struct {
	ID            pulid.ID              `json:"id"`
	TenantInfo    pagination.TenantInfo `json:"-"`
	ExpandDetails bool                  `json:"expandDetails"`
}

type RecurringShipmentSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest `json:"-"`
}

type MatchRecurringShipmentsRequest struct {
	TenantInfo            pagination.TenantInfo `json:"-"`
	CustomerID            pulid.ID              `json:"customerId"`
	OriginLocationID      pulid.ID              `json:"originLocationId"`
	DestinationLocationID pulid.ID              `json:"destinationLocationId"`
}

func (r *MatchRecurringShipmentsRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if r.CustomerID.IsNil() {
		multiErr.Add("customerId", errortypes.ErrRequired, "Customer ID is required")
	}

	if r.OriginLocationID.IsNil() {
		multiErr.Add("originLocationId", errortypes.ErrRequired, "Origin location ID is required")
	}

	if r.DestinationLocationID.IsNil() {
		multiErr.Add(
			"destinationLocationId",
			errortypes.ErrRequired,
			"Destination location ID is required",
		)
	}

	if r.TenantInfo.OrgID.IsNil() {
		multiErr.Add("orgId", errortypes.ErrRequired, "Organization ID is required")
	}

	if r.TenantInfo.BuID.IsNil() {
		multiErr.Add("buId", errortypes.ErrRequired, "Business unit ID is required")
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

type LanePatternSummary struct {
	ShipmentCount   int   `json:"shipmentCount"   bun:"shipment_count"`
	FirstShipmentAt int64 `json:"firstShipmentAt" bun:"first_shipment_at"`
	LastShipmentAt  int64 `json:"lastShipmentAt"  bun:"last_shipment_at"`
}

type MatchRecurringShipmentsResponse struct {
	Matches []*recurringshipment.RecurringShipment `json:"matches"`
	Pattern *LanePatternSummary                    `json:"pattern,omitempty"`
}

type DetectLanePatternRequest struct {
	TenantInfo            pagination.TenantInfo `json:"-"`
	CustomerID            pulid.ID              `json:"customerId"`
	OriginLocationID      pulid.ID              `json:"originLocationId"`
	DestinationLocationID pulid.ID              `json:"destinationLocationId"`
	LookbackDays          int                   `json:"lookbackDays"`
	MinShipments          int                   `json:"minShipments"`
}

type GenerateRecurringShipmentRequest struct {
	TenantInfo          pagination.TenantInfo        `json:"-"`
	RecurringShipmentID pulid.ID                     `json:"recurringShipmentId"`
	OccurrenceAt        *int64                       `json:"occurrenceAt"`
	Trigger             recurringshipment.RunTrigger `json:"-"`
	RequestedBy         pulid.ID                     `json:"-"`
}

func (r *GenerateRecurringShipmentRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if r.RecurringShipmentID.IsNil() {
		multiErr.Add(
			"recurringShipmentId",
			errortypes.ErrRequired,
			"Recurring shipment ID is required",
		)
	}

	if r.TenantInfo.OrgID.IsNil() {
		multiErr.Add("orgId", errortypes.ErrRequired, "Organization ID is required")
	}

	if r.TenantInfo.BuID.IsNil() {
		multiErr.Add("buId", errortypes.ErrRequired, "Business unit ID is required")
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

type GenerateRecurringShipmentResult struct {
	Series   *recurringshipment.RecurringShipment    `json:"series"`
	Run      *recurringshipment.RecurringShipmentRun `json:"run"`
	Shipment *shipment.Shipment                      `json:"shipment,omitempty"`
}

type UpdateRecurringShipmentStatusRequest struct {
	TenantInfo          pagination.TenantInfo    `json:"-"`
	RecurringShipmentID pulid.ID                 `json:"recurringShipmentId"`
	Status              recurringshipment.Status `json:"status"`
	Version             int64                    `json:"version"`
}

type RecordRecurringGenerationFailureRequest struct {
	TenantInfo          pagination.TenantInfo `json:"-"`
	RecurringShipmentID pulid.ID              `json:"recurringShipmentId"`
	OccurrenceAt        int64                 `json:"occurrenceAt"`
	Detail              string                `json:"detail"`
}

type ListDueRecurringShipmentsRequest struct {
	Now   int64 `json:"now"`
	Limit int   `json:"limit"`
}

type ListRecurringShipmentRunsRequest struct {
	TenantInfo          pagination.TenantInfo    `json:"-"`
	RecurringShipmentID pulid.ID                 `json:"recurringShipmentId"`
	Filter              *pagination.QueryOptions `json:"filter"`
}

type RecurringShipmentRepository interface {
	List(
		ctx context.Context,
		req *ListRecurringShipmentsRequest,
	) (*pagination.ListResult[*recurringshipment.RecurringShipment], error)
	ListConnection(
		ctx context.Context,
		req *ListRecurringShipmentConnectionRequest,
	) (*pagination.CursorListResult[*recurringshipment.RecurringShipment], error)
	GetByID(
		ctx context.Context,
		req *GetRecurringShipmentByIDRequest,
	) (*recurringshipment.RecurringShipment, error)
	Create(
		ctx context.Context,
		entity *recurringshipment.RecurringShipment,
	) (*recurringshipment.RecurringShipment, error)
	Update(
		ctx context.Context,
		entity *recurringshipment.RecurringShipment,
	) (*recurringshipment.RecurringShipment, error)
	UpdateStatus(
		ctx context.Context,
		req *UpdateRecurringShipmentStatusRequest,
	) (*recurringshipment.RecurringShipment, error)
	SelectOptions(
		ctx context.Context,
		req *RecurringShipmentSelectOptionsRequest,
	) (*pagination.ListResult[*recurringshipment.RecurringShipment], error)
	Match(
		ctx context.Context,
		req *MatchRecurringShipmentsRequest,
	) ([]*recurringshipment.RecurringShipment, error)
	DetectLanePattern(
		ctx context.Context,
		req *DetectLanePatternRequest,
	) (*LanePatternSummary, error)
	ListDue(
		ctx context.Context,
		req *ListDueRecurringShipmentsRequest,
	) ([]*recurringshipment.RecurringShipment, error)
	Generate(
		ctx context.Context,
		req *GenerateRecurringShipmentRequest,
	) (*GenerateRecurringShipmentResult, error)
	RecordGenerationFailure(
		ctx context.Context,
		req *RecordRecurringGenerationFailureRequest,
	) (*recurringshipment.RecurringShipment, error)
	ListRuns(
		ctx context.Context,
		req *ListRecurringShipmentRunsRequest,
	) (*pagination.ListResult[*recurringshipment.RecurringShipmentRun], error)
}
