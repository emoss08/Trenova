package distanceprofilerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceprofile"
	"github.com/emoss08/trenova/internal/modules/places/domain/distanceprofile/distanceprofiledb"
	"github.com/emoss08/trenova/internal/modules/places/ports/distanceprofileports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const busyMessage = "Distance profile is busy. Retry the request."

var distanceProfileErrors = dberror.Mapping{Entity: "DistanceProfile"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) distanceprofileports.DistanceProfileRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.distance-profile-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *distanceprofileports.ListDistanceProfileRequest,
) (*pagination.ListResult[*distanceprofile.DistanceProfile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*distanceprofile.DistanceProfile], error) {
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				distanceprofiledb.DistanceProfiles.Query(),
				&listquery.Request[distanceprofile.DistanceProfile]{Filter: req.Filter},
			)
			if err != nil {
				r.l.Error("failed to list distance profiles", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *distanceprofileports.ListDistanceProfileConnectionRequest,
) (*pagination.CursorListResult[*distanceprofile.DistanceProfile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*distanceprofile.DistanceProfile], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				distanceprofiledb.DistanceProfiles.Query(),
				&listquery.Request[distanceprofile.DistanceProfile]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.DistanceProfileColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list distance profiles", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req distanceprofileports.GetDistanceProfileByIDRequest,
) (*distanceprofile.DistanceProfile, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*distanceprofile.DistanceProfile, error) {
			entity, err := distanceprofiledb.DistanceProfiles.Get(ctx, r.db.Keel(ctx), req.ID)
			if err != nil {
				return nil, distanceProfileErrors.Map(err, req.ID.String())
			}
			return entity, nil
		},
	)
}

func (r *repository) GetDefault(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*distanceprofile.DistanceProfile, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*distanceprofile.DistanceProfile, error) {
			dp := distanceprofiledb.DistanceProfiles
			entity, err := dp.Where(dp.IsDefault.IsTrue()).One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, distanceProfileErrors.Map(err, "")
			}
			return entity, nil
		},
	)
}

func (r *repository) EnsureDefault(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*distanceprofile.DistanceProfile, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*distanceprofile.DistanceProfile, error) {
			existing, err := r.GetDefault(ctx, tenantInfo)
			if err == nil {
				return existing, nil
			}
			if !errortypes.IsNotFoundError(err) {
				return nil, err
			}

			entity := distanceprofile.NewDefault(tenantInfo.OrgID, tenantInfo.BuID)
			inserted, err := distanceprofiledb.DistanceProfiles.CreateOrSkip(
				ctx,
				r.db.Keel(ctx),
				[]*distanceprofile.DistanceProfile{entity},
			)
			if err != nil {
				return nil, distanceProfileErrors.Map(err, entity.ID.String())
			}
			if inserted == 0 {
				return r.GetDefault(ctx, tenantInfo)
			}
			return entity, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *distanceprofile.DistanceProfile,
) (*distanceprofile.DistanceProfile, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*distanceprofile.DistanceProfile, error) {
			if entity.IsDefault {
				if err := r.clearDefault(ctx, pulid.Nil); err != nil {
					return nil, dberror.MapRetryableTransactionError(err, busyMessage)
				}
			}
			if err := distanceprofiledb.DistanceProfiles.Create(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				return nil, dberror.MapRetryableTransactionError(
					distanceProfileErrors.Map(err, entity.ID.String()),
					busyMessage,
				)
			}
			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *distanceprofile.DistanceProfile,
) (*distanceprofile.DistanceProfile, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*distanceprofile.DistanceProfile, error) {
			if entity.IsDefault {
				if err := r.clearDefault(ctx, entity.ID); err != nil {
					return nil, dberror.MapRetryableTransactionError(err, busyMessage)
				}
			}
			if err := distanceprofiledb.DistanceProfiles.Save(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				return nil, dberror.MapRetryableTransactionError(
					distanceProfileErrors.Map(err, entity.ID.String()),
					busyMessage,
				)
			}
			return entity, nil
		},
	)
}

func (r *repository) Delete(
	ctx context.Context,
	req distanceprofileports.DeleteDistanceProfileRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		dp := distanceprofiledb.DistanceProfiles
		deleted, err := dp.Where(dp.ID.Eq(req.ID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return distanceProfileErrors.Map(err, req.ID.String())
		}

		return dberror.CheckFoundCount(deleted, distanceProfileErrors.Entity)
	})
}

func (r *repository) SetDefault(
	ctx context.Context,
	req distanceprofileports.GetDistanceProfileByIDRequest,
) (*distanceprofile.DistanceProfile, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*distanceprofile.DistanceProfile, error) {
			current, err := r.GetByID(ctx, req)
			if err != nil {
				return nil, err
			}
			if current.Status != distanceprofile.StatusActive {
				return nil, errortypes.NewValidationError(
					"isDefault",
					errortypes.ErrInvalid,
					"Default profile must be active",
				)
			}
			if err = r.clearDefault(ctx, current.ID); err != nil {
				return nil, dberror.MapRetryableTransactionError(err, busyMessage)
			}

			dp := distanceprofiledb.DistanceProfiles
			current.IsDefault = true
			if err = dp.Save(ctx, r.db.Keel(ctx), current, dp.IsDefault); err != nil {
				return nil, dberror.MapRetryableTransactionError(
					distanceProfileErrors.Map(err, current.ID.String()),
					busyMessage,
				)
			}
			return current, nil
		},
	)
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *distanceprofileports.DistanceProfileSelectOptionsRequest,
) (*pagination.ListResult[*distanceprofile.DistanceProfile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*distanceprofile.DistanceProfile], error) {
			dp := distanceprofiledb.DistanceProfiles
			q := dp.Pick(
				dp.ID,
				dp.Name,
				dp.Description,
				dp.Status,
				dp.IsDefault,
				dp.Provider,
				dp.DataVersion,
				dp.Region,
				dp.RoutingType,
				dp.DistanceUnits,
				dp.LocationGranularity,
				dp.ProfileName,
			).
				Where(dp.Status.Eq(distanceprofile.StatusActive)).
				Order(dp.IsDefault.Desc(), dp.Name.Asc())

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(
						dp.Name.ContainsFold(s),
						dp.Description.ContainsFold(s),
						dp.ProfileName.ContainsFold(s),
						dp.RoutingType.ContainsFold(s),
						dp.DistanceUnits.ContainsFold(s),
					)
				},
			)
		},
	)
}

func (r *repository) clearDefault(ctx context.Context, keep pulid.ID) error {
	dp := distanceprofiledb.DistanceProfiles
	_, err := dp.Where(
		dp.IsDefault.IsTrue(),
		keel.When(!keep.IsNil(), dp.ID.Ne(keep)),
	).Update(ctx, r.db.Keel(ctx), dp.IsDefault.To(false))
	return err
}
