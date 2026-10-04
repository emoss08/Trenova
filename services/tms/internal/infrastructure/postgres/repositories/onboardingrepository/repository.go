package onboardingrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/onboarding"
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

const entityName = "Onboarding"

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.OnboardingRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.onboarding-repository"),
	}
}

func (r *repository) Get(
	ctx context.Context,
	req repositories.GetOnboardingRequest,
) (*onboarding.Onboarding, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*onboarding.Onboarding, error) {
		entity := new(onboarding.Onboarding)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Apply(buncolgen.OnboardingApplyTenant(req.TenantInfo)).
			Limit(1).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, entityName)
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *onboarding.Onboarding,
) (*onboarding.Onboarding, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*onboarding.Onboarding, error) {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
			r.l.Error("failed to create onboarding",
				zap.String("organizationId", entity.OrganizationID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) Complete(
	ctx context.Context,
	entity *onboarding.Onboarding,
) (*onboarding.Onboarding, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*onboarding.Onboarding, error) {
		cols := buncolgen.OnboardingColumns
		tenantInfo := pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		}
		updated := new(onboarding.Onboarding)

		result, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(updated).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.OnboardingScopeTenantUpdate(uq, tenantInfo).
					Where(cols.ID.Eq(), entity.ID).
					Where(cols.Version.Eq(), entity.Version).
					Where(cols.Status.Eq(), onboarding.StatusPending)
			}).
			Set(cols.Status.Set(), entity.Status).
			Set(cols.OperationType.Set(), entity.OperationType).
			Set(cols.SampleDataLoaded.Set(), entity.SampleDataLoaded).
			Set(cols.CompletedAt.Set(), entity.CompletedAt).
			Set(cols.CompletedByID.Set(), entity.CompletedByID).
			Set(cols.Version.Inc(1)).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Returning("*").
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to complete onboarding",
				zap.String("onboardingId", entity.ID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		if err = dberror.CheckRowsAffected(result, entityName, entity.ID.String()); err != nil {
			return nil, err
		}

		return updated, nil
	})
}
