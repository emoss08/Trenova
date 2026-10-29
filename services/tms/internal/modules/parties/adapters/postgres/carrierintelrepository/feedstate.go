package carrierintelrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel/carrierinteldb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelenrollment"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/zap"
)

type feedStateRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewFeedStateRepository(p Params) carrierintelports.CarrierIntelFeedStateRepository {
	return &feedStateRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-intel-feed-state-repository"),
	}
}

func (r *feedStateRepository) Get(
	ctx context.Context,
	key carrierintelports.CarrierIntelFeedStateKey,
) (*carrierintelenrollment.CarrierIntelFeedState, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*carrierintelenrollment.CarrierIntelFeedState, error) {
			fs := carrierinteldb.CarrierIntelFeedStates
			entity, err := fs.Where(fs.Provider.Eq(key.Provider), fs.FeedType.Eq(key.FeedType)).
				One(ctx, r.db.Keel(ctx))
			if dberror.IsNotFoundError(err) {
				return nil, nil
			}
			if err != nil {
				r.l.Error("failed to get carrier intel feed state", zap.Error(err))
				return nil, fmt.Errorf("get carrier intel feed state: %w", err)
			}

			return entity, nil
		},
	)
}

func (r *feedStateRepository) Upsert(
	ctx context.Context,
	entity *carrierintelenrollment.CarrierIntelFeedState,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		fs := carrierinteldb.CarrierIntelFeedStates
		_, err := fs.Upsert(
			ctx,
			r.db.Keel(ctx),
			[]*carrierintelenrollment.CarrierIntelFeedState{entity},
			keel.OnConflict(fs.OrganizationID, fs.BusinessUnitID, fs.Provider, fs.FeedType).Update(
				fs.Cursor,
				fs.LastPolledAt,
				fs.LastSuccessAt,
				fs.NextPollAfter,
				fs.PausedReason,
				fs.PausedAt,
				fs.FailureCount,
				fs.LastError,
			),
		)
		if err != nil {
			r.l.Error("failed to upsert carrier intel feed state", zap.Error(err))
			return fmt.Errorf("upsert carrier intel feed state: %w", err)
		}

		return nil
	})
}
