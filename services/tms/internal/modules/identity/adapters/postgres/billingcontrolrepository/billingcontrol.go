package billingcontrolrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
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
) (*tenant.BillingControl, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tenant.BillingControl, error) {
		bc := tenantdb.BillingControls
		entity, err := bc.Where(bc.OrganizationID.Eq(orgID)).One(ctx, r.db.Keel(ctx))
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

func (r *repository) ListAutomaticTransfer(ctx context.Context) ([]*tenant.BillingControl, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"list organizations that transfer freight to billing automatically",
	)
	bc := tenantdb.BillingControls
	return r.listAll(
		ctx,
		"list automatic transfer billing controls",
		bc.BillingQueueTransferMode.Eq(tenant.BillingQueueTransferModeAutomaticWhenReady),
	)
}

func (r *repository) ListPaymentReminders(ctx context.Context) ([]*tenant.BillingControl, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations that email customers payment reminders")
	return r.listAll(
		ctx,
		"list payment reminder billing controls",
		tenantdb.BillingControls.PaymentRemindersEnabled.IsTrue(),
	)
}

func (r *repository) ListMonthlyStatements(ctx context.Context) ([]*tenant.BillingControl, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations that email customers monthly statements")
	return r.listAll(
		ctx,
		"list monthly statement billing controls",
		tenantdb.BillingControls.MonthlyStatementsEnabled.IsTrue(),
	)
}

func (r *repository) listAll(
	ctx context.Context,
	operation string,
	filter keel.Pred,
) ([]*tenant.BillingControl, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*tenant.BillingControl, error) {
		items, err := tenantdb.BillingControls.AllTenants().
			Where(filter).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", operation, err)
		}

		return items, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *tenant.BillingControl,
) (*tenant.BillingControl, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tenant.BillingControl, error) {
		if err := tenantdb.BillingControls.Save(ctx, r.db.Keel(ctx), entity); err != nil {
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
