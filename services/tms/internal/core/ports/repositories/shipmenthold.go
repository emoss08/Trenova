package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
)

type GetShipmentHoldByShipmentIDRequest struct {
	ShipmentID pulid.ID
	OrgID      pulid.ID
	BuID       pulid.ID
	UserID     pulid.ID
}

type GetShipmentHoldByIDRequest struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}
type HoldShipmentRequest struct {
	ShipmentID   pulid.ID `json:"shipmentId"   query:"shipmentId"`
	OrgID        pulid.ID `json:"orgId"        query:"orgId"`
	BuID         pulid.ID `json:"buId"         query:"buId"`
	UserID       pulid.ID `json:"userId"       query:"userId"`
	HoldReasonID pulid.ID `json:"holdReasonId" query:"holdReasonId"`
}

func (hr *HoldShipmentRequest) Validate() *errors.MultiError {
	me := errors.NewMultiError()

	err := validation.ValidateStruct(
		hr,
		validation.Field(&hr.ShipmentID, validation.Required.Error("Shipment ID is required")),
		validation.Field(&hr.OrgID, validation.Required.Error("Organization ID is required")),
		validation.Field(&hr.BuID, validation.Required.Error("Business Unit ID is required")),
		validation.Field(&hr.UserID, validation.Required.Error("User ID is required")),
		validation.Field(&hr.HoldReasonID, validation.Required.Error("Hold Reason ID is required")),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, me)
		}
	}

	if me.HasErrors() {
		return me
	}

	return nil
}

type ShipmentHoldRepository interface {
	GetByShipmentID(
		ctx context.Context,
		req *GetShipmentHoldByShipmentIDRequest,
	) (*ports.ListResult[*shipment.ShipmentHold], error)
	GetByID(
		ctx context.Context,
		req *GetShipmentHoldByIDRequest,
	) (*shipment.ShipmentHold, error)
	Create(ctx context.Context, hold *shipment.ShipmentHold) (*shipment.ShipmentHold, error)
	Update(ctx context.Context, hold *shipment.ShipmentHold) (*shipment.ShipmentHold, error)
}
