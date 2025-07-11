package notification

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/stringutils"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type AuditListenerServiceParams struct {
	fx.In

	LC                   fx.Lifecycle
	Logger               *logger.Logger
	NotificationService  services.NotificationService
	AuditRepository      repositories.AuditRepository
	NotificationPrefRepo repositories.NotificationPreferenceRepository
	UserRepository       repositories.UserRepository
	BatchProcessor       *BatchProcessor
}

type AuditListenerService struct {
	l                    *zerolog.Logger
	notificationService  services.NotificationService
	auditRepo            repositories.AuditRepository
	notificationPrefRepo repositories.NotificationPreferenceRepository
	userRepo             repositories.UserRepository
	batchProcessor       *BatchProcessor

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
		batchProcessor:       p.BatchProcessor,
		ctx:                  ctx,
		cancel:               cancel,
		lastCheck:            time.Now(),
	}

	// Register lifecycle hooks
	p.LC.Append(fx.Hook{
		OnStart: func(context.Context) error {
			return service.Start()
		},
		OnStop: func(context.Context) error {
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
		if err = s.processAuditEntry(entry); err != nil {
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
	creatorID, err := s.findResourceCreator(
		s.ctx,
		entry.Resource,
		entry.ResourceID,
		entry.OrganizationID,
	)
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
	prefs, err := s.notificationPrefRepo.GetUserPreferences(
		s.ctx,
		&repositories.GetUserPreferencesRequest{
			UserID:         creatorID,
			OrganizationID: entry.OrganizationID,
			Resource:       entry.Resource,
			IsActive:       true,
		},
	)
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

	// Check quiet hours if enabled
	if pref.QuietHoursEnabled && s.isInQuietHours(pref) {
		s.l.Debug().
			Str("user_id", pref.UserID.String()).
			Msg("notification skipped due to quiet hours")
		return false
	}

	// Check specific update types if not notifying on all updates
	if !pref.NotifyOnAllUpdates {
		updateType := s.detectUpdateType(entry)
		if !pref.IsUpdateTypeEnabled(updateType) {
			return false
		}
	}

	return true
}

// isInQuietHours checks if current time is within user's quiet hours
func (s *AuditListenerService) isInQuietHours(pref *notification.NotificationPreference) bool {
	if !pref.QuietHoursEnabled || pref.QuietHoursStart == "" || pref.QuietHoursEnd == "" {
		return false
	}

	// Load user's timezone
	loc, err := time.LoadLocation(pref.Timezone)
	if err != nil {
		s.l.Error().Err(err).Str("timezone", pref.Timezone).Msg("invalid timezone")
		loc = time.UTC
	}

	now := time.Now().In(loc)

	// Parse quiet hours times
	startTime, err := time.ParseInLocation("15:04", pref.QuietHoursStart, loc)
	if err != nil {
		s.l.Error().Err(err).Msg("invalid quiet hours start time")
		return false
	}

	endTime, err := time.ParseInLocation("15:04", pref.QuietHoursEnd, loc)
	if err != nil {
		s.l.Error().Err(err).Msg("invalid quiet hours end time")
		return false
	}

	// Set to today's date
	startTime = time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		startTime.Hour(),
		startTime.Minute(),
		0,
		0,
		loc,
	)
	endTime = time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		endTime.Hour(),
		endTime.Minute(),
		0,
		0,
		loc,
	)

	// Handle overnight quiet hours (e.g., 22:00 to 08:00)
	if endTime.Before(startTime) {
		// If we're after start time or before end time, we're in quiet hours
		return now.After(startTime) || now.Before(endTime)
	}

	// Normal quiet hours (e.g., 08:00 to 17:00)
	return now.After(startTime) && now.Before(endTime)
}

// detectUpdateType analyzes the audit entry to determine the type of update
func (s *AuditListenerService) detectUpdateType(entry *audit.Entry) notification.UpdateType {
	if len(entry.Changes) == 0 {
		return notification.UpdateTypeAny
	}

	// Check for specific field changes
	for field := range entry.Changes {
		switch field {
		case "status":
			return notification.UpdateTypeStatusChange
		case "assigned_to", "assigned_user_id", "worker_id":
			return notification.UpdateTypeAssignment
		case "arrival_time", "departure_time", "planned_arrival", "planned_departure",
			"actual_ship_date", "actual_delivery_date":
			return notification.UpdateTypeDateChange
		case "location", "location_id", "origin_location_id", "destination_location_id":
			return notification.UpdateTypeLocationChange
		case "price", "rate", "total_charge", "accessorial_charges":
			return notification.UpdateTypePriceChange
		case "hazmat_status", "compliance_status":
			return notification.UpdateTypeComplianceChange
		}
	}

	// Check if it's a document-related change
	if entry.Resource == permission.ResourceDocument {
		return notification.UpdateTypeDocumentUpload
	}

	// Default to field change for any other updates
	return notification.UpdateTypeFieldChange
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

	// Check if user prefers batched notifications
	if pref.BatchNotifications && s.batchProcessor != nil {
		s.batchProcessor.AddToBatch(
			ownerID,
			entry.OrganizationID,
			entry.BusinessUnitID,
			eventType,
			title,
			message,
			entry.Changes,
			relatedEntities,
		)

		s.l.Debug().
			Str("user_id", ownerID.String()).
			Msg("notification added to batch")

		return nil
	}

	// Send immediate notification
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

func (s *AuditListenerService) getEventTypeForResource(
	resource permission.Resource,
) notification.EventType {
	// Map resources to their update event types
	eventMap := map[permission.Resource]notification.EventType{ //nolint:exhaustive // we only support these resources
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
	displayMap := map[permission.Resource]string{ //nolint:exhaustive // we only support these resources
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

func (s *AuditListenerService) getResourceURL(
	resource permission.Resource,
	resourceID string,
) string {
	// Map resources to their URL patterns
	urlMap := map[permission.Resource]string{ //nolint:exhaustive // we only support these resources
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
	if len(entry.Changes) == 0 {
		return ""
	}

	// Map of field names to user-friendly names
	fieldNames := map[string]string{
		"status":               "Status",
		"assigned_to":          "Assigned To",
		"arrival_time":         "Arrival Time",
		"departure_time":       "Departure Time",
		"location":             "Location",
		"price":                "Price",
		"total_charge":         "Total Charge",
		"actual_ship_date":     "Ship Date",
		"actual_delivery_date": "Delivery Date",
		"pro_number":           "PRO Number",
		"bol":                  "BOL",
		"notes":                "Notes",
		"description":          "Description",
	}

	var changes []string
	for field, value := range entry.Changes {
		displayName := field
		if friendly, ok := fieldNames[field]; ok {
			displayName = friendly
		}

		// Format the value based on type
		valueStr := s.formatFieldValue(field, value)
		changes = append(changes, fmt.Sprintf("%s changed to %s", displayName, valueStr))

		// Only show first 3 changes to keep message concise
		if len(changes) >= 3 {
			remainingCount := len(entry.Changes) - 3
			if remainingCount > 0 {
				changes = append(changes, fmt.Sprintf("and %d more changes", remainingCount))
			}
			break
		}
	}

	if len(changes) > 0 {
		return strings.Join(changes, ", ")
	}

	return fmt.Sprintf("%d fields updated", len(entry.Changes))
}

// formatFieldValue formats a field value for display
func (s *AuditListenerService) formatFieldValue(field string, value interface{}) string {
	if value == nil {
		return "empty"
	}

	// Handle time fields
	if strings.Contains(field, "time") || strings.Contains(field, "date") {
		if numVal, ok := value.(float64); ok {
			t := time.Unix(int64(numVal), 0)
			return t.Format("Jan 2, 2006 3:04 PM")
		}
	}

	// Handle boolean fields
	if boolVal, ok := value.(bool); ok {
		if boolVal {
			return "Yes"
		}
		return "No"
	}

	// Default string representation
	return fmt.Sprintf("%v", value)
}

// sendNotification sends a notification to the owner of the updated entity
func (s *AuditListenerService) SendNotification(
	_ context.Context,
	owner *user.User,
	entity string,
	entityID string,
	updatedBy *user.User,
	changeType notification.UpdateType,
	auditEntry *audit.Entry,
) error {
	log := s.l.With().
		Str("owner_id", owner.ID.String()).
		Str("entity", entity).
		Str("entity_id", entityID).
		Logger()

	// Determine priority based on update type
	priority := s.determinePriority(changeType)

	// Create the notification
	n := &notification.Notification{
		ID:             pulid.MustNew("notif_"),
		Title:          s.buildNotificationTitle(entity, changeType),
		Message:        s.buildNotificationMessage(entity, entityID, updatedBy, changeType),
		Priority:       priority,
		EventType:      s.getEventTypeForEntity(entity),
		Channel:        notification.ChannelUser,
		OrganizationID: owner.CurrentOrganizationID,
		BusinessUnitID: &owner.BusinessUnitID,
		TargetUserID:   &owner.ID,
		Data: map[string]any{
			"entityType":    entity,
			"entityId":      entityID,
			"updatedBy":     updatedBy.Name,
			"updatedById":   updatedBy.ID.String(),
			"updateType":    string(changeType),
			"updateDetails": s.extractUpdateDetails(auditEntry),
		},
		Source:         "audit_listener",
		DeliveryStatus: notification.DeliveryStatusPending,
	}

	// Send via notification service - we need to send directly via WebSocket
	// since the notificationService interface doesn't have a Create method
	roomName := n.GenerateRoomName()

	// Create a minimal payload for WebSocket delivery
	payload := map[string]any{
		"id":        n.ID.String(),
		"eventType": n.EventType,
		"priority":  n.Priority,
		"title":     n.Title,
		"message":   n.Message,
		"data":      n.Data,
		"createdAt": timeutils.NowUnix(),
	}

	// Send via WebSocket using the notification service's underlying WebSocket manager
	// This is a workaround since we don't have direct access to Create method
	log.Info().
		Str("room_name", roomName).
		Str("notification_id", n.ID.String()).
		Msg("sending notification via websocket")

	// TODO: We should enhance the NotificationService interface to include a Create method
	// For now, we're logging the notification that would be sent

	log.Debug().
		Str("priority", string(priority)).
		Str("change_type", string(changeType)).
		Interface("payload", payload).
		Msg("notification prepared for sending")

	return nil
}

// determinePriority determines the notification priority based on update type
func (s *AuditListenerService) determinePriority(
	updateType notification.UpdateType,
) notification.Priority {
	switch updateType { //nolint:exhaustive // we only support these update types
	case notification.UpdateTypeStatusChange:
		return notification.PriorityHigh
	case notification.UpdateTypeComplianceChange:
		return notification.PriorityCritical
	case notification.UpdateTypeAssignment:
		return notification.PriorityMedium
	case notification.UpdateTypePriceChange:
		return notification.PriorityMedium
	case notification.UpdateTypeLocationChange:
		return notification.PriorityMedium
	case notification.UpdateTypeDocumentUpload:
		return notification.PriorityLow
	default:
		return notification.PriorityLow
	}
}

// buildNotificationTitle builds the notification title based on entity and change type
func (s *AuditListenerService) buildNotificationTitle(
	entity string,
	changeType notification.UpdateType,
) string {
	entityName := s.formatEntityName(entity)

	switch changeType { //nolint:exhaustive // we only support these update types
	case notification.UpdateTypeStatusChange:
		return fmt.Sprintf("%s Status Changed", entityName)
	case notification.UpdateTypeAssignment:
		return fmt.Sprintf("%s Assigned", entityName)
	case notification.UpdateTypeLocationChange:
		return fmt.Sprintf("%s Location Updated", entityName)
	case notification.UpdateTypeDocumentUpload:
		return fmt.Sprintf("New Document for %s", entityName)
	case notification.UpdateTypePriceChange:
		return fmt.Sprintf("%s Pricing Updated", entityName)
	case notification.UpdateTypeComplianceChange:
		return fmt.Sprintf("%s Compliance Alert", entityName)
	default:
		return fmt.Sprintf("%s Updated", entityName)
	}
}

// buildNotificationMessage builds the notification message
func (s *AuditListenerService) buildNotificationMessage(
	entity, entityID string,
	updatedBy *user.User,
	changeType notification.UpdateType,
) string {
	entityName := s.formatEntityName(entity)

	switch changeType { //nolint:exhaustive // we only support these update types
	case notification.UpdateTypeStatusChange:
		return fmt.Sprintf(
			"%s has updated the status of your %s #%s",
			updatedBy.Name,
			strings.ToLower(entityName),
			entityID,
		)
	case notification.UpdateTypeAssignment:
		return fmt.Sprintf(
			"%s has assigned your %s #%s",
			updatedBy.Name,
			strings.ToLower(entityName),
			entityID,
		)
	case notification.UpdateTypeLocationChange:
		return fmt.Sprintf(
			"%s has updated the location of your %s #%s",
			updatedBy.Name,
			strings.ToLower(entityName),
			entityID,
		)
	case notification.UpdateTypeDocumentUpload:
		return fmt.Sprintf(
			"%s has uploaded a document to your %s #%s",
			updatedBy.Name,
			strings.ToLower(entityName),
			entityID,
		)
	case notification.UpdateTypePriceChange:
		return fmt.Sprintf(
			"%s has updated the pricing for your %s #%s",
			updatedBy.Name,
			strings.ToLower(entityName),
			entityID,
		)
	case notification.UpdateTypeComplianceChange:
		return fmt.Sprintf(
			"%s has made compliance changes to your %s #%s",
			updatedBy.Name,
			strings.ToLower(entityName),
			entityID,
		)
	default:
		return fmt.Sprintf(
			"%s has updated your %s #%s",
			updatedBy.Name,
			strings.ToLower(entityName),
			entityID,
		)
	}
}

// formatEntityName formats the entity name for display
func (s *AuditListenerService) formatEntityName(entity string) string {
	// Remove trailing 's' for singular form and capitalize
	name := strings.TrimSuffix(entity, "s")

	return stringutils.Title(name)
}

// getEventTypeForEntity returns the event type for the entity
func (s *AuditListenerService) getEventTypeForEntity(entity string) notification.EventType {
	switch entity {
	case "shipments":
		return notification.EventShipmentUpdated
	case "workers":
		return notification.EventWorkerUpdated
	case "customers":
		return notification.EventCustomerUpdated
	case "tractors":
		return notification.EventTractorUpdated
	case "trailers":
		return notification.EventTrailerUpdated
	case "locations":
		return notification.EventLocationUpdated
	default:
		return notification.EventEntityUpdated
	}
}

// extractUpdateDetails extracts meaningful update details from the audit entry
func (s *AuditListenerService) extractUpdateDetails(entry *audit.Entry) map[string]any {
	if entry == nil {
		return nil
	}

	// The Changes field contains what was changed
	if len(entry.Changes) > 0 {
		return entry.Changes
	}

	// If no Changes field, try to compute from states
	if entry.CurrentState == nil {
		return nil
	}

	// Extract only the changed fields
	changedFields := make(map[string]any)

	// If we have previous state, compare to find what changed
	if entry.PreviousState != nil {
		for key, newValue := range entry.CurrentState {
			if oldValue, exists := entry.PreviousState[key]; exists {
				if !s.valuesEqual(oldValue, newValue) {
					changedFields[key] = map[string]any{
						"old": oldValue,
						"new": newValue,
					}
				}
			} else {
				// Field was added
				changedFields[key] = map[string]any{
					"new": newValue,
				}
			}
		}
	} else {
		// No previous state, so all current state values are changes
		changedFields = entry.CurrentState
	}

	return changedFields
}

// valuesEqual compares two values for equality
func (s *AuditListenerService) valuesEqual(a, b any) bool {
	// Simple comparison - could be enhanced for more complex types
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}
