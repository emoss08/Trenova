package onboardingrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/onboarding"
	"github.com/emoss08/trenova/internal/modules/identity/domain/onboarding/onboardingdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/onboardingports"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const entityName = "Onboarding"

var onboardingErrors = dberror.Mapping{Entity: entityName}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) onboardingports.OnboardingRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.onboarding-repository"),
	}
}

func (r *repository) Get(
	ctx context.Context,
	_ onboardingports.GetOnboardingRequest,
) (*onboarding.Onboarding, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*onboarding.Onboarding, error) {
		entity, err := onboardingdb.Onboardings.Query().First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, onboardingErrors.Map(err, "")
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *onboarding.Onboarding,
) (*onboarding.Onboarding, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*onboarding.Onboarding, error) {
		if err := onboardingdb.Onboardings.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create onboarding",
				zap.String("organizationId", entity.OrganizationID.String()),
				zap.Error(err),
			)
			return nil, onboardingErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Complete(
	ctx context.Context,
	entity *onboarding.Onboarding,
) (*onboarding.Onboarding, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*onboarding.Onboarding, error) {
		o := onboardingdb.Onboardings
		completedAt := o.CompletedAt.ToNull()
		if entity.CompletedAt != nil {
			completedAt = o.CompletedAt.To(*entity.CompletedAt)
		}

		rows, err := o.Where(
			o.ID.Eq(entity.ID),
			o.Version.Eq(entity.Version),
			o.Status.Eq(onboarding.StatusPending),
		).UpdateReturning(ctx, r.db.Keel(ctx), o.Status.To(entity.Status), o.OperationType.To(entity.OperationType), o.SampleDataLoaded.To(entity.SampleDataLoaded), completedAt, o.CompletedByID.To(entity.CompletedByID))
		if err != nil {
			r.l.Error("failed to complete onboarding",
				zap.String("onboardingId", entity.ID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		if err = dberror.CheckAffected(
			int64(len(rows)),
			entityName,
			entity.ID.String(),
		); err != nil {
			return nil, err
		}

		return rows[0], nil
	})
}
