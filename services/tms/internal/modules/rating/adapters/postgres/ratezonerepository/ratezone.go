package ratezonerepository

import (
	"cmp"
	"context"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratezone"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratezone/ratezonedb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratezoneports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const busyMessage = "Rate zone is busy. Retry the request."

var rateZoneErrors = dberror.Mapping{
	Entity: "RateZone",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_rate_zones_code",
			Field:   "code",
			Message: "A rate zone with this code already exists in your organization",
		},
		{
			Name:    "uq_rate_zone_members",
			Field:   "members",
			Message: "A place can appear only once in a rate zone",
		},
	},
}

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

func mapRateZoneError(err error, id string) error {
	return dberror.MapRetryableTransactionError(rateZoneErrors.Map(err, id), busyMessage)
}

func (r *repository) List(
	ctx context.Context,
	req *ratezoneports.ListRateZoneRequest,
) (*pagination.ListResult[*ratezone.RateZone], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*ratezone.RateZone], error) {
			rz := ratezonedb.RateZones
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				rz.Query(),
				&listquery.Request[ratezone.RateZone]{
					Filter: req.Filter,
					Then:   []keel.Order{rz.CreatedAt.Desc()},
				},
			)
			if err != nil {
				r.l.Error("failed to list rate zones", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *ratezoneports.ListRateZoneConnectionRequest,
) (*pagination.CursorListResult[*ratezone.RateZone], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*ratezone.RateZone], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				ratezonedb.RateZones.Query(),
				&listquery.Request[ratezone.RateZone]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.RateZoneColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list rate zones", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func zoneQuery(req *ratezoneports.GetRateZoneByIDRequest) *keel.Query[ratezone.RateZone] {
	rz := ratezonedb.RateZones
	q := rz.Where(rz.ID.Eq(req.RateZoneID))
	if req.IncludeMembers {
		members := rz.Members()
		q = q.Load(members.Order(members.MatchKey.Asc()))
	}

	return q
}

func (r *repository) GetByID(
	ctx context.Context,
	req *ratezoneports.GetRateZoneByIDRequest,
) (*ratezone.RateZone, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ratezone.RateZone, error) {
		entity, err := zoneQuery(req).One(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get rate zone",
				zap.String("id", req.RateZoneID.String()),
				zap.Error(err),
			)
			return nil, rateZoneErrors.Map(err, req.RateZoneID.String())
		}

		return entity, nil
	})
}

type laneZoneRow struct {
	ID       pulid.ID
	Code     string
	MatchKey string
}

func laneZonesQuery(keys []string) *keel.Query[ratezone.RateZoneMember] {
	m := ratezonedb.RateZoneMembers
	rz := ratezonedb.RateZones

	return m.Join(rz, rz.ID.EqCol(m.RateZoneID)).
		Where(m.MatchKey.In(keys...), rz.Status.Eq(domaintypes.StatusActive)).
		Group(rz.ID, rz.Code, m.MatchKey).
		Pick(rz.ID, rz.Code, m.MatchKey)
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

		rows, err := keel.Into[laneZoneRow](ctx, r.db.Keel(ctx), laneZonesQuery(keys))
		if err != nil {
			r.l.Error("failed to resolve lane zones", zap.Error(err))
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
	_ pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]string, error) {
		codes := make(map[pulid.ID]string, len(ids))
		if len(ids) == 0 {
			return codes, nil
		}

		rz := ratezonedb.RateZones
		zones, err := rz.Pick(rz.ID, rz.Code).
			Where(rz.ID.In(ids...)).
			Limit(len(ids)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to read rate zone codes", zap.Error(err))
			return nil, err
		}

		for _, zone := range zones {
			codes[zone.ID] = zone.Code
		}

		return codes, nil
	})
}

func (r *repository) insertMembers(
	ctx context.Context,
	db *keel.DB,
	entity *ratezone.RateZone,
) error {
	members := make([]*ratezone.RateZoneMember, 0, len(entity.Members))
	for _, member := range entity.Members {
		if member != nil {
			members = append(members, member)
		}
	}
	if len(members) == 0 {
		return nil
	}

	return ratezonedb.RateZoneMembers.CreateMany(ctx, db, members)
}

func (r *repository) Create(
	ctx context.Context,
	entity *ratezone.RateZone,
) (*ratezone.RateZone, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ratezone.RateZone, error) {
		db := r.db.Keel(ctx)
		err := ratezonedb.RateZones.Create(ctx, db, entity)
		if err == nil {
			stampMembers(entity, false)
			err = r.insertMembers(ctx, db, entity)
		}
		if err != nil {
			r.l.Error("failed to create rate zone",
				zap.String("code", entity.Code),
				zap.Error(err),
			)
			return nil, mapRateZoneError(err, entity.ID.String())
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
		db := r.db.Keel(ctx)
		m := ratezonedb.RateZoneMembers

		err := ratezonedb.RateZones.Save(ctx, db, entity)
		if err == nil {
			_, err = m.Where(m.RateZoneID.Eq(entity.ID)).Delete(ctx, db)
		}
		if err == nil {
			stampMembers(entity, true)
			err = r.insertMembers(ctx, db, entity)
		}
		if err != nil {
			r.l.Error("failed to update rate zone",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, mapRateZoneError(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Delete(
	ctx context.Context,
	req *ratezoneports.GetRateZoneByIDRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		err := ratezonedb.RateZones.DeleteRow(
			ctx,
			r.db.Keel(ctx),
			&ratezone.RateZone{ID: req.RateZoneID},
		)
		if err != nil {
			r.l.Error("failed to delete rate zone",
				zap.String("id", req.RateZoneID.String()),
				zap.Error(err),
			)
			return rateZoneErrors.Map(err, req.RateZoneID.String())
		}

		return nil
	})
}

func selectOptionsQuery() *keel.Query[ratezone.RateZone] {
	rz := ratezonedb.RateZones

	return rz.Pick(rz.ID, rz.Code, rz.Name, rz.Description, rz.Kind, rz.Status).
		Where(rz.Status.Eq(domaintypes.StatusActive))
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*ratezone.RateZone], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*ratezone.RateZone], error) {
			rz := ratezonedb.RateZones

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				selectOptionsQuery(),
				req,
				func(s string) keel.Pred {
					return keel.Or(rz.Code.ContainsFold(s), rz.Name.ContainsFold(s))
				},
			)
		},
	)
}
