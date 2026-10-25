package routingguiderepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender/tenderdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const rankShift = 10000

var guideErrors = dberror.Mapping{Entity: "Routing guide"}

var entryErrors = dberror.Mapping{Entity: "Routing guide entry"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) tenderports.RoutingGuideRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.routing-guide-repository"),
	}
}

type guideQuery = keel.Query[*tender.RoutingGuide]

func rankedEntries() keel.Loader {
	e := tenderdb.RoutingGuides.Entries()
	return e.Order(e.Rank.Asc()).Load(e.Carrier())
}

func withOptions(opts tenderports.RoutingGuideFilterOptions) func(*guideQuery) *guideQuery {
	return func(q *guideQuery) *guideQuery {
		if opts.IncludeEntries {
			return q.Load(rankedEntries())
		}

		return q
	}
}

func (r *repository) List(
	ctx context.Context,
	req *tenderports.ListRoutingGuideRequest,
) (*pagination.ListResult[*tender.RoutingGuide], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*tender.RoutingGuide], error) {
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				tenderdb.RoutingGuides.Query(),
				&listquery.Request[tender.RoutingGuide]{
					Filter: req.Filter,
					Shape:  withOptions(req.RoutingGuideFilterOptions),
				},
			)
			if err != nil {
				r.l.Error("failed to list routing guides", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *tenderports.ListRoutingGuideConnectionRequest,
) (*pagination.CursorListResult[*tender.RoutingGuide], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*tender.RoutingGuide], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				tenderdb.RoutingGuides.Query(),
				&listquery.Request[tender.RoutingGuide]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.RoutingGuideColumns,
					Shape:   withOptions(req.RoutingGuideFilterOptions),
				},
			)
			if err != nil {
				r.l.Error("failed to list routing guide connection", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req tenderports.GetRoutingGuideByIDRequest,
) (*tender.RoutingGuide, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tender.RoutingGuide, error) {
		rg := tenderdb.RoutingGuides
		entity, err := withOptions(req.RoutingGuideFilterOptions)(rg.Where(rg.ID.Eq(req.ID))).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get routing guide", zap.Error(err))
			return nil, guideErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req tenderports.GetRoutingGuidesByIDsRequest,
) ([]*tender.RoutingGuide, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*tender.RoutingGuide, error) {
		if len(req.RoutingGuideIDs) == 0 {
			return []*tender.RoutingGuide{}, nil
		}

		rg := tenderdb.RoutingGuides
		entities, err := withOptions(req.RoutingGuideFilterOptions)(
			rg.Where(rg.ID.In(req.RoutingGuideIDs...)),
		).
			Limit(len(req.RoutingGuideIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get routing guides by ids", zap.Error(err))
			return nil, fmt.Errorf("get routing guides by ids: %w", err)
		}

		return entities, nil
	})
}

func liveEntries(entity *tender.RoutingGuide) []*tender.RoutingGuideEntry {
	entries := make([]*tender.RoutingGuideEntry, 0, len(entity.Entries))
	for _, entry := range entity.Entries {
		if entry == nil {
			continue
		}
		entry.RoutingGuideID = entity.ID
		entry.OrganizationID = entity.OrganizationID
		entry.BusinessUnitID = entity.BusinessUnitID
		entries = append(entries, entry)
	}

	return entries
}

func keptEntryIDs(entity *tender.RoutingGuide) []pulid.ID {
	kept := make([]pulid.ID, 0, len(entity.Entries))
	for _, entry := range entity.Entries {
		if entry != nil && !entry.ID.IsNil() {
			kept = append(kept, entry.ID)
		}
	}
	return kept
}

