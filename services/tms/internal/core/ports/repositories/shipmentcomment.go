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

type ShipmentCommentListFilters struct {
	Types          []shipment.CommentType     `json:"types,omitempty"`
	Priorities     []shipment.CommentPriority `json:"priorities,omitempty"`
	AuthorIDs      []pulid.ID                 `json:"authorIds,omitempty"`
	MentionsUserID *pulid.ID                  `json:"mentionsUserId,omitempty"`
	PinnedOnly     bool                       `json:"pinnedOnly,omitempty"`
	UnresolvedOnly bool                       `json:"unresolvedOnly,omitempty"`
	Search         string                     `json:"search,omitempty"`
}

func (f ShipmentCommentListFilters) HasAny() bool {
	return len(f.Types) > 0 || len(f.Priorities) > 0 || len(f.AuthorIDs) > 0 ||
		f.MentionsUserID != nil || f.PinnedOnly || f.UnresolvedOnly || f.Search != ""
}

type ListShipmentCommentsRequest struct {
	Filter          *pagination.QueryOptions   `json:"filter"`
	Cursor          pagination.CursorInfo      `json:"cursor"`
	ShipmentID      pulid.ID                   `json:"shipmentId"`
	ParentCommentID *pulid.ID                  `json:"parentCommentId,omitempty"`
	Filters         ShipmentCommentListFilters `json:"filters"`
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

type SetShipmentCommentPinnedRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ShipmentID pulid.ID              `json:"shipmentId"`
	CommentID  pulid.ID              `json:"commentId"`
	ActorID    pulid.ID              `json:"actorId"`
	Pinned     bool                  `json:"pinned"`
}

type SetShipmentCommentResolvedRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ShipmentID pulid.ID              `json:"shipmentId"`
	CommentID  pulid.ID              `json:"commentId"`
	ActorID    pulid.ID              `json:"actorId"`
	Resolved   bool                  `json:"resolved"`
}

type AcknowledgeShipmentCommentRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ShipmentID pulid.ID              `json:"shipmentId"`
	CommentID  pulid.ID              `json:"commentId"`
	UserID     pulid.ID              `json:"userId"`
}

type SoftDeleteShipmentCommentRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ShipmentID pulid.ID              `json:"shipmentId"`
	CommentID  pulid.ID              `json:"commentId"`
	ActorID    pulid.ID              `json:"actorId"`
}

type ShipmentCommentRepository interface {
	ListByShipmentID(
		ctx context.Context,
		req *ListShipmentCommentsRequest,
	) (*pagination.CursorListResult[*shipment.ShipmentComment], error)
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
	SoftDelete(
		ctx context.Context,
		req *SoftDeleteShipmentCommentRequest,
	) (*shipment.ShipmentComment, error)
	SetPinned(
		ctx context.Context,
		req *SetShipmentCommentPinnedRequest,
	) (*shipment.ShipmentComment, error)
	SetResolved(
		ctx context.Context,
		req *SetShipmentCommentResolvedRequest,
	) (*shipment.ShipmentComment, error)
	Acknowledge(
		ctx context.Context,
		req *AcknowledgeShipmentCommentRequest,
	) (bool, error)
	CountReplies(
		ctx context.Context,
		req *GetShipmentCommentByIDRequest,
	) (int, error)
	CountPinnedByShipmentID(
		ctx context.Context,
		req *GetShipmentCommentCountRequest,
	) (int, error)
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

func (r *SetShipmentCommentPinnedRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(&r.CommentID, validation.Required.Error("Comment ID is required")),
		validation.Field(&r.ShipmentID, validation.Required.Error("Shipment ID is required")),
		validation.Field(&r.ActorID, validation.Required.Error("Actor ID is required")),
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

func (r *SetShipmentCommentResolvedRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(&r.CommentID, validation.Required.Error("Comment ID is required")),
		validation.Field(&r.ShipmentID, validation.Required.Error("Shipment ID is required")),
		validation.Field(&r.ActorID, validation.Required.Error("Actor ID is required")),
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

func (r *AcknowledgeShipmentCommentRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(&r.CommentID, validation.Required.Error("Comment ID is required")),
		validation.Field(&r.ShipmentID, validation.Required.Error("Shipment ID is required")),
		validation.Field(&r.UserID, validation.Required.Error("User ID is required")),
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

func (r *SoftDeleteShipmentCommentRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	err := validation.ValidateStruct(
		r,
		validation.Field(&r.CommentID, validation.Required.Error("Comment ID is required")),
		validation.Field(&r.ShipmentID, validation.Required.Error("Shipment ID is required")),
		validation.Field(&r.ActorID, validation.Required.Error("Actor ID is required")),
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
