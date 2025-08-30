package auditjobs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/temporaltype"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	AuditRepository repositories.AuditRepository
	Logger          *logger.Logger
}

type Activities struct {
	ar     repositories.AuditRepository
	logger *logger.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		ar:     p.AuditRepository,
		logger: p.Logger,
	}
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
			"duration":           duration.String(),
			"avgProcessingTime":  duration.Milliseconds() / int64(len(payload.Entries)),
			"organizationId":     payload.OrganizationID.String(),
			"businessUnitId":     payload.BusinessUnitID.String(),
		},
	}, nil
}

func (a *Activities) FlushAuditBufferActivity(
	ctx context.Context,
) ([]*audit.Entry, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Flushing audit buffer")

	entries := GetBufferedEntries()
	if len(entries) == 0 {
		logger.Info("No entries in buffer to flush")
		return nil, nil
	}

	ClearBuffer()

	logger.Info(
		"Flushed audit buffer",
		"entryCount", len(entries),
	)

	return entries, nil
}

func (a *Activities) GetAuditBufferStatusActivity(
	ctx context.Context,
) (*AuditBufferStatus, error) {
	status := GetBufferStatus()
	activity.GetLogger(ctx).Info(
		"Retrieved audit buffer status",
		"bufferedEntries", status.BufferedEntries,
	)
	return status, nil
}

var (
	auditBuffer       []*audit.Entry
	bufferMutex       = &sync.RWMutex{}
	lastFlush         time.Time
	overflowCount     int
	maxBufferSize     = 10000
)

func AddToBuffer(entry *audit.Entry) error {
	bufferMutex.Lock()
	defer bufferMutex.Unlock()

	if len(auditBuffer) >= maxBufferSize {
		overflowCount++
		return fmt.Errorf("audit buffer full: max size %d reached", maxBufferSize)
	}

	auditBuffer = append(auditBuffer, entry)
	return nil
}

func GetBufferedEntries() []*audit.Entry {
	bufferMutex.Lock()
	defer bufferMutex.Unlock()

	if len(auditBuffer) == 0 {
		return nil
	}

	entries := make([]*audit.Entry, len(auditBuffer))
	copy(entries, auditBuffer)
	return entries
}

func ClearBuffer() {
	bufferMutex.Lock()
	defer bufferMutex.Unlock()

	auditBuffer = make([]*audit.Entry, 0)
	lastFlush = time.Now()
}

func GetBufferStatus() *AuditBufferStatus {
	bufferMutex.RLock()
	defer bufferMutex.RUnlock()

	return &AuditBufferStatus{
		BufferedEntries: len(auditBuffer),
		LastFlush:       lastFlush.Unix(),
		OverflowCount:   overflowCount,
	}
}

func SetMaxBufferSize(size int) {
	bufferMutex.Lock()
	defer bufferMutex.Unlock()
	maxBufferSize = size
}