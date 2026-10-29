package invoiceadjustmentcontrolrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/invoiceadjustmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/invoiceadjustmentcontrol/invoiceadjustmentcontroldb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/resultcache"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var invoiceAdjustmentControlErrors = dberror.Mapping{Entity: "InvoiceAdjustmentControl"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) tenantports.InvoiceAdjustmentControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.invoiceadjustmentcontrol-repository"),
	}
}

func (r *repository) GetByOrgID(
	ctx context.Context,
	orgID pulid.ID,
) (*invoiceadjustmentcontrol.InvoiceAdjustmentControl, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*invoiceadjustmentcontrol.InvoiceAdjustmentControl, error) {
			iac := invoiceadjustmentcontroldb.InvoiceAdjustmentControls
			entity, err := iac.Where(iac.OrganizationID.Eq(orgID)).
				Cache(resultcache.Settings()...).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to get invoice adjustment control",
					zap.String("orgID", orgID.String()),
					zap.Error(err),
				)
				return nil, invoiceAdjustmentControlErrors.Map(err, orgID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *invoiceadjustmentcontrol.InvoiceAdjustmentControl,
) (*invoiceadjustmentcontrol.InvoiceAdjustmentControl, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*invoiceadjustmentcontrol.InvoiceAdjustmentControl, error) {
			if err := invoiceadjustmentcontroldb.InvoiceAdjustmentControls.Save(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				r.l.Error(
					"failed to update invoice adjustment control",
					zap.String("orgID", entity.OrganizationID.String()),
					zap.Error(err),
				)
				return nil, invoiceAdjustmentControlErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}
