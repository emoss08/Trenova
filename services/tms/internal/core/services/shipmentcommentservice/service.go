package shipmentcommentservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.ShipmentCommentRepository
	ShipmentRepo repositories.ShipmentRepository
	UserRepo     repositories.UserRepository
	AuditService services.AuditService
	Realtime     services.RealtimeService
}

type service struct {
	l            *zap.Logger
	repo         repositories.ShipmentCommentRepository
	shipmentRepo repositories.ShipmentRepository
	userRepo     repositories.UserRepository
	auditService services.AuditService
	realtime     services.RealtimeService
}

func New(p Params) services.ShipmentCommentService {
	return &service{
		l:            p.Logger.Named("service.shipment-comment"),
		repo:         p.Repo,
		shipmentRepo: p.ShipmentRepo,
		userRepo:     p.UserRepo,
		auditService: p.AuditService,
		realtime:     p.Realtime,
	}
}

func (s *service) ListByShipmentID(
	ctx context.Context,
	req *repositories.ListShipmentCommentsRequest,
) (*pagination.ListResult[*shipment.ShipmentComment], error) {
	if req == nil || req.Filter == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Comment list request is required",
		)
	}
	if req.ShipmentID.IsNil() {
		return nil, errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrRequired,
			"Shipment ID is required",
		)
	}

	if err := s.ensureShipmentExists(ctx, req.ShipmentID, req.Filter.TenantInfo); err != nil {
		return nil, err
	}

	comments, err := s.repo.ListByShipmentID(ctx, req)
	if err != nil {
		return nil, err
	}

	for _, comment := range comments.Items {
		normalizeCommentView(comment)
	}

	return comments, nil
}

func (s *service) GetCountByShipmentID(
	ctx context.Context,
	req *repositories.GetShipmentCommentCountRequest,
) (int, error) {
	if req == nil {
		return 0, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Comment count request is required",
		)
	}
	if multiErr := req.Validate(); multiErr != nil {
		return 0, multiErr
	}

	if err := s.ensureShipmentExists(ctx, req.ShipmentID, req.TenantInfo); err != nil {
		return 0, err
	}

	return s.repo.GetCountByShipmentID(ctx, req)
}

func (s *service) Create(
	ctx context.Context,
	entity *shipment.ShipmentComment,
	actor *services.RequestActor,
) (*shipment.ShipmentComment, error) {
	if entity == nil {
		return nil, errortypes.NewValidationError(
			"comment",
			errortypes.ErrRequired,
			"Shipment comment is required",
		)
	}

	userID, err := requireCommentUser(actor)
	if err != nil {
		return nil, err
	}

	entity.UserID = userID
	entity.Comment = strings.TrimSpace(entity.Comment)

	if multiErr := s.validateComment(entity); multiErr != nil {
		return nil, multiErr
	}

	if err := s.ensureShipmentExists(ctx, entity.ShipmentID, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}); err != nil {
		return nil, err
	}

	if err := s.populateMentions(ctx, entity); err != nil {
		return nil, err
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	normalizeCommentView(created)
	auditActor := actor.AuditActor()
	s.logCommentAction(
		created,
		auditActor,
		permission.OpCreate,
		nil,
		created,
		"Shipment comment created",
	)
	s.publishCommentInvalidation(ctx, created, auditActor, "created", created)

	return created, nil
}

func (s *service) Update(
	ctx context.Context,
	entity *shipment.ShipmentComment,
	actor *services.RequestActor,
) (*shipment.ShipmentComment, error) {
	if entity == nil {
		return nil, errortypes.NewValidationError(
			"comment",
			errortypes.ErrRequired,
			"Shipment comment is required",
		)
	}

	userID, err := requireCommentUser(actor)
	if err != nil {
		return nil, err
	}

	if entity.ID.IsNil() {
		return nil, errortypes.NewValidationError(
			"commentId",
			errortypes.ErrRequired,
			"Comment ID is required",
		)
	}

	entity.UserID = userID
	entity.Comment = strings.TrimSpace(entity.Comment)

	if multiErr := s.validateComment(entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.getOwnedComment(ctx, &repositories.GetShipmentCommentByIDRequest{
		CommentID:  entity.ID,
		ShipmentID: entity.ShipmentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	}, userID)
	if err != nil {
		return nil, err
	}

	if err := s.ensureShipmentExists(ctx, entity.ShipmentID, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}); err != nil {
		return nil, err
	}

	if err := s.populateMentions(ctx, entity); err != nil {
		return nil, err
	}

	entity.Version = original.Version
	entity.CreatedAt = original.CreatedAt
	if original.Comment != entity.Comment {
		now := timeutils.NowUnix()
		entity.EditedAt = &now
	} else {
		entity.EditedAt = original.EditedAt
	}

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	normalizeCommentView(updated)
	auditActor := actor.AuditActor()
	s.logCommentAction(
		updated,
		auditActor,
		permission.OpUpdate,
		original,
		updated,
		"Shipment comment updated",
	)
	s.publishCommentInvalidation(ctx, updated, auditActor, "updated", updated)

	return updated, nil
}

