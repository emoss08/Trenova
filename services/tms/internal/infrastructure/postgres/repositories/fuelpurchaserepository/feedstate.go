package fuelpurchaserepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// GetFeedState returns how far a provider's feed has been read. A feed that has
// never run yields a zero-valued state rather than an error, because the caller's
// next move is the same either way: read from the fallback window.
func (r *repository) GetFeedState(
	ctx context.Context,
	req *repositories.GetFuelFeedStateRequest,
) (*fuelpurchase.CardFeedState, error) {
	cols := buncolgen.CardFeedStateColumns

	state := new(fuelpurchase.CardFeedState)
	err := r.db.DBForContext(ctx).NewSelect().
		Model(state).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CardFeedStateScopeTenant(sq, req.TenantInfo).
				Where(cols.Provider.Eq(), req.Provider).
				Where(cols.FeedType.Eq(), req.FeedType)
		}).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &fuelpurchase.CardFeedState{
				OrganizationID: req.TenantInfo.OrgID,
				BusinessUnitID: req.TenantInfo.BuID,
				Provider:       req.Provider,
				FeedType:       req.FeedType,
			}, nil
		}

		r.l.Error("failed to read fuel feed state", zap.Error(err))

		return nil, fmt.Errorf("read fuel feed state: %w", err)
	}

	return state, nil
}

// SaveFeedState writes the watermark. It is an upsert on the composite key so the
// first run of a feed and every run after it take the same path.
func (r *repository) SaveFeedState(ctx context.Context, state *fuelpurchase.CardFeedState) error {
	cols := buncolgen.CardFeedStateColumns
	conflict := "CONFLICT (" + cols.OrganizationID.Name + ", " + cols.BusinessUnitID.Name +
		", " + cols.Provider.Name + ", " + cols.FeedType.Name + ") DO UPDATE"

	if _, err := r.db.DBForContext(ctx).NewInsert().
		Model(state).
		On(conflict).
		Set(cols.Cursor.SetExcluded()).
		Set(cols.LastPolledAt.SetExcluded()).
		Set(cols.LastSuccessAt.SetExcluded()).
		Set(cols.FailureCount.SetExcluded()).
		Set(cols.LastError.SetExcluded()).
		Exec(ctx); err != nil {
		r.l.Error("failed to save fuel feed state", zap.Error(err))

		return fmt.Errorf("save fuel feed state: %w", err)
	}

	return nil
}
