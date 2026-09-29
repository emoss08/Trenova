package shipmentcommentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *service) GetByID(
	ctx context.Context,
	req *repositories.GetShipmentCommentByIDRequest,
) (*shipment.ShipmentComment, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Comment request is required",
		)
	}
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	comment, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}

	normalizeCommentView(comment)
	if err = s.loadAttachments(ctx, req.TenantInfo, comment); err != nil {
		return nil, err
	}

	return comment, nil
}

func (s *service) PreviewPin(
	ctx context.Context,
	req *services.ToggleShipmentCommentRequest,
	actor *services.RequestActor,
	pinned bool,
) (*services.ShipmentCommentChange, error) {
	comment, userID, err := s.planPin(ctx, req, actor, pinned)
	if err != nil {
		return nil, err
	}

	after := *comment
	if comment.IsPinned() != pinned {
		if pinned {
			now := timeutils.NowUnix()
			after.PinnedAt = &now
			after.PinnedByID = pulid.PtrOrNil(userID)
		} else {
			after.PinnedAt = nil
			after.PinnedByID = nil
		}
	}

	return &services.ShipmentCommentChange{Before: comment, After: &after}, nil
}

func (s *service) planPin(
	ctx context.Context,
	req *services.ToggleShipmentCommentRequest,
	actor *services.RequestActor,
	pinned bool,
) (*shipment.ShipmentComment, pulid.ID, error) {
	comment, userID, err := s.getToggleTarget(ctx, req, actor)
	if err != nil {
		return nil, pulid.Nil, err
	}

	if comment.IsReply() {
		return nil, pulid.Nil, errortypes.NewValidationError(
			"commentId",
			errortypes.ErrInvalid,
			"Replies cannot be pinned",
		)
	}

	if pinned && !comment.IsPinned() {
		pinnedCount, countErr := s.repo.CountPinnedByShipmentID(
			ctx,
			&repositories.GetShipmentCommentCountRequest{
				TenantInfo: req.TenantInfo,
				ShipmentID: req.ShipmentID,
			},
		)
		if countErr != nil {
			return nil, pulid.Nil, countErr
		}
		if pinnedCount >= shipment.MaxPinnedComments {
			return nil, pulid.Nil, errortypes.NewConflictError(
				"This shipment already has the maximum number of pinned comments",
			)
		}
	}

	return comment, userID, nil
}

func (s *service) PreviewResolve(
	ctx context.Context,
	req *services.ToggleShipmentCommentRequest,
	actor *services.RequestActor,
	resolved bool,
) (*services.ShipmentCommentChange, error) {
	comment, userID, err := s.getToggleTarget(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	after := *comment
	if comment.IsResolved() != resolved {
		if resolved {
			now := timeutils.NowUnix()
			after.ResolvedAt = &now
			after.ResolvedByID = pulid.PtrOrNil(userID)
		} else {
			after.ResolvedAt = nil
			after.ResolvedByID = nil
		}
	}

	return &services.ShipmentCommentChange{Before: comment, After: &after}, nil
}

func (s *service) PreviewUpdate(
	ctx context.Context,
	req *services.UpdateShipmentCommentRequest,
	actor *services.RequestActor,
) (*services.ShipmentCommentChange, error) {
	original, entity, err := s.planUpdate(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	return &services.ShipmentCommentChange{Before: original, After: entity}, nil
}

func (s *service) planUpdate(
	ctx context.Context,
	req *services.UpdateShipmentCommentRequest,
	actor *services.RequestActor,
) (original, entity *shipment.ShipmentComment, err error) {
	if req == nil || req.Entity == nil {
		return nil, nil, errortypes.NewValidationError(
			"comment",
			errortypes.ErrRequired,
			"Shipment comment is required",
		)
	}
	entity = req.Entity

	userID, err := requireCommentUser(actor)
	if err != nil {
		return nil, nil, err
	}

	if entity.ID.IsNil() {
		return nil, nil, errortypes.NewValidationError(
			"commentId",
			errortypes.ErrRequired,
			"Comment ID is required",
		)
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}

	original, err = s.getEditableComment(ctx, &repositories.GetShipmentCommentByIDRequest{
		CommentID:  entity.ID,
		ShipmentID: entity.ShipmentID,
		TenantInfo: tenantInfo,
	}, userID, req.AsModerator)
	if err != nil {
		return nil, nil, err
	}

	if original.IsDeleted() {
		return nil, nil, errortypes.NewValidationError(
			"commentId",
			errortypes.ErrInvalid,
			"Deleted comments cannot be edited",
		)
	}

	if multiErr := s.prepareCommentUpdate(entity, original); multiErr != nil {
		return nil, nil, multiErr
	}

	return original, entity, nil
}

func (s *service) PreviewDelete(
	ctx context.Context,
	req *services.DeleteShipmentCommentRequest,
	actor *services.RequestActor,
) (*services.ShipmentCommentDeletion, error) {
	original, replies, err := s.planDelete(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	return &services.ShipmentCommentDeletion{Comment: original, Tombstoned: replies > 0}, nil
}

func (s *service) planDelete(
	ctx context.Context,
	req *services.DeleteShipmentCommentRequest,
	actor *services.RequestActor,
) (original *shipment.ShipmentComment, replies int, err error) {
	if req == nil {
		return nil, 0, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Delete comment request is required",
		)
	}

	repoReq := &repositories.DeleteShipmentCommentRequest{
		TenantInfo: req.TenantInfo,
		ShipmentID: req.ShipmentID,
		CommentID:  req.CommentID,
	}
	if multiErr := repoReq.Validate(); multiErr != nil {
		return nil, 0, multiErr
	}

	userID, err := requireCommentUser(actor)
	if err != nil {
		return nil, 0, err
	}

	original, err = s.getEditableComment(ctx, &repositories.GetShipmentCommentByIDRequest{
		CommentID:  req.CommentID,
		ShipmentID: req.ShipmentID,
		TenantInfo: req.TenantInfo,
	}, userID, req.AsModerator)
	if err != nil {
		return nil, 0, err
	}

	if original.IsDeleted() {
		return nil, 0, errortypes.NewValidationError(
			"commentId",
			errortypes.ErrInvalid,
			"This comment has already been deleted",
		)
	}

	if err = s.ensureShipmentExists(ctx, req.ShipmentID, req.TenantInfo); err != nil {
		return nil, 0, err
	}

	replies, err = s.repo.CountReplies(ctx, &repositories.GetShipmentCommentByIDRequest{
		CommentID:  req.CommentID,
		ShipmentID: req.ShipmentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, 0, err
	}

	return original, replies, nil
}
