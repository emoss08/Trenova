package distancecontrolrepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/distancecontrol"
	"github.com/emoss08/trenova/internal/core/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB                  *postgres.Connection
	Logger              *zap.Logger
	DistanceProfileRepo repositories.DistanceProfileRepository
}

type repository struct {
	db                  *postgres.Connection
	l                   *zap.Logger
	distanceProfileRepo repositories.DistanceProfileRepository
}

func New(p Params) repositories.DistanceControlRepository {
	return &repository{
		db:                  p.DB,
		l:                   p.Logger.Named("postgres.distance-control-repository"),
		distanceProfileRepo: p.DistanceProfileRepo,
	}
}

func (r *repository) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*distancecontrol.DistanceControl, error) {
	entity := new(distancecontrol.DistanceControl)
	err := r.db.DBForContext(ctx).NewSelect().
		Model(entity).
		Where("dc.organization_id = ?", tenantInfo.OrgID).
		Where("dc.business_unit_id = ?", tenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "DistanceControl")
	}
	return entity, nil
}

func (r *repository) EnsureDefault(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*distancecontrol.DistanceControl, error) {
	existing, err := r.Get(ctx, tenantInfo)
	if err == nil {
		return existing, nil
	}
	if !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	practical, err := r.distanceProfileRepo.EnsureDefault(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	shortestID := r.shortestProfileID(ctx, tenantInfo)
	entity := distancecontrol.NewDefault(tenantInfo.OrgID, tenantInfo.BuID, practical.ID, shortestID)
	if _, err = r.db.DBForContext(ctx).NewInsert().
		Model(entity).
		On("CONFLICT (organization_id, business_unit_id) DO NOTHING").
		Exec(ctx); err != nil {
		return nil, err
	}
	return r.Get(ctx, tenantInfo)
}

func (r *repository) Update(
	ctx context.Context,
	entity *distancecontrol.DistanceControl,
) (*distancecontrol.DistanceControl, error) {
	previousVersion := entity.Version
	entity.Version++
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		result, err := r.db.DBForContext(c).NewUpdate().
			Model(entity).
			WherePK().
			Where("version = ?", previousVersion).
			Returning("*").
			Exec(c)
		if err != nil {
			return err
		}
		return dberror.CheckRowsAffected(result, "DistanceControl", entity.ID.String())
	})
	if err != nil {
		return nil, dberror.MapRetryableTransactionError(err, "Distance control is busy. Retry the request.")
	}
	return entity, nil
}

func (r *repository) ResolveProfile(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	purpose string,
) (pulid.ID, error) {
	control, err := r.EnsureDefault(ctx, tenantInfo)
	if err != nil {
		return "", err
	}
	return control.ProfileIDForPurpose(purpose), nil
}

func (r *repository) shortestProfileID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) pulid.ID {
	entity := new(distanceprofile.DistanceProfile)
	err := r.db.DBForContext(ctx).NewSelect().
		Model(entity).
		Where("dp.organization_id = ?", tenantInfo.OrgID).
		Where("dp.business_unit_id = ?", tenantInfo.BuID).
		Where("dp.status = ?", distanceprofile.StatusActive).
		Where("lower(dp.routing_type) = ?", strings.ToLower("Shortest")).
		Order("dp.is_default DESC", "dp.updated_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return ""
	}
	return entity.ID
}
