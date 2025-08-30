/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package auditjobs

import (
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/pkg/types/temporaltype"
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
	LastFlush       int64 `json:"lastFlush"`
	OverflowCount   int   `json:"overflowCount"`
}
