package aitrainingservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	_ services.AITrainingExportOperator = (*Operator)(nil)

	errStarterUnavailable = errors.New("no training export starter is configured")
)

type OperatorParams struct {
	fx.In

	Logger  *zap.Logger
	Exports repositories.AITrainingExportRepository
	Records repositories.AITrainingRecordRepository
	Starter services.AITrainingExportStarter `optional:"true"`
}

type Operator struct {
	l       *zap.Logger
	exports repositories.AITrainingExportRepository
	records repositories.AITrainingRecordRepository
	starter services.AITrainingExportStarter
	now     func() int64
}

func NewOperator(p OperatorParams) *Operator {
	return &Operator{
		l:       p.Logger.Named("service.aitraining-operator"),
		exports: p.Exports,
		records: p.Records,
		starter: p.Starter,
		now:     timeutils.NowUnix,
	}
}

func AsOperator(o *Operator) services.AITrainingExportOperator { return o }

func (o *Operator) Start(
	ctx context.Context,
	req *services.StartAITrainingExportRequest,
) (*aitraining.TrainingExport, error) {
	if o.starter == nil {
		return nil, errortypes.NewBusinessError(
			"Training exports cannot run on this installation",
		).WithInternal(errStarterUnavailable)
	}
	if req == nil {
		return nil, errortypes.NewValidationError("request", errortypes.ErrRequired, "Request is required")
	}

	now := o.now()
	entity := &aitraining.TrainingExport{
		Task:               aicorrection.TaskShipmentDraftExtraction,
		Status:             aitraining.ExportStatusQueued,
		Format:             aitraining.ExampleFormat,
		CapturedFrom:       req.CapturedFrom,
		CapturedTo:         req.CapturedTo,
		MaxPerOrganization: req.MaxPerOrganization,
		ValidationPercent:  req.ValidationPercent,
		RequestedBy:        strings.TrimSpace(req.RequestedBy),
		Note:               strings.TrimSpace(req.Note),
	}
	if entity.MaxPerOrganization == 0 {
		entity.MaxPerOrganization = aitraining.DefaultMaxPerOrganization
	}
	if entity.CapturedTo == 0 {
		entity.CapturedTo = now
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if entity.CapturedTo > now {
		multiErr.Add("capturedTo", errortypes.ErrInvalid, "The window cannot end in the future")
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := o.exports.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	workflowID, err := o.starter.StartAITrainingExport(ctx, created.ID)
	if err != nil {
		o.l.Error("failed to start training export", zap.Error(err))
		created.Status = aitraining.ExportStatusFailed
		created.FailureMessage = "The export could not be started"
		finished := o.now()
		created.FinishedAt = &finished
		if _, updateErr := o.exports.Update(ctx, created); updateErr != nil {
			o.l.Error("failed to mark unstarted training export", zap.Error(updateErr))
		}

		return nil, fmt.Errorf("start training export: %w", err)
	}

	created.WorkflowID = workflowID
	updated, err := o.exports.Update(ctx, created)
	if err != nil {
		return nil, err
	}

	o.l.Info("training export started",
		zap.String("exportId", updated.ID.String()),
		zap.String("requestedBy", updated.RequestedBy),
		zap.Int64("capturedFrom", updated.CapturedFrom),
		zap.Int64("capturedTo", updated.CapturedTo),
	)

	return updated, nil
}

func (o *Operator) List(ctx context.Context, limit int) ([]*aitraining.TrainingExport, error) {
	return o.exports.List(ctx, repositories.ListAITrainingExportsRequest{Limit: limit})
}

func (o *Operator) Get(ctx context.Context, id pulid.ID) (*aitraining.TrainingExport, error) {
	return o.exports.GetByID(ctx, id)
}

func (o *Operator) Cancel(ctx context.Context, id pulid.ID) (*aitraining.TrainingExport, error) {
	entity, err := o.exports.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !entity.Status.IsActive() {
		return nil, errortypes.NewBusinessError(
			"Only a queued or running export can be canceled; this one is {0}",
			stringutils.HumanizeCamelCase(entity.Status.String()),
		)
	}

	entity.Status = aitraining.ExportStatusCanceled
	finished := o.now()
	entity.FinishedAt = &finished

	return o.exports.Update(ctx, entity)
}

func (o *Operator) WithdrawnExamples(
	ctx context.Context,
	req repositories.ListWithdrawnTrainingExamplesRequest,
) ([]repositories.WithdrawnTrainingExample, error) {
	if _, err := o.exports.GetByID(ctx, req.ExportID); err != nil {
		return nil, err
	}

	return o.records.ListWithdrawn(ctx, req)
}
