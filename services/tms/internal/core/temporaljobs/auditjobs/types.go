package auditjobs

import (
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

type ProcessAuditBatchPayload struct {
	temporaltype.BasePayload
	Entries []*audit.Entry `json:"entries"`
	BatchID pulid.ID       `json:"batchId"`
}

type ProcessAuditBatchResult struct {
	ProcessedCount int            `json:"processedCount"`
	FailedCount    int            `json:"failedCount"`
	BatchID        pulid.ID       `json:"batchId"`
	ProcessedAt    int64          `json:"processedAt"`
	Errors         []string       `json:"errors,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type AuditBufferStatus struct {
	BufferedEntries int   `json:"bufferedEntries"`
	DLQEntries      int   `json:"dlqEntries"`
	LastFlush       int64 `json:"lastFlush"`
}

type DeleteAuditEntriesResult struct {
	TotalDeleted int    `json:"totalDeleted,omitempty"`
	Result       string `json:"result,omitempty"`
}

type FlushFromRedisResult struct {
	Batches    [][]*audit.Entry `json:"batches"`
	EntryCount int              `json:"entryCount"`
}

type MoveToDLQPayload struct {
	Entries      []*audit.Entry `json:"entries"`
	ErrorMessage string         `json:"errorMessage"`
}

type DLQRetryResult struct {
	RetryCount     int        `json:"retryCount"`
	SuccessCount   int        `json:"successCount"`
	FailedCount    int        `json:"failedCount"`
	ExhaustedCount int        `json:"exhaustedCount"`
	RecoveredIDs   []pulid.ID `json:"recoveredIds,omitempty"`
	FailedIDs      []pulid.ID `json:"failedIds,omitempty"`
}
