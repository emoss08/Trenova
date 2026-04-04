package auditjobs

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
)

const (
	defaultBatchSize   = 500
	defaultMaxEntries  = 5000
	defaultDLQMaxRetry = 5
)

type ActivitiesParams struct {
	fx.In

	AuditRepository         repositories.AuditRepository
	AuditBufferRepository   repositories.AuditBufferRepository
	AuditDLQRepository      repositories.AuditDLQRepository
	DataRetentionRepository repositories.DataRetentionRepository
	RealtimeService         services.RealtimeService
	Metrics                 *metrics.Registry
}

type Activities struct {
	ar      repositories.AuditRepository
	abr     repositories.AuditBufferRepository
	adlq    repositories.AuditDLQRepository
	dr      repositories.DataRetentionRepository
	rt      services.RealtimeService
	metrics *metrics.Registry
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		ar:      p.AuditRepository,
		abr:     p.AuditBufferRepository,
		adlq:    p.AuditDLQRepository,
		dr:      p.DataRetentionRepository,
		rt:      p.RealtimeService,
		metrics: p.Metrics,
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
			ProcessedAt:    timeutils.NowUnix(),
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
			ProcessedAt:    timeutils.NowUnix(),
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

	a.publishRealtimeInvalidation(ctx, logger, payload.Entries)

	return &ProcessAuditBatchResult{
		ProcessedCount: len(payload.Entries),
		FailedCount:    0,
		BatchID:        payload.BatchID,
		ProcessedAt:    timeutils.NowUnix(),
		Metadata: map[string]any{
			"duration":          duration.String(),
			"avgProcessingTime": duration.Milliseconds() / int64(len(payload.Entries)),
			"organizationId":    payload.OrganizationID.String(),
			"businessUnitId":    payload.BusinessUnitID.String(),
		},
	}, nil
}

func (a *Activities) publishRealtimeInvalidation(
	ctx context.Context,
	logger interface {
		Warn(string, ...any)
	},
	entries []*audit.Entry,
) {
	if a.rt == nil || len(entries) == 0 {
		return
	}

	type tenantBatch struct {
		orgID         pulid.ID
		buID          pulid.ID
		actorUserID   pulid.ID
		actorID       pulid.ID
		actorType     services.PrincipalType
		actorAPIKeyID pulid.ID
		record        pulid.ID
		count         int
	}

	tenantBatches := make(map[string]*tenantBatch, len(entries))
	for _, entry := range entries {
		if entry == nil || entry.OrganizationID.IsNil() || entry.BusinessUnitID.IsNil() {
			continue
		}

		key := entry.RealtimeBatchKey()
		batch, ok := tenantBatches[key]
		if !ok {
			batch = &tenantBatch{
				orgID:         entry.OrganizationID,
				buID:          entry.BusinessUnitID,
				actorUserID:   entry.UserID,
				actorID:       entry.PrincipalID,
				actorType:     services.PrincipalType(entry.PrincipalType),
				actorAPIKeyID: entry.APIKeyID,
				record:        entry.ID,
			}
			tenantBatches[key] = batch
		}

		batch.count++
	}

	for _, batch := range tenantBatches {
		action := "created"
		recordID := batch.record
		if batch.count > 1 {
			action = "bulk_created"
			recordID = pulid.ID("")
		}

		if err := realtimeinvalidation.Publish(ctx, a.rt, &realtimeinvalidation.PublishParams{
			OrganizationID: batch.orgID,
			BusinessUnitID: batch.buID,
			ActorUserID:    batch.actorUserID,
			ActorType:      batch.actorType,
			ActorID:        batch.actorID,
			ActorAPIKeyID:  batch.actorAPIKeyID,
			Resource:       "audit-logs",
			Action:         action,
			RecordID:       recordID,
		}); err != nil {
			logger.Warn(
				"Failed to publish audit realtime invalidation",
				"error", err,
				"organizationID", batch.orgID.String(),
				"businessUnitID", batch.buID.String(),
				"action", action,
			)
		}
	}
}

