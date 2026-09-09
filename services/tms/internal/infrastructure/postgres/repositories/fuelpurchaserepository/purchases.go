package fuelpurchaserepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func (r *repository) applyPurchaseFilters(
	q *bun.SelectQuery,
	req *repositories.ListFuelPurchasesRequest,
) *bun.SelectQuery {
	cols := buncolgen.FuelPurchaseColumns

	if !req.TractorID.IsNil() {
		q = q.Where(cols.TractorID.Eq(), req.TractorID)
	}
	if !req.WorkerID.IsNil() {
		q = q.Where(cols.WorkerID.Eq(), req.WorkerID)
	}
	if !req.JurisdictionID.IsNil() {
		q = q.Where(cols.JurisdictionID.Eq(), req.JurisdictionID)
	}
	if !req.FuelCardID.IsNil() {
		q = q.Where(cols.FuelCardID.Eq(), req.FuelCardID)
	}
	if !req.ImportBatchID.IsNil() {
		q = q.Where(cols.ImportBatchID.Eq(), req.ImportBatchID)
	}
	if len(req.FuelTypes) > 0 {
		q = q.Where(cols.FuelType.In(), bun.In(req.FuelTypes))
	}
	if len(req.Sources) > 0 {
		q = q.Where(cols.Source.In(), bun.In(req.Sources))
	}
	if req.TaxPaid != nil {
		q = q.Where(cols.TaxPaid.Eq(), *req.TaxPaid)
	}
	if req.From > 0 {
		q = q.Where(cols.PurchasedAt.Gte(), req.From)
	}
	if req.To > 0 {
		q = q.Where(cols.PurchasedAt.Lt(), req.To)
	}

	return q
}

func applyPurchaseRelations(
	q *bun.SelectQuery,
	includeTractor, includeWorker, includeJurisdiction, includeFuelCard bool,
) *bun.SelectQuery {
	rel := buncolgen.FuelPurchaseRelations

	if includeTractor {
		q = q.Relation(rel.Tractor)
	}
	if includeWorker {
		q = q.Relation(rel.Worker)
	}
	if includeJurisdiction {
		q = q.Relation(rel.Jurisdiction)
	}
	if includeFuelCard {
		q = q.Relation(rel.FuelCard)
	}

	return q
}

func (r *repository) ListPurchases(
	ctx context.Context,
	req *repositories.ListFuelPurchasesRequest,
) (*pagination.CursorListResult[*fuelpurchase.FuelPurchase], error) {
	dba := r.db.DBForContext(ctx)

	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*fuelpurchase.FuelPurchase)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.FuelPurchaseTable.Alias,
					req.Filter,
					(*fuelpurchase.FuelPurchase)(nil),
				)
				return r.applyPurchaseFilters(sq, req)
			}).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count fuel purchases", zap.Error(err))
			return nil, fmt.Errorf("count fuel purchases: %w", err)
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*fuelpurchase.FuelPurchase]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(items *[]*fuelpurchase.FuelPurchase) *bun.SelectQuery {
				q := dba.NewSelect().
					Model(items).
					ColumnExpr(buncolgen.FuelPurchaseTable.All())
				return applyPurchaseRelations(
					q,
					req.IncludeTractor,
					req.IncludeWorker,
					req.IncludeJurisdiction,
					req.IncludeFuelCard,
				)
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				sq, applyErr := querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.FuelPurchaseTable.Alias,
					req.Filter,
					req.Cursor,
					(*fuelpurchase.FuelPurchase)(nil),
				)
				if applyErr != nil {
					return sq, applyErr
				}
				return r.applyPurchaseFilters(sq, req), nil
			},
		},
	)
	if err != nil {
		r.l.Error("failed to list fuel purchases", zap.Error(err))
		return nil, fmt.Errorf("list fuel purchases: %w", err)
	}

	return result, nil
}

func (r *repository) GetPurchaseByID(
	ctx context.Context,
	req *repositories.GetFuelPurchaseByIDRequest,
) (*fuelpurchase.FuelPurchase, error) {
	entity := new(fuelpurchase.FuelPurchase)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FuelPurchaseScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.FuelPurchaseColumns.ID.Eq(), req.ID)
		})
	q = applyPurchaseRelations(
		q,
		req.IncludeTractor,
		req.IncludeWorker,
		req.IncludeJurisdiction,
		req.IncludeFuelCard,
	)

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "FuelPurchase")
	}

	return entity, nil
}

