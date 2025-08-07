/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package notification

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

var ErrAlreadyRunning = eris.New("🚨 Batch Processor is already running")

type BatchProcessorParams struct {
	fx.In
	LC fx.Lifecycle

	Logger               *logger.Logger
	NotificationService  services.NotificationService
	NotificationPrefRepo repositories.NotificationPreferenceRepository
}

// PendingNotification represents a notification waiting to be batched
type PendingNotification struct {
	UserID          pulid.ID
	OrganizationID  pulid.ID
	BusinessUnitID  pulid.ID
	EventType       notification.EventType
	Title           string
	Message         string
	Data            map[string]any
	RelatedEntities []notification.RelatedEntity
	QueuedAt        time.Time
}

type BatchProcessor struct {
	l                    *zerolog.Logger
	notificationService  services.NotificationService
	notificationPrefRepo repositories.NotificationPreferenceRepository

	// Batch storage
	batches map[string][]*PendingNotification // key: userID
	mu      sync.RWMutex

	// Control mechanisms
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	ticker    *time.Ticker
	isRunning bool
}

func NewBatchProcessor(p BatchProcessorParams) *BatchProcessor {
	log := p.Logger.With().
		Str("service", "batch_processor").
		Logger()

	ctx, cancel := context.WithCancel(context.Background())

	processor := &BatchProcessor{
		l:                    &log,
		notificationService:  p.NotificationService,
		notificationPrefRepo: p.NotificationPrefRepo,
		batches:              make(map[string][]*PendingNotification),
		ctx:                  ctx,
		cancel:               cancel,
	}

	// Register lifecycle hooks
	p.LC.Append(fx.Hook{
		OnStart: func(context.Context) error {
			return processor.Start()
		},
		OnStop: func(context.Context) error {
			return processor.Stop()
		},
	})

	return processor
}

// Start begins the batch processor
func (bp *BatchProcessor) Start() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if bp.isRunning {
		return ErrAlreadyRunning
	}

	bp.l.Info().Msg("🚀 Starting Batch Processor")

	// Check every minute for batches to send
	bp.ticker = time.NewTicker(1 * time.Minute)
	bp.isRunning = true

	bp.wg.Add(1)
	go bp.process()

	return nil
}

// Stop gracefully shuts down the batch processor
func (bp *BatchProcessor) Stop() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if !bp.isRunning {
		return nil
	}

	bp.l.Info().Msg("stopping batch processor")

	// Send any remaining batches before stopping
	bp.flushAllBatches()

	bp.cancel()
	if bp.ticker != nil {
		bp.ticker.Stop()
	}

	// Wait for the processor to finish
	done := make(chan struct{})
	go func() {
		bp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		bp.l.Info().Msg("batch processor stopped successfully")
	case <-time.After(10 * time.Second):
		bp.l.Warn().Msg("batch processor stop timeout")
	}

	bp.isRunning = false
	return nil
}

// AddToBatch adds a notification to the user's batch
func (bp *BatchProcessor) AddToBatch(
	userID pulid.ID,
	orgID pulid.ID,
	buID pulid.ID,
	eventType notification.EventType,
	title string,
	message string,
	data map[string]any,
	relatedEntities []notification.RelatedEntity,
) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	key := userID.String()

	pending := &PendingNotification{
		UserID:          userID,
		OrganizationID:  orgID,
		BusinessUnitID:  buID,
		EventType:       eventType,
		Title:           title,
		Message:         message,
		Data:            data,
		RelatedEntities: relatedEntities,
		QueuedAt:        time.Now(),
	}

	bp.batches[key] = append(bp.batches[key], pending)

	bp.l.Debug().
		Str("user_id", userID.String()).
		Int("batch_size", len(bp.batches[key])).
		Msg("notification added to batch")
}

// process is the main loop that processes batches
func (bp *BatchProcessor) process() {
	defer bp.wg.Done()

	for {
		select {
		case <-bp.ctx.Done():
			return
		case <-bp.ticker.C:
			bp.processBatches()
		}
	}
}

// processBatches checks all batches and sends those that are ready
func (bp *BatchProcessor) processBatches() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	for userID, batch := range bp.batches {
		if len(batch) == 0 {
			continue
		}

		// Get user's batch interval preference
		shouldSend, err := bp.shouldSendBatch(userID, batch)
		if err != nil {
			bp.l.Error().Err(err).Str("user_id", userID).Msg("error checking batch send criteria")
			continue
		}

		if shouldSend {
			bp.sendBatch(userID, batch)
			// Clear the batch after sending
			delete(bp.batches, userID)
		}
	}
}

