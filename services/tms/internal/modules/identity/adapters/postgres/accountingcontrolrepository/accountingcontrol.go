package accountingcontrolrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol/accountingcontroldb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/resultcache"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var accountingControlErrors = dberror.Mapping{Entity: "AccountingControl"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) tenantports.AccountingControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accountingcontrol-repository"),
	}
}

func (r *repository) GetByOrgID(
	ctx context.Context,
	orgID pulid.ID,
) (*accountingcontrol.AccountingControl, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingcontrol.AccountingControl, error) {
			ac := accountingcontroldb.AccountingControls
			entity, err := ac.Where(ac.OrganizationID.Eq(orgID)).
				Cache(resultcache.Settings()...).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to get accounting control",
					zap.String("orgID", orgID.String()),
					zap.Error(err),
				)
				return nil, accountingControlErrors.Map(err, orgID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) ListAll(ctx context.Context) ([]*accountingcontrol.AccountingControl, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"list every organization's accounting controls for the fiscal calendar sweep",
	)
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingcontrol.AccountingControl, error) {
			entities, err := accountingcontroldb.AccountingControls.AllTenants().
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list accounting controls", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) ListWithScheduledPeriodClose(
	ctx context.Context,
) ([]*accountingcontrol.AccountingControl, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations with scheduled fiscal period close")
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingcontrol.AccountingControl, error) {
			ac := accountingcontroldb.AccountingControls
			entities, err := ac.AllTenants().
				Where(
					ac.PeriodCloseMode.Eq(accountingcontrol.PeriodCloseModeSystemScheduled),
					ac.RequirePeriodCloseApproval.IsFalse(),
				).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to list accounting controls with scheduled period close",
					zap.Error(err),
				)
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *accountingcontrol.AccountingControl,
) (*accountingcontrol.AccountingControl, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingcontrol.AccountingControl, error) {
			if err := accountingcontroldb.AccountingControls.Save(ctx, r.db.Keel(ctx), entity); err != nil {
				r.l.Error(
					"failed to update accounting control",
					zap.String("orgID", entity.OrganizationID.String()),
					zap.Error(err),
				)
				return nil, accountingControlErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}
