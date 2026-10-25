package organizationrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
)

type dependencyCount struct {
	target *int
	count  func(total *int64) keel.BatchItem
}

func (r *repository) countDependencies(ctx context.Context, deps []dependencyCount) error {
	return dbtx.ReadErr(ctx, r.db, func(ctx context.Context) error {
		totals := make([]int64, len(deps))
		items := make([]keel.BatchItem, len(deps))
		for i, dep := range deps {
			items[i] = dep.count(&totals[i])
		}

		if err := keel.Batch(ctx, r.db.Keel(ctx), items...); err != nil {
			return fmt.Errorf("count organization dependencies: %w", err)
		}

		for i, dep := range deps {
			*dep.target = int(totals[i])
		}

		return nil
	})
}