// shouldSendBatch checks if a batch should be sent based on user preferences
func (bp *BatchProcessor) shouldSendBatch(
	userIDStr string,
	batch []*PendingNotification,
) (bool, error) {
	if len(batch) == 0 {
		return false, nil
	}

	userID, err := pulid.Parse(userIDStr)
	if err != nil {
		return false, err
	}

	// Get user's notification preferences
	prefs, err := bp.notificationPrefRepo.GetUserPreferences(
		bp.ctx,
		&repositories.GetUserPreferencesRequest{
			UserID:         userID,
			OrganizationID: batch[0].OrganizationID,
			IsActive:       true,
		},
	)
	if err != nil {
		return false, err
	}

	// Find the preference with batch settings
	var batchInterval int
	for _, pref := range prefs {
		if pref.BatchNotifications && pref.BatchIntervalMinutes > 0 {
			batchInterval = pref.BatchIntervalMinutes
			break
		}
	}

	// If no batch preference found, send immediately
	if batchInterval == 0 {
		return true, nil
	}

	// Check if oldest notification has exceeded the batch interval
	oldestNotification := batch[0]
	timeSinceQueued := time.Since(oldestNotification.QueuedAt)

	return timeSinceQueued >= time.Duration(batchInterval)*time.Minute, nil
}

// sendBatch sends a batched notification to the user
func (bp *BatchProcessor) sendBatch(userIDStr string, batch []*PendingNotification) {
	if len(batch) == 0 {
		return
	}

	userID, _ := pulid.Parse(userIDStr)

	// Create a summary notification
	title := fmt.Sprintf("You have %d updates", len(batch))

	// Group notifications by type
	typeGroups := make(map[string]int)
	allRelatedEntities := make([]notification.RelatedEntity, 0)

	for _, notif := range batch {
		typeGroups[string(notif.EventType)]++
		allRelatedEntities = append(allRelatedEntities, notif.RelatedEntities...)
	}

	// Build summary message
	var summaryParts []string
	for eventType, count := range typeGroups {
		summaryParts = append(
			summaryParts,
			fmt.Sprintf("%d %s", count, bp.getEventTypeDisplay(eventType)),
		)
	}

	message := fmt.Sprintf("Summary: %s", summaryParts[0])
	if len(summaryParts) > 1 {
		message = fmt.Sprintf("Summary: %s and %d more types", summaryParts[0], len(summaryParts)-1)
	}

	// Create batch data with all notifications
	batchData := map[string]any{
		"notifications": batch,
		"count":         len(batch),
		"types":         typeGroups,
	}

	// Send the batched notification
	req := &services.SendNotificationRequest{
		EventType: notification.EventBatchSummary,
		Priority:  notification.PriorityMedium,
		Targeting: notification.Targeting{
			Channel:        notification.ChannelUser,
			OrganizationID: batch[0].OrganizationID,
			BusinessUnitID: &batch[0].BusinessUnitID,
			TargetUserID:   &userID,
		},
		Title:           title,
		Message:         message,
		Data:            batchData,
		RelatedEntities: allRelatedEntities,
		Actions: []notification.Action{
			{
				ID:       "view_all",
				Label:    "View All Updates",
				Type:     "link",
				Style:    "primary",
				Endpoint: "/notifications",
			},
		},
		Source: "batch_processor",
		Tags:   []string{"batch", "summary"},
	}

	if err := bp.notificationService.SendNotification(bp.ctx, req); err != nil {
		bp.l.Error().Err(err).Str("user_id", userIDStr).Msg("failed to send batch notification")
	}
}

// flushAllBatches sends all pending batches immediately
func (bp *BatchProcessor) flushAllBatches() {
	for userID, batch := range bp.batches {
		if len(batch) > 0 {
			bp.sendBatch(userID, batch)
		}
	}
	bp.batches = make(map[string][]*PendingNotification)
}

// getEventTypeDisplay returns a user-friendly display name for an event type
func (bp *BatchProcessor) getEventTypeDisplay(eventType string) string {
	displayMap := map[string]string{
		string(notification.EventShipmentUpdated): "shipment updates",
		string(notification.EventWorkerUpdated):   "worker updates",
		string(notification.EventCustomerUpdated): "customer updates",
		string(notification.EventTractorUpdated):  "tractor updates",
		string(notification.EventTrailerUpdated):  "trailer updates",
		string(notification.EventLocationUpdated): "location updates",
		string(notification.EventEntityUpdated):   "entity updates",
	}

	if display, ok := displayMap[eventType]; ok {
		return display
	}

	return "updates"
}
