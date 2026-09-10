package fuelpurchaserepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func (r *repository) applyCardFilters(
	q *bun.SelectQuery,
	req *repositories.ListFuelCardsRequest,
) *bun.SelectQuery {
	cols := buncolgen.FuelCardColumns

	if len(req.Statuses) > 0 {
		q = q.Where(cols.Status.In(), bun.In(req.Statuses))
	}
	if req.Provider != "" {
		q = q.Where(cols.Provider.Eq(), req.Provider)
	}
	if !req.AssignedTractorID.IsNil() {
		q = q.Where(cols.AssignedTractorID.Eq(), req.AssignedTractorID)
	}
	if !req.AssignedWorkerID.IsNil() {
		q = q.Where(cols.AssignedWorkerID.Eq(), req.AssignedWorkerID)
	}
	if req.UnassignedOnly {
		q = q.Where(cols.AssignedTractorID.IsNull()).
			Where(cols.AssignedWorkerID.IsNull())
	}
	if req.DiscoveredOnly {
		q = q.Where(cols.DiscoveredAt.IsNotNull())
	}

	return q
}

func (r *repository) ListCards(
	ctx context.Context,
	req *repositories.ListFuelCardsRequest,
) (*pagination.CursorListResult[*fuelpurchase.FuelCard], error) {
	dba := r.db.DBForContext(ctx)

	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*fuelpurchase.FuelCard)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.FuelCardTable.Alias,
					req.Filter,
					(*fuelpurchase.FuelCard)(nil),
				)
				return r.applyCardFilters(sq, req)
			}).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count fuel cards", zap.Error(err))
			return nil, fmt.Errorf("count fuel cards: %w", err)
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*fuelpurchase.FuelCard]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: totalCount,
		Query: func(items *[]*fuelpurchase.FuelCard) *bun.SelectQuery {
			q := dba.NewSelect().
				Model(items).
				ColumnExpr(buncolgen.FuelCardTable.All())
			if req.IncludeAssignments {
				q = q.Relation(buncolgen.FuelCardRelations.AssignedWorker).
					Relation(buncolgen.FuelCardRelations.AssignedTractor)
			}
			return q
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			sq, applyErr := querybuilder.ApplyCursorFilters(
				sq,
				buncolgen.FuelCardTable.Alias,
				req.Filter,
				req.Cursor,
				(*fuelpurchase.FuelCard)(nil),
			)
			if applyErr != nil {
				return sq, applyErr
			}
			return r.applyCardFilters(sq, req), nil
		},
	})
	if err != nil {
		r.l.Error("failed to list fuel cards", zap.Error(err))
		return nil, fmt.Errorf("list fuel cards: %w", err)
	}

	return result, nil
}

func (r *repository) GetCardByID(
	ctx context.Context,
	req *repositories.GetFuelCardByIDRequest,
) (*fuelpurchase.FuelCard, error) {
	entity := new(fuelpurchase.FuelCard)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FuelCardScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.FuelCardColumns.ID.Eq(), req.ID)
		})

	if req.IncludeAssignments {
		q = q.Relation(buncolgen.FuelCardRelations.AssignedWorker).
			Relation(buncolgen.FuelCardRelations.AssignedTractor)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "FuelCard")
	}

	return entity, nil
}

func (r *repository) GetCardsByIDs(
	ctx context.Context,
	req *repositories.GetFuelCardsByIDsRequest,
) ([]*fuelpurchase.FuelCard, error) {
	entities := make([]*fuelpurchase.FuelCard, 0, len(req.IDs))
	if len(req.IDs) == 0 {
		return entities, nil
	}

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FuelCardScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.FuelCardColumns.ID.In(), bun.List(req.IDs))
		}).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to get fuel cards by ids", zap.Error(err))
		return nil, fmt.Errorf("get fuel cards by ids: %w", err)
	}

	return entities, nil
}

