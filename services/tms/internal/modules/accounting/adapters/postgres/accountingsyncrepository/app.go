package accountingsyncrepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/accountingsynccol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const appCredentialEntity = "Accounting app"

type AppCredentialParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type appCredentialRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewAppCredentialRepository(
	p AppCredentialParams,
) accountingsyncports.AccountingAppCredentialRepository {
	return &appCredentialRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-app-credential-repository"),
	}
}

func (r *appCredentialRepository) GetByType(
	ctx context.Context,
	req accountingsyncports.GetAccountingAppCredentialRequest,
) (*accountingsync.AccountingAppCredential, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*accountingsync.AccountingAppCredential, error) {
		entity := new(accountingsync.AccountingAppCredential)
		cols := accountingsynccol.AccountingAppCredentialColumns

		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Where(cols.IntegrationType.Eq(), req.IntegrationType).
			Apply(accountingsynccol.AccountingAppCredentialApplyTenant(req.TenantInfo)).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, appCredentialEntity)
		}

		return entity, nil
	})
}

func (r *appCredentialRepository) GetForWebhook(
	ctx context.Context,
	id pulid.ID,
	integrationType integration.Type,
) (*accountingsync.AccountingAppCredential, error) {
	ctx = dbscope.WithSystem(ctx, "load the accounting app whose webhook was called before its tenant is known")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*accountingsync.AccountingAppCredential, error) {
		entity := new(accountingsync.AccountingAppCredential)
		cols := accountingsynccol.AccountingAppCredentialColumns

		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Where(cols.ID.Eq(), id).
			Where(cols.IntegrationType.Eq(), integrationType).
			Limit(1).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, appCredentialEntity)
		}

		return entity, nil
	})
}

func (r *appCredentialRepository) Create(
	ctx context.Context,
	entity *accountingsync.AccountingAppCredential,
) (*accountingsync.AccountingAppCredential, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*accountingsync.AccountingAppCredential, error) {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *appCredentialRepository) Update(
	ctx context.Context,
	entity *accountingsync.AccountingAppCredential,
) (*accountingsync.AccountingAppCredential, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*accountingsync.AccountingAppCredential, error) {
		cols := accountingsynccol.AccountingAppCredentialColumns
		ov := entity.Version
		entity.Version++

		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WherePK().
			Where(cols.Version.Eq(), ov).
			Exec(ctx)
		if err != nil {
			entity.Version = ov
			return nil, err
		}
		if err = dberror.CheckRowsAffected(
			results,
			appCredentialEntity,
			entity.ID.String(),
		); err != nil {
			entity.Version = ov
			return nil, err
		}

		return entity, nil
	})
}

func (r *appCredentialRepository) Delete(
	ctx context.Context,
	req accountingsyncports.GetAccountingAppCredentialRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := accountingsynccol.AccountingAppCredentialColumns

		_, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*accountingsync.AccountingAppCredential)(nil)).
			WhereGroup(" AND ", func(q *bun.DeleteQuery) *bun.DeleteQuery {
				return accountingsynccol.AccountingAppCredentialScopeTenantDelete(q, req.TenantInfo).
					Where(cols.IntegrationType.Eq(), req.IntegrationType)
			}).
			Exec(ctx)
		return err
	})
}
