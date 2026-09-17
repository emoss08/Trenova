package carrierintelrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type feedStateRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewFeedStateRepository(p Params) repositories.CarrierIntelFeedStateRepository {
	return &feedStateRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-intel-feed-state-repository"),
	}
}

func (r *feedStateRepository) Get(
	ctx context.Context,
	key repositories.CarrierIntelFeedStateKey,
) (*carrierintel.CarrierIntelFeedState, error) {
	cols := buncolgen.CarrierIntelFeedStateColumns
	entity := new(carrierintel.CarrierIntelFeedState)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelFeedStateScopeTenant(sq, key.TenantInfo).
				Where(cols.Provider.Eq(), key.Provider).
				Where(cols.FeedType.Eq(), key.FeedType)
		}).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // nil feed state means the feed has never been polled
		}
		r.l.Error("failed to get carrier intel feed state", zap.Error(err))
		return nil, fmt.Errorf("get carrier intel feed state: %w", err)
	}

	return entity, nil
}

func (r *feedStateRepository) Upsert(
	ctx context.Context,
	entity *carrierintel.CarrierIntelFeedState,
) error {
	cols := buncolgen.CarrierIntelFeedStateColumns
	_, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		On("CONFLICT (organization_id, business_unit_id, provider, feed_type) DO UPDATE").
		Set(cols.Cursor.SetExcluded()).
		Set(cols.LastPolledAt.SetExcluded()).
		Set(cols.LastSuccessAt.SetExcluded()).
		Set(cols.NextPollAfter.SetExcluded()).
		Set(cols.PausedReason.SetExcluded()).
		Set(cols.PausedAt.SetExcluded()).
		Set(cols.FailureCount.SetExcluded()).
		Set(cols.LastError.SetExcluded()).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to upsert carrier intel feed state", zap.Error(err))
		return fmt.Errorf("upsert carrier intel feed state: %w", err)
	}

	return nil
}
