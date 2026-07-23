package settlementcontrolrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
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

func New(p Params) repositories.SettlementControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.settlement-control-repository"),
	}
}

func (r *repository) GetOrCreate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.SettlementControl, error) {
	entity, err := r.selectControl(ctx, tenantInfo)
	if err == nil {
		return entity, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, dberror.HandleNotFoundError(err, "SettlementControl")
	}

	control := &tenant.SettlementControl{
		ID:                            pulid.MustNew("stlc_"),
		BusinessUnitID:                tenantInfo.BuID,
		OrganizationID:                tenantInfo.OrgID,
		PayPeriodFrequency:            tenant.PayPeriodFrequencyWeekly,
		PeriodEndDayOfWeek:            6,
		PayDelayDays:                  5,
		PayTrigger:                    tenant.PayTriggerShipmentDelivered,
		AllowNegativeNet:              true,
		VarianceThresholdPct:          decimal.NewFromInt(25),
		VarianceLookbackWeeks:         8,
		EscrowInterestFrequencyMonths: 3,
	}
	if _, err = r.db.DBForContext(ctx).
		NewInsert().
		Model(control).
		On("CONFLICT (organization_id, business_unit_id) DO NOTHING").
		Exec(ctx); err != nil {
		return nil, fmt.Errorf("create default settlement control: %w", err)
	}

	entity, err = r.selectControl(ctx, tenantInfo)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "SettlementControl")
	}
	return entity, nil
}

func (r *repository) selectControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.SettlementControl, error) {
	entity := new(tenant.SettlementControl)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("stlc.organization_id = ?", tenantInfo.OrgID).
		Where("stlc.business_unit_id = ?", tenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) ListAutoGenerate(
	ctx context.Context,
) ([]*tenant.SettlementControl, error) {
	items := make([]*tenant.SettlementControl, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("stlc.auto_generate_batches = TRUE").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list auto-generate settlement controls: %w", err)
	}
	return items, nil
}

func (r *repository) ListAll(ctx context.Context) ([]*tenant.SettlementControl, error) {
	items := make([]*tenant.SettlementControl, 0)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list settlement controls: %w", err)
	}
	return items, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *tenant.SettlementControl,
) (*tenant.SettlementControl, error) {
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Where("id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("version = ?", entity.Version).
		Set("pay_period_frequency = ?", entity.PayPeriodFrequency).
		Set("period_end_day_of_week = ?", entity.PeriodEndDayOfWeek).
		Set("pay_delay_days = ?", entity.PayDelayDays).
		Set("pay_trigger = ?", entity.PayTrigger).
		Set("auto_generate_batches = ?", entity.AutoGenerateBatches).
		Set("auto_approve_clean = ?", entity.AutoApproveClean).
		Set("allow_negative_net = ?", entity.AllowNegativeNet).
		Set("variance_threshold_pct = ?", entity.VarianceThresholdPct).
		Set("variance_lookback_weeks = ?", entity.VarianceLookbackWeeks).
		Set("default_escrow_interest_rate = ?", entity.DefaultEscrowInterestRate).
		Set("escrow_interest_frequency_months = ?", entity.EscrowInterestFrequencyMonths).
		Set("version = version + 1").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("update settlement control: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "SettlementControl", entity.ID.String()); err != nil {
		return nil, err
	}
	return r.GetOrCreate(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
}
