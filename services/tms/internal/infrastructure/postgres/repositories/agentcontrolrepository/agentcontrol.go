package agentcontrolrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
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

func New(p Params) repositories.AgentControlRepository {
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
		return &tenant.AgentControl{
			BusinessUnitID:     tenantInfo.BuID,
			OrganizationID:     tenantInfo.OrgID,
			ShadowMode:         true,
			PromotionThreshold: tenant.DefaultPromotionThreshold,
			BriefingEnabled:    true,
			BriefingHourLocal:  tenant.DefaultBriefingHourLocal,
		}
	}

	return dbtx.GetOrCreate(ctx, r.db, dbtx.GetOrCreateSpec[*tenant.AgentControl]{
		Find: func(ctx context.Context) (*tenant.AgentControl, error) {
			return r.selectControl(ctx, tenantInfo)
		},
		Create: func(ctx context.Context) error {
			if _, insertErr := r.db.DBForContext(ctx).
				NewInsert().
				Model(newDefault()).
				On("CONFLICT (organization_id, business_unit_id) DO NOTHING").
				Exec(ctx); insertErr != nil {
				return fmt.Errorf("create default agent control: %w", insertErr)
			}

			return nil
		},
		Default: newDefault,
		MapError: func(err error) error {
			return dberror.HandleNotFoundError(err, "AgentControl")
		},
	})
}

func (r *repository) selectControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.AgentControl, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tenant.AgentControl, error) {
		entity := new(tenant.AgentControl)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.AgentControlScopeTenant(sq, tenantInfo)
			}).
			Scan(ctx)
		if err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *tenant.AgentControl,
) (*tenant.AgentControl, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tenant.AgentControl, error) {
		cols := buncolgen.AgentControlColumns

		res, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.AgentControlScopeTenantUpdate(uq, pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				}).Where(cols.ID.Eq(), entity.ID).
					Where(cols.Version.Eq(), entity.Version)
			}).
			Set(cols.ShadowMode.Set(), entity.ShadowMode).
			Set(cols.EarnedAutonomy.Set(), entity.EarnedAutonomy).
			Set(cols.PromotionThreshold.Set(), entity.PromotionThreshold).
			Set(cols.PersonMonthlyMessages.Set(), entity.PersonMonthlyMessages).
			Set(cols.LearningOff.Set(), entity.LearningOff).
			Set(cols.AITrainingConsent.Set(), entity.AITrainingConsent).
			Set(cols.AITrainingConsentChangedAt.Set(), entity.AITrainingConsentChangedAt).
			Set(cols.AITrainingConsentChangedByID.Set(), entity.AITrainingConsentChangedByID).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Set(cols.Version.Inc(1)).
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("update agent control: %w", err)
		}

		if err = dberror.CheckRowsAffected(res, "AgentControl", entity.ID.String()); err != nil {
			return nil, err
		}

		return r.GetOrCreate(ctx, pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		})
	})
}
