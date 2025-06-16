package notification

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type EntityUpdateServiceParams struct {
	fx.In

	Logger               *logger.Logger
	NotificationService  services.NotificationService
	AuditRepository      repositories.AuditRepository
	NotificationPrefRepo repositories.NotificationPreferenceRepository
	UserRepository       repositories.UserRepository
}

type EntityUpdateService struct {
	l                    *zerolog.Logger
	notificationService  services.NotificationService
	auditRepo            repositories.AuditRepository
	notificationPrefRepo repositories.NotificationPreferenceRepository
	userRepo             repositories.UserRepository
}

func NewEntityUpdateService(p EntityUpdateServiceParams) *EntityUpdateService {
	log := p.Logger.With().
		Str("service", "entity_update_notification").
		Logger()

	return &EntityUpdateService{
		l:                    &log,
		notificationService:  p.NotificationService,
		auditRepo:            p.AuditRepository,
		notificationPrefRepo: p.NotificationPrefRepo,
		userRepo:             p.UserRepository,
	}
}

// HandleEntityUpdate checks if notifications should be sent when an entity is updated
func (s *EntityUpdateService) HandleEntityUpdate(
	ctx context.Context,
	req *EntityUpdateRequest,
) error {
	log := s.l.With().
		Str("entity_type", string(req.EntityType)).
		Str("entity_id", req.EntityID).
		Str("updated_by", req.UpdatedByUserID.String()).
		Logger()

	// Find the original creator of the entity
	creatorID, err := s.findEntityCreator(ctx, req.EntityType, req.EntityID, req.OrganizationID)
	if err != nil {
		log.Warn().Err(err).Msg("failed to find entity creator")
		return nil // Don't fail the update, just skip notification
	}

	// Skip if the updater is the same as the creator
	if creatorID == req.UpdatedByUserID {
		log.Debug().Msg("updater is the same as creator, skipping notification")
		return nil
	}

	// Get notification preferences for the creator
	prefs, err := s.notificationPrefRepo.GetUserPreferences(ctx, &repositories.GetUserPreferencesRequest{
		UserID:         creatorID,
		OrganizationID: req.OrganizationID,
		EntityType:     req.EntityType,
		IsActive:       true,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get user notification preferences")
		return nil // Don't fail the update
	}

	// Check each preference to see if we should send a notification
	for _, pref := range prefs {
		if s.shouldSendNotification(pref, req) {
			if err := s.sendUpdateNotification(ctx, pref, req, creatorID); err != nil {
				log.Error().Err(err).Msg("failed to send update notification")
				// Continue with other preferences even if one fails
			}
		}
	}

	return nil
}

// shouldSendNotification checks if a notification should be sent based on preferences
func (s *EntityUpdateService) shouldSendNotification(
	pref *notification.NotificationPreference,
	req *EntityUpdateRequest,
) bool {
	// Check if the user wants to be notified about this type of update
	if !pref.IsUpdateTypeEnabled(req.UpdateType) {
		return false
	}

	// Check if the updater should trigger notifications
	if !pref.ShouldNotifyUser(req.UpdatedByUserID) {
		return false
	}

	// Check if notifications are active
	if !pref.IsActive {
		return false
	}

	// TODO: Add quiet hours check if needed

	return true
}

// sendUpdateNotification sends the actual notification to the user
func (s *EntityUpdateService) sendUpdateNotification(
	ctx context.Context,
	pref *notification.NotificationPreference,
	req *EntityUpdateRequest,
	ownerID pulid.ID,
) error {
	// Get the updater's information
	updater, err := s.userRepo.GetByID(ctx, repositories.GetUserByIDOptions{
		UserID: req.UpdatedByUserID,
		OrgID:  req.OrganizationID,
		BuID:   req.BusinessUnitID,
	})
	if err != nil {
		return fmt.Errorf("failed to get updater information: %w", err)
	}

	// Build the notification
	eventType := s.getEventTypeForEntity(req.EntityType)
	title := fmt.Sprintf("%s Updated", req.EntityName)
	message := fmt.Sprintf(
		"%s updated the %s of %s #%s",
		updater.Name,
		s.getUpdateTypeDescription(req.UpdateType),
		req.EntityType,
		req.EntityCode,
	)

	// Add specific details based on update type
	if req.Details != nil {
		if oldValue, ok := req.Details["old_value"]; ok {
			if newValue, ok := req.Details["new_value"]; ok {
				message += fmt.Sprintf(" from '%v' to '%v'", oldValue, newValue)
			}
		}
	}

	// Create related entity information
	relatedEntities := []notification.RelatedEntity{
		{
			Type: string(req.EntityType),
			ID:   pulid.MustParse(req.EntityID),
			Name: req.EntityName,
			URL:  req.EntityURL,
		},
	}

	// Send the notification
	notifReq := &services.SendNotificationRequest{
		EventType: eventType,
		Priority:  notification.PriorityMedium,
		Targeting: notification.Targeting{
			Channel:        notification.ChannelUser,
			OrganizationID: req.OrganizationID,
			BusinessUnitID: &req.BusinessUnitID,
			TargetUserID:   &ownerID,
		},
		Title:           title,
		Message:         message,
		Data:            req.Details,
		RelatedEntities: relatedEntities,
		Actions: []notification.Action{
			{
				ID:       "view",
				Label:    "View Details",
				Type:     "link",
				Style:    "primary",
				Endpoint: req.EntityURL,
			},
		},
		Source: "entity_update_service",
		Tags:   []string{"entity-update", string(req.EntityType), string(req.UpdateType)},
	}

	return s.notificationService.SendNotification(ctx, notifReq)
}

// findEntityCreator finds who created the entity by looking at audit logs
func (s *EntityUpdateService) findEntityCreator(
	ctx context.Context,
	entityType notification.EntityType,
	entityID string,
	orgID pulid.ID,
) (pulid.ID, error) {
	// Map entity type to permission resource
	resource := s.mapEntityTypeToResource(entityType)

	// Find the create audit entry
	entries, err := s.auditRepo.GetByResourceAndAction(ctx, &repositories.GetAuditByResourceRequest{
		Resource:       resource,
		ResourceID:     entityID,
		Action:         permission.ActionCreate,
		OrganizationID: orgID,
		Limit:          1,
	})
	if err != nil {
		return pulid.Nil, fmt.Errorf("failed to get audit entries: %w", err)
	}

	if len(entries) == 0 {
		return pulid.Nil, fmt.Errorf("no create audit entry found for entity")
	}

	return entries[0].UserID, nil
}

// Helper methods

func (s *EntityUpdateService) mapEntityTypeToResource(entityType notification.EntityType) permission.Resource {
	switch entityType {
	case notification.EntityTypeShipment:
		return permission.ResourceShipment
	case notification.EntityTypeWorker:
		return permission.ResourceWorker
	case notification.EntityTypeCustomer:
		return permission.ResourceCustomer
	case notification.EntityTypeTractor:
		return permission.ResourceTractor
	case notification.EntityTypeTrailer:
		return permission.ResourceTrailer
	case notification.EntityTypeLocation:
		return permission.ResourceLocation
	default:
		return permission.Resource(entityType)
	}
}

func (s *EntityUpdateService) getEventTypeForEntity(entityType notification.EntityType) notification.EventType {
	switch entityType {
	case notification.EntityTypeShipment:
		return notification.EventShipmentUpdated
	case notification.EntityTypeWorker:
		return notification.EventWorkerUpdated
	case notification.EntityTypeCustomer:
		return notification.EventCustomerUpdated
	case notification.EntityTypeTractor:
		return notification.EventTractorUpdated
	case notification.EntityTypeTrailer:
		return notification.EventTrailerUpdated
	case notification.EntityTypeLocation:
		return notification.EventLocationUpdated
	default:
		return notification.EventEntityUpdated
	}
}

func (s *EntityUpdateService) getUpdateTypeDescription(updateType notification.UpdateType) string {
	switch updateType {
	case notification.UpdateTypeStatusChange:
		return "status"
	case notification.UpdateTypeAssignment:
		return "assignment"
	case notification.UpdateTypeDateChange:
		return "dates"
	case notification.UpdateTypeLocationChange:
		return "location"
	case notification.UpdateTypeDocumentUpload:
		return "documents"
	case notification.UpdateTypePriceChange:
		return "pricing"
	case notification.UpdateTypeComplianceChange:
		return "compliance"
	default:
		return "details"
	}
}

// EntityUpdateRequest contains information about an entity update
type EntityUpdateRequest struct {
	EntityType      notification.EntityType
	EntityID        string
	EntityCode      string // Pro number, code, etc.
	EntityName      string
	EntityURL       string
	UpdateType      notification.UpdateType
	UpdatedByUserID pulid.ID
	OrganizationID  pulid.ID
	BusinessUnitID  pulid.ID
	Details         map[string]any
}
