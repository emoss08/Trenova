package tenantbootstrap

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type Recorder func(ctx context.Context, table string, id pulid.ID) error

type Scope struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Now            int64
	Record         Recorder
}

func (s Scope) record(ctx context.Context, table string, id pulid.ID) error {
	if s.Record == nil {
		return nil
	}

	return s.Record(ctx, table, id)
}

func (s Scope) now() int64 {
	if s.Now > 0 {
		return s.Now
	}

	return timeutils.NowUnix()
}