func (r *repository) GetPurchasesByIDs(
	ctx context.Context,
	req *repositories.GetFuelPurchasesByIDsRequest,
) ([]*fuelpurchase.FuelPurchase, error) {
	entities := make([]*fuelpurchase.FuelPurchase, 0, len(req.IDs))
	if len(req.IDs) == 0 {
		return entities, nil
	}

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FuelPurchaseScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.FuelPurchaseColumns.ID.In(), bun.List(req.IDs))
		}).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to get fuel purchases by ids", zap.Error(err))
		return nil, fmt.Errorf("get fuel purchases by ids: %w", err)
	}

	return entities, nil
}

type referenceRow struct {
	ID        pulid.ID `bun:"id"`
	Reference string   `bun:"transaction_reference"`
}

func (r *repository) FindReferences(
	ctx context.Context,
	req *repositories.FindFuelPurchaseReferencesRequest,
) (map[string]pulid.ID, error) {
	refs := make([]string, 0, len(req.References))
	for _, ref := range req.References {
		normalized := strings.ToUpper(strings.TrimSpace(ref))
		if normalized != "" {
			refs = append(refs, normalized)
		}
	}
	found := make(map[string]pulid.ID, len(refs))
	if len(refs) == 0 {
		return found, nil
	}

	cols := buncolgen.FuelPurchaseColumns
	rows := make([]referenceRow, 0, len(refs))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*fuelpurchase.FuelPurchase)(nil)).
		Column(cols.ID.String(), cols.TransactionReference.String()).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FuelPurchaseScopeTenant(sq, req.TenantInfo).
				Where(cols.TransactionReference.In(), bun.In(refs))
		}).
		Scan(ctx, &rows)
	if err != nil {
		r.l.Error("failed to find fuel purchase references", zap.Error(err))
		return nil, fmt.Errorf("find fuel purchase references: %w", err)
	}

	for _, row := range rows {
		found[row.Reference] = row.ID
	}

	return found, nil
}

func (r *repository) CreatePurchase(
	ctx context.Context,
	entity *fuelpurchase.FuelPurchase,
) (*fuelpurchase.FuelPurchase, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateReference()
		}
		r.l.Error("failed to create fuel purchase", zap.Error(err))
		return nil, fmt.Errorf("create fuel purchase: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdatePurchase(
	ctx context.Context,
	entity *fuelpurchase.FuelPurchase,
) (*fuelpurchase.FuelPurchase, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.FuelPurchaseColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateReference()
		}
		r.l.Error("failed to update fuel purchase", zap.Error(err))
		return nil, fmt.Errorf("update fuel purchase: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, "FuelPurchase", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) DeletePurchase(
	ctx context.Context,
	req *repositories.DeleteFuelPurchaseRequest,
) error {
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*fuelpurchase.FuelPurchase)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.FuelPurchaseScopeTenantDelete(dq, req.TenantInfo).
				Where(buncolgen.FuelPurchaseColumns.ID.Eq(), req.ID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to delete fuel purchase", zap.Error(err))
		return fmt.Errorf("delete fuel purchase: %w", err)
	}

	return dberror.CheckRowsAffected(results, "FuelPurchase", req.ID.String())
}

func (r *repository) AccumulateFuel(
	ctx context.Context,
	req *repositories.AccumulateFuelRequest,
) ([]*repositories.FuelAccumulationRow, error) {
	cols := buncolgen.FuelPurchaseColumns
	rows := make([]*repositories.FuelAccumulationRow, 0, 64)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*fuelpurchase.FuelPurchase)(nil)).
		ColumnExpr(cols.TractorID.As("tractor_id")).
		ColumnExpr(cols.JurisdictionID.As("jurisdiction_id")).
		ColumnExpr(cols.FuelType.As("fuel_type")).
		ColumnExpr(buncolgen.Sum(cols.Gallons, "gallons")).
		ColumnExpr(buncolgen.Expr(
			"COALESCE(SUM({0}) FILTER (WHERE {1}), 0) AS tax_paid_gallons",
			cols.Gallons,
			cols.TaxPaid,
		)).
		ColumnExpr(buncolgen.Count("purchase_count")).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FuelPurchaseScopeTenant(sq, req.TenantInfo).
				Where(cols.PurchasedAt.Gte(), req.Start).
				Where(cols.PurchasedAt.Lt(), req.End).
				Where(cols.FuelType.In(), bun.In(domaintypes.IFTAFuelTypes()))
		}).
		GroupExpr(cols.TractorID.Qualified()).
		GroupExpr(cols.JurisdictionID.Qualified()).
		GroupExpr(cols.FuelType.Qualified()).
		OrderExpr(cols.TractorID.OrderAsc()).
		OrderExpr(cols.JurisdictionID.OrderAsc()).
		OrderExpr(cols.FuelType.OrderAsc()).
		Scan(ctx, &rows)
	if err != nil {
		r.l.Error("failed to accumulate fuel purchases", zap.Error(err))
		return nil, fmt.Errorf("accumulate fuel purchases: %w", err)
	}

	return rows, nil
}
