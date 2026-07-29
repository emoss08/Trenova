package shipmentcommentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *service) Pin(
	ctx context.Context,
	req *services.ToggleShipmentCommentRequest,
	actor *services.RequestActor,
) (*shipment.ShipmentComment, error) {
	return s.setPinned(ctx, req, actor, true)
}

func (s *service) Unpin(
	ctx context.Context,
	req *services.ToggleShipmentCommentRequest,
	actor *services.RequestActor,
) (*shipment.ShipmentComment, error) {
	return s.setPinned(ctx, req, actor, false)
}

func (s *service) setPinned(
	ctx context.Context,
	req *services.ToggleShipmentCommentRequest,
	actor *services.RequestActor,
	pinned bool,
) (*shipment.ShipmentComment, error) {
	comment, userID, err := s.getToggleTarget(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	if comment.IsReply() {
		return nil, errortypes.NewValidationError(
			"commentId",
			errortypes.ErrInvalid,
			"Replies cannot be pinned",
		)
	}

	if comment.IsPinned() == pinned {
		return comment, nil
	}

	if pinned {
		pinnedCount, countErr := s.repo.CountPinnedByShipmentID(
			ctx,
			&repositories.GetShipmentCommentCountRequest{
				TenantInfo: req.TenantInfo,
				ShipmentID: req.ShipmentID,
			},
		)
		if countErr != nil {
			return nil, countErr
		}
		if pinnedCount >= shipment.MaxPinnedComments {
			return nil, errortypes.NewConflictError(
				"This shipment already has the maximum number of pinned comments",
			)
		}
	}

	updated, err := s.repo.SetPinned(ctx, &repositories.SetShipmentCommentPinnedRequest{
		TenantInfo: req.TenantInfo,
		ShipmentID: req.ShipmentID,
		CommentID:  req.CommentID,
		ActorID:    userID,
		Pinned:     pinned,
	})
	if err != nil {
		return nil, err
	}

	normalizeCommentView(updated)
	if err = s.loadAttachments(ctx, req.TenantInfo, updated); err != nil {
		return nil, err
	}

	auditActor := actor.AuditActor()
	op := permission.OpPin
	action := actionPinned
	auditComment := "Shipment comment pinned"
	if !pinned {
		op = permission.OpUnpin
		action = actionUnpinned
		auditComment = "Shipment comment unpinned"
	}
	s.logCommentAction(updated, auditActor, op, comment, updated, auditComment)
	s.publishCommentInvalidation(ctx, updated, auditActor, action, updated)

	return updated, nil
}

func (s *service) Resolve(
	ctx context.Context,
	req *services.ToggleShipmentCommentRequest,
	actor *services.RequestActor,
) (*shipment.ShipmentComment, error) {
	return s.setResolved(ctx, req, actor, true)
}

func (s *service) Unresolve(
	ctx context.Context,
	req *services.ToggleShipmentCommentRequest,
	actor *services.RequestActor,
) (*shipment.ShipmentComment, error) {
	return s.setResolved(ctx, req, actor, false)
}

func (s *service) setResolved(
	ctx context.Context,
	req *services.ToggleShipmentCommentRequest,
	actor *services.RequestActor,
	resolved bool,
) (*shipment.ShipmentComment, error) {
	comment, userID, err := s.getToggleTarget(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	if comment.IsResolved() == resolved {
		return comment, nil
	}

	updated, err := s.repo.SetResolved(ctx, &repositories.SetShipmentCommentResolvedRequest{
		TenantInfo: req.TenantInfo,
		ShipmentID: req.ShipmentID,
		CommentID:  req.CommentID,
		ActorID:    userID,
		Resolved:   resolved,
	})
	if err != nil {
		return nil, err
	}

	normalizeCommentView(updated)
	if err = s.loadAttachments(ctx, req.TenantInfo, updated); err != nil {
		return nil, err
	}

	auditActor := actor.AuditActor()
	action := actionResolved
	auditComment := "Shipment comment resolved"
	if !resolved {
		action = actionUnresolved
		auditComment = "Shipment comment reopened"
	}
	s.logCommentAction(updated, auditActor, permission.OpResolve, comment, updated, auditComment)
	s.publishCommentInvalidation(ctx, updated, auditActor, action, updated)

	return updated, nil
}

func (s *service) Acknowledge(
	ctx context.Context,
	req *services.ToggleShipmentCommentRequest,
	actor *services.RequestActor,
) (*shipment.ShipmentComment, error) {
	comment, userID, err := s.getToggleTarget(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	if !comment.RequiresAcknowledgment {
		return nil, errortypes.NewValidationError(
			"commentId",
			errortypes.ErrInvalid,
			"This comment does not request acknowledgment",
		)
	}

	inserted, err := s.repo.Acknowledge(ctx, &repositories.AcknowledgeShipmentCommentRequest{
		TenantInfo: req.TenantInfo,
		ShipmentID: req.ShipmentID,
		CommentID:  req.CommentID,
		UserID:     userID,
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.GetByID(ctx, &repositories.GetShipmentCommentByIDRequest{
		CommentID:  req.CommentID,
		ShipmentID: req.ShipmentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	normalizeCommentView(updated)
	if err = s.loadAttachments(ctx, req.TenantInfo, updated); err != nil {
		return nil, err
	}

	if inserted {
		auditActor := actor.AuditActor()
		s.logCommentAction(
			updated,
			auditActor,
			permission.OpUpdate,
			comment,
			updated,
			"Shipment comment acknowledged",
		)
		s.publishCommentInvalidation(ctx, updated, auditActor, actionAcknowledged, updated)
	}

	return updated, nil
}

func (s *service) getToggleTarget(
	ctx context.Context,
	req *services.ToggleShipmentCommentRequest,
	actor *services.RequestActor,
) (*shipment.ShipmentComment, pulid.ID, error) {
	if req == nil {
		return nil, pulid.Nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Comment request is required",
		)
	}

	getReq := &repositories.GetShipmentCommentByIDRequest{
		CommentID:  req.CommentID,
		ShipmentID: req.ShipmentID,
		TenantInfo: req.TenantInfo,
	}
	if multiErr := getReq.Validate(); multiErr != nil {
		return nil, pulid.Nil, multiErr
	}

	userID, err := requireCommentUser(actor)
	if err != nil {
		return nil, pulid.Nil, err
	}

	comment, err := s.repo.GetByID(ctx, getReq)
	if err != nil {
		return nil, pulid.Nil, err
	}

	if comment.IsDeleted() {
		return nil, pulid.Nil, errortypes.NewValidationError(
			"commentId",
			errortypes.ErrInvalid,
			"Deleted comments cannot be modified",
		)
	}

	normalizeCommentView(comment)
	if err = s.loadAttachments(ctx, req.TenantInfo, comment); err != nil {
		return nil, pulid.Nil, err
	}

	return comment, userID, nil
}
