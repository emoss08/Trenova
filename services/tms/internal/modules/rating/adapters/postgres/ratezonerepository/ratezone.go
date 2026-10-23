package ratezonerepository

import (
	"cmp"
	"context"
	"slices"

	"github.com/emoss08/trenova/pkg/buncolgen/ratezonecol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratezone"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratezoneports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
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

func New(p Params) ratezoneports.RateZoneRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.ratezone-repository"),
	}
}

func orderMembers(sq *bun.SelectQuery) *bun.SelectQuery {
	return sq.Order(ratezonecol.RateZoneMemberColumns.MatchKey.OrderAsc())
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *ratezoneports.ListRateZoneRequest,
) *bun.SelectQuery {
	cols := ratezonecol.RateZoneColumns
	q = querybuilder.ApplyFilters(
		q,
		ratezonecol.RateZoneTable.Alias,
		req.Filter,
		(*ratezone.RateZone)(nil),
	)

	return q.Apply(ratezonecol.RateZoneApplyTenant(req.Filter.TenantInfo)).
		Limit(int64(req.Filter.Pagination.SafeLimit())).
		Offset(int64(req.Filter.Pagination.SafeOffset())).
		Order(cols.CreatedAt.OrderDesc())
}

func (r *repository) List(
	ctx context.Context,
	req *ratezoneports.ListRateZoneRequest,
) (*pagination.ListResult[*ratezone.RateZone], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*ratezone.RateZone], error) {
			log := r.l.With(zap.String("operation", "List"))

			entities := make([]*ratezone.RateZone, 0, req.Filter.Pagination.SafeLimit())
			total, err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
					return r.filterQuery(sq, req)
				}).
				ScanAndCount(ctx)
			if err != nil {
				log.Error("failed to scan and count rate zones", zap.Error(err))
				return nil, err
			}

			return &pagination.ListResult[*ratezone.RateZone]{
				Items: entities,
				Total: int(total),
			}, nil
		},
	)
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *ratezoneports.ListRateZoneConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q,
		ratezonecol.RateZoneTable.Alias,
		req.Filter,
		req.Cursor,
		(*ratezone.RateZone)(nil),
	)
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *ratezoneports.ListRateZoneConnectionRequest,
) *bun.SelectQuery {
	return querybuilder.ApplyFiltersWithoutSort(
		q,
		ratezonecol.RateZoneTable.Alias,
		req.Filter,
		(*ratezone.RateZone)(nil),
	)
}

func applyRateZoneColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(ratezonecol.RateZoneTable.All())
	}

	return q.Column(columns...)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *ratezoneports.ListRateZoneConnectionRequest,
) (*pagination.CursorListResult[*ratezone.RateZone], error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*ratezone.RateZone], error) {
			log := r.l.With(zap.String("operation", "ListConnection"))

			dba := r.db.DBForContext(ctx)
			var totalCount *int
			if req.Cursor.IncludeTotalCount {
				total, err := dba.
					NewSelect().
					Model((*ratezone.RateZone)(nil)).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return r.applyTotalCountFilters(sq, req)
					}).
					Count(ctx)
				if err != nil {
					log.Error("failed to count rate zones", zap.Error(err))
					return nil, err
				}
				totalCount = new(int(total))
			}

			result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*ratezone.RateZone]{
				Filter:     req.Filter,
				Cursor:     req.Cursor,
				TotalCount: totalCount,
				Query: func(entities *[]*ratezone.RateZone) *bun.SelectQuery {
					return dba.
						NewSelect().
						Model(entities).
						Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
							return applyRateZoneColumns(sq, req.RateZoneColumns)
						})
				},
				Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
					return r.applyCursorPageFilters(sq, req)
				},
			})
			if err != nil {
				log.Error("failed to scan rate zones", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req *ratezoneports.GetRateZoneByIDRequest,
) (*ratezone.RateZone, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ratezone.RateZone, error) {
		log := r.l.With(
			zap.String("operation", "GetByID"),
			zap.String("id", req.RateZoneID.String()),
		)

		entity := new(ratezone.RateZone)
		cols := ratezonecol.RateZoneColumns

		q := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return ratezonecol.RateZoneScopeTenant(sq, req.TenantInfo).
					Where(cols.ID.Eq(), req.RateZoneID)
			})

		if req.IncludeMembers {
			q = q.Relation(buncolgen.Rel(ratezonecol.RateZoneRelations.Members), orderMembers)
		}

		if err := q.Scan(ctx); err != nil {
			log.Error("failed to get rate zone", zap.Error(err))
			return nil, dberror.HandleNotFoundError(err, "RateZone")
		}

		return entity, nil
	})
}

