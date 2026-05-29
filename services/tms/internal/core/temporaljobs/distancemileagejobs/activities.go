package distancemileagejobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultStoredMileageBatchSize  = 250
	defaultStoredMileageTotalLimit = 2500
)

type ActivitiesParams struct {
	fx.In

	StoredMileageRepo   repositories.StoredMileageRepository
	StoredMileageBuffer repositories.StoredMileageBufferRepository
	Logger              *zap.Logger
}

type Activities struct {
	storedMileageRepo   repositories.StoredMileageRepository
	storedMileageBuffer repositories.StoredMileageBufferRepository
	l                   *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		storedMileageRepo:   p.StoredMileageRepo,
		storedMileageBuffer: p.StoredMileageBuffer,
		l:                   p.Logger.Named("temporal.distance-mileage.activities"),
	}
}

func (a *Activities) FlushStoredMileageBufferActivity(
	ctx context.Context,
) (*FlushStoredMileageBufferResult, error) {
	batches, err := a.storedMileageBuffer.PopTenantBatches(
		ctx,
		defaultStoredMileageBatchSize,
		defaultStoredMileageTotalLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("pop stored mileage buffer: %w", err)
	}
	count := 0
	for _, batch := range batches {
		count += len(batch)
	}
	return &FlushStoredMileageBufferResult{RecordCount: count, Batches: batches}, nil
}

func (a *Activities) UpsertStoredMileageBatchActivity(
	ctx context.Context,
	payload *UpsertStoredMileageBatchPayload,
) (*UpsertStoredMileageBatchResult, error) {
	if payload == nil || len(payload.Records) == 0 {
		return &UpsertStoredMileageBatchResult{}, nil
	}
	if err := a.storedMileageRepo.BulkUpsert(ctx, payload.Records); err != nil {
		return nil, fmt.Errorf("bulk upsert stored mileage: %w", err)
	}
	a.l.Info("stored mileage batch upserted", zap.Int("count", len(payload.Records)))
	return &UpsertStoredMileageBatchResult{ProcessedCount: len(payload.Records)}, nil
}
