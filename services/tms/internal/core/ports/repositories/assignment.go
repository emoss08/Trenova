package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListAssignmentsRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetAssignmentByIDRequest struct {
	TenantInfo   pagination.TenantInfo `json:"tenantInfo"`
	AssignmentID pulid.ID              `json:"assignmentId"`
}

type AssignShipmentMoveRequest struct {
	TenantInfo        pagination.TenantInfo `json:"-"`
	ShipmentMoveID    pulid.ID              `json:"shipmentMoveId"`
	PrimaryWorkerID   pulid.ID              `json:"primaryWorkerId"`
	TractorID         pulid.ID              `json:"tractorId"`
	TrailerID         *pulid.ID             `json:"trailerId,omitempty"`
	SecondaryWorkerID *pulid.ID             `json:"secondaryWorkerId,omitempty"`
}

func (r *AssignShipmentMoveRequest) Validate() *errortypes.MultiError {
	me := errortypes.NewMultiError()

	if r == nil {
		me.Add("", errortypes.ErrInvalid, "Request is required")
		return me
	}

	if r.TenantInfo.OrgID.IsNil() {
		me.Add("tenantInfo.orgId", errortypes.ErrRequired, "Organization ID is required")
	}
	if r.TenantInfo.BuID.IsNil() {
		me.Add("tenantInfo.buId", errortypes.ErrRequired, "Business unit ID is required")
	}
	if r.ShipmentMoveID.IsNil() {
		me.Add("shipmentMoveId", errortypes.ErrRequired, "Shipment move ID is required")
	}
	if r.PrimaryWorkerID.IsNil() {
		me.Add("primaryWorkerId", errortypes.ErrRequired, "Primary worker ID is required")
	}
	if r.TractorID.IsNil() {
		me.Add("tractorId", errortypes.ErrRequired, "Tractor ID is required")
	}

	if me.HasErrors() {
		return me
	}

	return nil
}

type ReassignShipmentMoveRequest struct {
	TenantInfo        pagination.TenantInfo `json:"-"`
	ShipmentMoveID    pulid.ID              `json:"shipmentMoveId"`
	PrimaryWorkerID   pulid.ID              `json:"primaryWorkerId"`
	TractorID         pulid.ID              `json:"tractorId"`
	TrailerID         *pulid.ID             `json:"trailerId,omitempty"`
	SecondaryWorkerID *pulid.ID             `json:"secondaryWorkerId,omitempty"`
}

func (r *ReassignShipmentMoveRequest) Validate() *errortypes.MultiError {
	assignReq := &AssignShipmentMoveRequest{
		TenantInfo:        r.TenantInfo,
		ShipmentMoveID:    r.ShipmentMoveID,
		PrimaryWorkerID:   r.PrimaryWorkerID,
		TractorID:         r.TractorID,
		TrailerID:         r.TrailerID,
		SecondaryWorkerID: r.SecondaryWorkerID,
	}

	return assignReq.Validate()
}

type UnassignShipmentMoveRequest struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	ShipmentMoveID pulid.ID              `json:"shipmentMoveId"`
}

func (r *UnassignShipmentMoveRequest) Validate() *errortypes.MultiError {
	me := errortypes.NewMultiError()

	if r == nil {
		me.Add("", errortypes.ErrInvalid, "Request is required")
		return me
	}

	if r.TenantInfo.OrgID.IsNil() {
		me.Add("tenantInfo.orgId", errortypes.ErrRequired, "Organization ID is required")
	}
	if r.TenantInfo.BuID.IsNil() {
		me.Add("tenantInfo.buId", errortypes.ErrRequired, "Business unit ID is required")
	}
	if r.ShipmentMoveID.IsNil() {
		me.Add("shipmentMoveId", errortypes.ErrRequired, "Shipment move ID is required")
	}

	if me.HasErrors() {
		return me
	}

	return nil
}

type ActualTimelineEventType string

const (
	ActualTimelineEventTypeArrival   = ActualTimelineEventType("arrival")
	ActualTimelineEventTypeDeparture = ActualTimelineEventType("departure")
)

type ActualTimelineDirection string

const (
	ActualTimelineDirectionPrevious = ActualTimelineDirection("previous")
	ActualTimelineDirectionNext     = ActualTimelineDirection("next")
)

type FindNearestActualTimelineEventRequest struct {
	TenantInfo        pagination.TenantInfo
	ExcludeShipmentID pulid.ID
	Timestamp         int64
	Direction         ActualTimelineDirection
}

type ActualTimelineEvent struct {
	Timestamp      int64                   `json:"timestamp"      bun:"timestamp"`
	EventType      ActualTimelineEventType `json:"eventType"      bun:"event_type"`
	StopID         pulid.ID                `json:"stopId"         bun:"stop_id"`
	ShipmentMoveID pulid.ID                `json:"shipmentMoveId" bun:"shipment_move_id"`
	ShipmentID     pulid.ID                `json:"shipmentId"     bun:"shipment_id"`
	LocationName   string                  `json:"locationName"   bun:"location_name"`
}

type FindOverlappingActualTimelineWindowRequest struct {
	TenantInfo        pagination.TenantInfo
	ExcludeShipmentID pulid.ID
	Timestamp         int64
}

type ActualTimelineWindow struct {
	StartTimestamp int64    `json:"startTimestamp" bun:"start_timestamp"`
	EndTimestamp   int64    `json:"endTimestamp"   bun:"end_timestamp"`
	ShipmentMoveID pulid.ID `json:"shipmentMoveId" bun:"shipment_move_id"`
	ShipmentID     pulid.ID `json:"shipmentId"     bun:"shipment_id"`
}

type AssignmentRepository interface {
	List(
		ctx context.Context,
		req *ListAssignmentsRequest,
	) (*pagination.ListResult[*shipment.Assignment], error)
	GetByID(
		ctx context.Context,
		req *GetAssignmentByIDRequest,
	) (*shipment.Assignment, error)
	GetByMoveID(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		moveID pulid.ID,
	) (*shipment.Assignment, error)
	GetMoveByID(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		moveID pulid.ID,
	) (*shipment.ShipmentMove, error)
	FindInProgressByTractorID(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		tractorID pulid.ID,
		excludeMoveID pulid.ID,
	) (*shipment.Assignment, error)
	FindInProgressByTrailerID(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		trailerID pulid.ID,
		excludeMoveID pulid.ID,
	) (*shipment.Assignment, error)
	FindNearestActualEventByTractorID(
		ctx context.Context,
		req FindNearestActualTimelineEventRequest,
		tractorID pulid.ID,
	) (*ActualTimelineEvent, error)
	FindNearestActualEventByPrimaryWorkerID(
		ctx context.Context,
		req FindNearestActualTimelineEventRequest,
		workerID pulid.ID,
	) (*ActualTimelineEvent, error)
	FindOverlappingActualWindowByTractorID(
		ctx context.Context,
		req FindOverlappingActualTimelineWindowRequest,
		tractorID pulid.ID,
	) (*ActualTimelineWindow, error)
	FindOverlappingActualWindowByPrimaryWorkerID(
		ctx context.Context,
		req FindOverlappingActualTimelineWindowRequest,
		workerID pulid.ID,
	) (*ActualTimelineWindow, error)
	Create(
		ctx context.Context,
		entity *shipment.Assignment,
	) (*shipment.Assignment, error)
	Update(
		ctx context.Context,
		entity *shipment.Assignment,
	) (*shipment.Assignment, error)
	Unassign(
		ctx context.Context,
		entity *shipment.Assignment,
	) (*shipment.Assignment, error)
}
