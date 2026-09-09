package iftarepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	taxRatePeriodConflict = "CONFLICT ON CONSTRAINT uq_ifta_tax_rates_period DO UPDATE"
	defaultRateCount      = 128
)

func filterOrEmpty(filter *pagination.QueryOptions) *pagination.QueryOptions {
	if filter == nil {
		return &pagination.QueryOptions{}
	}
	return filter
}

func withoutSort(filter *pagination.QueryOptions) *pagination.QueryOptions {
	filterCopy := *filter
	filterCopy.Sort = nil
	filterCopy.Cursor = pagination.CursorInfo{}
	filterCopy.CursorSort = nil
	filterCopy.CursorColumns = nil
	filterCopy.CursorError = nil
	filterCopy.UseCursor = false
	return &filterCopy
}

func (r *repository) applyTaxRateFilters(
	q *bun.SelectQuery,
	req *repositories.ListTaxRatesRequest,
) *bun.SelectQuery {
	cols := buncolgen.TaxRateColumns
	if req.Year > 0 {
		q = q.Where(cols.Year.Eq(), req.Year)
	}
	if req.Quarter > 0 {
		q = q.Where(cols.Quarter.Eq(), req.Quarter)
	}
	if !req.JurisdictionID.IsNil() {
		q = q.Where(cols.JurisdictionID.Eq(), req.JurisdictionID)
	}
	if req.FuelType != "" {
		q = q.Where(cols.FuelType.Eq(), req.FuelType)
	}
	return q
}

func (r *repository) ListTaxRates(
	ctx context.Context,
	req *repositories.ListTaxRatesRequest,
) (*pagination.CursorListResult[*ifta.TaxRate], error) {
	dba := r.db.DBForContext(ctx)
	filter := filterOrEmpty(req.Filter)
	cols := buncolgen.TaxRateColumns

	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.NewSelect().
			Model((*ifta.TaxRate)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutTenantScope(
					sq,
					buncolgen.TaxRateTable.Alias,
					withoutSort(filter),
					(*ifta.TaxRate)(nil),
				)
				return r.applyTaxRateFilters(sq, req)
			}).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count ifta tax rates", zap.Error(err))
			return nil, fmt.Errorf("count ifta tax rates: %w", err)
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*ifta.TaxRate]{
		Filter:     filter,
		Cursor:     req.Cursor,
		TotalCount: totalCount,
		Query: func(items *[]*ifta.TaxRate) *bun.SelectQuery {
			q := dba.NewSelect().
				Model(items).
				ColumnExpr(buncolgen.TaxRateTable.All())
			if req.IncludeJurisdiction {
				q = q.Relation(buncolgen.TaxRateRelations.Jurisdiction)
			}
			return q
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			sq, applyErr := querybuilder.ApplyCursorFiltersWithoutTenantScope(
				sq,
				buncolgen.TaxRateTable.Alias,
				filter,
				req.Cursor,
				(*ifta.TaxRate)(nil),
			)
			if applyErr != nil {
				return sq, applyErr
			}
			sq = r.applyTaxRateFilters(sq, req)
			if len(filter.Sort) == 0 {
				sq = sq.Order(cols.Year.OrderDesc()).
					Order(cols.Quarter.OrderDesc()).
					Order(cols.FuelType.OrderAsc())
			}
			return sq, nil
		},
	})
	if err != nil {
		r.l.Error("failed to list ifta tax rates", zap.Error(err))
		return nil, fmt.Errorf("list ifta tax rates: %w", err)
	}

	return result, nil
}

func (r *repository) GetTaxRateByID(ctx context.Context, id pulid.ID) (*ifta.TaxRate, error) {
	entity := new(ifta.TaxRate)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(buncolgen.TaxRateRelations.Jurisdiction).
		Where(buncolgen.TaxRateColumns.ID.Eq(), id).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "IFTA tax rate")
	}

	return entity, nil
}

func (r *repository) UpsertTaxRates(
	ctx context.Context,
	rates []*ifta.TaxRate,
) ([]*ifta.TaxRate, error) {
	if len(rates) == 0 {
		return []*ifta.TaxRate{}, nil
	}

	cols := buncolgen.TaxRateColumns
	_, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(&rates).
		On(taxRatePeriodConflict).
		Set(cols.RatePerGallon.SetExcluded()).
		Set(cols.SurchargeRatePerGallon.SetExcluded()).
		Set(cols.SourceNote.SetExcluded()).
		Set(cols.SourceURL.SetExcluded()).
		Set(cols.Version.Inc(1)).
		Set(cols.UpdatedAt.SetExcluded()).
		Returning("*").
		Exec(ctx)
	if err != nil {
		if dberror.IsForeignKeyConstraintViolation(err) {
			return nil, errortypes.NewValidationError(
				"jurisdictionId",
				errortypes.ErrInvalid,
				"One of the rates references a jurisdiction that does not exist",
			)
		}
		r.l.Error("failed to upsert ifta tax rates", zap.Error(err))
		return nil, fmt.Errorf("upsert ifta tax rates: %w", err)
	}

	return rates, nil
}

func (r *repository) DeleteTaxRate(ctx context.Context, id pulid.ID, version int64) error {
	cols := buncolgen.TaxRateColumns
	result, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*ifta.TaxRate)(nil)).
		Where(cols.ID.Eq(), id).
		Where(cols.Version.Eq(), version).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to delete ifta tax rate", zap.Error(err))
		return fmt.Errorf("delete ifta tax rate: %w", err)
	}

	return dberror.CheckRowsAffected(result, "IFTA tax rate", id.String())
}

func (r *repository) ResolveRates(
	ctx context.Context,
	req *repositories.ResolveRatesRequest,
) (map[ifta.RateKey]*ifta.TaxRate, error) {
	cols := buncolgen.TaxRateColumns
	entities := make([]*ifta.TaxRate, 0, defaultRateCount)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where(cols.Year.Eq(), req.Year).
		Where(cols.Quarter.Eq(), req.Quarter).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to resolve ifta tax rates", zap.Error(err))
		return nil, fmt.Errorf("resolve ifta tax rates: %w", err)
	}

	out := make(map[ifta.RateKey]*ifta.TaxRate, len(entities))
	for _, entity := range entities {
		out[entity.Key()] = entity
	}

	return out, nil
}
