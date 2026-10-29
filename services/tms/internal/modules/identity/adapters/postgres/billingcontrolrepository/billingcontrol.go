package billingcontrolrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol/billingcontroldb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/resultcache"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var billingControlErrors = dberror.Mapping{Entity: "BillingControl"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) tenantports.BillingControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.billingcontrol-repository"),
	}
}

func (r *repository) GetByOrgID(
	ctx context.Context,
	orgID pulid.ID,
) (*billingcontrol.BillingControl, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*billingcontrol.BillingControl, error) {
		bc := billingcontroldb.BillingControls
		entity, err := bc.Where(bc.OrganizationID.Eq(orgID)).
			Cache(resultcache.Settings()...).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to get billing control",
				zap.String("orgID", orgID.String()),
				zap.Error(err),
			)
			return nil, billingControlErrors.Map(err, orgID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListAutomaticTransfer(
	ctx context.Context,
) ([]*billingcontrol.BillingControl, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"list organizations that transfer freight to billing automatically",
	)
	bc := billingcontroldb.BillingControls
	return r.listAll(
		ctx,
		"list automatic transfer billing controls",
		bc.BillingQueueTransferMode.Eq(billingcontrol.BillingQueueTransferModeAutomaticWhenReady),
	)
}

func (r *repository) ListPaymentReminders(
	ctx context.Context,
) ([]*billingcontrol.BillingControl, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations that email customers payment reminders")
	return r.listAll(
		ctx,
		"list payment reminder billing controls",
		billingcontroldb.BillingControls.PaymentRemindersEnabled.IsTrue(),
	)
}

func (r *repository) ListMonthlyStatements(
	ctx context.Context,
) ([]*billingcontrol.BillingControl, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations that email customers monthly statements")
	return r.listAll(
		ctx,
		"list monthly statement billing controls",
		billingcontroldb.BillingControls.MonthlyStatementsEnabled.IsTrue(),
	)
}

func (r *repository) listAll(
	ctx context.Context,
	operation string,
	filter keel.Pred,
) ([]*billingcontrol.BillingControl, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*billingcontrol.BillingControl, error) {
			items, err := billingcontroldb.BillingControls.AllTenants().
				Where(filter).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("%s: %w", operation, err)
			}

			return items, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *billingcontrol.BillingControl,
) (*billingcontrol.BillingControl, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*billingcontrol.BillingControl, error) {
		if err := billingcontroldb.BillingControls.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to update billing control",
				zap.String("orgID", entity.OrganizationID.String()),
				zap.Error(err),
			)
			return nil, billingControlErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}