// FindCardsByLastFour resolves many cards in one round trip, which is what a feed
// needs: it has to know which of a file's cards already exist before it can
// decide which to create. Cancelled cards are excluded so a reissued last four
// resolves to the live card.
func (r *repository) FindCardsByLastFour(
	ctx context.Context,
	req *repositories.FindFuelCardsByLastFourRequest,
) (map[string]*fuelpurchase.FuelCard, error) {
	cols := buncolgen.FuelCardColumns

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

	entities := make([]*fuelpurchase.FuelCard, 0, len(wanted))
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.FuelCardScopeTenant(sq, req.TenantInfo).
				Where(cols.LastFour.In(), bun.List(wanted)).
				Where(cols.Status.Ne(), fuelpurchase.CardStatusCancelled)
			if req.Provider != "" {
				sq = sq.Where(cols.Provider.Eq(), req.Provider)
			}

			return sq
		}).
		Order(cols.CreatedAt.OrderDesc()).
		Scan(ctx); err != nil {
		r.l.Error("failed to find fuel cards by last four", zap.Error(err))

		return nil, fmt.Errorf("find fuel cards by last four: %w", err)
	}

	// Ordered newest first, so the first row for a last four wins and later
	// duplicates are ignored, matching what the single-card lookup returns.
	for _, entity := range entities {
		if _, taken := found[entity.LastFour]; taken {
			continue
		}
		found[entity.LastFour] = entity
	}

	return found, nil
}

func (r *repository) FindCardByLastFour(
	ctx context.Context,
	req *repositories.FindFuelCardByLastFourRequest,
) (*fuelpurchase.FuelCard, error) {
	cols := buncolgen.FuelCardColumns
	lastFour := strings.TrimSpace(req.LastFour)
	if lastFour == "" {
		return nil, nil
	}

	entity := new(fuelpurchase.FuelCard)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.FuelCardScopeTenant(sq, req.TenantInfo).
				Where(cols.LastFour.Eq(), lastFour).
				Where(cols.Status.Ne(), fuelpurchase.CardStatusCancelled)
			if req.Provider != "" {
				sq = sq.Where(cols.Provider.Eq(), req.Provider)
			}
			return sq
		}).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		r.l.Error("failed to find fuel card by last four", zap.Error(err))
		return nil, fmt.Errorf("find fuel card by last four: %w", err)
	}

	return entity, nil
}

func (r *repository) ListActiveCards(
	ctx context.Context,
	req *repositories.ListActiveFuelCardsRequest,
) ([]*fuelpurchase.FuelCard, error) {
	cols := buncolgen.FuelCardColumns
	limit := limitOr(req.Limit, defaultActiveCardLimit, maxActiveCardLimit)
	entities := make([]*fuelpurchase.FuelCard, 0, limit)
	term := strings.TrimSpace(req.Query)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.FuelCardScopeTenant(sq, req.TenantInfo).
				Where(cols.Status.Eq(), fuelpurchase.CardStatusActive)
			if term != "" {
				pattern := "%" + term + "%"
				sq = sq.WhereGroup(" AND ", func(tq *bun.SelectQuery) *bun.SelectQuery {
					return tq.Where(cols.Label.ILike(), pattern).
						WhereOr(cols.LastFour.Like(), pattern)
				})
			}
			return sq
		}).
		Order(cols.Label.OrderAsc()).
		Order(cols.LastFour.OrderAsc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list active fuel cards", zap.Error(err))
		return nil, fmt.Errorf("list active fuel cards: %w", err)
	}

	return entities, nil
}

func (r *repository) CreateCard(
	ctx context.Context,
	entity *fuelpurchase.FuelCard,
) (*fuelpurchase.FuelCard, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateCard()
		}
		r.l.Error("failed to create fuel card", zap.Error(err))
		return nil, fmt.Errorf("create fuel card: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdateCard(
	ctx context.Context,
	entity *fuelpurchase.FuelCard,
) (*fuelpurchase.FuelCard, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.FuelCardColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateCard()
		}
		r.l.Error("failed to update fuel card", zap.Error(err))
		return nil, fmt.Errorf("update fuel card: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, "FuelCard", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}
