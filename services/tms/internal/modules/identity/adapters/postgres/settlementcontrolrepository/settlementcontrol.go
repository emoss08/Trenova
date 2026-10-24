package settlementcontrolrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var settlementControlErrors = dberror.Mapping{Entity: "SettlementControl"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) tenantports.SettlementControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.settlement-control-repository"),
	}
}

func (r *repository) GetOrCreate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.SettlementControl, error) {
	newDefault := func() *tenant.SettlementControl {
		return &tenant.SettlementControl{
			ID:                            pulid.MustNew("stlc_"),
			BusinessUnitID:                tenantInfo.BuID,
			OrganizationID:                tenantInfo.OrgID,
			PayPeriodFrequency:            tenant.PayPeriodFrequencyWeekly,
			PeriodEndDayOfWeek:            6,
			PayDelayDays:                  5,
			PayTrigger:                    tenant.PayTriggerShipmentDelivered,
			AllowNegativeNet:              true,
			AutoAttachAccruals:            true,
			VarianceThresholdPct:          decimal.NewFromInt(25),
			VarianceLookbackWeeks:         8,
			EscrowInterestFrequencyMonths: 3,
		}
	}

	return dbtx.GetOrCreate(ctx, r.db, dbtx.GetOrCreateSpec[*tenant.SettlementControl]{
		Find: r.selectControl,
		Create: func(ctx context.Context) error {
			sc := tenantdb.SettlementControls
			if _, err := sc.CreateOrSkip(
				ctx,
				r.db.Keel(ctx),
				[]*tenant.SettlementControl{newDefault()},
				sc.OrganizationID,
				sc.BusinessUnitID,
			); err != nil {
				return fmt.Errorf("create default settlement control: %w", err)
			}

			return nil
		},
		Default: newDefault,
		MapError: func(err error) error {
			return settlementControlErrors.Map(err, tenantInfo.OrgID.String())
		},
	})
}

func (r *repository) selectControl(ctx context.Context) (*tenant.SettlementControl, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tenant.SettlementControl, error) {
		return tenantdb.SettlementControls.Query().One(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) ListAutoGenerate(
	ctx context.Context,
) ([]*tenant.SettlementControl, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations that auto-generate settlements")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*tenant.SettlementControl, error) {
		sc := tenantdb.SettlementControls
		items, err := sc.AllTenants().
			Where(sc.AutoGenerateBatches.IsTrue()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list auto-generate settlement controls: %w", err)
		}
		return items, nil
	})
}

func (r *repository) ListAll(ctx context.Context) ([]*tenant.SettlementControl, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"list every organization's settlement controls for escrow interest",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*tenant.SettlementControl, error) {
		items, err := tenantdb.SettlementControls.AllTenants().All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list settlement controls: %w", err)
		}
		return items, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *tenant.SettlementControl,
) (*tenant.SettlementControl, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tenant.SettlementControl, error) {
		sc := tenantdb.SettlementControls
		if err := sc.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			sc.PayPeriodFrequency,
			sc.PeriodEndDayOfWeek,
			sc.PayDelayDays,
			sc.PayTrigger,
			sc.AutoGenerateBatches,
			sc.AutoApproveClean,
			sc.AllowNegativeNet,
			sc.VarianceThresholdPct,
			sc.VarianceLookbackWeeks,
			sc.DefaultEscrowInterestRate,
			sc.EscrowInterestFrequencyMonths,
		); err != nil {
			return nil, settlementControlErrors.Map(err, entity.ID.String())
		}

		return r.GetOrCreate(ctx, pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		})
	})
}