// syncEntries makes the guide's entries exactly the payload's: rows it no
// longer carries are deleted, kept rows are updated in place (their identity and
// created_at survive the edit) and new rows are inserted. Kept rows move out of
// the live rank range first: uq_routing_guide_entries_rank is not deferrable, so
// a rank swap applied row by row would collide mid-statement. The +10000 offset
// stays within smallint bounds and above the rank >= 1 check.
func syncEntries(ctx context.Context, db *keel.DB, entity *tender.RoutingGuide) error {
	e := tenderdb.RoutingGuideEntries
	kept := keptEntryIDs(entity)

	if _, err := e.Where(
		e.RoutingGuideID.Eq(entity.ID),
		keel.When(len(kept) > 0, e.ID.NotIn(kept...)),
	).Delete(ctx, db); err != nil {
		return err
	}

	if len(kept) > 0 {
		if _, err := e.Where(e.RoutingGuideID.Eq(entity.ID), e.ID.In(kept...)).Update(
			ctx,
			db,
			e.Rank.Add(rankShift),
			e.Version.ToExpr(e.Version),
			e.UpdatedAt.ToExpr(e.UpdatedAt),
		); err != nil {
			return err
		}
	}

	entries := liveEntries(entity)
	if len(entries) == 0 {
		return nil
	}

	created := make([]int64, len(entries))
	for i, entry := range entries {
		created[i] = entry.CreatedAt
	}

	inserted, err := e.Upsert(
		ctx,
		db,
		entries,
		keel.OnConflict(e.ID, e.BusinessUnitID, e.OrganizationID).
			Update(e.CarrierID, e.Rank, e.RateMethod, e.Rate, e.OfferTTLSeconds, e.Channel),
	)
	if err != nil {
		return err
	}

	for i, entry := range entries {
		if !inserted[i] && created[i] != 0 {
			entry.CreatedAt = created[i]
		}
	}

	return nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *tender.RoutingGuide,
) (*tender.RoutingGuide, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tender.RoutingGuide, error) {
		db := r.db.Keel(ctx)
		if err := tenderdb.RoutingGuides.Create(ctx, db, entity); err != nil {
			r.l.Error(
				"failed to create routing guide",
				zap.String("name", entity.Name),
				zap.Error(err),
			)
			return nil, guideErrors.Map(err, entity.ID.String())
		}

		if err := tenderdb.RoutingGuideEntries.CreateMany(
			ctx,
			db,
			liveEntries(entity),
		); err != nil {
			r.l.Error("failed to create routing guide entries", zap.Error(err))
			return nil, entryErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *tender.RoutingGuide,
) (*tender.RoutingGuide, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*tender.RoutingGuide, error) {
		log := r.l.With(
			zap.String("operation", "Update"),
			zap.String("id", entity.ID.String()),
		)

		db := r.db.Keel(ctx)
		if err := tenderdb.RoutingGuides.Save(ctx, db, entity); err != nil {
			log.Error("failed to update routing guide", zap.Error(err))
			return nil, dberror.MapRetryableTransactionError(
				guideErrors.Map(err, entity.ID.String()),
				"Routing guide is busy. Retry the request.",
			)
		}

		if err := syncEntries(ctx, db, entity); err != nil {
			log.Error("failed to update routing guide entries", zap.Error(err))
			return nil, dberror.MapRetryableTransactionError(
				entryErrors.Map(err, entity.ID.String()),
				"Routing guide is busy. Retry the request.",
			)
		}

		return entity, nil
	})
}

func (r *repository) Delete(
	ctx context.Context,
	req tenderports.DeleteRoutingGuideRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		rg := tenderdb.RoutingGuides
		n, err := rg.Where(rg.ID.Eq(req.ID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to delete routing guide", zap.Error(err))
			return guideErrors.Map(err, req.ID.String())
		}

		return dberror.CheckAffected(n, "Routing guide", req.ID.String())
	})
}

func (r *repository) matchTier(
	ctx context.Context,
	specificity int16,
	preds ...keel.Pred,
) (*tender.RoutingGuide, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tender.RoutingGuide, error) {
		rg := tenderdb.RoutingGuides
		entity, err := rg.Where(
			rg.Status.Eq(domaintypes.StatusActive),
			rg.Specificity.Eq(specificity),
		).
			Where(preds...).
			Load(rankedEntries()).
			Order(rg.CreatedAt.Asc()).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil //nolint:nilnil // nil guide represents an optional absence
			}
			r.l.Error("failed to match routing guide tier", zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

func upperEq(col keel.StringCol, value string) keel.Pred {
	return col.Upper().Eq(strings.ToUpper(value))
}

func lowerEq(col keel.StringCol, value string) keel.Pred {
	return col.Lower().Eq(strings.ToLower(value))
}

func (r *repository) MatchLane(
	ctx context.Context,
	req *tenderports.MatchLaneRequest,
) (*tender.RoutingGuide, error) {
	rg := tenderdb.RoutingGuides

	if !req.OriginLocationID.IsNil() && !req.DestinationLocationID.IsNil() {
		guide, err := r.matchTier(ctx, tender.SpecificityLocationPair,
			rg.OriginLocationID.Eq(req.OriginLocationID),
			rg.DestinationLocationID.Eq(req.DestinationLocationID),
		)
		if err != nil || guide != nil {
			return guide, err
		}
	}

	if req.OriginCity != "" && req.OriginState != "" &&
		req.DestinationCity != "" && req.DestinationState != "" {
		guide, err := r.matchTier(ctx, tender.SpecificityCityPair,
			lowerEq(rg.OriginCity, req.OriginCity),
			upperEq(rg.OriginState, req.OriginState),
			lowerEq(rg.DestinationCity, req.DestinationCity),
			upperEq(rg.DestinationState, req.DestinationState),
		)
		if err != nil || guide != nil {
			return guide, err
		}
	}

	if req.OriginState != "" && req.DestinationState != "" {
		guide, err := r.matchTier(ctx, tender.SpecificityStatePair,
			upperEq(rg.OriginState, req.OriginState),
			upperEq(rg.DestinationState, req.DestinationState),
		)
		if err != nil || guide != nil {
			return guide, err
		}
	}

	return nil, nil //nolint:nilnil // nil guide represents an optional absence
}
