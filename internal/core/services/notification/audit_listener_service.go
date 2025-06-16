package notification

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type AuditListenerServiceParams struct {
	fx.In
	fx.Lifecycle

	Logger               *logger.Logger
	NotificationService  services.NotificationService
	AuditRepository      repositories.AuditRepository
	NotificationPrefRepo repositories.NotificationPreferenceRepository
	UserRepository       repositories.UserRepository
}

type AuditListenerService struct {
	l                    *zerolog.Logger
	notificationService  services.NotificationService
	auditRepo            repositories.AuditRepository
	notificationPrefRepo repositories.NotificationPreferenceRepository
	userRepo             repositories.UserRepository

	// Control mechanisms
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	ticker    *time.Ticker
	lastCheck time.Time
	isRunning bool
	mu        sync.RWMutex
}

func NewAuditListenerService(p AuditListenerServiceParams) *AuditListenerService {
	log := p.Logger.With().
		Str("service", "audit_listener").
		Logger()

	ctx, cancel := context.WithCancel(context.Background())

	service := &AuditListenerService{
		l:                    &log,
		notificationService:  p.NotificationService,
		auditRepo:            p.AuditRepository,
		notificationPrefRepo: p.NotificationPrefRepo,
		userRepo:             p.UserRepository,
		ctx:                  ctx,
		cancel:               cancel,
		lastCheck:            time.Now(),
	}

	// Register lifecycle hooks
	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return service.Start()
		},
		OnStop: func(ctx context.Context) error {
			return service.Stop()
		},
	})

	return service
}

// Start begins the audit listener service
func (s *AuditListenerService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("audit listener service is already running")
	}

	s.l.Info().Msg("starting audit listener service")

	// Check every 5 seconds for new audit entries
	s.ticker = time.NewTicker(5 * time.Second)
	s.isRunning = true

	s.wg.Add(1)
	go s.listen()

	return nil
}

// Stop gracefully shuts down the audit listener service
func (s *AuditListenerService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return nil
	}

	s.l.Info().Msg("stopping audit listener service")

	s.cancel()
	if s.ticker != nil {
		s.ticker.Stop()
	}

	// Wait for the listener to finish
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.l.Info().Msg("audit listener service stopped successfully")
	case <-time.After(10 * time.Second):
		s.l.Warn().Msg("audit listener service stop timeout")
	}

	s.isRunning = false
	return nil
}

// listen is the main loop that checks for new audit entries
func (s *AuditListenerService) listen() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.ticker.C:
			if err := s.checkForUpdates(); err != nil {
				s.l.Error().Err(err).Msg("error checking for updates")
			}
		}
	}
}

// checkForUpdates queries for recent audit entries and processes them
func (s *AuditListenerService) checkForUpdates() error {
	// Get audit entries since last check
	entries, err := s.getRecentAuditEntries()
	if err != nil {
		return fmt.Errorf("failed to get recent audit entries: %w", err)
	}

	// Process each audit entry
	for _, entry := range entries {
		if err := s.processAuditEntry(entry); err != nil {
			s.l.Error().
				Err(err).
				Str("audit_id", entry.ID.String()).
				Msg("failed to process audit entry")
			// Continue processing other entries
		}
	}

	// Update last check time
	s.mu.Lock()
	s.lastCheck = time.Now()
	s.mu.Unlock()

	return nil
}

