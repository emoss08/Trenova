package repositories

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type ListShipmentCommentsRequest struct {
	Filter     *pagination.QueryOptions `json:"filter"`
	ShipmentID pulid.ID                 `json:"shipmentId"`
}

type GetShipmentCommentCountRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ShipmentID pulid.ID              `json:"shipmentId"`
}

type GetShipmentCommentByIDRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ShipmentID pulid.ID              `json:"shipmentId"`
	CommentID  pulid.ID              `json:"commentId"`
}

type DeleteShipmentCommentRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ShipmentID pulid.ID              `json:"shipmentId"`
	CommentID  pulid.ID              `json:"commentId"`
}

type ShipmentCommentRepository interface {
	ListByShipmentID(
		ctx context.Context,
		req *ListShipmentCommentsRequest,
	) (*pagination.ListResult[*shipment.ShipmentComment], error)
	GetCountByShipmentID(
		ctx context.Context,
		req *GetShipmentCommentCountRequest,
	) (int, error)
	GetByID(
		ctx context.Context,
		req *GetShipmentCommentByIDRequest,
	) (*shipment.ShipmentComment, error)
	Create(
		ctx context.Context,
		entity *shipment.ShipmentComment,
	) (*shipment.ShipmentComment, error)
	Update(
		ctx context.Context,
		entity *shipment.ShipmentComment,
	) (*shipment.ShipmentComment, error)
	Delete(
		ctx context.Context,
		req *DeleteShipmentCommentRequest,
	) error
}

func (r *GetShipmentCommentCountRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(&r.ShipmentID, validation.Required.Error("Shipment ID is required")),
		validation.Field(
			&r.TenantInfo.OrgID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&r.TenantInfo.BuID,
			validation.Required.Error("Business unit ID is required"),
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

func (r *GetShipmentCommentByIDRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(&r.CommentID, validation.Required.Error("Comment ID is required")),
		validation.Field(&r.ShipmentID, validation.Required.Error("Shipment ID is required")),
		validation.Field(
			&r.TenantInfo.OrgID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&r.TenantInfo.BuID,
			validation.Required.Error("Business unit ID is required"),
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

func (r *DeleteShipmentCommentRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(&r.CommentID, validation.Required.Error("Comment ID is required")),
		validation.Field(&r.ShipmentID, validation.Required.Error("Shipment ID is required")),
		validation.Field(
			&r.TenantInfo.OrgID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&r.TenantInfo.BuID,
			validation.Required.Error("Business unit ID is required"),
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
