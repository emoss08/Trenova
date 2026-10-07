package aiproviderrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type DismissalParams struct {
	fx.In

	DB *postgres.Connection
}

type dismissalRepository struct {
	db *postgres.Connection
}

func NewDismissalRepository(p DismissalParams) repositories.AIProviderFailureDismissalRepository {
	return &dismissalRepository{db: p.DB}
}

func (r *dismissalRepository) List(
	ctx context.Context,
	req *repositories.ListFailureDismissalsRequest,
) ([]*aiprovider.FailureDismissal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*aiprovider.FailureDismissal, error) {
		cols := buncolgen.FailureDismissalColumns
		dismissals := make([]*aiprovider.FailureDismissal, 0)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&dismissals).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.FailureDismissalScopeTenant(sq, req.TenantInfo).
					Where(cols.UserID.Eq(), req.UserID)
			}).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("list provider failure dismissals: %w", err)
		}
		return dismissals, nil
	})
}

func (r *dismissalRepository) Upsert(ctx context.Context, dismissal *aiprovider.FailureDismissal) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := buncolgen.FailureDismissalColumns
		_, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(dismissal).
			On("CONFLICT (organization_id, business_unit_id, user_id, provider_id) DO UPDATE").
			Set(cols.FailureAt.SetExcluded()).
			Set(cols.CreatedAt.SetExcluded()).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("dismiss provider failure: %w", err)
		}
		return nil
	})
}

func (r *dismissalRepository) Delete(
	ctx context.Context,
	req *repositories.DeleteFailureDismissalRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := buncolgen.FailureDismissalColumns
		_, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*aiprovider.FailureDismissal)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.FailureDismissalScopeTenantDelete(dq, req.TenantInfo).
					Where(cols.UserID.Eq(), req.UserID).
					Where(cols.ProviderID.Eq(), req.ProviderID)
			}).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("restore provider failure: %w", err)
		}
		return nil
	})
}
