package tenantbootstrap

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type Scope struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Now            int64
}

func (s Scope) now() int64 {
	if s.Now > 0 {
		return s.Now
	}

	return timeutils.NowUnix()
}

func scoped[T any](ctx context.Context, q *keel.Query[T]) *keel.Query[T] {
	if dbscope.IsSystem(ctx) {
		return q.AllTenants()
	}

	return q
}
