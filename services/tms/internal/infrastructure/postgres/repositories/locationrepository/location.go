package locationrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/geofence"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/quotaservice"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/quotatx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dbdialect"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/postgis"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
	Quota  services.QuotaGuard `optional:"true"`
}

type repository struct {
	db    *postgres.Connection
	l     *zap.Logger
	quota services.QuotaGuard
}

func New(p Params) repositories.LocationRepository {
	return &repository{
		db:    p.DB,
		l:     p.Logger.Named("postgres.location-repository"),
		quota: quotaservice.OrUnlimited(p.Quota),
	}
}

func WithGeofenceGeometry(q *bun.SelectQuery) *bun.SelectQuery {
	return q.ColumnExpr("loc.*").ColumnExpr("loc.geofence_geometry AS geofence_geometry")
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListLocationRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.LocationTable.Alias,
		req.Filter,
		(*location.Location)(nil),
	)

	relations := buncolgen.LocationRelations

	q = q.Relation(relations.State).Relation(relations.LocationCategory)
	q = q.Apply(WithGeofenceGeometry)

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListLocationRequest,
) (*pagination.ListResult[*location.Location], error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*pagination.ListResult[*location.Location], error) {
		log := r.l.With(
			zap.String("operation", "List"),
			zap.Any("request", req),
		)

		entities := make([]*location.Location, 0, req.Filter.Pagination.SafeLimit())
		total, err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				return r.filterQuery(sq, req)
			}).ScanAndCount(ctx)
		if err != nil {
			log.Error("failed to scan and count locations", zap.Error(err))
			return nil, err
		}
		if err = location.HydrateGeofences(entities...); err != nil {
			log.Error("failed to hydrate location geofences", zap.Error(err))
			return nil, err
		}

		return &pagination.ListResult[*location.Location]{
			Items: entities,
			Total: total,
		}, nil
	})
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *repositories.ListLocationConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q,
		buncolgen.LocationTable.Alias,
		req.Filter,
		req.Cursor,
		(*location.Location)(nil),
	)
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *repositories.ListLocationConnectionRequest,
) *bun.SelectQuery {
	return querybuilder.ApplyFiltersWithoutSort(
		q,
		buncolgen.LocationTable.Alias,
		req.Filter,
		(*location.Location)(nil),
	)
}

func applyLocationColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(buncolgen.LocationTable.All())
	}

	return q.Column(columns...)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListLocationConnectionRequest,
) (*pagination.CursorListResult[*location.Location], error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*pagination.CursorListResult[*location.Location], error) {
		log := r.l.With(
			zap.String("operation", "ListConnection"),
			zap.Any("request", req),
		)

		dba := r.db.DBForContext(ctx)
		var totalCount *int
		if req.Cursor.IncludeTotalCount {
			total, err := dba.
				NewSelect().
				Model((*location.Location)(nil)).
				Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
					return r.applyTotalCountFilters(sq, req)
				}).
				Count(ctx)
			if err != nil {
				log.Error("failed to count locations", zap.Error(err))
				return nil, err
			}
			totalCount = &total
		}

		result, err := dbhelper.CursorList(
			ctx,
			dbhelper.CursorListParams[*location.Location]{
				Filter:     req.Filter,
				Cursor:     req.Cursor,
				TotalCount: totalCount,
				Query: func(entities *[]*location.Location) *bun.SelectQuery {
					relations := buncolgen.LocationRelations
					return dba.
						NewSelect().
						Model(entities).
						Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
							return applyLocationColumns(sq, req.LocationColumns)
						}).
						Relation(relations.State).
						Relation(relations.LocationCategory)
				},
				Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
					return r.applyCursorPageFilters(sq, req)
				},
			})
		if err != nil {
			log.Error("failed to scan locations", zap.Error(err))
			return nil, err
		}

		return result, nil
	})
}

func applyLocationGeofence(
	insertQuery *bun.InsertQuery,
	updateQuery *bun.UpdateQuery,
	entity *location.Location,
) error {
	switch entity.GeofenceType {
	case geofence.TypeAuto, geofence.TypeCircle:
		if entity.Longitude == nil || entity.Latitude == nil || entity.GeofenceRadiusMeters == nil {
			return errors.New("circle geofences require longitude, latitude, and radius")
		}

		expression, args := postgis.CirclePolygonExpression(
			*entity.Longitude,
			*entity.Latitude,
			*entity.GeofenceRadiusMeters,
		)

		if insertQuery != nil {
			insertQuery.Value("geofence_geometry", expression, args...)
		}
		if updateQuery != nil {
			updateQuery.Set("geofence_geometry = "+expression, args...)
		}

		return nil
	case geofence.TypeRectangle, geofence.TypeDraw:
		geometry, err := entity.GeofencePolygon()
		if err != nil {
			return err
		}

		geometryJSON, err := geometry.GeoJSONString()
		if err != nil {
			return err
		}

		if insertQuery != nil {
			if err = dbdialect.RequireFromBun(insertQuery.DB(), dbdialect.CapPostGIS); err != nil {
				return err
			}
			insertQuery.Value(
				"geofence_geometry",
				"ST_SetSRID(ST_GeomFromGeoJSON(?), 4326)",
				geometryJSON,
			)
		}
		if updateQuery != nil {
			if err = dbdialect.RequireFromBun(updateQuery.DB(), dbdialect.CapPostGIS); err != nil {
				return err
			}
			updateQuery.Set(
				"geofence_geometry = ST_SetSRID(ST_GeomFromGeoJSON(?), 4326)",
				geometryJSON,
			)
		}

		return nil
	default:
		return fmt.Errorf("unsupported location geofence type %q", entity.GeofenceType)
	}
}

