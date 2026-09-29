package aitrainingservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var _ services.AIRetrainingService = (*Retrainer)(nil)

const (
	unstartedExportGrace = int64(time.Hour / time.Second)
	maxTrainerRunes      = aitraining.MaxTrainerRunes
)

type RetrainerParams struct {
	fx.In

	Logger      *zap.Logger
	Config      *config.Config
	Cycles      repositories.RetrainingCycleRepository
	Exports     repositories.AITrainingExportRepository
	Corrections repositories.AICorrectionRepository
	Operator    services.AITrainingExportOperator
	Alerter     services.RetrainingAlerter `optional:"true"`
}

type Retrainer struct {
	l           *zap.Logger
	cfg         *config.AIRetrainingConfig
	cycles      repositories.RetrainingCycleRepository
	exports     repositories.AITrainingExportRepository
	corrections repositories.AICorrectionRepository
	operator    services.AITrainingExportOperator
	alerter     services.RetrainingAlerter
	now         func() int64
}

func NewRetrainer(p RetrainerParams) *Retrainer { //nolint:gocritic // fx params are passed by value
	return &Retrainer{
		l:           p.Logger.Named("service.ai-retraining"),
		cfg:         &p.Config.AIRetraining,
		cycles:      p.Cycles,
		exports:     p.Exports,
		corrections: p.Corrections,
		operator:    p.Operator,
		alerter:     p.Alerter,
		now:         timeutils.NowUnix,
	}
}

func AsRetrainer(r *Retrainer) services.AIRetrainingService { return r }

func (r *Retrainer) LeaseDuration() time.Duration {
	return r.cfg.GetLeaseDuration()
}

func (r *Retrainer) policy() aitraining.RetrainingPolicy {
	return aitraining.RetrainingPolicy{
		LookbackDays:         r.cfg.GetLookbackDays(),
		MinNewExamples:       r.cfg.GetMinNewExamples(),
		MinIntervalDays:      r.cfg.GetMinIntervalDays(),
		RetrainOnDrift:       r.cfg.GetRetrainOnDrift(),
		MaxPerOrganization:   r.cfg.GetMaxPerOrganization(),
		ValidationPercent:    r.cfg.GetValidationPercent(),
		StructuredOutputMode: aiprovider.StructuredOutputMode(r.cfg.GetStructuredOutputMode()),
		Gate: aitraining.RetrainingGate{
			MinAccuracyPercent:  r.cfg.GetMinAccuracyPercent(),
			MaxRegressionPoints: r.cfg.GetMaxRegressionPoints(),
		},
	}
}

func (r *Retrainer) Plan(
	ctx context.Context,
	req *services.PlanAIRetrainingRequest,
) (*aitraining.RetrainingCycle, error) {
	if req == nil {
		req = &services.PlanAIRetrainingRequest{}
	}
	if !req.Manual && !r.cfg.IsEnabled() {
		return nil, nil //nolint:nilnil // a schedule that is off records no cycle
	}
	if _, err := r.Reconcile(ctx); err != nil {
		return nil, err
	}

	now := r.now()
	policy := r.policy()
	open, err := r.first(ctx, aitraining.OpenRetrainingStatuses())
	if err != nil {
		return nil, err
	}
	last, err := r.first(ctx, aitraining.BaselineRetrainingStatuses())
	if err != nil {
		return nil, err
	}
	exportActive, err := r.exportActive(ctx)
	if err != nil {
		return nil, err
	}

	input := &aitraining.RetrainingPlanInput{
		Now:          now,
		Policy:       policy,
		Manual:       req.Manual,
		RequestedBy:  strings.TrimSpace(req.RequestedBy),
		Note:         strings.TrimSpace(req.Note),
		Last:         last,
		OpenCycle:    open != nil,
		ExportActive: exportActive,
	}
	if !input.OpenCycle && !input.ExportActive {
		window := aitraining.NewRetrainingWindow(now, &policy, last)
		if input.NewExamples, err = r.corrections.CountTrainable(
			ctx,
			&repositories.CountTrainableAICorrectionsRequest{
				Task:               aicorrection.TaskShipmentDraftExtraction,
				CapturedFrom:       window.NewSince,
				CapturedTo:         window.CapturedTo,
				PerOrganizationCap: policy.MaxPerOrganization,
			},
		); err != nil {
			return nil, err
		}
		if !req.Manual && policy.RetrainOnDrift {
			if input.DriftingSeries, err = r.driftingSeries(ctx, now); err != nil {
				return nil, err
			}
		}
	}

	cycle := aitraining.PlanRetraining(input)
	if cycle.Status == aitraining.RetrainingStatusSkipped && req.Manual {
		return nil, errortypes.NewBusinessError(cycle.SkipReason.Message())
	}

	multiErr := errortypes.NewMultiError()
	cycle.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := r.cycles.Create(ctx, cycle)
	if err != nil {
		return nil, err
	}
	if created.Status == aitraining.RetrainingStatusSkipped {
		r.l.Info("retraining skipped",
			zap.String("cycleId", created.ID.String()),
			zap.String("reason", created.SkipReason.String()),
			zap.Int("newExamples", created.NewExamples),
			zap.Int("minNewExamples", created.MinNewExamples),
		)
		r.alert(ctx, created)
		return created, nil
	}

	return r.startExport(ctx, created)
}