func (a *Activities) FlushFromRedisActivity(
	ctx context.Context,
) (*FlushFromRedisResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Flushing audit buffer from Redis")

	startTime := time.Now()
	result := &FlushFromRedisResult{
		Batches: make([][]*audit.Entry, 0),
	}

	totalFetched := 0
	for totalFetched < defaultMaxEntries {
		entries, err := a.abr.Pop(ctx, defaultBatchSize)
		if err != nil {
			logger.Error("Failed to pop from Redis buffer", "error", err)
			break
		}

		if len(entries) == 0 {
			break
		}

		result.Batches = append(result.Batches, entries)
		totalFetched += len(entries)
		result.EntryCount += len(entries)

		activity.RecordHeartbeat(ctx, fmt.Sprintf("Fetched %d entries", totalFetched))
	}

	duration := time.Since(startTime)

	if a.metrics != nil {
		a.metrics.Audit.RecordBufferFlush(true, duration.Seconds(), result.EntryCount)
	}

	if result.EntryCount == 0 {
		logger.Info("No entries in Redis buffer to flush")
		return result, nil
	}

	logger.Info(
		"Flushed audit buffer from Redis",
		"entryCount", result.EntryCount,
		"batchCount", len(result.Batches),
		"duration", duration.String(),
	)

	return result, nil
}

func (a *Activities) MoveToDLQActivity(
	ctx context.Context,
	payload *MoveToDLQPayload,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Moving failed entries to DLQ",
		"entryCount", len(payload.Entries),
		"errorMessage", payload.ErrorMessage,
	)

	now := timeutils.NowUnix()
	dlqEntries := make([]*audit.DLQEntry, 0, len(payload.Entries))

	for _, entry := range payload.Entries {
		entryData, err := entryToMap(entry)
		if err != nil {
			logger.Error("Failed to convert entry to map",
				"entryID", entry.ID.String(),
				"error", err,
			)
			continue
		}

		dlqEntry := &audit.DLQEntry{
			ID:              pulid.MustNew("dlq_"),
			OriginalEntryID: entry.ID,
			EntryData:       entryData,
			FailureTime:     now,
			LastError:       payload.ErrorMessage,
			Status:          audit.DLQStatusPending,
			OrganizationID:  entry.OrganizationID,
			BusinessUnitID:  entry.BusinessUnitID,
			NextRetryAt:     now + 60,
		}

		dlqEntries = append(dlqEntries, dlqEntry)
	}

	if err := a.adlq.InsertBatch(ctx, dlqEntries); err != nil {
		logger.Error("Failed to insert DLQ entries", "error", err)
		return temporaltype.NewRetryableError("Failed to insert DLQ entries", err).ToTemporalError()
	}

	if a.metrics != nil {
		a.metrics.Audit.RecordDLQPush(len(dlqEntries))
	}

	logger.Info("Moved entries to DLQ", "count", len(dlqEntries))
	return nil
}

