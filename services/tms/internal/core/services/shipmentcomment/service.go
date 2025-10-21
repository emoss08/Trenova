package shipmentcomment

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/core/services/moderation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger              *zap.Logger
	Repo                repositories.ShipmentCommentRepository
	AuditService        services.AuditService
	NotificationService services.NotificationService
	UserRepository      repositories.UserRepository
	ModerationService   *moderation.Service
}

type Service struct {
	l    *zap.Logger
	repo repositories.ShipmentCommentRepository
	as   services.AuditService
	ns   services.NotificationService
	ur   repositories.UserRepository
	modS *moderation.Service
}

//nolint:gocritic // This is a constructor
func NewService(p Params) *Service {
	return &Service{
		l:    p.Logger.With(zap.String("service", "shipmentcomment")),
		repo: p.Repo,
		as:   p.AuditService,
		ns:   p.NotificationService,
		ur:   p.UserRepository,
		modS: p.ModerationService,
	}
}

func (s *Service) ListByShipmentID(
	ctx context.Context,
	req repositories.GetCommentsByShipmentIDRequest,
) (*pagination.ListResult[*shipment.ShipmentComment], error) {
	return s.repo.ListByShipmentID(ctx, req)
}

func (s *Service) GetCountByShipmentID(
	ctx context.Context,
	req repositories.GetShipmentCommentCountRequest,
) (int, error) {
	return s.repo.GetCountByShipmentID(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *shipment.ShipmentComment,
) (*shipment.ShipmentComment, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("userID", entity.UserID.String()),
	)

	result, err := s.modS.ModerateText(ctx, entity.Comment)
	if err != nil {
		log.Error("failed to moderate text", zap.Error(err))
		return nil, err
	}

	if result.Flagged {
		return nil, errortypes.NewValidationError(
			"comment",
			errortypes.ErrInvalid,
			result.Reason,
		)
	}

	entity, err = s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipment,
			ResourceID:     entity.GetID(),
			Operation:      permission.OpCreate,
			UserID:         entity.UserID,
			CurrentState:   jsonutils.MustToJSON(entity),
			BusinessUnitID: entity.BusinessUnitID,
			OrganizationID: entity.OrganizationID,
		},
		audit.WithComment("Shipment comment created"),
	)
	if err != nil {
		log.Error("failed to log shipment comment creation", zap.Error(err))
	}

	ownerName, err := s.ur.GetNameByID(ctx, entity.UserID)
	if err != nil {
		log.Error("failed to get owner name", zap.Error(err))
		return nil, err
	}

	if len(entity.MentionedUsers) > 0 {
		s.processMentionNotifications(ctx, entity, ownerName, log)
	}

	return entity, nil
}

func (s *Service) processMentionNotifications(
	ctx context.Context,
	entity *shipment.ShipmentComment,
	ownerName string,
	log *zap.Logger,
) {
	mentionedUserIDs := s.getUniqueMentionedUserIDs(entity.MentionedUsers)

	mentionedUsers, err := s.ur.GetByIDs(ctx, repositories.GetUsersByIDsRequest{
		OrgID:   entity.OrganizationID,
		BuID:    entity.BusinessUnitID,
		UserIDs: mentionedUserIDs,
	})
	if err != nil {
		log.Error("failed to get mentioned users", zap.Error(err))
		return
	}

	notificationRequests := s.buildNotificationRequests(entity, mentionedUsers, ownerName)
	if len(notificationRequests) == 0 {
		return
	}

	if err = s.ns.SendBulkCommentNotifications(ctx, notificationRequests); err != nil {
		log.Error("failed to send shipment comment notifications", zap.Error(err))
	}
}

func (s *Service) getUniqueMentionedUserIDs(
	mentionedUsers []*shipment.ShipmentCommentMention,
) []pulid.ID {
	uniqueUserIDs := make(map[pulid.ID]struct{})
	for _, mu := range mentionedUsers {
		uniqueUserIDs[mu.MentionedUserID] = struct{}{}
	}

	result := make([]pulid.ID, 0, len(uniqueUserIDs))
	for userID := range uniqueUserIDs {
		result = append(result, userID)
	}
	return result
}

