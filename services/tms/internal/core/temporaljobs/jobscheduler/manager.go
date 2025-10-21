package jobscheduler

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ManagerParams struct {
	fx.In

	Client client.Client
	Logger *zap.Logger
}

type Manager struct {
	client    client.Client
	schedules map[string]client.ScheduleHandle
	l         *zap.Logger
}

func NewManager(p ManagerParams) *Manager {
	return &Manager{
		client:    p.Client,
		schedules: make(map[string]client.ScheduleHandle),
		l:         p.Logger.With(zap.String("component", "job-scheduler-manager")),
	}
}

func (m *Manager) CreateOrUpdateSchedule(ctx context.Context, config *ScheduleConfig) error {
	log := m.l.With(zap.String("operation", "CreateOrUpdateSchedule"), zap.Any("config", config))
	sc := m.client.ScheduleClient()
	handle := sc.GetHandle(ctx, config.ID)

	existing, err := handle.Describe(ctx)
	if err == nil {
		if m.needsUpdate(existing, config) {
			if err = m.updateSchedule(ctx, handle, config); err != nil {
				log.Error("failed to update schedule", zap.Error(err))
				return fmt.Errorf("failed to update schedule %s: %w", config.ID, err)
			}
			log.Info("schedule updated")
		} else {
			log.Debug("schedule unchanged")
		}
		m.schedules[config.ID] = handle
		return nil
	}

	opts := BuildSchedule(config)
	nh, err := sc.Create(ctx, opts)
	if err != nil {
		log.Error("failed to create schedule", zap.Error(err))
		return fmt.Errorf("failed to create schedule %s: %w", config.ID, err)
	}

	m.schedules[config.ID] = nh
	log.Info("schedule created")
	return nil
}

func (m *Manager) needsUpdate(existing *client.ScheduleDescription, config *ScheduleConfig) bool {
	if CompareScheduleSpecs(existing.Schedule.Spec, &config.Schedule) {
		return true
	}

	if action, ok := existing.Schedule.Action.(*client.ScheduleWorkflowAction); ok {
		if action.Workflow != config.WorkflowType || action.TaskQueue != config.TaskQueue {
			return true
		}
	}

	return false
}

func (m *Manager) updateSchedule(
	ctx context.Context,
	handle client.ScheduleHandle,
	config *ScheduleConfig,
) error {
	return handle.Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: func(input client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
			UpdateScheduleSpec(&input, &config.Schedule)
			return &client.ScheduleUpdate{
				Schedule: &input.Description.Schedule,
			}, nil
		},
	})
}

func (m *Manager) DeleteSchedule(ctx context.Context, scheduleID string) error {
	handle, exists := m.schedules[scheduleID]
	if !exists {
		handle = m.client.ScheduleClient().GetHandle(ctx, scheduleID)
	}

	if err := handle.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete schedule %s: %w", scheduleID, err)
	}

	delete(m.schedules, scheduleID)
	m.l.Info("schedule deleted", zap.String("scheduleID", scheduleID))
	return nil
}

func (m *Manager) GetHandle(ctx context.Context, scheduleID string) client.ScheduleHandle {
	if handle, exists := m.schedules[scheduleID]; exists {
		return handle
	}

	return m.client.ScheduleClient().GetHandle(ctx, scheduleID)
}

func (m *Manager) Clear(ctx context.Context) error {
	log := m.l.With(zap.String("operation", "Clear"))
	for id, handle := range m.schedules {
		if err := handle.Delete(ctx); err != nil {
			log.Error("failed to delete schedule", zap.String("scheduleID", id), zap.Error(err))
		} else {
			log.Info("schedule deleted", zap.String("scheduleID", id))
		}
	}

	m.schedules = make(map[string]client.ScheduleHandle)
	log.Info("schedules cleared")
	return nil
}
