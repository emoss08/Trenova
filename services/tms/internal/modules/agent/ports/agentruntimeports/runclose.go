package agentruntimeports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// RunCloser ends a run that waited on decisions. A run's workflow ends once
// its proposals are filed; the run is completed by the decision that leaves
// nothing pending, or by the expiry sweep once the rest expired.
type RunCloser interface {
	// CloseIfDecided completes the run when none of its proposals is still
	// pending, and has it reflected on.
	CloseIfDecided(ctx context.Context, tenantInfo pagination.TenantInfo, runID pulid.ID) error
	// CloseSettled completes the runs whose proposals all expired.
	CloseSettled(ctx context.Context, now int64) (int, error)
	// StopWaiting fails a run that waits on decisions and expires what it
	// left pending. It reports false when the run was not waiting.
	StopWaiting(ctx context.Context, tenantInfo pagination.TenantInfo, runID pulid.ID) (bool, error)
}
