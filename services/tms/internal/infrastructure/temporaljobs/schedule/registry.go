package schedule

import (
	"fmt"
	"maps"
	"sync"

	"go.uber.org/zap"
)

type Registry struct {
	providers []Provider
	schedules map[string]*Schedule
	mu        sync.RWMutex
	logger    *zap.Logger
}

func NewRegistry(logger *zap.Logger) *Registry {
	return &Registry{
		providers: make([]Provider, 0),
		schedules: make(map[string]*Schedule),
		logger:    logger.Named("schedule-registry"),
	}
}

func (r *Registry) RegisterProvider(provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers = append(r.providers, provider)
}

func (r *Registry) CollectSchedules() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.schedules = make(map[string]*Schedule)

	for _, provider := range r.providers {
		for _, sched := range provider.GetSchedules() {
			if err := sched.Validate(); err != nil {
				return fmt.Errorf("invalid schedule %q: %w", sched.ID, err)
			}

			if existing, exists := r.schedules[sched.ID]; exists {
				return fmt.Errorf("%w: schedule ID %q already registered (existing: %s, new: %s)",
					ErrDuplicateScheduleID, sched.ID, existing.Description, sched.Description)
			}

			r.schedules[sched.ID] = sched
			r.logger.Debug("collected schedule",
				zap.String("id", sched.ID),
				zap.String("hash", sched.Hash()),
				zap.String("description", sched.Description),
			)
		}
	}

	r.logger.Info("collected all schedules",
		zap.Int("count", len(r.schedules)),
		zap.Int("providers", len(r.providers)),
	)

	return nil
}

func (r *Registry) GetSchedules() map[string]*Schedule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*Schedule, len(r.schedules))
	maps.Copy(result, r.schedules)
	return result
}

func (r *Registry) GetScheduleIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.schedules))
	for id := range r.schedules {
		ids = append(ids, id)
	}
	return ids
}

func (r *Registry) GetSchedule(id string) (*Schedule, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sched, exists := r.schedules[id]
	return sched, exists
}

func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.schedules)
}

func (r *Registry) ProviderCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.providers)
}
