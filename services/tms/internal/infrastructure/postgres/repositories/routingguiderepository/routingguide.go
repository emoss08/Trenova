package routingguiderepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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

func New(p Params) repositories.RoutingGuideRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.routing-guide-repository"),
	}
}

func (r *repository) addOptions(
	q *bun.SelectQuery,
	opts repositories.RoutingGuideFilterOptions,
) *bun.SelectQuery {
	if opts.IncludeEntries {
		q = q.Relation("Entries", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Order(buncolgen.RoutingGuideEntryColumns.Rank.OrderAsc())
		})
		q = q.Relation("Entries.Carrier")
	}

	return q
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListRoutingGuideRequest,
) (*pagination.ListResult[*tender.RoutingGuide], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*tender.RoutingGuide, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = querybuilder.ApplyFilters(
				sq,
				buncolgen.RoutingGuideTable.Alias,
				req.Filter,
				(*tender.RoutingGuide)(nil),
			)
			return r.addOptions(sq, req.RoutingGuideFilterOptions).
				Limit(req.Filter.Pagination.SafeLimit()).
				Offset(req.Filter.Pagination.SafeOffset())
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count routing guides", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*tender.RoutingGuide]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetRoutingGuideByIDRequest,
) (*tender.RoutingGuide, error) {
	cols := buncolgen.RoutingGuideColumns
	entity := new(tender.RoutingGuide)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, req.RoutingGuideFilterOptions)
		}).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RoutingGuideScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to get routing guide", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Routing guide")
	}

	return entity, nil
}

// stampEntries pins entries to the guide and tenant. IDs are left alone: kept
// rows retain their identity; BeforeAppendModel mints IDs for new rows only.
func (r *repository) stampEntries(entity *tender.RoutingGuide) {
	for _, entry := range entity.Entries {
		entry.RoutingGuideID = entity.ID
		entry.OrganizationID = entity.OrganizationID
		entry.BusinessUnitID = entity.BusinessUnitID
	}
}

func (r *repository) insertEntries(ctx context.Context, entity *tender.RoutingGuide) error {
	if len(entity.Entries) == 0 {
		return nil
	}

	_, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(&entity.Entries).
		Returning("*").
		Exec(ctx)
	return err
}

func (r *repository) deleteMissingEntries(ctx context.Context, entity *tender.RoutingGuide) error {
	kept := make([]pulid.ID, 0, len(entity.Entries))
	for _, entry := range entity.Entries {
		if entry != nil && !entry.ID.IsNil() {
			kept = append(kept, entry.ID)
		}
	}

	_, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*tender.RoutingGuideEntry)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			dq = buncolgen.RoutingGuideEntryScopeTenantDelete(dq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).Where(buncolgen.RoutingGuideEntryColumns.RoutingGuideID.Eq(), entity.ID)
			if len(kept) > 0 {
				dq = dq.Where(buncolgen.RoutingGuideEntryColumns.ID.NotIn(), bun.List(kept))
			}
			return dq
		}).
		Exec(ctx)
	return err
}

// upsertEntries inserts new (ID-less) rows and updates kept ones in place,
// preserving row identity across guide edits.
func (r *repository) upsertEntries(ctx context.Context, entity *tender.RoutingGuide) error {
	if len(entity.Entries) == 0 {
		return nil
	}

	_, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(&entity.Entries).
		On("CONFLICT (id, business_unit_id, organization_id) DO UPDATE").
		Set("carrier_id = EXCLUDED.carrier_id").
		Set("rank = EXCLUDED.rank").
		Set("rate_method = EXCLUDED.rate_method").
		Set("rate = EXCLUDED.rate").
		Set("offer_ttl_seconds = EXCLUDED.offer_ttl_seconds").
		Set("channel = EXCLUDED.channel").
		Set("version = rge.version + 1").
		Set("updated_at = ?", timeutils.NowUnix()).
		Returning("*").
		Exec(ctx)
	return err
}

func (r *repository) Create(
	ctx context.Context,
	entity *tender.RoutingGuide,
) (*tender.RoutingGuide, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("name", entity.Name),
	)

	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error("failed to create routing guide", zap.Error(err))
		return nil, err
	}

	r.stampEntries(entity)
	if err := r.insertEntries(ctx, entity); err != nil {
		log.Error("failed to insert routing guide entries", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *tender.RoutingGuide,
) (*tender.RoutingGuide, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update routing guide", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "Routing guide", entity.ID.String()); err != nil {
		return nil, err
	}

	if err = r.deleteMissingEntries(ctx, entity); err != nil {
		log.Error("failed to delete routing guide entries", zap.Error(err))
		return nil, err
	}

	r.stampEntries(entity)
	if err = r.upsertEntries(ctx, entity); err != nil {
		log.Error("failed to upsert routing guide entries", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	req repositories.DeleteRoutingGuideRequest,
) error {
	cols := buncolgen.RoutingGuideColumns
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*tender.RoutingGuide)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.RoutingGuideScopeTenantDelete(dq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to delete routing guide", zap.Error(err))
		return err
	}

	return dberror.CheckRowsAffected(results, "Routing guide", req.ID.String())
}

func (r *repository) matchTier(
	ctx context.Context,
	req *repositories.MatchLaneRequest,
	specificity int16,
	apply func(*bun.SelectQuery) *bun.SelectQuery,
) (*tender.RoutingGuide, error) {
	cols := buncolgen.RoutingGuideColumns
	entity := new(tender.RoutingGuide)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("Entries", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Order(buncolgen.RoutingGuideEntryColumns.Rank.OrderAsc())
		}).
		Relation("Entries.Carrier").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.RoutingGuideScopeTenant(sq, req.TenantInfo).
				Where(cols.Status.Eq(), domaintypes.StatusActive).
				Where(cols.Specificity.Eq(), specificity)
			return apply(sq)
		}).
		Order(cols.CreatedAt.OrderAsc()).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // nil guide represents an optional absence
		}
		r.l.Error("failed to match routing guide tier", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) MatchLane(
	ctx context.Context,
	req *repositories.MatchLaneRequest,
) (*tender.RoutingGuide, error) {
	cols := buncolgen.RoutingGuideColumns

	if !req.OriginLocationID.IsNil() && !req.DestinationLocationID.IsNil() {
		guide, err := r.matchTier(ctx, req, tender.SpecificityLocationPair,
			func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.
					Where(cols.OriginLocationID.Eq(), req.OriginLocationID).
					Where(cols.DestinationLocationID.Eq(), req.DestinationLocationID)
			})
		if err != nil || guide != nil {
			return guide, err
		}
	}

	if req.OriginCity != "" && req.OriginState != "" &&
		req.DestinationCity != "" && req.DestinationState != "" {
		guide, err := r.matchTier(ctx, req, tender.SpecificityCityPair,
			func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.
					Where("LOWER(rgd.origin_city) = ?", strings.ToLower(req.OriginCity)).
					Where(cols.OriginState.Eq(), req.OriginState).
					Where("LOWER(rgd.destination_city) = ?", strings.ToLower(req.DestinationCity)).
					Where(cols.DestinationState.Eq(), req.DestinationState)
			})
		if err != nil || guide != nil {
			return guide, err
		}
	}

	if req.OriginState != "" && req.DestinationState != "" {
		guide, err := r.matchTier(ctx, req, tender.SpecificityStatePair,
			func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.
					Where(cols.OriginState.Eq(), req.OriginState).
					Where(cols.DestinationState.Eq(), req.DestinationState)
			})
		if err != nil || guide != nil {
			return guide, err
		}
	}

	return nil, nil //nolint:nilnil // nil guide represents an optional absence
}
