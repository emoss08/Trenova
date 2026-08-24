package carriersettlementcontrolrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
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

func New(p Params) repositories.CarrierSettlementControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-settlement-control-repository"),
	}
}

func (r *repository) GetOrCreate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.CarrierSettlementControl, error) {
	entity, err := r.selectControl(ctx, tenantInfo)
	if err == nil {
		return entity, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, dberror.HandleNotFoundError(err, "CarrierSettlementControl")
	}

	control := &tenant.CarrierSettlementControl{
		ID:                 pulid.MustNew("carstlc_"),
		BusinessUnitID:     tenantInfo.BuID,
		OrganizationID:     tenantInfo.OrgID,
		PayTrigger:         tenant.PayTriggerShipmentDelivered,
		PayPeriodFrequency: tenant.PayPeriodFrequencyWeekly,
		PeriodEndDayOfWeek: 6,
		PayDelayDays:       5,
	}
	if _, err = r.db.DBForContext(ctx).
		NewInsert().
		Model(control).
		On("CONFLICT (organization_id, business_unit_id) DO NOTHING").
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("create default carrier settlement control: %w", err)
	}

	entity, err = r.selectControl(ctx, tenantInfo)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "CarrierSettlementControl")
	}
	return entity, nil
}

func (r *repository) selectControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.CarrierSettlementControl, error) {
	entity := new(tenant.CarrierSettlementControl)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierSettlementControlScopeTenant(sq, tenantInfo)
		}).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) ListAutoGenerate(
	ctx context.Context,
) ([]*tenant.CarrierSettlementControl, error) {
	cols := buncolgen.CarrierSettlementControlColumns
	items := make([]*tenant.CarrierSettlementControl, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where(cols.AutoGenerateBatches.IsTrue()).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list auto-generate carrier settlement controls: %w", err)
	}
	return items, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *tenant.CarrierSettlementControl,
) (*tenant.CarrierSettlementControl, error) {
	cols := buncolgen.CarrierSettlementControlColumns
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.CarrierSettlementControlScopeTenantUpdate(uq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).
				Where(cols.ID.Eq(), entity.ID).
				Where(cols.Version.Eq(), entity.Version)
		}).
		Set(cols.PayTrigger.Set(), entity.PayTrigger).
		Set(cols.PayPeriodFrequency.Set(), entity.PayPeriodFrequency).
		Set(cols.PeriodEndDayOfWeek.Set(), entity.PeriodEndDayOfWeek).
		Set(cols.PayDelayDays.Set(), entity.PayDelayDays).
		Set(cols.AutoGenerateBatches.Set(), entity.AutoGenerateBatches).
		Set(cols.AutoPostOnApprove.Set(), entity.AutoPostOnApprove).
		Set(cols.VarianceToleranceMinor.Set(), entity.VarianceToleranceMinor).
		Set(cols.AutoMatchInboundInvoices.Set(), entity.AutoMatchInboundInvoices).
		Set(cols.AutoAcceptWithinTolerance.Set(), entity.AutoAcceptWithinTolerance).
		Set(cols.DefaultAPAccountID.Set(), entity.DefaultAPAccountID).
		Set(
			cols.DefaultPurchasedTransportationAccountID.Set(),
			entity.DefaultPurchasedTransportationAccountID,
		).
		Set(cols.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update carrier settlement control: %w", err)
	}
	if err = dberror.CheckRowsAffected(
		res,
		"CarrierSettlementControl",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}
	return r.GetOrCreate(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
}
