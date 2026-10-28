package accountingsyncrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection/accountingconnectiondb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const appCredentialEntity = "Accounting app"

var appCredentialErrors = dberror.Mapping{Entity: appCredentialEntity}

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
) accountingconnectionports.AccountingAppCredentialRepository {
	return &appCredentialRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-app-credential-repository"),
	}
}

func (r *appCredentialRepository) GetByType(
	ctx context.Context,
	req accountingconnectionports.GetAccountingAppCredentialRequest,
) (*accountingconnection.AccountingAppCredential, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingconnection.AccountingAppCredential, error) {
			ac := accountingconnectiondb.AccountingAppCredentials
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
) (*accountingconnection.AccountingAppCredential, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"load the accounting app whose webhook was called before its tenant is known",
	)
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingconnection.AccountingAppCredential, error) {
			ac := accountingconnectiondb.AccountingAppCredentials
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
	entity *accountingconnection.AccountingAppCredential,
) (*accountingconnection.AccountingAppCredential, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingconnection.AccountingAppCredential, error) {
			err := accountingconnectiondb.AccountingAppCredentials.Create(
				ctx,
				r.db.Keel(ctx),
				entity,
			)
			if err != nil {
				return nil, appCredentialErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *appCredentialRepository) Update(
	ctx context.Context,
	entity *accountingconnection.AccountingAppCredential,
) (*accountingconnection.AccountingAppCredential, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingconnection.AccountingAppCredential, error) {
			ov := entity.Version
			err := accountingconnectiondb.AccountingAppCredentials.Save(ctx, r.db.Keel(ctx), entity)
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
	req accountingconnectionports.GetAccountingAppCredentialRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		ac := accountingconnectiondb.AccountingAppCredentials
		_, err := ac.Where(ac.IntegrationType.Eq(req.IntegrationType)).Delete(ctx, r.db.Keel(ctx))
		return err
	})
}
