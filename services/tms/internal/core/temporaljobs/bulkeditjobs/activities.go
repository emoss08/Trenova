package bulkeditjobs

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/pkg/errortypes"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	Runner Runner
}

type Activities struct {
	runner Runner
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{runner: p.Runner}
}

func (a *Activities) PrepareEditActivity(
	ctx context.Context,
	payload *EditPayload,
) (*PreparedEdit, error) {
	entity, err := a.runner.Prepare(ctx, tenantOf(payload.BasePayload), payload.EditID)
	if err != nil {
		return nil, classify(err)
	}
	return &PreparedEdit{TotalCount: len(entity.Targets)}, nil
}

func (a *Activities) ProcessBatchActivity(
	ctx context.Context,
	payload *BatchPayload,
) (int, error) {
	activity.RecordHeartbeat(ctx, payload.EditID.String())
	processed, err := a.runner.ProcessBatch(
		ctx,
		tenantOf(payload.BasePayload),
		payload.EditID,
		payload.Limit,
	)
	if err != nil {
		return 0, classify(err)
	}
	return processed, nil
}

func (a *Activities) FinalizeEditActivity(
	ctx context.Context,
	payload *FinalizePayload,
) (*EditResult, error) {
	entity, err := a.runner.Finalize(
		ctx,
		tenantOf(payload.BasePayload),
		payload.EditID,
		payload.Failure,
	)
	if err != nil {
		return nil, err
	}
	return &EditResult{
		Status:         entity.Status,
		ChangedCount:   entity.ChangedCount,
		FailedCount:    entity.FailedCount,
		ProcessedCount: entity.ProcessedCount,
	}, nil
}

func classify(err error) error {
	var validation *errortypes.Error
	var multi *errortypes.MultiError
	var business *errortypes.BusinessError
	if errors.As(err, &validation) || errors.As(err, &multi) || errors.As(err, &business) {
		return temporal.NewNonRetryableApplicationError(err.Error(), ErrTypeNotRunnable, err)
	}
	return err
}
