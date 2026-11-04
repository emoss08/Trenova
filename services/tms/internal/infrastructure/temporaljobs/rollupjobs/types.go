// Package rollupjobs keeps the analytics rollups current. Writes to a rollup's source table mark
// the tenant days they touch; a refresh every minute recomputes those days from the source, so a
// rollup trails its source by about a minute and never by a write.
package rollupjobs

import "github.com/emoss08/keel/analytics/rollup"

const (
	RefreshWorkflowName = "RefreshRollupsWorkflow"
	RefreshScheduleID   = "rollup-refresh"
)

// Rollups is every rollup the refresh keeps, supplied once by the worker's bootstrap from the
// same list the analytics CLI generates migrations from.
type Rollups []*rollup.Rollup

// RefreshResult is what one refresh run did across every rollup.
type RefreshResult struct {
	Batches int   `json:"batches"`
	Slices  int   `json:"slices"`
	Tenants int   `json:"tenants"`
	Rows    int64 `json:"rows"`
	Pending int64 `json:"pending"`
}