func (s *service) Delete(
	ctx context.Context,
	req *repositories.DeleteShipmentCommentRequest,
	actor *services.RequestActor,
) error {
	if req == nil {
		return errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Delete comment request is required",
		)
	}
	if multiErr := req.Validate(); multiErr != nil {
		return multiErr
	}

	userID, err := requireCommentUser(actor)
	if err != nil {
		return err
	}

	original, err := s.getOwnedComment(ctx, &repositories.GetShipmentCommentByIDRequest{
		CommentID:  req.CommentID,
		ShipmentID: req.ShipmentID,
		TenantInfo: req.TenantInfo,
	}, userID)
	if err != nil {
		return err
	}

	if err := s.ensureShipmentExists(ctx, req.ShipmentID, req.TenantInfo); err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, req); err != nil {
		return err
	}

	auditActor := actor.AuditActor()
	s.logCommentAction(
		original,
		auditActor,
		permission.OpDelete,
		original,
		nil,
		"Shipment comment deleted",
	)
	s.publishCommentInvalidation(ctx, original, auditActor, "deleted", original)

	return nil
}

func (s *service) validateComment(entity *shipment.ShipmentComment) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *service) ensureShipmentExists(
	ctx context.Context,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	_, err := s.shipmentRepo.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
	})
	return err
}

func (s *service) populateMentions(
	ctx context.Context,
	entity *shipment.ShipmentComment,
) error {
	entity.MentionedUserIDs = uniqueMentionIDs(entity.MentionedUserIDs)
	entity.MentionedUsers = make(
		[]*shipment.ShipmentCommentMention,
		0,
		len(entity.MentionedUserIDs),
	)

	if len(entity.MentionedUserIDs) == 0 {
		return nil
	}

	users, err := s.userRepo.GetByIDs(ctx, repositories.GetUsersByIDsRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		UserIDs: entity.MentionedUserIDs,
	})
	if err != nil {
		return err
	}

	if len(users) != len(entity.MentionedUserIDs) {
		return errortypes.NewValidationError(
			"mentionedUserIds",
			errortypes.ErrInvalid,
			"All mentioned users must belong to the current tenant",
		)
	}

	userByID := make(map[pulid.ID]*tenant.User, len(users))
	for _, user := range users {
		userByID[user.ID] = user
	}

	for _, userID := range entity.MentionedUserIDs {
		entity.MentionedUsers = append(entity.MentionedUsers, &shipment.ShipmentCommentMention{
			CommentID:       entity.ID,
			MentionedUserID: userID,
			ShipmentID:      entity.ShipmentID,
			OrganizationID:  entity.OrganizationID,
			BusinessUnitID:  entity.BusinessUnitID,
			MentionedUser:   userByID[userID],
		})
	}

	return nil
}

func (s *service) getOwnedComment(
	ctx context.Context,
	req *repositories.GetShipmentCommentByIDRequest,
	userID pulid.ID,
) (*shipment.ShipmentComment, error) {
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	comment, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}

	if comment.UserID != userID {
		return nil, errortypes.NewAuthorizationError(
			"You are not the owner of this shipment comment",
		)
	}

	normalizeCommentView(comment)

	return comment, nil
}

func (s *service) logCommentAction(
	entity *shipment.ShipmentComment,
	actor services.AuditActor,
	op permission.Operation,
	previous any,
	current any,
	comment string,
) {
	params := &services.LogActionParams{
		Resource:       permission.ResourceShipmentComment,
		ResourceID:     entity.ID.String(),
		Operation:      op,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
	}
	if current != nil {
		params.CurrentState = jsonutils.MustToJSON(current)
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
	}

	opts := []services.LogOption{
		auditservice.WithComment(comment),
		auditservice.WithMetadata(map[string]any{
			"shipmentId": entity.ShipmentID.String(),
			"commentId":  entity.ID.String(),
		}),
	}
	if previous != nil && current != nil {
		opts = append(opts, auditservice.WithDiff(previous, current))
	}

	if err := s.auditService.LogAction(params, opts...); err != nil {
		s.l.Error("failed to log shipment comment action", zap.Error(err))
	}
}

func (s *service) publishCommentInvalidation(
	ctx context.Context,
	entity *shipment.ShipmentComment,
	actor services.AuditActor,
	action string,
	payload any,
) {
	err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		ActorUserID:    actor.UserID,
		ActorType:      actor.PrincipalType,
		ActorID:        actor.PrincipalID,
		ActorAPIKeyID:  actor.APIKeyID,
		Resource:       permission.ResourceShipmentComment.String(),
		Action:         action,
		RecordID:       entity.ShipmentID,
		Entity:         payload,
	})
	if err != nil {
		s.l.Warn("failed to publish shipment comment invalidation", zap.Error(err))
	}
}

func uniqueMentionIDs(ids []pulid.ID) []pulid.ID {
	if len(ids) == 0 {
		return nil
	}

	unique := make([]pulid.ID, 0, len(ids))
	seen := make(map[pulid.ID]struct{}, len(ids))
	for _, id := range ids {
		if id.IsNil() {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}

	return unique
}

func normalizeCommentView(entity *shipment.ShipmentComment) {
	if entity == nil {
		return
	}

	entity.MentionedUserIDs = entity.MentionedUserIDs[:0]
	for _, mention := range entity.MentionedUsers {
		if mention == nil {
			continue
		}
		entity.MentionedUserIDs = append(entity.MentionedUserIDs, mention.MentionedUserID)
	}
	entity.MentionedUserIDs = uniqueMentionIDs(entity.MentionedUserIDs)
}

func requireCommentUser(actor *services.RequestActor) (pulid.ID, error) {
	if actor == nil || actor.UserID.IsNil() {
		return pulid.Nil, errortypes.NewAuthorizationError(
			"Shipment comment actions require a user actor",
		)
	}

	return actor.UserID, nil
}
