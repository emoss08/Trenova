package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
)

type GetAssignmentByIDOptions struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type AssignmentRequest struct {
	ShipmentID        pulid.ID  `json:"shipmentId"`
	UserID            pulid.ID  `json:"userId"`
	PrimaryWorkerID   pulid.ID  `json:"primaryWorkerId"`
	TractorID         pulid.ID  `json:"tractorId"`
	TrailerID         pulid.ID  `json:"trailerId"`
	OrgID             pulid.ID  `json:"orgId"`
	BuID              pulid.ID  `json:"buId"`
	SecondaryWorkerID *pulid.ID `json:"secondaryWorkerId,omitempty"`
}

func (a *AssignmentRequest) Validate(ctx context.Context) *errors.MultiError {
	me := errors.NewMultiError()

	err := validation.ValidateStructWithContext(
		ctx,
		a,
		validation.Field(&a.ShipmentID, validation.Required.Error("Shipment ID is required")),
		validation.Field(&a.UserID, validation.Required.Error("User ID is required")),
		validation.Field(
			&a.PrimaryWorkerID,
			validation.Required.Error("Primary Worker ID is required"),
		),
		validation.Field(&a.TractorID, validation.Required.Error("Tractor ID is required")),
		validation.Field(&a.TrailerID, validation.Required.Error("Trailer ID is required")),
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

type ListAssignmentsRequest struct {
	Filter *ports.LimitOffsetQueryOptions
}

type AssignmentRepository interface {
	List(
		ctx context.Context,
		req ListAssignmentsRequest,
	) (*ports.ListResult[*shipment.Assignment], error)
	BulkAssign(ctx context.Context, req *AssignmentRequest) ([]*shipment.Assignment, error)
	SingleAssign(ctx context.Context, a *shipment.Assignment) (*shipment.Assignment, error)
	Reassign(ctx context.Context, a *shipment.Assignment) (*shipment.Assignment, error)
	GetByID(ctx context.Context, opts GetAssignmentByIDOptions) (*shipment.Assignment, error)
}
