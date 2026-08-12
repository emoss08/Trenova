package organizationrepository

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

// dependencyCount describes one tenant-scoped count backing a capability
// disable guard: the model to count, where to store the total, and the scope
// that narrows it to the rows that still block the capability.
type dependencyCount struct {
	label  string
	model  any
	target *int
	scope  func(*bun.SelectQuery) *bun.SelectQuery
}

func (r *repository) countDependencies(ctx context.Context, deps []dependencyCount) error {
	for _, dep := range deps {
		total, err := r.db.DBForContext(ctx).
			NewSelect().
			Model(dep.model).
			WhereGroup(" AND ", dep.scope).
			Count(ctx)
		if err != nil {
			return fmt.Errorf("count %s: %w", dep.label, err)
		}

		*dep.target = total
	}

	return nil
}
