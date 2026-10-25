package fuelpurchaserepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase/fuelpurchasedb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/fuelpurchaseports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

func cardFilters(req *fuelpurchaseports.ListFuelCardsRequest) []keel.Pred {
	fc := fuelpurchasedb.FuelCards

	return []keel.Pred{
		when(len(req.Statuses) > 0, fc.Status.In(req.Statuses...)),
		when(req.Provider != "", fc.Provider.Eq(req.Provider)),
		when(!req.AssignedTractorID.IsNil(), fc.AssignedTractorID.Eq(req.AssignedTractorID)),
		when(!req.AssignedWorkerID.IsNil(), fc.AssignedWorkerID.Eq(req.AssignedWorkerID)),
		when(
			req.UnassignedOnly,
			keel.And(fc.AssignedTractorID.IsNull(), fc.AssignedWorkerID.IsNull()),
		),
		when(req.DiscoveredOnly, fc.DiscoveredAt.IsNotNull()),
	}
}

func cardAssignments(include bool) []keel.Loader {
	if !include {
		return nil
	}

	fc := fuelpurchasedb.FuelCards
	return []keel.Loader{fc.AssignedWorker(), fc.AssignedTractor()}
}

func (r *repository) ListCards(
	ctx context.Context,
	req *fuelpurchaseports.ListFuelCardsRequest,
) (*pagination.CursorListResult[*fuelpurchase.FuelCard], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*fuelpurchase.FuelCard], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				fuelpurchasedb.FuelCards.Where(cardFilters(req)...),
				&listquery.Request[fuelpurchase.FuelCard]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(q *keel.Query[fuelpurchase.FuelCard]) *keel.Query[fuelpurchase.FuelCard] {
						return q.Load(cardAssignments(req.IncludeAssignments)...)
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list fuel cards", zap.Error(err))
				return nil, fmt.Errorf("list fuel cards: %w", err)
			}

			return result, nil
		},
	)
}

func (r *repository) GetCardByID(
	ctx context.Context,
	req *fuelpurchaseports.GetFuelCardByIDRequest,
) (*fuelpurchase.FuelCard, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*fuelpurchase.FuelCard, error) {
		fc := fuelpurchasedb.FuelCards
		entity, err := fc.Where(fc.ID.Eq(req.ID)).
			Load(cardAssignments(req.IncludeAssignments)...).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, cardErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetCardsByIDs(
	ctx context.Context,
	req *fuelpurchaseports.GetFuelCardsByIDsRequest,
) ([]*fuelpurchase.FuelCard, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*fuelpurchase.FuelCard, error) {
		if len(req.IDs) == 0 {
			return []*fuelpurchase.FuelCard{}, nil
		}

		fc := fuelpurchasedb.FuelCards
		entities, err := fc.Where(fc.ID.In(req.IDs...)).
			Limit(len(req.IDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get fuel cards by ids", zap.Error(err))
			return nil, fmt.Errorf("get fuel cards by ids: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) FindCardsByLastFour(
	ctx context.Context,
	req *fuelpurchaseports.FindFuelCardsByLastFourRequest,
) (map[string]*fuelpurchase.FuelCard, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[string]*fuelpurchase.FuelCard, error) {
			wanted := make([]string, 0, len(req.LastFours))
			seen := make(map[string]struct{}, len(req.LastFours))
			for _, lastFour := range req.LastFours {
				lastFour = strings.TrimSpace(lastFour)
				if lastFour == "" {
					continue
				}
				if _, duplicate := seen[lastFour]; duplicate {
					continue
				}
				seen[lastFour] = struct{}{}
				wanted = append(wanted, lastFour)
			}

			found := make(map[string]*fuelpurchase.FuelCard, len(wanted))
			if len(wanted) == 0 {
				return found, nil
			}

			fc := fuelpurchasedb.FuelCards
			entities, err := fc.Where(
				fc.LastFour.In(wanted...),
				fc.Status.Ne(fuelpurchase.CardStatusCancelled),
				when(req.Provider != "", fc.Provider.Eq(req.Provider)),
			).Order(fc.CreatedAt.Desc()).All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to find fuel cards by last four", zap.Error(err))
				return nil, fmt.Errorf("find fuel cards by last four: %w", err)
			}

			for _, entity := range entities {
				if _, taken := found[entity.LastFour]; taken {
					continue
				}
				found[entity.LastFour] = entity
			}

			return found, nil
		},
	)
}

func (r *repository) FindCardByLastFour(
	ctx context.Context,
	req *fuelpurchaseports.FindFuelCardByLastFourRequest,
) (*fuelpurchase.FuelCard, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*fuelpurchase.FuelCard, error) {
		lastFour := strings.TrimSpace(req.LastFour)
		if lastFour == "" {
			return nil, nil
		}

		fc := fuelpurchasedb.FuelCards
		entity, err := fc.Where(
			fc.LastFour.Eq(lastFour),
			fc.Status.Ne(fuelpurchase.CardStatusCancelled),
			when(req.Provider != "", fc.Provider.Eq(req.Provider)),
		).Order(fc.CreatedAt.Desc()).First(ctx, r.db.Keel(ctx))
		if dberror.IsNotFoundError(err) {
			return nil, nil
		}
		if err != nil {
			r.l.Error("failed to find fuel card by last four", zap.Error(err))
			return nil, fmt.Errorf("find fuel card by last four: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) ListActiveCards(
	ctx context.Context,
	req *fuelpurchaseports.ListActiveFuelCardsRequest,
) ([]*fuelpurchase.FuelCard, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*fuelpurchase.FuelCard, error) {
		fc := fuelpurchasedb.FuelCards
		term := strings.TrimSpace(req.Query)

		entities, err := fc.Where(
			fc.Status.Eq(fuelpurchase.CardStatusActive),
			when(term != "", keel.Or(fc.Label.ContainsFold(term), fc.LastFour.Contains(term))),
		).
			Order(fc.Label.Asc(), fc.LastFour.Asc()).
			Limit(limitOr(req.Limit, defaultActiveCardLimit, maxActiveCardLimit)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list active fuel cards", zap.Error(err))
			return nil, fmt.Errorf("list active fuel cards: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) CreateCard(
	ctx context.Context,
	entity *fuelpurchase.FuelCard,
) (*fuelpurchase.FuelCard, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fuelpurchase.FuelCard, error) {
		if err := fuelpurchasedb.FuelCards.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, duplicateCard()
			}
			r.l.Error("failed to create fuel card", zap.Error(err))
			return nil, cardErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateCard(
	ctx context.Context,
	entity *fuelpurchase.FuelCard,
) (*fuelpurchase.FuelCard, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*fuelpurchase.FuelCard, error) {
		if err := fuelpurchasedb.FuelCards.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, duplicateCard()
			}
			r.l.Error("failed to update fuel card", zap.Error(err))
			return nil, cardErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}
