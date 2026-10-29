package agentcontrolrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/agentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/agentcontrol/agentcontroldb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/resultcache"
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
) (*agentcontrol.AgentControl, error) {
	newDefault := func() *agentcontrol.AgentControl {
		control := &agentcontrol.AgentControl{
			BusinessUnitID:     tenantInfo.BuID,
			OrganizationID:     tenantInfo.OrgID,
			ShadowMode:         true,
			PromotionThreshold: agentcontrol.DefaultPromotionThreshold,
			BriefingEnabled:    true,
			BriefingHourLocal:  agentcontrol.DefaultBriefingHourLocal,
		}
		control.ApplyAgentMoneyDefaults()

		return control
	}

	return dbtx.GetOrCreate(ctx, r.db, dbtx.GetOrCreateSpec[*agentcontrol.AgentControl]{
		Find: r.selectControl,
		Create: func(ctx context.Context) error {
			ac := agentcontroldb.AgentControls
			if _, err := ac.CreateOrSkip(
				ctx,
				r.db.Keel(ctx),
				[]*agentcontrol.AgentControl{newDefault()},
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

func (r *repository) selectControl(ctx context.Context) (*agentcontrol.AgentControl, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentcontrol.AgentControl, error) {
		return agentcontroldb.AgentControls.Cache(resultcache.Settings()...).
			One(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *agentcontrol.AgentControl,
) (*agentcontrol.AgentControl, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentcontrol.AgentControl, error) {
		ac := agentcontroldb.AgentControls
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
