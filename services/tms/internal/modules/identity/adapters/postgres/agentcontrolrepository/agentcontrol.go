package agentcontrolrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var agentControlErrors = dberror.Mapping{Entity: "AgentControl"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) tenantports.AgentControlRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-control-repository"),
	}
}

func (r *repository) GetOrCreate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.AgentControl, error) {
	newDefault := func() *tenant.AgentControl {
		control := &tenant.AgentControl{
			BusinessUnitID:     tenantInfo.BuID,
			OrganizationID:     tenantInfo.OrgID,
			ShadowMode:         true,
			PromotionThreshold: tenant.DefaultPromotionThreshold,
			BriefingEnabled:    true,
			BriefingHourLocal:  tenant.DefaultBriefingHourLocal,
		}
		control.ApplyAgentMoneyDefaults()

		return control
	}

	return dbtx.GetOrCreate(ctx, r.db, dbtx.GetOrCreateSpec[*tenant.AgentControl]{
		Find: r.selectControl,
		Create: func(ctx context.Context) error {
			ac := tenantdb.AgentControls
			if _, err := ac.CreateOrSkip(
				ctx,
				r.db.Keel(ctx),
				[]*tenant.AgentControl{newDefault()},
				ac.OrganizationID,
				ac.BusinessUnitID,
			); err != nil {
				return fmt.Errorf("create default agent control: %w", err)
			}

			return nil
		},
		Default: newDefault,
		MapError: func(err error) error {
			return agentControlErrors.Map(err, tenantInfo.OrgID.String())
		},
	})
}

func (r *repository) selectControl(ctx context.Context) (*tenant.AgentControl, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tenant.AgentControl, error) {
		return tenantdb.AgentControls.Query().One(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *tenant.AgentControl,
) (*tenant.AgentControl, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tenant.AgentControl, error) {
		ac := tenantdb.AgentControls
		if err := ac.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			ac.ShadowMode,
			ac.EarnedAutonomy,
			ac.PromotionThreshold,
			ac.PersonMonthlyMessages,
			ac.LearningOff,
			ac.AgentMoneyCallCeilingMinor,
			ac.AgentMoneyRunCalls,
			ac.AgentMoneyRunAmountMinor,
			ac.AgentMoneyDayCalls,
			ac.AgentMoneyDayAmountMinor,
			ac.AITrainingConsent,
			ac.AITrainingConsentChangedAt,
			ac.AITrainingConsentChangedByID,
		); err != nil {
			return nil, agentControlErrors.Map(err, entity.ID.String())
		}

		return r.GetOrCreate(ctx, pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		})
	})
}
