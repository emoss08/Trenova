package ratezonerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ratezone"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
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

func New(p Params) repositories.RateZoneRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.ratezone-repository"),
	}
}

func orderMembers(sq *bun.SelectQuery) *bun.SelectQuery {
	return sq.Order(buncolgen.RateZoneMemberColumns.MatchKey.OrderAsc())
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListRateZoneRequest,
) *bun.SelectQuery {
	cols := buncolgen.RateZoneColumns
	q = querybuilder.ApplyFilters(
		q,
		buncolgen.RateZoneTable.Alias,
		req.Filter,
		(*ratezone.RateZone)(nil),
	)

	return q.Apply(buncolgen.RateZoneApplyTenant(req.Filter.TenantInfo)).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		Order(cols.CreatedAt.OrderDesc())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListRateZoneRequest,
) (*pagination.ListResult[*ratezone.RateZone], error) {
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

	return &pagination.ListResult[*ratezone.RateZone]{Items: entities, Total: total}, nil
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *repositories.ListRateZoneConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q,
		buncolgen.RateZoneTable.Alias,
		req.Filter,
		req.Cursor,
		(*ratezone.RateZone)(nil),
	)
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *repositories.ListRateZoneConnectionRequest,
) *bun.SelectQuery {
	return querybuilder.ApplyFiltersWithoutSort(
		q,
		buncolgen.RateZoneTable.Alias,
		req.Filter,
		(*ratezone.RateZone)(nil),
	)
}

func applyRateZoneColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(buncolgen.RateZoneTable.All())
	}

	return q.Column(columns...)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListRateZoneConnectionRequest,
) (*pagination.CursorListResult[*ratezone.RateZone], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
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

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*ratezone.RateZone]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: &total,
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
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetRateZoneByIDRequest,
) (*ratezone.RateZone, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.RateZoneID.String()),
	)

	entity := new(ratezone.RateZone)
	cols := buncolgen.RateZoneColumns

	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateZoneScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.RateZoneID)
		})

	if req.IncludeMembers {
		q = q.Relation(buncolgen.Rel(buncolgen.RateZoneRelations.Members), orderMembers)
	}

	if err := q.Scan(ctx); err != nil {
		log.Error("failed to get rate zone", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "RateZone")
	}

	return entity, nil
}

// ResolveMembership turns a place's match keys into the zones that contain it.
//
// This runs twice on every rating, once per end of the lane, so it is served
// entirely from the covering index on (organization, business unit, match key)
// and returns nothing but the zone ids.
func (r *repository) ResolveMembership(
	ctx context.Context,
	req *repositories.ResolveZoneMembershipRequest,
) ([]pulid.ID, error) {
	if len(req.MatchKeys) == 0 {
		return nil, nil
	}

	log := r.l.With(zap.String("operation", "ResolveMembership"))

	memberCols := buncolgen.RateZoneMemberColumns
	zoneCols := buncolgen.RateZoneColumns

	zoneIDs := make([]pulid.ID, 0, len(req.MatchKeys))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*ratezone.RateZoneMember)(nil)).
		ColumnExpr("DISTINCT "+memberCols.RateZoneID.String()).
		Join(
			"JOIN "+buncolgen.RateZoneTable.As(buncolgen.RateZoneTable.Alias)+
				" ON "+zoneCols.ID.Qualified()+" = "+memberCols.RateZoneID.Qualified()+
				" AND "+zoneCols.OrganizationID.Qualified()+" = "+memberCols.OrganizationID.Qualified()+
				" AND "+zoneCols.BusinessUnitID.Qualified()+" = "+memberCols.BusinessUnitID.Qualified(),
		).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateZoneMemberScopeTenant(sq, req.TenantInfo).
				Where(memberCols.MatchKey.In(), bun.List(req.MatchKeys)).
				Where(zoneCols.Status.Eq(), domaintypes.StatusActive)
		}).
		Scan(ctx, &zoneIDs)
	if err != nil {
		log.Error("failed to resolve zone membership", zap.Error(err))
		return nil, err
	}

	return zoneIDs, nil
}

// stampMembers re-seats the tenancy and parent on every member.
//
// None of those three are the caller's to supply: a payload that named a
// different zone would move a member between zones as a side effect of editing
// the one it arrived under.
func stampMembers(entity *ratezone.RateZone, resetIDs bool) {
	for _, member := range entity.Members {
		if member == nil {
			continue
		}

		if resetIDs {
			member.ID = pulid.Nil
		}

		member.RateZoneID = entity.ID
		member.OrganizationID = entity.OrganizationID
		member.BusinessUnitID = entity.BusinessUnitID
		member.ApplyMatchKey()
	}
}

func (r *repository) insertMembers(ctx context.Context, entity *ratezone.RateZone) error {
	if len(entity.Members) == 0 {
		return nil
	}

	_, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(&entity.Members).
		Returning("*").
		Exec(ctx)

	return err
}

func (r *repository) Create(
	ctx context.Context,
	entity *ratezone.RateZone,
) (*ratezone.RateZone, error) {
	log := r.l.With(zap.String("operation", "Create"), zap.String("code", entity.Code))

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
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
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	zoneCols := buncolgen.RateZoneColumns
	memberCols := buncolgen.RateZoneMemberColumns

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
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

		if uErr = dberror.CheckRowsAffected(results, "RateZone", entity.ID.String()); uErr != nil {
			return uErr
		}

		if _, dErr := r.db.DBForContext(c).
			NewDelete().
			Model((*ratezone.RateZoneMember)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.RateZoneMemberScopeTenantDelete(dq, pagination.TenantInfo{
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
}

func (r *repository) Delete(
	ctx context.Context,
	req *repositories.GetRateZoneByIDRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.RateZoneID.String()),
	)

	cols := buncolgen.RateZoneColumns
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*ratezone.RateZone)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.RateZoneScopeTenantDelete(dq, req.TenantInfo).
				Where(cols.ID.Eq(), req.RateZoneID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete rate zone", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(results, "RateZone", req.RateZoneID.String())
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*ratezone.RateZone], error) {
	cols := buncolgen.RateZoneColumns

	return dbhelper.SelectOptions[*ratezone.RateZone](
		ctx,
		r.db.DB(),
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
}
