package ailogjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	AILogRepository repositories.AILogRepository
}

type Activities struct {
	ailogRepository repositories.AILogRepository
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		ailogRepository: p.AILogRepository,
	}
}

func (a *Activities) InsertAILogActivity(
	ctx context.Context,
	payload *InsertAILogPayload,
) (*InsertAILogResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info(
		"Starting insert ai log activity",
		"log", payload.Log,
	)

	err := a.ailogRepository.Insert(ctx, payload.Log)
	if err != nil {
		logger.Error("Failed to insert ai log", "error", err, "log", payload.Log)

		appErr := temporaltype.ClassifyError(err)
		if appErr.Type == temporaltype.ErrorTypeRetryable {
			return nil, temporaltype.NewRetryableError(
				"Failed to insert ai log",
				err,
			).ToTemporalError()
		}

		return nil, appErr.ToTemporalError()
	}

	activity.RecordHeartbeat(ctx, "Inserted ai log")

	logger.Info("Inserted ai log", "log", payload.Log)

	return &InsertAILogResult{
		ID: payload.Log.ID,
	}, nil
}
