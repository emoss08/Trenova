package aitrainingservice

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (f *exportStore) List(
	_ context.Context,
	req repositories.ListAITrainingExportsRequest,
) ([]*aitraining.TrainingExport, error) {
	items := make([]*aitraining.TrainingExport, 0, len(f.items))
	for _, item := range f.items {
		clone := *item
		items = append(items, &clone)
	}
	slices.SortFunc(items, func(a, b *aitraining.TrainingExport) int {
		if a.CreatedAt != b.CreatedAt {
			return int(b.CreatedAt - a.CreatedAt)
		}
		return strings.Compare(b.ID.String(), a.ID.String())
	})
	if req.Limit > 0 && len(items) > req.Limit {
		items = items[:req.Limit]
	}
	return items, nil
}

type cycleStore struct {
	items   []*aitraining.RetrainingCycle
	created int
}

func (f *cycleStore) Create(
	_ context.Context,
	entity *aitraining.RetrainingCycle,
) (*aitraining.RetrainingCycle, error) {
	if entity.Status.IsOpen() {
		for _, item := range f.items {
			if item.Status.IsOpen() {
				return nil, errortypes.NewBusinessError("A retraining cycle is already open")
			}
		}
	}
	f.created++
	entity.ID = pulid.MustNew("airc_")
	if entity.CreatedAt == 0 {
		entity.CreatedAt = testNow
	}
	stored := *entity
	f.items = append(f.items, &stored)
	return entity, nil
}

func (f *cycleStore) add(entity *aitraining.RetrainingCycle) *aitraining.RetrainingCycle {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("airc_")
	}
	stored := *entity
	f.items = append(f.items, &stored)
	return entity
}

func (f *cycleStore) find(id pulid.ID) *aitraining.RetrainingCycle {
	for _, item := range f.items {
		if item.ID == id {
			return item
		}
	}
	return nil
}

func (f *cycleStore) GetByID(_ context.Context, id pulid.ID) (*aitraining.RetrainingCycle, error) {
	item := f.find(id)
	if item == nil {
		return nil, errortypes.NewNotFoundError("Retraining cycle not found")
	}
	clone := *item
	return &clone, nil
}

func (f *cycleStore) List(
	_ context.Context,
	req repositories.ListRetrainingCyclesRequest,
) ([]*aitraining.RetrainingCycle, error) {
	out := make([]*aitraining.RetrainingCycle, 0, len(f.items))
	for i := len(f.items) - 1; i >= 0; i-- {
		item := f.items[i]
		if len(req.Statuses) > 0 && !slices.Contains(req.Statuses, item.Status) {
			continue
		}
		clone := *item
		out = append(out, &clone)
		if req.Limit > 0 && len(out) == req.Limit {
			break
		}
	}
	return out, nil
}

func (f *cycleStore) Update(
	_ context.Context,
	entity *aitraining.RetrainingCycle,
) (*aitraining.RetrainingCycle, error) {
	current := f.find(entity.ID)
	if current == nil {
		return nil, errortypes.NewNotFoundError("Retraining cycle not found")
	}
	if current.Version != entity.Version {
		return nil, dberror.CreateVersionMismatchError("RetrainingCycle", entity.ID.String())
	}
	entity.Version++
	*current = *entity
	clone := *entity
	return &clone, nil
}

type trainableStore struct {
	repositories.AICorrectionRepository
	trainable   int
	weekly      []aicorrection.WeekTotal
	countCalls  []repositories.CountTrainableAICorrectionsRequest
	weeklyCalls int
}

func (f *trainableStore) CountTrainable(
	_ context.Context,
	req *repositories.CountTrainableAICorrectionsRequest,
) (int, error) {
	f.countCalls = append(f.countCalls, *req)
	return f.trainable, nil
}

func (f *trainableStore) WeeklyTrainableTotalsByProvider(
	_ context.Context,
	_ *repositories.WeeklyTrainableAICorrectionTotalsRequest,
) ([]aicorrection.WeekTotal, error) {
	f.weeklyCalls++
	return f.weekly, nil
}

type fakeOperator struct {
	services.AITrainingExportOperator
	exports  *exportStore
	started  []services.StartAITrainingExportRequest
	canceled []pulid.ID
	startErr error
}

func (f *fakeOperator) Start(
	ctx context.Context,
	req *services.StartAITrainingExportRequest,
) (*aitraining.TrainingExport, error) {
	f.started = append(f.started, *req)
	if f.startErr != nil {
		return nil, f.startErr
	}
	return f.exports.Create(ctx, &aitraining.TrainingExport{
		Status:       aitraining.ExportStatusQueued,
		CapturedFrom: req.CapturedFrom,
		CapturedTo:   req.CapturedTo,
	})
}

func (f *fakeOperator) Cancel(ctx context.Context, id pulid.ID) (*aitraining.TrainingExport, error) {
	f.canceled = append(f.canceled, id)
	export, err := f.exports.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !export.Status.IsActive() {
		return nil, errortypes.NewBusinessError("Only a queued or running export can be canceled")
	}
	export.Status = aitraining.ExportStatusCanceled
	return f.exports.Update(ctx, export)
}

var errStartRefused = errors.New("temporal unavailable")

type retrainingFixture struct {
	service     *Retrainer
	cycles      *cycleStore
	exports     *exportStore
	corrections *trainableStore
	operator    *fakeOperator
	cfg         *config.Config
	now         int64
}

func newRetrainingFixture() *retrainingFixture {
	f := &retrainingFixture{
		cycles:      &cycleStore{},
		exports:     newExportStore(),
		corrections: &trainableStore{},
		cfg:         &config.Config{AIRetraining: config.AIRetrainingConfig{Enabled: true}},
		now:         testNow,
	}
	f.operator = &fakeOperator{exports: f.exports}
	f.service = NewRetrainer(RetrainerParams{
		Logger:      zap.NewNop(),
		Config:      f.cfg,
		Cycles:      f.cycles,
		Exports:     f.exports,
		Corrections: f.corrections,
		Operator:    f.operator,
	})
	f.service.now = func() int64 { return f.now }
	return f
}
