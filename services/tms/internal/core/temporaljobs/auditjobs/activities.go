package auditjobs

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	AuditRepository         repositories.AuditRepository
	DataRetentionRepository repositories.DataRetentionRepository
}

type Activities struct {
	ar     repositories.AuditRepository
	dr     repositories.DataRetentionRepository
	buffer *Buffer
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		ar:     p.AuditRepository,
		dr:     p.DataRetentionRepository,
		buffer: NewBuffer(10000),
	}
}

func (a *Activities) DeleteAuditEntriesActivity(
	ctx context.Context,
) (*DeleteAuditEntriesResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting audit entries deletion activity")

	activity.RecordHeartbeat(ctx, "fetching data retention entities")
	entities, err := a.dr.List(ctx)
	if err != nil {
		logger.Error("Failed to get data retention entities", "error", err)
		return nil, temporaltype.NewRetryableError(
			"Failed to fetch data retention configuration",
			err,
		).ToTemporalError()
	}

	if entities.Total == 0 {
		logger.Info("No data retention entities found, skipping deletion")
		return &DeleteAuditEntriesResult{
			TotalDeleted: 0,
			Result:       "No data retention entities configured",
		}, nil
	}

	totalDeleted := 0
	deletedOrgIDs := make([]pulid.ID, 0, entities.Total)
	failedOrgIDs := make([]pulid.ID, 0)

	for _, entity := range entities.Items {
		if entity.AuditRetentionPeriod <= 0 {
			logger.Warn("Invalid retention period for organization",
				"orgID", entity.OrganizationID,
				"retentionPeriod", entity.AuditRetentionPeriod,
			)
			continue
		}

		timestamp := time.Now().AddDate(0, 0, -entity.AuditRetentionPeriod).Unix()

		activity.RecordHeartbeat(
			ctx,
			fmt.Sprintf("deleting audit entries for org %s", entity.OrganizationID),
		)

		deletedRows, drErr := a.ar.DeleteAuditEntries(ctx, timestamp)
		if drErr != nil {
			logger.Error("Failed to delete audit entries for organization",
				"error", drErr,
				"orgID", entity.OrganizationID,
				"retentionDays", entity.AuditRetentionPeriod,
			)

			failedOrgIDs = append(failedOrgIDs, entity.OrganizationID)
			continue
		}

		deletedOrgIDs = append(deletedOrgIDs, entity.OrganizationID)
		totalDeleted += int(deletedRows)

		logger.Info("Deleted audit entries for organization",
			"orgID", entity.OrganizationID,
			"deletedCount", deletedRows,
			"retentionDays", entity.AuditRetentionPeriod,
		)
	}

	if len(failedOrgIDs) > 0 && len(deletedOrgIDs) == 0 {
		return nil, temporaltype.NewRetryableError(
			fmt.Sprintf("Failed to delete audit entries for all organizations: %v", failedOrgIDs),
			nil,
		).ToTemporalError()
	}

	result := fmt.Sprintf(
		"Deleted %d audit entries for %d organizations. Successful: %v",
		totalDeleted,
		len(deletedOrgIDs),
		deletedOrgIDs,
	)

	if len(failedOrgIDs) > 0 {
		result += fmt.Sprintf(". Failed: %v", failedOrgIDs)
	}

	logger.Info("Audit deletion completed",
		"totalDeleted", totalDeleted,
		"successfulOrgs", len(deletedOrgIDs),
		"failedOrgs", len(failedOrgIDs),
	)

	return &DeleteAuditEntriesResult{
		TotalDeleted: totalDeleted,
		Result:       result,
	}, nil
}

func (a *Activities) ProcessAuditBatchActivity(
	ctx context.Context,
	payload *ProcessAuditBatchPayload,
) (*ProcessAuditBatchResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info(
		"Starting audit batch processing",
		"batchId", payload.BatchID.String(),
		"entryCount", len(payload.Entries),
	)

	if len(payload.Entries) == 0 {
		return &ProcessAuditBatchResult{
			ProcessedCount: 0,
			FailedCount:    0,
			BatchID:        payload.BatchID,
			ProcessedAt:    time.Now().Unix(),
			Metadata: map[string]any{
				"message": "No entries to process",
			},
		}, nil
	}

	activity.RecordHeartbeat(ctx, fmt.Sprintf("Processing %d audit entries", len(payload.Entries)))

	startTime := time.Now()
	err := a.ar.InsertAuditEntries(ctx, payload.Entries)
	duration := time.Since(startTime)

	if err != nil {
		logger.Error(
			"Failed to insert audit entries",
			"error", err,
			"batchId", payload.BatchID.String(),
			"duration", duration.String(),
		)

		appErr := temporaltype.ClassifyError(err)
		if appErr.Type == temporaltype.ErrorTypeRetryable {
			return nil, temporaltype.NewRetryableError(
				"Failed to insert audit entries",
				err,
			).ToTemporalError()
		}

		return &ProcessAuditBatchResult{
			ProcessedCount: 0,
			FailedCount:    len(payload.Entries),
			BatchID:        payload.BatchID,
			ProcessedAt:    time.Now().Unix(),
			Errors:         []string{err.Error()},
			Metadata: map[string]any{
				"duration": duration.String(),
				"error":    err.Error(),
			},
		}, err
	}

	activity.RecordHeartbeat(ctx, "Audit entries successfully inserted")

	logger.Info(
		"Successfully processed audit batch",
		"batchId", payload.BatchID.String(),
		"processedCount", len(payload.Entries),
		"duration", duration.String(),
	)

	return &ProcessAuditBatchResult{
		ProcessedCount: len(payload.Entries),
		FailedCount:    0,
		BatchID:        payload.BatchID,
		ProcessedAt:    time.Now().Unix(),
		Metadata: map[string]any{
			"duration":          duration.String(),
			"avgProcessingTime": duration.Milliseconds() / int64(len(payload.Entries)),
			"organizationId":    payload.OrganizationID.String(),
			"businessUnitId":    payload.BusinessUnitID.String(),
		},
	}, nil
}

func (a *Activities) FlushAuditBufferActivity(
	ctx context.Context,
) ([]*audit.Entry, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Flushing audit buffer")

	entries := a.buffer.FlushAndReset()
	if len(entries) == 0 {
		logger.Info("No entries in buffer to flush")
		return nil, nil
	}

	logger.Info(
		"Flushed audit buffer",
		"entryCount", len(entries),
	)

	return entries, nil
}

func (a *Activities) GetAuditBufferStatusActivity(
	ctx context.Context,
) (*AuditBufferStatus, error) {
	size := a.buffer.Size()
	state := a.buffer.GetState()

	status := &AuditBufferStatus{
		BufferedEntries: size,
		LastFlush:       time.Now().Unix(),
		OverflowCount:   0,
	}

	activity.GetLogger(ctx).Info(
		"Retrieved audit buffer status",
		"bufferedEntries", status.BufferedEntries,
		"circuitState", state,
	)
	return status, nil
}

func (a *Activities) AddToBuffer(entry *audit.Entry) error {
	if !a.buffer.Add(entry) {
		if a.buffer.GetState() == CircuitOpen {
			return ErrBufferCircuitBreakerOpen
		}
		return ErrBufferFull
	}
	return nil
}
