package schedule

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/workfeature"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
)

type ReconcileResult struct {
	Created  []string
	Updated  []string
	Deleted  []string
	Skipped  []string
	Errors   []error
	Withheld []platformcatalog.FeatureKey
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
	features platformcatalogports.FeatureGate
	logger   *zap.Logger
}

func NewReconciler(
	c client.Client,
	registry *Registry,
	features platformcatalogports.FeatureGate,
	logger *zap.Logger,
) *Reconciler {
	return &Reconciler{
		client:   c,
		registry: registry,
		features: features,
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
	withheld, err := r.withheldFeatures(ctx, desiredSchedules)
	if err != nil {
		return nil, fmt.Errorf("failed to read the plan's features: %w", err)
	}
	result.Withheld = withheld

	existingSchedules, err := r.listExistingSchedules(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing schedules: %w", err)
	}

	log.Debug("starting reconciliation",
		zap.Int("desired", len(desiredSchedules)),
		zap.Int("existing", len(existingSchedules)),
	)

	for id, sched := range desiredSchedules {
		want := desired{Schedule: sched, withheld: slices.Contains(withheld, sched.Feature)}
		existing, exists := existingSchedules[id]
		if !exists {
			if err = r.createSchedule(ctx, want); err != nil {
				log.Error("failed to create schedule", zap.String("id", id), zap.Error(err))
				result.Errors = append(result.Errors, fmt.Errorf("create %s: %w", id, err))
				continue
			}
			result.Created = append(result.Created, id)
			log.Info("created schedule", zap.String("id", id), zap.Bool("withheld", want.withheld))
			continue
		}

		if r.needsUpdate(existing, want) {
			if err = r.updateSchedule(ctx, want); err != nil {
				log.Error("failed to update schedule", zap.String("id", id), zap.Error(err))
				result.Errors = append(result.Errors, fmt.Errorf("update %s: %w", id, err))
				continue
			}
			result.Updated = append(result.Updated, id)
			log.Info("updated schedule", zap.String("id", id), zap.Bool("withheld", want.withheld))
		} else {
			result.Skipped = append(result.Skipped, id)
			log.Debug("schedule unchanged", zap.String("id", id))
		}
	}

	for id, existing := range existingSchedules {
		if _, wanted := desiredSchedules[id]; !wanted {
			if !ownedByRegistry(existing) {
				continue
			}
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
		zap.Strings("withheldFeatures", featureStrings(result.Withheld)),
	)

	return result, nil
}

// Withheld is the sorted list of features the registered schedules name that
// the plan does not include. It asks the gate once per feature and touches
// nothing in Temporal.
func (r *Reconciler) Withheld(ctx context.Context) ([]platformcatalog.FeatureKey, error) {
	return r.withheldFeatures(ctx, r.registry.GetSchedules())
}

func (r *Reconciler) withheldFeatures(
	ctx context.Context,
	schedules map[string]*Schedule,
) ([]platformcatalog.FeatureKey, error) {
	named := make(map[platformcatalog.FeatureKey]struct{}, len(schedules))
	for _, sched := range schedules {
		named[sched.Feature] = struct{}{}
	}

	withheld := make([]platformcatalog.FeatureKey, 0)
	for _, feature := range slices.Sorted(maps.Keys(named)) {
		on, err := r.features.EnabledAnywhere(ctx, feature)
		if err != nil {
			return nil, fmt.Errorf("feature %q: %w", feature, err)
		}
		if !on {
			withheld = append(withheld, feature)
		}
	}

	return withheld, nil
}

func featureStrings(features []platformcatalog.FeatureKey) []string {
	out := make([]string, len(features))
	for i, feature := range features {
		out[i] = string(feature)
	}
	return out
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

func (r *Reconciler) needsUpdate(existing *client.ScheduleListEntry, want desired) bool {
	if hash, ok := hashFromNote(existing.Note); ok {
		return hash != want.hash()
	}

	if existing.Memo != nil && existing.Memo.Fields != nil {
		if hashPayload, ok := existing.Memo.GetFields()["scheduleHash"]; ok {
			hash := string(hashPayload.GetData())
			return hash != want.hash()
		}
	}
	return true
}

func (r *Reconciler) createSchedule(ctx context.Context, want desired) error {
	ctx = workfeature.Starting(ctx, want.Feature)
	sc := r.client.ScheduleClient()
	opts := want.options()

	_, err := sc.Create(ctx, opts)
	if err != nil {
		if isScheduleAlreadyExists(err) {
			r.logger.Debug("schedule already exists, updating instead", zap.String("id", want.ID))
			return r.updateSchedule(ctx, want)
		}
		return err
	}
	return nil
}

func isScheduleAlreadyExists(err error) bool {
	if errors.Is(err, temporal.ErrScheduleAlreadyRunning) {
		return true
	}

	var alreadyExists *serviceerror.AlreadyExists
	return errors.As(err, &alreadyExists)
}

func (r *Reconciler) updateSchedule(ctx context.Context, want desired) error {
	ctx = workfeature.Starting(ctx, want.Feature)
	sc := r.client.ScheduleClient()
	handle := sc.GetHandle(ctx, want.ID)

	return handle.Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: func(input client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
			opts := want.options()

			input.Description.Schedule.Spec = &opts.Spec
			input.Description.Schedule.Action = opts.Action

			// Overlap and paused are part of the definition too. Leaving them
			// out meant a change to either was hashed, reported as updated, and
			// never applied.
			if input.Description.Schedule.Policy == nil {
				input.Description.Schedule.Policy = &client.SchedulePolicies{}
			}
			input.Description.Schedule.Policy.Overlap = opts.Overlap

			// The paused state is read against the schedule's current one,
			// so a pause a person made by hand is kept (desired.state).
			if input.Description.Schedule.State == nil {
				input.Description.Schedule.State = &client.ScheduleState{}
			}
			state := input.Description.Schedule.State
			state.Paused, state.Note = want.state(*state)

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

// ownedByRegistry reports whether the registry created a schedule and may
// therefore delete it as an orphan. Schedules created before ownership was
// recorded carry no marker at all, and back then the registry was the only
// thing that created schedules, so an unmarked schedule is the registry's.
// A schedule marked as anything else belongs to whatever marked it.
func ownedByRegistry(entry *client.ScheduleListEntry) bool {
	if entry.Memo == nil || entry.Memo.GetFields() == nil {
		return true
	}

	payload, ok := entry.Memo.GetFields()[ManagedByMemoKey]
	if !ok {
		return true
	}

	var owner string
	if err := converter.GetDefaultDataConverter().FromPayload(payload, &owner); err != nil {
		return false
	}

	return owner == ManagedByRegistry
}
