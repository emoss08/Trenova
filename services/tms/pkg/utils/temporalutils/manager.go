package temporalutils

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/client"
	"github.com/rs/zerolog"
)

// ScheduleManager provides high-level schedule management operations
type ScheduleManager struct {
	client    client.Client
	schedules map[string]client.ScheduleHandle
	logger    *zerolog.Logger
}

// NewScheduleManager creates a new schedule manager
func NewScheduleManager(c client.Client, logger *zerolog.Logger) *ScheduleManager {
	return &ScheduleManager{
		client:    c,
		schedules: make(map[string]client.ScheduleHandle),
		logger:    logger,
	}
}

// CreateOrUpdateSchedule creates a new schedule or updates an existing one
func (m *ScheduleManager) CreateOrUpdateSchedule(ctx context.Context, config ScheduleConfig) error {
	scheduleClient := m.client.ScheduleClient()
	handle := scheduleClient.GetHandle(ctx, config.ID)
	
	// Check if schedule exists
	existing, err := handle.Describe(ctx)
	if err == nil {
		// Schedule exists, check if update needed
		if m.needsUpdate(existing, config) {
			if err := m.updateSchedule(ctx, handle, config); err != nil {
				return fmt.Errorf("failed to update schedule %s: %w", config.ID, err)
			}
			m.logger.Info().
				Str("scheduleID", config.ID).
				Msg("schedule updated")
		} else {
			m.logger.Debug().
				Str("scheduleID", config.ID).
				Msg("schedule unchanged")
		}
		m.schedules[config.ID] = handle
		return nil
	}
	
	// Create new schedule
	opts := BuildSchedule(config)
	newHandle, err := scheduleClient.Create(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to create schedule %s: %w", config.ID, err)
	}
	
	m.schedules[config.ID] = newHandle
	m.logger.Info().
		Str("scheduleID", config.ID).
		Str("description", config.Description).
		Msg("schedule created")
	
	return nil
}

// needsUpdate checks if a schedule needs updating
func (m *ScheduleManager) needsUpdate(existing *client.ScheduleDescription, config ScheduleConfig) bool {
	// Check schedule spec
	if CompareScheduleSpecs(existing.Schedule.Spec, config.Schedule) {
		return true
	}
	
	// Check workflow action
	if action, ok := existing.Schedule.Action.(*client.ScheduleWorkflowAction); ok {
		if action.Workflow != config.WorkflowType || action.TaskQueue != config.TaskQueue {
			return true
		}
	}
	
	return false
}

// updateSchedule updates an existing schedule
func (m *ScheduleManager) updateSchedule(ctx context.Context, handle client.ScheduleHandle, config ScheduleConfig) error {
	return handle.Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: func(input client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
			UpdateScheduleSpec(&input, config.Schedule)
			return &client.ScheduleUpdate{
				Schedule: &input.Description.Schedule,
			}, nil
		},
	})
}

// DeleteSchedule deletes a schedule
func (m *ScheduleManager) DeleteSchedule(ctx context.Context, scheduleID string) error {
	handle, exists := m.schedules[scheduleID]
	if !exists {
		handle = m.client.ScheduleClient().GetHandle(ctx, scheduleID)
	}
	
	if err := handle.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete schedule %s: %w", scheduleID, err)
	}
	
	delete(m.schedules, scheduleID)
	m.logger.Info().
		Str("scheduleID", scheduleID).
		Msg("schedule deleted")
	
	return nil
}

// GetHandle returns the schedule handle for a given ID
func (m *ScheduleManager) GetHandle(ctx context.Context, scheduleID string) client.ScheduleHandle {
	if handle, exists := m.schedules[scheduleID]; exists {
		return handle
	}
	return m.client.ScheduleClient().GetHandle(ctx, scheduleID)
}

// Clear removes all managed schedules
func (m *ScheduleManager) Clear(ctx context.Context) error {
	for id, handle := range m.schedules {
		if err := handle.Delete(ctx); err != nil {
			m.logger.Error().
				Str("scheduleID", id).
				Err(err).
				Msg("failed to delete schedule")
		} else {
			m.logger.Info().
				Str("scheduleID", id).
				Msg("schedule deleted")
		}
	}
	m.schedules = make(map[string]client.ScheduleHandle)
	return nil
}