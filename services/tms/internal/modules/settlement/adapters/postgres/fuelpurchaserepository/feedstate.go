package fuelpurchaserepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase/fuelpurchasedb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/fuelpurchaseports"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/zap"
)

func (r *repository) GetFeedState(
	ctx context.Context,
	req *fuelpurchaseports.GetFuelFeedStateRequest,
) (*fuelpurchase.CardFeedState, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*fuelpurchase.CardFeedState, error) {
		fs := fuelpurchasedb.CardFeedStates
		state, err := fs.Where(
			fs.Provider.Eq(req.Provider),
			fs.FeedType.Eq(req.FeedType),
		).One(ctx, r.db.Keel(ctx))
		if dberror.IsNotFoundError(err) {
			return &fuelpurchase.CardFeedState{
				OrganizationID: req.TenantInfo.OrgID,
				BusinessUnitID: req.TenantInfo.BuID,
				Provider:       req.Provider,
				FeedType:       req.FeedType,
			}, nil
		}
		if err != nil {
			r.l.Error("failed to read fuel feed state", zap.Error(err))
			return nil, fmt.Errorf("read fuel feed state: %w", err)
		}

		return state, nil
	})
}

func (r *repository) SaveFeedState(ctx context.Context, state *fuelpurchase.CardFeedState) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		fs := fuelpurchasedb.CardFeedStates
		_, err := fs.Upsert(
			ctx,
			r.db.Keel(ctx),
			[]*fuelpurchase.CardFeedState{state},
			keel.OnConflict(fs.OrganizationID, fs.BusinessUnitID, fs.Provider, fs.FeedType).
				Update(fs.Cursor, fs.LastPolledAt, fs.LastSuccessAt, fs.FailureCount, fs.LastError),
		)
		if err != nil {
			r.l.Error("failed to save fuel feed state", zap.Error(err))
			return fmt.Errorf(
				"save fuel feed state: %w",
				feedStateErrors.Map(err, string(state.Provider)),
			)
		}

		return nil
	})
}
