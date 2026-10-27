package carriersettlementcontrolrepository

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
	"github.com/emoss08/trenova/pkg/resultcache"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var carrierSettlementControlErrors = dberror.Mapping{Entity: "CarrierSettlementControl"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) tenantports.CarrierSettlementControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-settlement-control-repository"),
	}
}

func (r *repository) GetOrCreate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.CarrierSettlementControl, error) {
	newDefault := func() *tenant.CarrierSettlementControl {
		return &tenant.CarrierSettlementControl{
			ID:                 pulid.MustNew("carstlc_"),
			BusinessUnitID:     tenantInfo.BuID,
			OrganizationID:     tenantInfo.OrgID,
			PayTrigger:         tenant.PayTriggerShipmentDelivered,
			PayPeriodFrequency: tenant.PayPeriodFrequencyWeekly,
			PeriodEndDayOfWeek: 6,
			PayDelayDays:       5,
		}
	}

	return dbtx.GetOrCreate(ctx, r.db, dbtx.GetOrCreateSpec[*tenant.CarrierSettlementControl]{
		Find: r.selectControl,
		Create: func(ctx context.Context) error {
			cc := tenantdb.CarrierSettlementControls
			if _, err := cc.CreateOrSkip(
				ctx,
				r.db.Keel(ctx),
				[]*tenant.CarrierSettlementControl{newDefault()},
				cc.OrganizationID,
				cc.BusinessUnitID,
			); err != nil {
				return fmt.Errorf("create default carrier settlement control: %w", err)
			}

			return nil
		},
		Default: newDefault,
		MapError: func(err error) error {
			return carrierSettlementControlErrors.Map(err, tenantInfo.OrgID.String())
		},
	})
}

func (r *repository) selectControl(ctx context.Context) (*tenant.CarrierSettlementControl, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*tenant.CarrierSettlementControl, error) {
			return tenantdb.CarrierSettlementControls.Cache(resultcache.Settings()...).
				One(ctx, r.db.Keel(ctx))
		},
	)
}

func (r *repository) ListAutoGenerate(
	ctx context.Context,
) ([]*tenant.CarrierSettlementControl, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations that auto-generate carrier settlements")
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*tenant.CarrierSettlementControl, error) {
			cc := tenantdb.CarrierSettlementControls
			items, err := cc.AllTenants().
				Where(cc.AutoGenerateBatches.IsTrue()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list auto-generate carrier settlement controls: %w", err)
			}
			return items, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *tenant.CarrierSettlementControl,
) (*tenant.CarrierSettlementControl, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*tenant.CarrierSettlementControl, error) {
			cc := tenantdb.CarrierSettlementControls
			if err := cc.Save(
				ctx,
				r.db.Keel(ctx),
				entity,
				cc.PayTrigger,
				cc.PayPeriodFrequency,
				cc.PeriodEndDayOfWeek,
				cc.PayDelayDays,
				cc.AutoGenerateBatches,
				cc.AutoPostOnApprove,
				cc.VarianceToleranceMinor,
				cc.AutoMatchInboundInvoices,
				cc.AutoAcceptWithinTolerance,
				cc.HoldUntilInvoiceMatched,
				cc.DefaultAPAccountID,
				cc.DefaultPurchasedTransportationAccountID,
			); err != nil {
				return nil, carrierSettlementControlErrors.Map(err, entity.ID.String())
			}

			return r.GetOrCreate(ctx, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			})
		},
	)
}
