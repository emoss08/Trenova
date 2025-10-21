package repositories

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type GetMoveByIDRequest struct {
	MoveID            pulid.ID
	OrgID             pulid.ID
	BuID              pulid.ID
	ExpandMoveDetails bool `query:"expandMoveDetails"`
}

type UpdateMoveStatusRequest struct {
	GetMoveReq GetMoveByIDRequest
	Status     shipment.MoveStatus
}

type BulkUpdateMoveStatusRequest struct {
	MoveIDs []pulid.ID
	Status  shipment.MoveStatus
}

type GetMovesByShipmentIDRequest struct {
	ShipmentID pulid.ID
	OrgID      pulid.ID
	BuID       pulid.ID
}

type SplitQuantity struct {
	Pieces *int `json:"pieces"`
	Weight *int `json:"weight"`
}

type SplitStopTimes struct {
	PlannedArrival   int64 `json:"plannedArrival"`
	PlannedDeparture int64 `json:"plannedDeparture"`
}

type SplitMoveRequest struct {
	MoveID                 pulid.ID       `json:"moveId"`
	OrgID                  pulid.ID       `json:"organizationId"`
	BuID                   pulid.ID       `json:"businessUnitId"`
	SplitLocationID        pulid.ID       `json:"splitLocationId"`
	SplitQuantities        SplitQuantity  `json:"splitQuantities"`
	SplitAfterStopSequence int            `json:"splitAfterStopSequence"`
	SplitDeliveryTimes     SplitStopTimes `json:"splitDeliveryTimes"`
	SplitPickupTimes       SplitStopTimes `json:"splitPickupTimes"`
}

func (smr *SplitMoveRequest) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		smr,
		validation.Field(&smr.MoveID, validation.Required.Error("Move ID is required")),
		validation.Field(&smr.OrgID, validation.Required.Error("Organization ID is required")),
		validation.Field(&smr.BuID, validation.Required.Error("Business Unit ID is required")),
		validation.Field(
			&smr.SplitLocationID,
			validation.Required.Error("Split Location ID is required"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

type HandleMoveDeletionsRequest struct {
	ExistingMoveMap map[pulid.ID]*shipment.ShipmentMove
	UpdatedMoveIDs  map[pulid.ID]struct{}
	MoveToDelete    []*shipment.ShipmentMove
}

type SplitMoveResponse struct {
	OriginalMove *shipment.ShipmentMove `json:"originalMove,omitempty"`
	NewMove      *shipment.ShipmentMove `json:"newMove,omitempty"`
}

type ShipmentMoveRepository interface {
	GetByID(ctx context.Context, req GetMoveByIDRequest) (*shipment.ShipmentMove, error)
	UpdateStatus(ctx context.Context, req *UpdateMoveStatusRequest) (*shipment.ShipmentMove, error)
	GetMovesByShipmentID(
		ctx context.Context,
		req GetMovesByShipmentIDRequest,
	) ([]*shipment.ShipmentMove, error)
	BulkUpdateStatus(
		ctx context.Context,
		req BulkUpdateMoveStatusRequest,
	) ([]*shipment.ShipmentMove, error)
	BulkInsert(
		ctx context.Context,
		moves []*shipment.ShipmentMove,
	) ([]*shipment.ShipmentMove, error)
	SplitMove(ctx context.Context, req *SplitMoveRequest) (*SplitMoveResponse, error)
	HandleMoveOperations(
		ctx context.Context,
		tx bun.IDB,
		shp *shipment.Shipment,
		isCreate bool,
	) error
}
