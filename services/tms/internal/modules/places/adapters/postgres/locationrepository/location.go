package locationrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/postgis"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/quotatx"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/places/domain/geofence"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var errCircleGeofence = errors.New("circle geofences require longitude, latitude, and radius")

var locationErrors = dberror.Mapping{Entity: "Location"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
	Quota  platformplanports.QuotaGuard `optional:"true"`
}

type repository struct {
	db    *postgres.Connection
	l     *zap.Logger
	quota platformplanports.QuotaGuard
}

func New(p Params) locationports.LocationRepository {
	return &repository{
		db:    p.DB,
		l:     p.Logger.Named("postgres.location-repository"),
		quota: quotaservice.OrUnlimited(p.Quota),
	}
}

func withRelations(q *keel.Query[location.Location]) *keel.Query[location.Location] {
	l := locationdb.Locations
	return q.Load(l.State(), l.LocationCategory())
}

func (r *repository) List(
	ctx context.Context,
	req *locationports.ListLocationRequest,
) (*pagination.ListResult[*location.Location], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*location.Location], error) {
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				locationdb.Locations.Query(),
				&listquery.Request[location.Location]{
					Filter: req.Filter,
					Shape:  withRelations,
				},
			)
			if err != nil {
				r.l.Error("failed to list locations", zap.Error(err))
				return nil, err
			}
			if err = location.HydrateGeofences(result.Items...); err != nil {
				r.l.Error("failed to hydrate location geofences", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *locationports.ListLocationConnectionRequest,
) (*pagination.CursorListResult[*location.Location], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*location.Location], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				locationdb.Locations.Query(),
				&listquery.Request[location.Location]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.LocationColumns,
					Shape:   withRelations,
				},
			)
			if err != nil {
				r.l.Error("failed to list locations", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func geofenceGeometry(entity *location.Location) (keel.Assignment, bool, error) {
	geometry := locationdb.Locations.GeofenceGeometry
	switch entity.GeofenceType {
	case geofence.TypeAuto, geofence.TypeCircle:
		unplaced := entity.Longitude == nil || entity.Latitude == nil
		if unplaced && entity.GeofenceType == geofence.TypeAuto {
			return geometry.ToNull(), false, nil
		}
		if unplaced || entity.GeofenceRadiusMeters == nil {
			return keel.Assignment{}, false, errCircleGeofence
		}

		return geometry.SetExpr(postgis.Buffer(
			postgis.Pt(*entity.Longitude, *entity.Latitude),
			*entity.GeofenceRadiusMeters,
		)), true, nil
	case geofence.TypeRectangle, geofence.TypeDraw:
		polygon, err := entity.GeofencePolygon()
		if err != nil {
			return keel.Assignment{}, false, err
		}

		return geometry.To(polygon), true, nil
	default:
		return keel.Assignment{}, false, fmt.Errorf(
			"unsupported location geofence type %q",
			entity.GeofenceType,
		)
	}
}

func (r *repository) writeGeofence(
	ctx context.Context,
	entity *location.Location,
	set keel.Assignment,
) error {
	l := locationdb.Locations
	_, err := l.Where(l.ID.Eq(entity.ID)).Update(
		ctx,
		r.db.Keel(ctx),
		set,
		l.Version.ToExpr(l.Version),
		l.UpdatedAt.ToExpr(l.UpdatedAt),
	)
	return err
}

func (r *repository) GetByID(
	ctx context.Context,
	req locationports.GetLocationByIDRequest,
) (*location.Location, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*location.Location, error) {
		l := locationdb.Locations
		entity, err := withRelations(l.Where(l.ID.Eq(req.ID))).One(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to get location",
				zap.String("id", req.ID.String()),
				zap.Error(err),
			)
			return nil, locationErrors.Map(err, req.ID.String())
		}
		if err = entity.PopulateGeofenceVertices(); err != nil {
			r.l.Error("failed to hydrate location geofence", zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *location.Location,
) (*location.Location, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*location.Location, error) {
		set, placed, err := geofenceGeometry(entity)
		if err != nil {
			r.l.Error("failed to prepare geofence for location insert", zap.Error(err))
			return nil, err
		}

		err = quotatx.Run(ctx, r.db, r.quota, func(c context.Context) error {
			if createErr := locationdb.Locations.Create(c, r.db.Keel(c), entity); createErr != nil {
				r.l.Error("failed to create location", zap.Error(createErr))
				return locationErrors.Map(createErr, entity.ID.String())
			}
			if !placed {
				return nil
			}

			return r.writeGeofence(c, entity, set)
		}, platformplanports.QuotaRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
			Meter:    platformcatalog.MeterLocationsTotal,
			Quantity: 1,
		})
		if err != nil {
			return nil, err
		}

		return r.GetByID(ctx, locationports.GetLocationByIDRequest{
			ID: entity.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		})
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *location.Location,
) (*location.Location, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*location.Location, error) {
		set, _, err := geofenceGeometry(entity)
		if err != nil {
			r.l.Error("failed to prepare geofence for location update", zap.Error(err))
			return nil, err
		}

		l := locationdb.Locations
		if err = l.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			l.LocationCategoryID,
			l.StateID,
			l.Status,
			l.Code,
			l.Name,
			l.Description,
			l.AddressLine1,
			l.AddressLine2,
			l.City,
			l.PostalCode,
			l.PlaceID,
			l.IsGeocoded,
			l.Longitude,
			l.Latitude,
			l.GeofenceType,
			l.GeofenceRadiusMeters,
		); err != nil {
			r.l.Error(
				"failed to update location",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, locationErrors.Map(err, entity.ID.String())
		}

		if err = r.writeGeofence(ctx, entity, set); err != nil {
			r.l.Error("failed to update location geofence", zap.Error(err))
			return nil, err
		}

		return r.GetByID(ctx, locationports.GetLocationByIDRequest{
			ID: entity.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
		})
	})
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *locationports.BulkUpdateLocationStatusRequest,
) ([]*location.Location, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*location.Location, error) {
		l := locationdb.Locations
		entities, err := l.Where(l.ID.In(req.LocationIDs...)).
			UpdateReturning(ctx, r.db.Keel(ctx), l.Status.To(req.Status))
		if err != nil {
			r.l.Error("failed to bulk update location status", zap.Error(err))
			return nil, err
		}
		if err = location.HydrateGeofences(entities...); err != nil {
			r.l.Error("failed to hydrate location geofences", zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckBulkAffected(
			int64(len(entities)),
			"Location",
			req.LocationIDs,
		); err != nil {
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req locationports.GetLocationsByIDsRequest,
) ([]*location.Location, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*location.Location, error) {
		if len(req.LocationIDs) == 0 {
			return []*location.Location{}, nil
		}

		l := locationdb.Locations
		entities, err := l.Where(l.ID.In(req.LocationIDs...)).
			Limit(len(req.LocationIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get locations", zap.Error(err))
			return nil, err
		}
		if err = location.HydrateGeofences(entities...); err != nil {
			r.l.Error("failed to hydrate location geofences", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *locationports.LocationSelectOptionsRequest,
) (*pagination.ListResult[*location.Location], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*location.Location], error) {
			l := locationdb.Locations
			q := l.Pick(
				l.ID,
				l.Code,
				l.Name,
				l.Description,
				l.Status,
				l.AddressLine1,
				l.City,
				l.StateID,
				l.PostalCode,
			).
				Where(l.Status.Eq(domaintypes.StatusActive)).
				Load(l.State())

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(
						l.Code.ContainsFold(s),
						l.Name.ContainsFold(s),
						l.Description.ContainsFold(s),
					)
				},
			)
		},
	)
}
