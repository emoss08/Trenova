package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
)

// AgentRunEventWriter records one run's account of itself.
//
// Record never returns an error and never blocks on the database in the common
// case. That is deliberate: the caller is a run in progress, it has nothing
// useful to do about a failed write, and an agent that finishes its work and
// then dies filing the paperwork is worse than one that files none.
type AgentRunEventWriter interface {
	Record(ctx context.Context, event StreamEvent)
	Flush(ctx context.Context)
}

// AgentRunEventRecorder opens a writer for a run or a conversation turn.
type AgentRunEventRecorder interface {
	Recorder(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		owner RunStepOwner,
	) AgentRunEventWriter
}

// RecordTrajectory is nil-safe, because the recorder is an optional dependency
// and a caller should not have to check before saying what it did.
func RecordTrajectory(writer AgentRunEventWriter, ctx context.Context, event StreamEvent) {
	if writer == nil {
		return
	}

	writer.Record(ctx, event)
}

// FlushTrajectory is nil-safe for the same reason.
func FlushTrajectory(writer AgentRunEventWriter, ctx context.Context) {
	if writer == nil {
		return
	}

	writer.Flush(ctx)
}
