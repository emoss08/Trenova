package conversationrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const maxThreadOwnersPerCall = 1000

var _ conversationports.ThreadOwnerRepository = (*repository)(nil)

func NewThreadOwners(p Params) conversationports.ThreadOwnerRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.thread-owner-repository"),
	}
}

func (r *repository) ThreadOwners(
	ctx context.Context,
	req conversationports.ThreadOwnersRequest,
) (map[pulid.ID]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]pulid.ID, error) {
		if req.TenantInfo.OrgID.IsNil() || req.TenantInfo.BuID.IsNil() {
			return nil, errors.New("thread owners need an organization and a business unit")
		}

		ids := sliceutils.Dedupe(req.ThreadIDs)
		if len(ids) > maxThreadOwnersPerCall {
			return nil, fmt.Errorf("at most %d threads per call", maxThreadOwnersPerCall)
		}

		owners := make(map[pulid.ID]pulid.ID, len(ids))
		if len(ids) == 0 {
			return owners, nil
		}

		th := conversationdb.Threads
		rows, err := th.Pick(th.ID, th.UserID).
			Where(th.ID.In(ids...)).
			Limit(len(ids)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read thread owners: %w", err)
		}

		for _, row := range rows {
			owners[row.ID] = row.UserID
		}

		return owners, nil
	})
}