func (r *Retrainer) startExport(
	ctx context.Context,
	cycle *aitraining.RetrainingCycle,
) (*aitraining.RetrainingCycle, error) {
	note := "Retraining cycle " + cycle.ID.String()
	if cycle.Note != "" {
		note += ": " + cycle.Note
	}
	export, err := r.operator.Start(ctx, &services.StartAITrainingExportRequest{
		CapturedFrom:       cycle.CapturedFrom,
		CapturedTo:         cycle.CapturedTo,
		MaxPerOrganization: cycle.MaxPerOrganization,
		ValidationPercent:  cycle.ValidationPercent,
		RequestedBy:        cycle.RequestedBy,
		Note:               note,
	})
	if err != nil {
		r.l.Error("failed to start retraining export",
			zap.String("cycleId", cycle.ID.String()),
			zap.Error(err),
		)
		cycle.Fail("The training export could not be started: "+err.Error(), r.now())
		if _, updateErr := r.cycles.Update(ctx, cycle); updateErr != nil {
			return nil, errors.Join(err, updateErr)
		}
		r.alert(ctx, cycle)

		return cycle, fmt.Errorf("start retraining export: %w", err)
	}

	cycle.StartedExport(export.ID)
	updated, err := r.cycles.Update(ctx, cycle)
	if err != nil {
		return nil, err
	}

	r.l.Info("retraining started",
		zap.String("cycleId", updated.ID.String()),
		zap.String("exportId", export.ID.String()),
		zap.String("trigger", updated.Trigger.String()),
		zap.Int("newExamples", updated.NewExamples),
		zap.Int("driftingProviders", updated.DriftingProviders),
	)
	r.alert(ctx, updated)

	return updated, nil
}

func (r *Retrainer) driftingSeries(ctx context.Context, now int64) (int, error) {
	window := aicorrection.NewTrendWindow(now)
	totals, err := r.corrections.WeeklyTrainableTotalsByProvider(
		ctx,
		&repositories.WeeklyTrainableAICorrectionTotalsRequest{
			Task:  aicorrection.TaskShipmentDraftExtraction,
			Since: window.BaselineStart,
		},
	)
	if err != nil {
		return 0, err
	}

	trends := aicorrection.BuildProviderTrends(window, totals)
	drifting := 0
	for i := range trends {
		if trends[i].Drifting {
			drifting++
		}
	}

	return drifting, nil
}

func (r *Retrainer) exportActive(ctx context.Context) (bool, error) {
	latest, err := r.exports.List(ctx, repositories.ListAITrainingExportsRequest{Limit: 1})
	if err != nil {
		return false, err
	}

	return len(latest) > 0 && latest[0].Status.IsActive(), nil
}

func (r *Retrainer) first(
	ctx context.Context,
	statuses []aitraining.RetrainingStatus,
) (*aitraining.RetrainingCycle, error) {
	cycles, err := r.cycles.List(ctx, repositories.ListRetrainingCyclesRequest{
		Statuses: statuses,
		Limit:    1,
	})
	if err != nil {
		return nil, err
	}
	if len(cycles) == 0 {
		return nil, nil //nolint:nilnil // no cycle in these statuses is a valid answer
	}

	return cycles[0], nil
}