func (a *Activities) RetryDLQEntriesActivity(
	ctx context.Context,
	limit int,
) (*DLQRetryResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting DLQ retry activity", "limit", limit)

	entries, err := a.adlq.GetPendingEntries(ctx, limit)
	if err != nil {
		logger.Error("Failed to get pending DLQ entries", "error", err)
		return nil, temporaltype.NewRetryableError(
			"Failed to get pending DLQ entries",
			err,
		).ToTemporalError()
	}

	if len(entries) == 0 {
		logger.Info("No pending DLQ entries to retry")
		return &DLQRetryResult{}, nil
	}

	result := &DLQRetryResult{
		RetryCount:   len(entries),
		RecoveredIDs: make([]pulid.ID, 0),
		FailedIDs:    make([]pulid.ID, 0),
	}

	for _, dlqEntry := range entries {
		activity.RecordHeartbeat(ctx, fmt.Sprintf("Retrying DLQ entry %s", dlqEntry.ID.String()))

		if dlqEntry.RetryCount >= defaultDLQMaxRetry {
			logger.Warn("DLQ entry exhausted max retries",
				"entryID", dlqEntry.ID.String(),
				"retryCount", dlqEntry.RetryCount,
			)
			_ = a.adlq.MarkAsFailed(ctx, dlqEntry.ID, "Max retries exhausted")
			result.ExhaustedCount++
			continue
		}

		auditEntry, convErr := mapToEntry(dlqEntry.EntryData)
		if convErr != nil {
			logger.Error("Failed to convert DLQ entry data",
				"entryID", dlqEntry.ID.String(),
				"error", convErr,
			)
			_ = a.adlq.MarkAsFailed(ctx, dlqEntry.ID, "Invalid entry data: "+convErr.Error())
			result.FailedCount++
			result.FailedIDs = append(result.FailedIDs, dlqEntry.ID)
			continue
		}

		if insertErr := a.ar.InsertAuditEntries(ctx, []*audit.Entry{auditEntry}); insertErr != nil {
			logger.Error("Failed to retry DLQ entry",
				"entryID", dlqEntry.ID.String(),
				"error", insertErr,
			)
			dlqEntry.RetryCount++
			dlqEntry.LastError = insertErr.Error()
			dlqEntry.NextRetryAt = timeutils.NowUnix() + int64(60*(1<<dlqEntry.RetryCount))
			_ = a.adlq.Update(ctx, dlqEntry)
			result.FailedCount++
			result.FailedIDs = append(result.FailedIDs, dlqEntry.ID)
			if a.metrics != nil {
				a.metrics.Audit.RecordDLQRetry(false)
			}
			continue
		}

		result.SuccessCount++
		result.RecoveredIDs = append(result.RecoveredIDs, dlqEntry.ID)
		if a.metrics != nil {
			a.metrics.Audit.RecordDLQRetry(true)
		}
	}

	if len(result.RecoveredIDs) > 0 {
		if mrErr := a.adlq.MarkAsRecovered(ctx, result.RecoveredIDs); mrErr != nil {
			logger.Error("Failed to mark DLQ entries as recovered", "error", mrErr)
		}
	}

	logger.Info("DLQ retry completed",
		"retryCount", result.RetryCount,
		"successCount", result.SuccessCount,
		"failedCount", result.FailedCount,
		"exhaustedCount", result.ExhaustedCount,
	)

	return result, nil
}

func (a *Activities) GetBufferStatusActivity(ctx context.Context) (*AuditBufferStatus, error) {
	logger := activity.GetLogger(ctx)

	size, err := a.abr.Size(ctx)
	if err != nil {
		logger.Error("Failed to get buffer size", "error", err)
		return nil, err
	}

	dlqSize, err := a.adlq.Count(ctx)
	if err != nil {
		logger.Error("Failed to get DLQ size", "error", err)
		dlqSize = 0
	}

	if a.metrics != nil {
		a.metrics.Audit.SetBufferSize(size)
		a.metrics.Audit.SetDLQSize(dlqSize)
	}

	status := &AuditBufferStatus{
		BufferedEntries: int(size),
		DLQEntries:      int(dlqSize),
		LastFlush:       timeutils.NowUnix(),
	}

	logger.Info("Retrieved audit buffer status",
		"bufferedEntries", status.BufferedEntries,
		"dlqEntries", status.DLQEntries,
	)

	return status, nil
}

func entryToMap(entry *audit.Entry) (map[string]any, error) {
	data, err := sonic.Marshal(entry)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal entry: %w", err)
	}

	var result map[string]any
	if err = sonic.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to map: %w", err)
	}

	return result, nil
}

func mapToEntry(data map[string]any) (*audit.Entry, error) {
	jsonData, err := sonic.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal map: %w", err)
	}

	var entry audit.Entry
	if err = sonic.Unmarshal(jsonData, &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to entry: %w", err)
	}

	return &entry, nil
}
