package accountingsyncrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync/accountingsyncdb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const appCredentialEntity = "Accounting app"

var appCredentialErrors = dberror.Mapping{
	Entity: appCredentialEntity,
	Unique: []dberror.Constraint{
		{
			Name:    "uq_accounting_app_credentials_tenant_type",
			Field:   "integrationType",
			Message: "An accounting app is already registered for this integration",
		},
	},
}

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
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingAppCredential, error) {
			ac := accountingsyncdb.AccountingAppCredentials
			entity, err := ac.Where(ac.IntegrationType.Eq(req.IntegrationType)).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, appCredentialErrors.Map(err, "")
			}

			return entity, nil
		},
	)
}

func (r *appCredentialRepository) GetForWebhook(
	ctx context.Context,
	id pulid.ID,
	integrationType integration.Type,
) (*accountingsync.AccountingAppCredential, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"load the accounting app whose webhook was called before its tenant is known",
	)
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingAppCredential, error) {
			ac := accountingsyncdb.AccountingAppCredentials
			entity, err := ac.AllTenants().
				Where(ac.ID.Eq(id), ac.IntegrationType.Eq(integrationType)).
				First(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, appCredentialErrors.Map(err, id.String())
			}

			return entity, nil
		},
	)
}

func (r *appCredentialRepository) Create(
	ctx context.Context,
	entity *accountingsync.AccountingAppCredential,
) (*accountingsync.AccountingAppCredential, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingAppCredential, error) {
			err := accountingsyncdb.AccountingAppCredentials.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				return nil, appCredentialErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *appCredentialRepository) Update(
	ctx context.Context,
	entity *accountingsync.AccountingAppCredential,
) (*accountingsync.AccountingAppCredential, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingAppCredential, error) {
			ov := entity.Version
			err := accountingsyncdb.AccountingAppCredentials.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				entity.Version = ov
				return nil, appCredentialErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *appCredentialRepository) Delete(
	ctx context.Context,
	req accountingsyncports.GetAccountingAppCredentialRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		ac := accountingsyncdb.AccountingAppCredentials
		_, err := ac.Where(ac.IntegrationType.Eq(req.IntegrationType)).Delete(ctx, r.db.Keel(ctx))
		return err
	})
}