func (r *Retrainer) Reconcile(ctx context.Context) (int, error) {
	cycle, err := r.first(ctx, []aitraining.RetrainingStatus{aitraining.RetrainingStatusExporting})
	if err != nil || cycle == nil {
		return 0, err
	}

	now := r.now()
	if cycle.ExportID == nil {
		if now-cycle.CreatedAt < unstartedExportGrace {
			return 0, nil
		}
		cycle.Fail("The training export was never started", now)
	} else {
		export, getErr := r.exports.GetByID(ctx, *cycle.ExportID)
		switch {
		case errortypes.IsNotFoundError(getErr):
			cycle.Fail("The training export no longer exists", now)
		case getErr != nil:
			return 0, getErr
		case !cycle.ExportFinished(export, now):
			return 0, nil
		}
	}

	updated, err := r.cycles.Update(ctx, cycle)
	if err != nil {
		if errortypes.IsVersionMismatchError(err) {
			return 0, nil
		}
		return 0, err
	}

	r.l.Info("retraining export settled",
		zap.String("cycleId", updated.ID.String()),
		zap.String("status", updated.Status.String()),
		zap.String("failure", updated.FailureMessage),
	)
	r.alert(ctx, updated)

	return 1, nil
}

func (r *Retrainer) ClaimNext(
	ctx context.Context,
	req *services.ClaimAIRetrainingRequest,
) (*aitraining.RetrainingCycle, error) {
	trainer, err := trainerName(req.Trainer)
	if err != nil {
		return nil, err
	}
	if _, err = r.Reconcile(ctx); err != nil {
		return nil, err
	}

	cycle, err := r.first(ctx, []aitraining.RetrainingStatus{
		aitraining.RetrainingStatusReady,
		aitraining.RetrainingStatusTraining,
	})
	if err != nil || cycle == nil {
		return nil, err
	}

	now := r.now()
	if !cycle.Claimable(now) {
		return nil, nil //nolint:nilnil // nothing to train is a valid answer
	}
	previous := cycle.Trainer
	if err = cycle.Claim(trainer, now, r.leaseSeconds()); err != nil {
		return nil, err
	}

	claimed, err := r.cycles.Update(ctx, cycle)
	if err != nil {
		if errortypes.IsVersionMismatchError(err) {
			return nil, nil //nolint:nilnil // another trainer claimed it first
		}
		return nil, err
	}

	fields := []zap.Field{
		zap.String("cycleId", claimed.ID.String()),
		zap.String("trainer", trainer),
		zap.Int("attempt", claimed.Attempts),
	}
	if previous != "" {
		fields = append(fields, zap.String("expiredTrainer", previous))
	}
	r.l.Info("retraining claimed", fields...)
	r.alert(ctx, claimed)

	return claimed, nil
}

func (r *Retrainer) Heartbeat(
	ctx context.Context,
	req *services.HeartbeatAIRetrainingRequest,
) (*aitraining.RetrainingCycle, error) {
	return r.mutateHeld(
		ctx,
		req.CycleID,
		req.Trainer,
		func(cycle *aitraining.RetrainingCycle, trainer string) error {
			return cycle.Extend(trainer, r.now(), r.leaseSeconds())
		},
	)
}

func (r *Retrainer) Record(
	ctx context.Context,
	req *services.RecordAIRetrainingRequest,
) (*aitraining.RetrainingCycle, error) {
	recorded, err := r.mutateHeld(
		ctx,
		req.CycleID,
		req.Trainer,
		func(cycle *aitraining.RetrainingCycle, trainer string) error {
			return cycle.Settle(trainer, req.Result, r.now())
		},
	)
	if err != nil {
		return nil, err
	}

	r.l.Info("retraining scored",
		zap.String("cycleId", recorded.ID.String()),
		zap.String("status", recorded.Status.String()),
		zap.Float64("modelAccuracy", recorded.ModelAccuracy()),
		zap.Float64("baselineAccuracy", recorded.BaselineAccuracy()),
		zap.String("model", recorded.ModelDirectory),
		zap.String("gate", recorded.GateMessage),
	)

	r.alert(ctx, recorded)

	return recorded, nil
}

