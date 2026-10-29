//nolint:gocognit // existing legacy workflow/API shape is intentionally kept stable
package invoiceaction

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/zap"
)

type RecorderParams struct {
	Logger       *zap.Logger
	AuditService auditports.AuditService
	Realtime     realtimeports.RealtimeService
}

// Recorder records what happens to an invoice: the audit entry and the
// realtime invalidations that refresh it, and its billing queue item, on
// screen.
type Recorder struct {
	l            *zap.Logger
	auditService auditports.AuditService
	realtime     realtimeports.RealtimeService
}

func NewRecorder(p RecorderParams) *Recorder {
	return &Recorder{l: p.Logger, auditService: p.AuditService, realtime: p.Realtime}
}
