package shipmentcomment

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/user"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger              *logger.Logger
	Repo                repositories.ShipmentCommentRepository
	PermService         services.PermissionService
	AuditService        services.AuditService
	NotificationService services.NotificationService
	UserRepo            repositories.UserRepository
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.ShipmentCommentRepository
	ps   services.PermissionService
	as   services.AuditService
	ns   services.NotificationService
	ur   repositories.UserRepository
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "shipmentcomment").
		Logger()

	return &Service{
		l:    &log,
		repo: p.Repo,
		ps:   p.PermService,
		as:   p.AuditService,
		ns:   p.NotificationService,
		ur:   p.UserRepo,
	}
}

func (s *Service) ListByShipmentID(
	ctx context.Context,
	req repositories.GetCommentsByShipmentIDRequest,
) (*ports.ListResult[*shipment.ShipmentComment], error) {
	log := s.l.With().
		Str("operation", "ListByShipmentID").
		Interface("req", req).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         req.Filter.TenantOpts.UserID,
			Resource:       permission.ResourceShipmentComment,
			Action:         permission.ActionRead,
			BusinessUnitID: req.Filter.TenantOpts.BuID,
			OrganizationID: req.Filter.TenantOpts.OrgID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read shipment comments",
		)
	}

	return s.repo.ListByShipmentID(ctx, req)
}

func (s *Service) GetCountByShipmentID(
	ctx context.Context,
	req repositories.GetShipmentCommentCountRequest,
	userID pulid.ID,
) (int, error) {
	log := s.l.With().
		Str("operation", "GetCountByShipmentID").
		Interface("req", req).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         userID,
			Resource:       permission.ResourceShipmentComment,
			Action:         permission.ActionRead,
			BusinessUnitID: req.BuID,
			OrganizationID: req.OrgID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return 0, err
	}

	if !result.Allowed {
		return 0, errors.NewAuthorizationError(
			"You do not have permission to read shipment comments",
		)
	}

	return s.repo.GetCountByShipmentID(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	comment *shipment.ShipmentComment,
) (*shipment.ShipmentComment, error) {
	log := s.l.With().
		Str("operation", "Create").
		Str("orgID", comment.OrganizationID.String()).
		Str("buID", comment.BusinessUnitID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         comment.UserID,
			Resource:       permission.ResourceShipmentComment,
			Action:         permission.ActionCreate,
			BusinessUnitID: comment.BusinessUnitID,
			OrganizationID: comment.OrganizationID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to create shipment comments",
		)
	}

	entity, err := s.repo.Create(ctx, comment)
	if err != nil {
		log.Error().Err(err).Msg("failed to create shipment comment")
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipmentComment,
			ResourceID:     entity.GetID(),
			Action:         permission.ActionCreate,
			UserID:         entity.UserID,
			CurrentState:   jsonutils.MustToJSON(entity),
			BusinessUnitID: entity.BusinessUnitID,
			OrganizationID: entity.OrganizationID,
		},
		audit.WithComment("Shipment comment created"),
		audit.WithMetadata(map[string]any{
			"shipmentID": entity.ShipmentID,
		}),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log shipment comment creation")
		// ! we will not return an error here because we want to continue the operation
		// ! even if the notification fails
	}

	ownerName, err := s.ur.GetNameByID(ctx, entity.UserID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get owner name")
		return nil, err
	}

	// Extract mentioned user IDs and deduplicate
	if len(comment.MentionedUsers) > 0 {
		uniqueUserIDs := make(map[pulid.ID]struct{})
		for _, mu := range comment.MentionedUsers {
			uniqueUserIDs[mu.MentionedUserID] = struct{}{}
		}

		mentionedUserIDs := make([]pulid.ID, 0, len(uniqueUserIDs))
		for userID := range uniqueUserIDs {
			mentionedUserIDs = append(mentionedUserIDs, userID)
		}

		log.Debug().
			Int("uniqueMentions", len(mentionedUserIDs)).
			Interface("uniqueUserIDs", mentionedUserIDs).
			Msg("deduplicated mentioned users")

		mentionedUsers, err := s.ur.GetByIDs(ctx, repositories.GetUsersByIDsOptions{
			OrgID:   entity.OrganizationID,
			BuID:    entity.BusinessUnitID,
			UserIDs: mentionedUserIDs,
		})
		if err != nil {
			log.Error().Err(err).Msg("failed to get mentioned users")
			// ! we will not return an error here because we want to continue the operation
			// ! even if the notification fails
		} else {
			userMap := make(map[pulid.ID]*user.User)
			for _, u := range mentionedUsers {
				userMap[u.ID] = u
			}

			notificationRequests := make([]*services.ShipmentCommentNotificationRequest, 0, len(userMap))
			processedUsers := make(map[pulid.ID]struct{})

			for _, mentionUserID := range comment.MentionedUsers {
				// Skip if we've already processed this user
				if _, processed := processedUsers[mentionUserID.MentionedUserID]; processed {
					continue
				}

				if _, exists := userMap[mentionUserID.MentionedUserID]; exists {
					notificationRequests = append(notificationRequests, &services.ShipmentCommentNotificationRequest{
						OrganizationID:  entity.OrganizationID,
						BusinessUnitID:  entity.BusinessUnitID,
						CommentID:       entity.ID,
						OwnerName:       ownerName,
						MentionedUserID: mentionUserID.MentionedUserID,
					})
					processedUsers[mentionUserID.MentionedUserID] = struct{}{}
				}
			}

			if len(notificationRequests) > 0 {
				if err = s.ns.SendBulkCommentNotifications(ctx, notificationRequests); err != nil {
					log.Error().Err(err).Msg("failed to send shipment comment notifications")
					// ! we will not return an error here because we want to continue the operation
					// ! even if the notification fails
				}
			}
		}
	}

	return entity, nil
}