// getRecentAuditEntries retrieves audit entries since the last check
func (s *AuditListenerService) getRecentAuditEntries() ([]*audit.Entry, error) {
	s.mu.RLock()
	lastCheckTimestamp := s.lastCheck.Unix()
	s.mu.RUnlock()

	// Query for UPDATE actions since last check
	entries, err := s.auditRepo.GetRecentEntries(s.ctx, &repositories.GetRecentEntriesRequest{
		SinceTimestamp: lastCheckTimestamp,
		Action:         permission.ActionUpdate,
		Limit:          100, // Process up to 100 entries at a time
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get recent entries: %w", err)
	}

	return entries, nil
}

// processAuditEntry processes a single audit entry and sends notifications if needed
func (s *AuditListenerService) processAuditEntry(entry *audit.Entry) error {
	// Skip if not an update action
	if entry.Action != permission.ActionUpdate {
		return nil
	}

	// Find the original creator of the resource
	creatorID, err := s.findResourceCreator(s.ctx, entry.Resource, entry.ResourceID, entry.OrganizationID)
	if err != nil {
		s.l.Debug().
			Err(err).
			Str("resource", string(entry.Resource)).
			Str("resource_id", entry.ResourceID).
			Msg("could not find resource creator")
		return nil // Don't fail, just skip
	}

	// Skip if the updater is the same as the creator
	if creatorID == entry.UserID {
		return nil
	}

	// Get notification preferences for the creator
	prefs, err := s.notificationPrefRepo.GetUserPreferences(s.ctx, &repositories.GetUserPreferencesRequest{
		UserID:         creatorID,
		OrganizationID: entry.OrganizationID,
		Resource:       entry.Resource,
		IsActive:       true,
	})
	if err != nil {
		return fmt.Errorf("failed to get user preferences: %w", err)
	}

	// Check each preference to see if we should send a notification
	for _, pref := range prefs {
		if s.shouldSendNotification(pref, entry) {
			if err := s.sendUpdateNotification(pref, entry, creatorID); err != nil {
				s.l.Error().Err(err).Msg("failed to send update notification")
				// Continue with other preferences even if one fails
			}
		}
	}

	return nil
}

// findResourceCreator finds who created the resource by looking at audit logs
func (s *AuditListenerService) findResourceCreator(
	ctx context.Context,
	resource permission.Resource,
	resourceID string,
	orgID pulid.ID,
) (pulid.ID, error) {
	// Find the create audit entry
	entries, err := s.auditRepo.GetByResourceAndAction(ctx, &repositories.GetAuditByResourceRequest{
		Resource:       resource,
		ResourceID:     resourceID,
		Action:         permission.ActionCreate,
		OrganizationID: orgID,
		Limit:          1,
	})
	if err != nil {
		return pulid.Nil, fmt.Errorf("failed to get audit entries: %w", err)
	}

	if len(entries) == 0 {
		return pulid.Nil, fmt.Errorf("no create audit entry found for resource")
	}

	return entries[0].UserID, nil
}

// shouldSendNotification checks if a notification should be sent based on preferences
func (s *AuditListenerService) shouldSendNotification(
	pref *notification.NotificationPreference,
	entry *audit.Entry,
) bool {
	// Check if the user wants to be notified about updates
	if !pref.NotifyOnAllUpdates && len(pref.UpdateTypes) == 0 {
		return false
	}

	// Check if the updater should trigger notifications
	if !pref.ShouldNotifyUser(entry.UserID) {
		return false
	}

	// Check if notifications are active
	if !pref.IsActive {
		return false
	}

	// TODO: Add quiet hours check if needed
	// TODO: Check specific update types based on audit entry changes

	return true
}

// sendUpdateNotification sends the actual notification to the user
func (s *AuditListenerService) sendUpdateNotification(
	pref *notification.NotificationPreference,
	entry *audit.Entry,
	ownerID pulid.ID,
) error {
	// Get the updater's information
	updater, err := s.userRepo.GetByID(s.ctx, repositories.GetUserByIDOptions{
		UserID: entry.UserID,
		OrgID:  entry.OrganizationID,
		BuID:   entry.BusinessUnitID,
	})
	if err != nil {
		return fmt.Errorf("failed to get updater information: %w", err)
	}

	// Build the notification
	eventType := s.getEventTypeForResource(entry.Resource)
	title := fmt.Sprintf("%s Updated", s.getResourceDisplayName(entry.Resource))

	// Extract specific changes from audit entry
	changeDetails := s.extractChangeDetails(entry)

	message := fmt.Sprintf(
		"%s updated %s",
		updater.Name,
		s.getResourceDisplayName(entry.Resource),
	)

	if changeDetails != "" {
		message += ": " + changeDetails
	}

	// Create related entity information
	resourceID, _ := pulid.MustParse(entry.ResourceID)
	relatedEntities := []notification.RelatedEntity{
		{
			Type: string(entry.Resource),
			ID:   resourceID,
			Name: s.getResourceDisplayName(entry.Resource),
			URL:  s.getResourceURL(entry.Resource, entry.ResourceID),
		},
	}

	// Send the notification
	notifReq := &services.SendNotificationRequest{
		EventType: eventType,
		Priority:  notification.PriorityMedium,
		Targeting: notification.Targeting{
			Channel:        notification.ChannelUser,
			OrganizationID: entry.OrganizationID,
			BusinessUnitID: &entry.BusinessUnitID,
			TargetUserID:   &ownerID,
		},
		Title:           title,
		Message:         message,
		Data:            entry.Changes,
		RelatedEntities: relatedEntities,
		Actions: []notification.Action{
			{
				ID:       "view",
				Label:    "View Details",
				Type:     "link",
				Style:    "primary",
				Endpoint: s.getResourceURL(entry.Resource, entry.ResourceID),
			},
		},
		Source: "audit_listener_service",
		Tags:   []string{"entity-update", string(entry.Resource), "audit-triggered"},
	}

	return s.notificationService.SendNotification(s.ctx, notifReq)
}

// Helper methods

func (s *AuditListenerService) getEventTypeForResource(resource permission.Resource) notification.EventType {
	// Map resources to their update event types
	eventMap := map[permission.Resource]notification.EventType{
		permission.ResourceShipment: notification.EventShipmentUpdated,
		permission.ResourceWorker:   notification.EventWorkerUpdated,
		permission.ResourceCustomer: notification.EventCustomerUpdated,
		permission.ResourceTractor:  notification.EventTractorUpdated,
		permission.ResourceTrailer:  notification.EventTrailerUpdated,
		permission.ResourceLocation: notification.EventLocationUpdated,
	}

	if eventType, ok := eventMap[resource]; ok {
		return eventType
	}

	return notification.EventEntityUpdated
}

func (s *AuditListenerService) getResourceDisplayName(resource permission.Resource) string {
	// Map resources to display names
	displayMap := map[permission.Resource]string{
		permission.ResourceShipment:              "Shipment",
		permission.ResourceWorker:                "Worker",
		permission.ResourceCustomer:              "Customer",
		permission.ResourceTractor:               "Tractor",
		permission.ResourceTrailer:               "Trailer",
		permission.ResourceLocation:              "Location",
		permission.ResourceCommodity:             "Commodity",
		permission.ResourceFleetCode:             "Fleet Code",
		permission.ResourceEquipmentType:         "Equipment Type",
		permission.ResourceEquipmentManufacturer: "Equipment Manufacturer",
	}

	if name, ok := displayMap[resource]; ok {
		return name
	}

	return string(resource)
}

func (s *AuditListenerService) getResourceURL(resource permission.Resource, resourceID string) string {
	// Map resources to their URL patterns
	urlMap := map[permission.Resource]string{
		permission.ResourceShipment:              "/shipments/%s",
		permission.ResourceWorker:                "/workers/%s",
		permission.ResourceCustomer:              "/customers/%s",
		permission.ResourceTractor:               "/equipment/tractors/%s",
		permission.ResourceTrailer:               "/equipment/trailers/%s",
		permission.ResourceLocation:              "/locations/%s",
		permission.ResourceCommodity:             "/commodities/%s",
		permission.ResourceFleetCode:             "/fleet-codes/%s",
		permission.ResourceEquipmentType:         "/equipment-types/%s",
		permission.ResourceEquipmentManufacturer: "/equipment-manufacturers/%s",
	}

	if pattern, ok := urlMap[resource]; ok {
		return fmt.Sprintf(pattern, resourceID)
	}

	return fmt.Sprintf("/%s/%s", resource, resourceID)
}

func (s *AuditListenerService) extractChangeDetails(entry *audit.Entry) string {
	// Extract meaningful change details from the audit entry
	// This is a simplified version - you might want to make this more sophisticated

	if len(entry.Changes) == 0 {
		return ""
	}

	// Look for specific fields that are commonly updated
	importantFields := []string{"status", "assigned_to", "location", "arrival_time", "departure_time"}

	var changes []string
	for _, field := range importantFields {
		if val, ok := entry.Changes[field]; ok {
			changes = append(changes, fmt.Sprintf("%s changed to %v", field, val))
		}
	}

	if len(changes) > 0 {
		return changes[0] // Return first change for brevity
	}

	return fmt.Sprintf("%d fields updated", len(entry.Changes))
}