type laneZoneRow struct {
	ID       pulid.ID `bun:"id"`
	Code     string   `bun:"code"`
	MatchKey string   `bun:"match_key"`
}

func (r *repository) ResolveLaneZones(
	ctx context.Context,
	req *ratezoneports.ResolveLaneZonesRequest,
) (*ratezoneports.LaneZones, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ratezoneports.LaneZones, error) {
		keys := make([]string, 0, len(req.OriginKeys)+len(req.DestinationKeys))
		keys = append(keys, req.OriginKeys...)
		keys = append(keys, req.DestinationKeys...)
		if len(keys) == 0 {
			return &ratezoneports.LaneZones{}, nil
		}

		memberCols := ratezonecol.RateZoneMemberColumns
		zoneCols := ratezonecol.RateZoneColumns

		rows := make([]laneZoneRow, 0, len(keys))
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*ratezone.RateZoneMember)(nil)).
			ColumnExpr("DISTINCT "+zoneCols.ID.Qualified()+" AS id").
			ColumnExpr(zoneCols.Code.Qualified()+" AS code").
			ColumnExpr(memberCols.MatchKey.Qualified()+" AS match_key").
			Join(
				"JOIN "+ratezonecol.RateZoneTable.As(ratezonecol.RateZoneTable.Alias)+
					" ON "+zoneCols.ID.Qualified()+" = "+memberCols.RateZoneID.Qualified()+
					" AND "+zoneCols.OrganizationID.Qualified()+" = "+memberCols.OrganizationID.Qualified()+
					" AND "+zoneCols.BusinessUnitID.Qualified()+" = "+memberCols.BusinessUnitID.Qualified(),
			).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return ratezonecol.RateZoneMemberScopeTenant(sq, req.TenantInfo).
					Where(memberCols.MatchKey.In(), bun.List(keys)).
					Where(zoneCols.Status.Eq(), domaintypes.StatusActive)
			}).
			Scan(ctx, &rows)
		if err != nil {
			r.l.With(zap.String("operation", "ResolveLaneZones")).
				Error("failed to resolve lane zones", zap.Error(err))
			return nil, err
		}

		return &ratezoneports.LaneZones{
			Origin:      laneSide(rows, req.OriginKeys),
			Destination: laneSide(rows, req.DestinationKeys),
		}, nil
	})
}

func laneSide(rows []laneZoneRow, keys []string) []ratezoneports.ZoneRef {
	rank := make(map[string]int, len(keys))
	for index, key := range keys {
		if _, seen := rank[key]; !seen {
			rank[key] = index
		}
	}

	best := make(map[pulid.ID]int, len(rows))
	refs := make([]ratezoneports.ZoneRef, 0, len(rows))
	for _, row := range rows {
		position, ok := rank[row.MatchKey]
		if !ok {
			continue
		}
		current, seen := best[row.ID]
		if !seen {
			refs = append(refs, ratezoneports.ZoneRef{ID: row.ID, Code: row.Code})
			best[row.ID] = position
			continue
		}
		if position < current {
			best[row.ID] = position
		}
	}

	slices.SortStableFunc(refs, func(a, b ratezoneports.ZoneRef) int {
		return cmp.Or(cmp.Compare(best[a.ID], best[b.ID]), cmp.Compare(a.Code, b.Code))
	})
	return refs
}

func (r *repository) GetCodesByIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]string, error) {
		codes := make(map[pulid.ID]string, len(ids))
		if len(ids) == 0 {
			return codes, nil
		}

		zoneCols := ratezonecol.RateZoneColumns
		zones := make([]*ratezone.RateZone, 0, len(ids))
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&zones).
			Column(zoneCols.ID.Bare(), zoneCols.Code.Bare()).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return ratezonecol.RateZoneScopeTenant(sq, tenantInfo).
					Where(zoneCols.ID.In(), bun.List(ids))
			}).
			Scan(ctx)
		if err != nil {
			r.l.With(zap.String("operation", "GetCodesByIDs")).
				Error("failed to read rate zone codes", zap.Error(err))
			return nil, err
		}

		for _, zone := range zones {
			codes[zone.ID] = zone.Code
		}
		return codes, nil
	})
}

