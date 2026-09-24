package conversationrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/uptrace/bun"
)

const maxThreadOwnersPerCall = 1000

var _ repositories.ThreadOwnerRepository = (*repository)(nil)

func NewThreadOwners(p Params) repositories.ThreadOwnerRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.thread-owner-repository"),
	}
}

type threadOwnerRow struct {
	ID     pulid.ID `bun:"id"`
	UserID pulid.ID `bun:"user_id"`
}

func (r *repository) ThreadOwners(
	ctx context.Context,
	req repositories.ThreadOwnersRequest,
) (map[pulid.ID]pulid.ID, error) {
	if req.TenantInfo.OrgID.IsNil() || req.TenantInfo.BuID.IsNil() {
		return nil, fmt.Errorf("thread owners need an organization and a business unit")
	}

	ids := sliceutils.Dedupe(req.ThreadIDs)
	if len(ids) > maxThreadOwnersPerCall {
		return nil, fmt.Errorf("at most %d threads per call", maxThreadOwnersPerCall)
	}

	owners := make(map[pulid.ID]pulid.ID, len(ids))
	if len(ids) == 0 {
		return owners, nil
	}

	cols := buncolgen.ThreadColumns
	rows := make([]threadOwnerRow, 0, len(ids))
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*conversation.Thread)(nil)).
		Column(cols.ID.String(), cols.UserID.String()).
		Apply(buncolgen.ThreadApplyTenant(req.TenantInfo)).
		Where(cols.ID.In(), bun.List(ids)).
		Scan(ctx, &rows); err != nil {
		return nil, fmt.Errorf("read thread owners: %w", err)
	}

	for _, row := range rows {
		owners[row.ID] = row.UserID
	}

	return owners, nil
}