var locationWritableColumns = []string{
	"location_category_id",
	"state_id",
	"status",
	"code",
	"name",
	"description",
	"address_line_1",
	"address_line_2",
	"city",
	"postal_code",
	"place_id",
	"is_geocoded",
	"longitude",
	"latitude",
	"geofence_type",
	"geofence_radius_meters",
	"version",
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetLocationByIDRequest,
) (*location.Location, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*location.Location, error) {
		log := r.l.With(
			zap.String("operation", "GetByID"),
			zap.String("id", req.ID.String()),
		)

		relations := buncolgen.LocationRelations

		entity := new(location.Location)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Apply(WithGeofenceGeometry).
			Relation(relations.State).
			Relation(relations.LocationCategory).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.LocationScopeTenant(sq, req.TenantInfo).
					Where(buncolgen.LocationColumns.ID.Eq(), req.ID)
			}).
			Scan(ctx)
		if err != nil {
			log.Error("failed to get location", zap.Error(err))
			return nil, dberror.HandleNotFoundError(err, "Location")
		}
		if err = entity.PopulateGeofenceVertices(); err != nil {
			log.Error("failed to hydrate location geofence", zap.Error(err))
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
		log := r.l.With(
			zap.String("operation", "Create"),
		)

		err := quotatx.Run(ctx, r.db, r.quota, func(c context.Context) error {
			query := r.db.DBForContext(c).NewInsert().Model(entity)
			if geofenceErr := applyLocationGeofence(query, nil, entity); geofenceErr != nil {
				log.Error("failed to prepare geofence for location insert", zap.Error(geofenceErr))
				return geofenceErr
			}

			if _, insertErr := query.Exec(c); insertErr != nil {
				log.Error("failed to create location", zap.Error(insertErr))
				return insertErr
			}

			return nil
		}, services.QuotaRequest{
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

		return r.GetByID(ctx, repositories.GetLocationByIDRequest{
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
		log := r.l.With(
			zap.String("operation", "Update"),
			zap.String("id", entity.ID.String()),
		)

		ov := entity.Version
		entity.Version++

		query := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			Column(locationWritableColumns...).
			WherePK().
			Where(buncolgen.LocationColumns.Version.Eq(), ov)

		if err := applyLocationGeofence(nil, query, entity); err != nil {
			log.Error("failed to prepare geofence for location update", zap.Error(err))
			return nil, err
		}

		results, err := query.Exec(ctx)
		if err != nil {
			log.Error("failed to update location", zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckRowsAffected(results, "Location", entity.ID.String()); err != nil {
			return nil, err
		}

		return r.GetByID(ctx, repositories.GetLocationByIDRequest{
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
	req *repositories.BulkUpdateLocationStatusRequest,
) ([]*location.Location, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*location.Location, error) {
		log := r.l.With(
			zap.String("operation", "BulkUpdateStatus"),
			zap.Any("request", req),
		)

		entities := make([]*location.Location, 0, len(req.LocationIDs))
		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(&entities).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.LocationScopeTenantUpdate(uq, req.TenantInfo).
					Where(buncolgen.LocationColumns.ID.In(), bun.List(req.LocationIDs))
			}).
			Set(buncolgen.LocationColumns.Status.Set(), req.Status).
			Returning("*").
			Exec(ctx)
		if err != nil {
			log.Error("failed to bulk update location status", zap.Error(err))
			return nil, err
		}
		if err = location.HydrateGeofences(entities...); err != nil {
			log.Error("failed to hydrate location geofences", zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckBulkRowsAffected(results, "Location", req.LocationIDs); err != nil {
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req repositories.GetLocationsByIDsRequest,
) ([]*location.Location, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*location.Location, error) {
		log := r.l.With(
			zap.String("operation", "GetByIDs"),
			zap.Any("request", req),
		)

		entities := make([]*location.Location, 0, len(req.LocationIDs))
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Apply(WithGeofenceGeometry).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.LocationScopeTenant(sq, req.TenantInfo).
					Where(buncolgen.LocationColumns.ID.In(), bun.List(req.LocationIDs))
			}).
			Scan(ctx)
		if err != nil {
			log.Error("failed to get locations", zap.Error(err))
			return nil, dberror.HandleNotFoundError(err, "Location")
		}
		if err = location.HydrateGeofences(entities...); err != nil {
			log.Error("failed to hydrate location geofences", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.LocationSelectOptionsRequest,
) (*pagination.ListResult[*location.Location], error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*pagination.ListResult[*location.Location], error) {
		cols := buncolgen.LocationColumns

		return dbhelper.SelectOptions[*location.Location](
			ctx,
			r.db.DBForContext(ctx),
			req.SelectQueryRequest,
			&dbhelper.SelectOptionsConfig{
				ColumnRefs: []buncolgen.Column{
					cols.ID,
					cols.Code,
					cols.Name,
					cols.Description,
					cols.Status,
					cols.AddressLine1,
					cols.City,
					cols.StateID,
					cols.PostalCode,
				},
				OrgColumnRef: &cols.OrganizationID,
				BuColumnRef:  &cols.BusinessUnitID,
				QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
					return q.Where(cols.Status.Eq(), domaintypes.StatusActive).
						Relation(buncolgen.LocationRelations.State)
				},
				EntityName: "Location",
				SearchColumnRefs: []buncolgen.Column{
					cols.Code,
					cols.Name,
					cols.Description,
				},
			},
		)
	})
}