func (s *Service) buildNotificationRequests(
	entity *shipment.ShipmentComment,
	mentionedUsers []*tenant.User,
	ownerName string,
) []*services.ShipmentCommentNotificationRequest {
	userMap := make(map[pulid.ID]*tenant.User)
	for _, u := range mentionedUsers {
		userMap[u.ID] = u
	}

	notificationRequests := make([]*services.ShipmentCommentNotificationRequest, 0, len(userMap))
	processedUsers := make(map[pulid.ID]struct{})

	for _, mu := range entity.MentionedUsers {
		if _, processed := processedUsers[mu.MentionedUserID]; processed {
			continue
		}

		if _, exists := userMap[mu.MentionedUserID]; exists {
			notificationRequests = append(
				notificationRequests,
				&services.ShipmentCommentNotificationRequest{
					OrganizationID:  entity.OrganizationID,
					BusinessUnitID:  entity.BusinessUnitID,
					CommentID:       entity.ID,
					OwnerName:       ownerName,
					OwnerID:         entity.UserID,
					MentionedUserID: mu.MentionedUserID,
				},
			)
			processedUsers[mu.MentionedUserID] = struct{}{}
		}
	}

	return notificationRequests
}

func (s *Service) Update(
	ctx context.Context,
	entity *shipment.ShipmentComment,
) (*shipment.ShipmentComment, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("buID", entity.BusinessUnitID.String()),
	)

	if err := s.validateCommentOwnership(ctx, &validateCommentOwnershipRequest{
		CommentID: entity.ID,
		OrgID:     entity.OrganizationID,
		BuID:      entity.BusinessUnitID,
		UserID:    entity.UserID,
	}); err != nil {
		return nil, err
	}

	entity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update shipment comment", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipmentComment,
			ResourceID:     entity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         entity.UserID,
			CurrentState:   jsonutils.MustToJSON(entity),
			BusinessUnitID: entity.BusinessUnitID,
			OrganizationID: entity.OrganizationID,
		},
		audit.WithComment("Shipment comment updated"),
		audit.WithMetadata(map[string]any{
			"shipmentID": entity.ShipmentID,
		}),
	)
	if err != nil {
		log.Error("failed to log shipment comment update", zap.Error(err))
	}

	return entity, nil
}

func (s *Service) Delete(
	ctx context.Context,
	req *repositories.DeleteCommentRequest,
) error {
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.String("orgID", req.OrgID.String()),
		zap.String("buID", req.BuID.String()),
	)

	if err := s.validateCommentOwnership(ctx, &validateCommentOwnershipRequest{
		CommentID: req.CommentID,
		OrgID:     req.OrgID,
		BuID:      req.BuID,
		UserID:    req.UserID,
	}); err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, req); err != nil {
		log.Error("failed to delete shipment comment", zap.Error(err))
		return err
	}

	err := s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipmentComment,
			ResourceID:     req.CommentID.String(),
			Operation:      permission.OpDelete,
			UserID:         req.UserID,
			BusinessUnitID: req.BuID,
			OrganizationID: req.OrgID,
		},
		audit.WithComment("Shipment comment deleted"),
	)
	if err != nil {
		log.Error("failed to log shipment comment deletion", zap.Error(err))
	}

	return nil
}

type validateCommentOwnershipRequest struct {
	CommentID pulid.ID
	OrgID     pulid.ID
	BuID      pulid.ID
	UserID    pulid.ID
}

func (s *Service) validateCommentOwnership(
	ctx context.Context,
	req *validateCommentOwnershipRequest,
) error {
	existing, err := s.repo.GetByID(ctx, repositories.GetCommentByIDRequest{
		CommentID: req.CommentID,
		OrgID:     req.OrgID,
		BuID:      req.BuID,
	})
	if err != nil {
		return err
	}

	if existing.UserID != req.UserID {
		return errortypes.NewValidationError(
			"commentID",
			errortypes.ErrInvalidOperation,
			"You are not the owner of this shipment comment",
		)
	}

	return nil
}
