package distancecontrolrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/places/domain/distancecontrol"
	"github.com/emoss08/trenova/internal/modules/places/domain/distancecontrol/distancecontroldb"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceprofile/distanceprofiledb"
	"github.com/emoss08/trenova/internal/modules/places/ports/distancecontrolports"
	"github.com/emoss08/trenova/internal/modules/places/ports/distanceprofileports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/resultcache"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const shortestRoutingType = "shortest"

var distanceControlErrors = dberror.Mapping{Entity: "DistanceControl"}

type Params struct {
	fx.In

	DB                  *postgres.Connection
	Logger              *zap.Logger
	DistanceProfileRepo distanceprofileports.DistanceProfileRepository
}

type repository struct {
	db                  *postgres.Connection
	l                   *zap.Logger
	distanceProfileRepo distanceprofileports.DistanceProfileRepository
}

func New(p Params) distancecontrolports.DistanceControlRepository {
	return &repository{
		db:                  p.DB,
		l:                   p.Logger.Named("postgres.distance-control-repository"),
		distanceProfileRepo: p.DistanceProfileRepo,
	}
}

func (r *repository) Get(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*distancecontrol.DistanceControl, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*distancecontrol.DistanceControl, error) {
			entity, err := distancecontroldb.DistanceControls.Cache(resultcache.Settings()...).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, distanceControlErrors.Map(err, "")
			}
			return entity, nil
		},
	)
}

func (r *repository) EnsureDefault(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*distancecontrol.DistanceControl, error) {
	found, findErr := r.Get(ctx, tenantInfo)
	if findErr == nil {
		return found, nil
	}
	if !errortypes.IsNotFoundError(findErr) {
		return nil, findErr
	}

	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*distancecontrol.DistanceControl, error) {
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
			entity := distancecontrol.NewDefault(
				tenantInfo.OrgID,
				tenantInfo.BuID,
				practical.ID,
				r.shortestProfileID(ctx),
			)
			dc := distancecontroldb.DistanceControls
			if _, err = dc.CreateOrSkip(
				ctx,
				r.db.Keel(ctx),
				[]*distancecontrol.DistanceControl{entity},
				dc.OrganizationID,
				dc.BusinessUnitID,
			); err != nil {
				return nil, err
			}
			return r.Get(ctx, tenantInfo)
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *distancecontrol.DistanceControl,
) (*distancecontrol.DistanceControl, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*distancecontrol.DistanceControl, error) {
			err := distancecontroldb.DistanceControls.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				return nil, dberror.MapRetryableTransactionError(
					distanceControlErrors.Map(err, entity.ID.String()),
					"Distance control is busy. Retry the request.",
				)
			}
			return entity, nil
		},
	)
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

func (r *repository) shortestProfileID(ctx context.Context) pulid.ID {
	dp := distanceprofiledb.DistanceProfiles
	entity, err := dp.Pick(dp.ID).
		Where(
			dp.Status.Eq(distanceprofile.StatusActive),
			dp.RoutingType.Lower().Eq(shortestRoutingType),
		).
		Order(dp.IsDefault.Desc(), dp.UpdatedAt.Desc()).
		First(ctx, r.db.Keel(ctx))
	if err != nil {
		return ""
	}
	return entity.ID
}