func (r *repository) insertMembers(ctx context.Context, entity *ratezone.RateZone) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(entity.Members) == 0 {
			return nil
		}

		_, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&entity.Members).
			Returning("*").
			Exec(ctx)

		return err
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *ratezone.RateZone,
) (*ratezone.RateZone, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ratezone.RateZone, error) {
		log := r.l.With(zap.String("operation", "Create"), zap.String("code", entity.Code))

		err := r.db.WithTx(ctx, kernelports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
			if _, iErr := r.db.DBForContext(c).
				NewInsert().
				Model(entity).
				Returning("*").
				Exec(c); iErr != nil {
				return iErr
			}

			stampMembers(entity, false)

			return r.insertMembers(c, entity)
		})
		if err != nil {
			log.Error("failed to create rate zone", zap.Error(err))
			return nil, dberror.MapRetryableTransactionError(
				err,
				"Rate zone is busy. Retry the request.",
			)
		}

		return entity, nil
	})
}

// Update replaces the member set wholesale.
//
// Members carry no identity a user would recognise and nothing points at them,
// so rebuilding the set is simpler than diffing it and cannot leave a stale row
// behind to keep matching a place the zone no longer covers.
func (r *repository) Update(
	ctx context.Context,
	entity *ratezone.RateZone,
) (*ratezone.RateZone, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ratezone.RateZone, error) {
		log := r.l.With(
			zap.String("operation", "Update"),
			zap.String("id", entity.ID.String()),
		)

		ov := entity.Version
		entity.Version++

		zoneCols := ratezonecol.RateZoneColumns
		memberCols := ratezonecol.RateZoneMemberColumns

		err := r.db.WithTx(ctx, kernelports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
			results, uErr := r.db.DBForContext(c).
				NewUpdate().
				Model(entity).
				WherePK().
				Where(zoneCols.Version.Eq(), ov).
				OmitZero().
				Returning("*").
				Exec(c)
			if uErr != nil {
				return uErr
			}

			if uErr = dberror.CheckRowsAffected(
				results,
				"RateZone",
				entity.ID.String(),
			); uErr != nil {
				return uErr
			}

			if _, dErr := r.db.DBForContext(c).
				NewDelete().
				Model((*ratezone.RateZoneMember)(nil)).
				WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
					return ratezonecol.RateZoneMemberScopeTenantDelete(dq, pagination.TenantInfo{
						OrgID: entity.OrganizationID,
						BuID:  entity.BusinessUnitID,
					}).Where(memberCols.RateZoneID.Eq(), entity.ID)
				}).
				Exec(c); dErr != nil {
				return dErr
			}

			stampMembers(entity, true)

			return r.insertMembers(c, entity)
		})
		if err != nil {
			log.Error("failed to update rate zone", zap.Error(err))
			return nil, dberror.MapRetryableTransactionError(
				err,
				"Rate zone is busy. Retry the request.",
			)
		}

		return entity, nil
	})
}

func (r *repository) Delete(
	ctx context.Context,
	req *ratezoneports.GetRateZoneByIDRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		log := r.l.With(
			zap.String("operation", "Delete"),
			zap.String("id", req.RateZoneID.String()),
		)

		cols := ratezonecol.RateZoneColumns
		results, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*ratezone.RateZone)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return ratezonecol.RateZoneScopeTenantDelete(dq, req.TenantInfo).
					Where(cols.ID.Eq(), req.RateZoneID)
			}).
			Exec(ctx)
		if err != nil {
			log.Error("failed to delete rate zone", zap.Error(err))
			return err
		}

		return dberror.CheckRowsAffected(results, "RateZone", req.RateZoneID.String())
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*ratezone.RateZone], error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*ratezone.RateZone], error) {
			cols := ratezonecol.RateZoneColumns

			return dbhelper.SelectOptions[*ratezone.RateZone](
				ctx,
				r.db.DBForContext(ctx),
				req,
				&dbhelper.SelectOptionsConfig{
					ColumnRefs: []buncolgen.Column{
						cols.ID,
						cols.Code,
						cols.Name,
						cols.Description,
						cols.Kind,
						cols.Status,
					},
					OrgColumnRef:     &cols.OrganizationID,
					BuColumnRef:      &cols.BusinessUnitID,
					SearchColumnRefs: []buncolgen.Column{cols.Code, cols.Name},
					EntityName:       "RateZone",
					QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
						return q.Where(cols.Status.Eq(), domaintypes.StatusActive)
					},
				},
			)
		},
	)
}
