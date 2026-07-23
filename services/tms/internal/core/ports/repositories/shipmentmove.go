package repositories

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type GetMoveByIDRequest struct {
	MoveID            pulid.ID              `json:"moveId"`
	TenantInfo        pagination.TenantInfo `json:"-"`
	ExpandMoveDetails bool                  `json:"expandMoveDetails"`
	ForUpdate         bool                  `json:"-"`
}

type GetMovesByShipmentIDRequest struct {
	ShipmentID        pulid.ID              `json:"shipmentId"`
	TenantInfo        pagination.TenantInfo `json:"-"`
	ExpandMoveDetails bool                  `json:"expandMoveDetails"`
}

type UpdateMoveStatusRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	MoveID     pulid.ID              `json:"moveId"`
	Status     shipment.MoveStatus   `json:"status"`
}

type StopActualAction string

const (
	StopActualActionArrive = StopActualAction("Arrive")
	StopActualActionDepart = StopActualAction("Depart")
)

func (a StopActualAction) IsValid() bool {
	return a == StopActualActionArrive || a == StopActualActionDepart
}

type RecordStopActualRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	MoveID     pulid.ID              `json:"moveId"`
	StopID     pulid.ID              `json:"stopId"`
	Action     StopActualAction      `json:"action"`
}

func (r *RecordStopActualRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if r == nil {
		multiErr.Add("", errortypes.ErrInvalid, "Request is required")
		return multiErr
	}
	if r.TenantInfo.OrgID.IsNil() {
		multiErr.Add("tenantInfo.orgId", errortypes.ErrRequired, "Organization ID is required")
	}
	if r.TenantInfo.BuID.IsNil() {
		multiErr.Add("tenantInfo.buId", errortypes.ErrRequired, "Business unit ID is required")
	}
	if r.MoveID.IsNil() {
		multiErr.Add("moveId", errortypes.ErrRequired, "Move ID is required")
	}
	if r.StopID.IsNil() {
		multiErr.Add("stopId", errortypes.ErrRequired, "Stop ID is required")
	}
	if !r.Action.IsValid() {
		multiErr.Add("action", errortypes.ErrInvalid, "Action must be Arrive or Depart")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (r *UpdateMoveStatusRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(&r.MoveID, validation.Required.Error("Move ID is required")),
		validation.Field(
			&r.TenantInfo.OrgID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&r.TenantInfo.BuID,
			validation.Required.Error("Business unit ID is required"),
		),
		validation.Field(
			&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				shipment.MoveStatusNew,
				shipment.MoveStatusAssigned,
				shipment.MoveStatusInTransit,
				shipment.MoveStatusCompleted,
				shipment.MoveStatusCanceled,
			).Error("Status must be a valid move status"),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

type BulkUpdateMoveStatusRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	MoveIDs    []pulid.ID            `json:"moveIds"`
	Status     shipment.MoveStatus   `json:"status"`
}

func (r *BulkUpdateMoveStatusRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(
			&r.MoveIDs,
			validation.Required.Error("Move IDs are required"),
			validation.Length(1, 500),
		),
		validation.Field(
			&r.TenantInfo.OrgID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&r.TenantInfo.BuID,
			validation.Required.Error("Business unit ID is required"),
		),
		validation.Field(
			&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				shipment.MoveStatusNew,
				shipment.MoveStatusAssigned,
				shipment.MoveStatusInTransit,
				shipment.MoveStatusCompleted,
				shipment.MoveStatusCanceled,
			).Error("Status must be a valid move status"),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

type SplitStopTimes struct {
	ScheduledWindowStart int64  `json:"scheduledWindowStart"`
	ScheduledWindowEnd   *int64 `json:"scheduledWindowEnd"`
}

type SplitMoveRequest struct {
	TenantInfo            pagination.TenantInfo `json:"-"`
	MoveID                pulid.ID              `json:"moveId"`
	NewDeliveryLocationID pulid.ID              `json:"newDeliveryLocationId"`
	SplitPickupTimes      SplitStopTimes        `json:"splitPickupTimes"`
	NewDeliveryTimes      SplitStopTimes        `json:"newDeliveryTimes"`
	Pieces                *int64                `json:"pieces,omitempty"`
	Weight                *int64                `json:"weight,omitempty"`
}

func (r *SplitMoveRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(&r.MoveID, validation.Required.Error("Move ID is required")),
		validation.Field(
			&r.NewDeliveryLocationID,
			validation.Required.Error("New delivery location ID is required"),
		),
		validation.Field(
			&r.TenantInfo.OrgID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&r.TenantInfo.BuID,
			validation.Required.Error("Business unit ID is required"),
		),
		validation.Field(
			&r.SplitPickupTimes.ScheduledWindowStart,
			validation.Required.Error("Split pickup scheduled window start is required"),
		),
		validation.Field(
			&r.NewDeliveryTimes.ScheduledWindowStart,
			validation.Required.Error("New delivery scheduled window start is required"),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	if r.SplitPickupTimes.ScheduledWindowEnd != nil &&
		*r.SplitPickupTimes.ScheduledWindowEnd < r.SplitPickupTimes.ScheduledWindowStart {
		multiErr.Add(
			"splitPickupTimes.scheduledWindowEnd",
			errortypes.ErrInvalid,
			"Split pickup scheduled window end must be greater than or equal to the scheduled window start",
		)
	}

	if r.NewDeliveryTimes.ScheduledWindowEnd != nil &&
		*r.NewDeliveryTimes.ScheduledWindowEnd < r.NewDeliveryTimes.ScheduledWindowStart {
		multiErr.Add(
			"newDeliveryTimes.scheduledWindowEnd",
			errortypes.ErrInvalid,
			"New delivery scheduled window end must be greater than or equal to the scheduled window start",
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

type SplitMoveResponse struct {
	OriginalMove *shipment.ShipmentMove `json:"originalMove,omitempty"`
	NewMove      *shipment.ShipmentMove `json:"newMove,omitempty"`
}

type ShipmentMoveRepository interface {
	SyncForShipment(
		ctx context.Context,
		tx bun.IDB,
		entity *shipment.Shipment,
	) error
	GetByID(
		ctx context.Context,
		req *GetMoveByIDRequest,
	) (*shipment.ShipmentMove, error)
	GetMovesByShipmentID(
		ctx context.Context,
		req *GetMovesByShipmentIDRequest,
	) ([]*shipment.ShipmentMove, error)
	UpdateStatus(
		ctx context.Context,
		req *UpdateMoveStatusRequest,
	) (*shipment.ShipmentMove, error)
	UpdateStopActuals(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		stop *shipment.Stop,
	) (*shipment.Stop, error)
	BulkUpdateStatus(
		ctx context.Context,
		req *BulkUpdateMoveStatusRequest,
	) ([]*shipment.ShipmentMove, error)
	SplitMove(
		ctx context.Context,
		req *SplitMoveRequest,
	) (*SplitMoveResponse, error)
}
