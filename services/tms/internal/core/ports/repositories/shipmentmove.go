/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

type GetMoveByIDOptions struct {
	// ID of the move
	MoveID pulid.ID

	// ID of the organization
	OrgID pulid.ID

	// ID of the business unit
	BuID pulid.ID

	// Expand move details (Optional)
	ExpandMoveDetails bool `query:"expandMoveDetails"`
}

type UpdateMoveStatusRequest struct {
	// Fetch the move
	GetMoveOpts GetMoveByIDOptions

	// Status of the move
	Status shipment.MoveStatus
}

type BulkUpdateMoveStatusRequest struct {
	// IDs of the moves
	MoveIDs []pulid.ID

	// Status of the move
	Status shipment.MoveStatus
}

type GetMovesByShipmentIDOptions struct {
	// ID of the shipment
	ShipmentID pulid.ID

	// ID of the organization
	OrgID pulid.ID

	// ID of the business unit
	BuID pulid.ID
}

type SplitQuantity struct {
	// Pieces to split
	Pieces *int `json:"pieces"`

	// Weight to split
	Weight *int `json:"weight"`
}

type SplitStopTimes struct {
	// Planned arrival time for the split stops
	PlannedArrival int64 `json:"plannedArrival"`

	// Planned departure time for the split stops
	PlannedDeparture int64 `json:"plannedDeparture"`
}

type SplitMoveRequest struct {
	// ID of the move
	MoveID pulid.ID `json:"moveId"`

	// ID of the organization
	OrgID pulid.ID `json:"organizationId"`

	// ID of the business unit
	BuID pulid.ID `json:"businessUnitId"`

	// Location where the split will occur
	SplitLocationID pulid.ID `json:"splitLocationId"`

	// Quantities to split
	SplitQuantities SplitQuantity `json:"splitQuantities"`

	// The sequence number after which to perform the split
	SplitAfterStopSequence int `json:"splitAfterStopSequence"`

	// Times for the split delivery stop
	SplitDeliveryTimes SplitStopTimes `json:"splitDeliveryTimes"`

	// Times for the split pickup stop
	SplitPickupTimes SplitStopTimes `json:"splitPickupTimes"`
}

func (smr *SplitMoveRequest) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(
		ctx,
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
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

type HandleMoveDeletionsRequest struct {
	ExistingMoveMap map[pulid.ID]*shipment.ShipmentMove
	UpdatedMoveIDs  map[pulid.ID]struct{}
	MoveToDelete    []*shipment.ShipmentMove
}

type SplitMoveResponse struct {
	// The original move after splitting
	OriginalMove *shipment.ShipmentMove `json:"originalMove,omitempty"`

	// The newly created move
	NewMove *shipment.ShipmentMove `json:"newMove,omitempty"`
}

type ShipmentMoveRepository interface {
	GetByID(ctx context.Context, opts GetMoveByIDOptions) (*shipment.ShipmentMove, error)
	UpdateStatus(ctx context.Context, opts *UpdateMoveStatusRequest) (*shipment.ShipmentMove, error)
	GetMovesByShipmentID(
		ctx context.Context,
		opts GetMovesByShipmentIDOptions,
	) ([]*shipment.ShipmentMove, error)
	BulkUpdateStatus(
		ctx context.Context,
		opts BulkUpdateMoveStatusRequest,
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
