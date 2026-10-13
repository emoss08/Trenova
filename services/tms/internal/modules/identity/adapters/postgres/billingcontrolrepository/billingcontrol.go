package billingcontrolrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/buncolgen/tenantcol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

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
		log := r.l.With(
			zap.String("operation", "GetByOrgID"),
			zap.String("orgID", orgID.String()),
		)

		entity := new(tenant.BillingControl)
		if err := r.db.DBForContext(ctx).NewSelect().
			Model(entity).
			Where("bc.organization_id = ?", orgID).
			Scan(ctx); err != nil {
			log.Error("failed to get billing control", zap.Error(err))
			return nil, dberror.HandleNotFoundError(err, "BillingControl")
		}

		return entity, nil
	})
}

func (r *repository) ListAutomaticTransfer(ctx context.Context) ([]*tenant.BillingControl, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations that transfer freight to billing automatically")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*tenant.BillingControl, error) {
		cols := tenantcol.BillingControlColumns
		items := make([]*tenant.BillingControl, 0)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&items).
			Where(cols.BillingQueueTransferMode.Eq(), tenant.BillingQueueTransferModeAutomaticWhenReady).
			Scan(ctx); err != nil {
			return nil, fmt.Errorf("list automatic transfer billing controls: %w", err)
		}

		return items, nil
	})
}

func (r *repository) ListPaymentReminders(ctx context.Context) ([]*tenant.BillingControl, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations that email customers payment reminders")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*tenant.BillingControl, error) {
		cols := tenantcol.BillingControlColumns
		items := make([]*tenant.BillingControl, 0)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&items).
			Where(cols.PaymentRemindersEnabled.IsTrue()).
			Scan(ctx); err != nil {
			return nil, fmt.Errorf("list payment reminder billing controls: %w", err)
		}

		return items, nil
	})
}

func (r *repository) ListMonthlyStatements(ctx context.Context) ([]*tenant.BillingControl, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations that email customers monthly statements")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*tenant.BillingControl, error) {
		cols := tenantcol.BillingControlColumns
		items := make([]*tenant.BillingControl, 0)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&items).
			Where(cols.MonthlyStatementsEnabled.IsTrue()).
			Scan(ctx); err != nil {
			return nil, fmt.Errorf("list monthly statement billing controls: %w", err)
		}

		return items, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *tenant.BillingControl,
) (*tenant.BillingControl, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tenant.BillingControl, error) {
		log := r.l.With(
			zap.String("operation", "Update"),
			zap.String("orgID", entity.OrganizationID.String()),
		)

		ov := entity.Version
		entity.Version++

		result, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WherePK().
			Where("version = ?", ov).
			Returning("*").
			Exec(ctx)
		if err != nil {
			log.Error("failed to update billing control", zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckRowsAffected(
			result,
			"BillingControl",
			entity.ID.String(),
		); err != nil {
			return nil, err
		}

		return entity, nil
	})
}