func (r *Retrainer) FailTraining(
	ctx context.Context,
	req *services.FailAIRetrainingRequest,
) (*aitraining.RetrainingCycle, error) {
	failed, err := r.mutateHeld(
		ctx,
		req.CycleID,
		req.Trainer,
		func(cycle *aitraining.RetrainingCycle, trainer string) error {
			return cycle.FailTraining(trainer, req.Message, r.now())
		},
	)
	if err != nil {
		return nil, err
	}

	r.l.Warn("retraining failed",
		zap.String("cycleId", failed.ID.String()),
		zap.String("trainer", failed.Trainer),
		zap.String("failure", failed.FailureMessage),
	)

	r.alert(ctx, failed)

	return failed, nil
}

func (r *Retrainer) mutateHeld(
	ctx context.Context,
	id pulid.ID,
	trainer string,
	mutate func(*aitraining.RetrainingCycle, string) error,
) (*aitraining.RetrainingCycle, error) {
	name, err := trainerName(trainer)
	if err != nil {
		return nil, err
	}

	const attempts = 2
	var lastErr error
	for range attempts {
		cycle, getErr := r.cycles.GetByID(ctx, id)
		if getErr != nil {
			return nil, getErr
		}
		if mutateErr := mutate(cycle, name); mutateErr != nil {
			return nil, mutateErr
		}
		updated, updateErr := r.cycles.Update(ctx, cycle)
		if updateErr == nil {
			return updated, nil
		}
		if !errortypes.IsVersionMismatchError(updateErr) {
			return nil, updateErr
		}
		lastErr = updateErr
	}

	return nil, lastErr
}

func (r *Retrainer) Cancel(ctx context.Context, id pulid.ID) (*aitraining.RetrainingCycle, error) {
	cycle, err := r.cycles.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	exporting := cycle.Status == aitraining.RetrainingStatusExporting && cycle.ExportID != nil
	if err = cycle.Cancel(r.now()); err != nil {
		return nil, err
	}
	if exporting {
		if _, cancelErr := r.operator.Cancel(ctx, *cycle.ExportID); cancelErr != nil &&
			!errortypes.IsBusinessError(cancelErr) {
			return nil, fmt.Errorf("cancel the retraining export: %w", cancelErr)
		}
	}

	canceled, err := r.cycles.Update(ctx, cycle)
	if err != nil {
		return nil, err
	}

	r.l.Info("retraining canceled", zap.String("cycleId", canceled.ID.String()))
	r.alert(ctx, canceled)

	return canceled, nil
}

func (r *Retrainer) List(ctx context.Context, limit int) ([]*aitraining.RetrainingCycle, error) {
	return r.cycles.List(ctx, repositories.ListRetrainingCyclesRequest{Limit: limit})
}

func (r *Retrainer) Get(ctx context.Context, id pulid.ID) (*aitraining.RetrainingCycle, error) {
	return r.cycles.GetByID(ctx, id)
}

func (r *Retrainer) alert(ctx context.Context, cycle *aitraining.RetrainingCycle) {
	if r.alerter == nil || cycle == nil {
		return
	}
	if cycle.Status == aitraining.RetrainingStatusSkipped && !r.cfg.Alerts.IncludeSkipped {
		return
	}
	if err := r.alerter.AlertRetraining(context.WithoutCancel(ctx), cycle); err != nil {
		r.l.Warn("failed to send retraining alert",
			zap.String("cycleId", cycle.ID.String()),
			zap.String("status", cycle.Status.String()),
			zap.Error(err),
		)
	}
}

func (r *Retrainer) leaseSeconds() int64 {
	return int64(r.LeaseDuration() / time.Second)
}

func trainerName(raw string) (string, error) {
	trainer := strings.TrimSpace(raw)
	if trainer == "" || len([]rune(trainer)) > maxTrainerRunes {
		return "", errortypes.NewValidationError(
			"trainer",
			errortypes.ErrInvalid,
			fmt.Sprintf("The trainer must be named, in at most %d characters", maxTrainerRunes),
		)
	}

	return trainer, nil
}
