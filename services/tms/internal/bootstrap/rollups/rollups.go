package rollups

import (
	"github.com/emoss08/keel/analytics/rollup"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/detentionrollup"
)

// All is every rollup Trenova keeps: the analytics CLI writes their migrations and the worker
// refreshes them.
func All() []*rollup.Rollup {
	return detentionrollup.All()
}
