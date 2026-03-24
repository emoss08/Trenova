package schedule

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/shared/intutils"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type ReconcileResult struct {
	Created []string
	Updated []string
	Deleted []string
	Skipped []string
	Errors  []error
}

func (r *ReconcileResult) HasErrors() bool {
	return len(r.Errors) > 0
}

func (r *ReconcileResult) Summary() string {
	return fmt.Sprintf("created=%d updated=%d deleted=%d skipped=%d errors=%d",
		len(r.Created), len(r.Updated), len(r.Deleted), len(r.Skipped), len(r.Errors))
}

type Reconciler struct {
	client   client.Client
	registry *Registry
	logger   *zap.Logger
}

func NewReconciler(c client.Client, registry *Registry, logger *zap.Logger) *Reconciler {
	return &Reconciler{
		client:   c,
		registry: registry,
		logger:   logger.Named("schedule-reconciler"),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) (*ReconcileResult, error) {
	log := r.logger.With(zap.String("operation", "Reconcile"))
	result := &ReconcileResult{
		Created: make([]string, 0),
		Updated: make([]string, 0),
		Deleted: make([]string, 0),
		Skipped: make([]string, 0),
		Errors:  make([]error, 0),
	}

	if err := r.registry.CollectSchedules(); err != nil {
		return nil, fmt.Errorf("failed to collect schedules: %w", err)
	}

	desiredSchedules := r.registry.GetSchedules()
	existingSchedules, err := r.listExistingSchedules(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing schedules: %w", err)
	}

	log.Debug("starting reconciliation",
		zap.Int("desired", len(desiredSchedules)),
		zap.Int("existing", len(existingSchedules)),
	)

	for id, desired := range desiredSchedules {
		existing, exists := existingSchedules[id]
		if !exists {
			if err = r.createSchedule(ctx, desired); err != nil {
				log.Error("failed to create schedule", zap.String("id", id), zap.Error(err))
				result.Errors = append(result.Errors, fmt.Errorf("create %s: %w", id, err))
				continue
			}
			result.Created = append(result.Created, id)
			log.Info("created schedule", zap.String("id", id))
			continue
		}

		if r.needsUpdate(existing, desired) {
			if err = r.updateSchedule(ctx, desired); err != nil {
				log.Error("failed to update schedule", zap.String("id", id), zap.Error(err))
				result.Errors = append(result.Errors, fmt.Errorf("update %s: %w", id, err))
				continue
			}
			result.Updated = append(result.Updated, id)
			log.Info("updated schedule", zap.String("id", id))
		} else {
			result.Skipped = append(result.Skipped, id)
			log.Debug("schedule unchanged", zap.String("id", id))
		}
	}

	for id := range existingSchedules {
		if _, desired := desiredSchedules[id]; !desired {
			if err = r.deleteSchedule(ctx, id); err != nil {
				log.Error("failed to delete orphan schedule", zap.String("id", id), zap.Error(err))
				result.Errors = append(result.Errors, fmt.Errorf("delete %s: %w", id, err))
				continue
			}
			result.Deleted = append(result.Deleted, id)
			log.Info("deleted orphan schedule", zap.String("id", id))
		}
	}

	log.Info("reconciliation complete",
		zap.Strings("created", result.Created),
		zap.Strings("updated", result.Updated),
		zap.Strings("deleted", result.Deleted),
		zap.Int("skipped", len(result.Skipped)),
		zap.Int("errors", len(result.Errors)),
	)

	return result, nil
}

func (r *Reconciler) listExistingSchedules(
	ctx context.Context,
) (map[string]*client.ScheduleListEntry, error) {
	sc := r.client.ScheduleClient()
	result := make(map[string]*client.ScheduleListEntry)

	iter, err := sc.List(ctx, client.ScheduleListOptions{
		PageSize: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}

	for iter.HasNext() {
		entry, entryErr := iter.Next()
		if entryErr != nil {
			return nil, fmt.Errorf("failed to iterate schedules: %w", entryErr)
		}
		result[entry.ID] = entry
	}

	return result, nil
}

func (r *Reconciler) needsUpdate(existing *client.ScheduleListEntry, desired *Schedule) bool {
	if existing.Memo != nil && existing.Memo.Fields != nil {
		if hashPayload, ok := existing.Memo.GetFields()["scheduleHash"]; ok {
			hash := string(hashPayload.GetData())
			return hash != desired.Hash()
		}
	}
	return true
}

func (r *Reconciler) createSchedule(ctx context.Context, sched *Schedule) error {
	sc := r.client.ScheduleClient()
	opts := sched.ToScheduleOptions()

	_, err := sc.Create(ctx, opts)
	if err != nil {
		var alreadyExists *serviceerror.AlreadyExists
		if errors.As(err, &alreadyExists) {
			r.logger.Debug("schedule already exists, updating instead", zap.String("id", sched.ID))
			return r.updateSchedule(ctx, sched)
		}
		return err
	}
	return nil
}

func (r *Reconciler) updateSchedule(ctx context.Context, sched *Schedule) error {
	sc := r.client.ScheduleClient()
	handle := sc.GetHandle(ctx, sched.ID)

	return handle.Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: func(input client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
			opts := sched.ToScheduleOptions()

			input.Description.Schedule.Spec = &opts.Spec
			input.Description.Schedule.Action = opts.Action

			return &client.ScheduleUpdate{
				Schedule: &input.Description.Schedule,
			}, nil
		},
	})
}

func (r *Reconciler) deleteSchedule(ctx context.Context, id string) error {
	sc := r.client.ScheduleClient()
	handle := sc.GetHandle(ctx, id)

	err := handle.Delete(ctx)
	if err != nil {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			r.logger.Debug("schedule already deleted", zap.String("id", id))
			return nil
		}
		return err
	}
	return nil
}

func (r *Reconciler) ReconcileWithRetry(
	ctx context.Context,
	maxRetries int,
) (*ReconcileResult, error) {
	var result *ReconcileResult
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		result, lastErr = r.Reconcile(ctx)
		if lastErr == nil && !result.HasErrors() {
			return result, nil
		}

		if i < maxRetries {
			backoff := min(
				time.Second*time.Duration(1<<intutils.SafeShiftAmount(i, 5)),
				30*time.Second,
			)

			r.logger.Warn("reconciliation had errors, retrying",
				zap.Int("attempt", i+1),
				zap.Int("maxRetries", maxRetries),
				zap.Duration("backoff", backoff),
				zap.Error(lastErr),
			)

			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(backoff):
			}
		}
	}

	if lastErr != nil {
		return result, lastErr
	}

	if result.HasErrors() {
		return result, errors.Join(result.Errors...)
	}

	return result, nil
}