func (s *Service) Update(
	ctx context.Context,
	comment *shipment.ShipmentComment,
	userID pulid.ID,
) (*shipment.ShipmentComment, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("orgID", comment.OrganizationID.String()).
		Str("buID", comment.BusinessUnitID.String()).
		Logger()

	if err := s.validateCommentOwnership(
		ctx,
		&validateCommentOwnershipRequest{
			CommentID: comment.ID,
			OrgID:     comment.OrganizationID,
			BuID:      comment.BusinessUnitID,
			UserID:    userID,
		},
	); err != nil {
		return nil, err
	}

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         userID,
			Resource:       permission.ResourceShipmentComment,
			Action:         permission.ActionUpdate,
			BusinessUnitID: comment.BusinessUnitID,
			OrganizationID: comment.OrganizationID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to update shipment comments",
		)
	}

	entity, err := s.repo.Update(ctx, comment)
	if err != nil {
		log.Error().Err(err).Msg("failed to update shipment comment")
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipmentComment,
			ResourceID:     entity.GetID(),
			Action:         permission.ActionUpdate,
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
		log.Error().Err(err).Msg("failed to log shipment comment update")
	}

	return entity, nil
}

func (s *Service) Delete(
	ctx context.Context,
	req repositories.DeleteCommentRequest,
) error {
	log := s.l.With().
		Str("operation", "Delete").
		Str("orgID", req.OrgID.String()).
		Str("buID", req.BuID.String()).
		Logger()

	if err := s.validateCommentOwnership(
		ctx,
		&validateCommentOwnershipRequest{
			CommentID: req.CommentID,
			OrgID:     req.OrgID,
			BuID:      req.BuID,
			UserID:    req.UserID,
		},
	); err != nil {
		return err
	}

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         req.UserID,
			Resource:       permission.ResourceShipmentComment,
			Action:         permission.ActionDelete,
			BusinessUnitID: req.BuID,
			OrganizationID: req.OrgID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return err
	}

	if !result.Allowed {
		return errors.NewAuthorizationError(
			"You do not have permission to delete shipment comments",
		)
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		log.Error().Err(err).Msg("failed to delete shipment comment")
		return err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipmentComment,
			ResourceID:     req.CommentID.String(),
			Action:         permission.ActionDelete,
			UserID:         req.UserID,
			BusinessUnitID: req.BuID,
			OrganizationID: req.OrgID,
		},
		audit.WithComment("Shipment comment deleted"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log shipment comment deletion")
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
		return errors.NewValidationError(
			"You do not have permission to delete this shipment comment",
			errors.ErrInvalidOperation,
			"You are not the owner of this shipment comment",
		)
	}

	return nil
}
