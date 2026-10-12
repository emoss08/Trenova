package iamrepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/iamcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type authEventRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewAuthEventRepository(p Params) iamports.AuthEventRepository {
	return &authEventRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.auth-event-repository"),
	}
}

func (r *authEventRepository) Create(ctx context.Context, event *iam.AuthEvent) error {
	if event.OrganizationID.IsNil() {
		ctx = dbscope.WithSystem(
			ctx,
			"record an authentication attempt that matched no organization",
		)
	}
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(event).Exec(ctx); err != nil {
			r.l.Error("failed to insert auth event", zap.Error(err))
			return err
		}

		return nil
	})
}

func (r *authEventRepository) DeleteBefore(ctx context.Context, before int64) (int64, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"delete authentication events past the retention period for every organization",
	)
	return postgres.DeleteUnderAuditRetention(ctx, r.db, func(tx bun.Tx) *bun.DeleteQuery {
		return tx.NewDelete().
			Model((*iam.AuthEvent)(nil)).
			Where(iamcol.AuthEventColumns.OccurredAt.Lt(), before)
	})
}

const maxSignInHistory = 50

func NewAuthEventHistoryRepository(p Params) iamports.AuthEventHistoryRepository {
	return &authEventRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.auth-event-history-repository"),
	}
}

func (r *authEventRepository) ListSignInsForUser(
	ctx context.Context,
	req iamports.ListUserSignInsRequest,
) ([]*iam.AuthEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*iam.AuthEvent, error) {
		cols := iamcol.AuthEventColumns
		limit := req.Limit
		if limit <= 0 || limit > maxSignInHistory {
			limit = maxSignInHistory
		}

		entities := make([]*iam.AuthEvent, 0, limit)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(&entities).
			Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
			Where(cols.UserID.Eq(), req.TenantInfo.UserID).
			Where(cols.Provider.NotIn(), bun.List(iamports.NonSignInAuthEventProviders())).
			Order(cols.OccurredAt.OrderDesc()).
			Limit(int64(limit)).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list sign-ins for user", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}
