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
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func (r *repository) applyMileageEntryFilters(
	q *bun.SelectQuery,
	req *repositories.ListMileageEntriesRequest,
) *bun.SelectQuery {
	cols := buncolgen.JurisdictionMileageEntryColumns
	if !req.TractorID.IsNil() {
		q = q.Where(cols.TractorID.Eq(), req.TractorID)
	}
	if !req.JurisdictionID.IsNil() {
		q = q.Where(cols.JurisdictionID.Eq(), req.JurisdictionID)
	}
	if req.Year > 0 {
		q = q.Where(cols.Year.Eq(), req.Year)
	}
	if req.Quarter > 0 {
		q = q.Where(cols.Quarter.Eq(), req.Quarter)
	}
	if len(req.Sources) > 0 {
		q = q.Where(cols.Source.In(), bun.In(req.Sources))
	}
	return q
}

func (r *repository) ListMileageEntries(
	ctx context.Context,
	req *repositories.ListMileageEntriesRequest,
) (*pagination.CursorListResult[*ifta.JurisdictionMileageEntry], error) {
	dba := r.db.DBForContext(ctx)
	filter := filterOrEmpty(req.Filter)
	cols := buncolgen.JurisdictionMileageEntryColumns
	rel := buncolgen.JurisdictionMileageEntryRelations

	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.NewSelect().
			Model((*ifta.JurisdictionMileageEntry)(nil)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.JurisdictionMileageEntryScopeTenant(sq, filter.TenantInfo)
			}).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.JurisdictionMileageEntryTable.Alias,
					filter,
					(*ifta.JurisdictionMileageEntry)(nil),
				)
				return r.applyMileageEntryFilters(sq, req)
			}).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count ifta mileage entries", zap.Error(err))
			return nil, fmt.Errorf("count ifta mileage entries: %w", err)
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*ifta.JurisdictionMileageEntry]{
			Filter:     filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(items *[]*ifta.JurisdictionMileageEntry) *bun.SelectQuery {
				q := dba.NewSelect().
					Model(items).
					ColumnExpr(buncolgen.JurisdictionMileageEntryTable.All()).
					WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
						return buncolgen.JurisdictionMileageEntryScopeTenant(sq, filter.TenantInfo)
					})
				if req.IncludeTractor {
					q = q.Relation(rel.Tractor)
				}
				if req.IncludeJurisdiction {
					q = q.Relation(rel.Jurisdiction)
				}
				return q
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				sq, applyErr := querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.JurisdictionMileageEntryTable.Alias,
					filter,
					req.Cursor,
					(*ifta.JurisdictionMileageEntry)(nil),
				)
				if applyErr != nil {
					return sq, applyErr
				}
				sq = r.applyMileageEntryFilters(sq, req)
				if len(filter.Sort) == 0 {
					sq = sq.Order(cols.TraveledAt.OrderDesc())
				}
				return sq, nil
			},
		},
	)
	if err != nil {
		r.l.Error("failed to list ifta mileage entries", zap.Error(err))
		return nil, fmt.Errorf("list ifta mileage entries: %w", err)
	}

	return result, nil
}

func (r *repository) GetMileageEntryByID(
	ctx context.Context,
	req *repositories.GetMileageEntryByIDRequest,
) (*ifta.JurisdictionMileageEntry, error) {
	rel := buncolgen.JurisdictionMileageEntryRelations
	entity := new(ifta.JurisdictionMileageEntry)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.JurisdictionMileageEntryScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.JurisdictionMileageEntryColumns.ID.Eq(), req.ID)
		})
	if req.IncludeTractor {
		q = q.Relation(rel.Tractor)
	}
	if req.IncludeJurisdiction {
		q = q.Relation(rel.Jurisdiction)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "IFTA mileage entry")
	}

	return entity, nil
}

func (r *repository) CreateMileageEntry(
	ctx context.Context,
	entity *ifta.JurisdictionMileageEntry,
) (*ifta.JurisdictionMileageEntry, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		if dberror.IsForeignKeyConstraintViolation(err) {
			return nil, errortypes.NewValidationError(
				"tractorId",
				errortypes.ErrInvalid,
				"The tractor, jurisdiction or move referenced by this entry does not exist",
			)
		}
		r.l.Error("failed to create ifta mileage entry", zap.Error(err))
		return nil, fmt.Errorf("create ifta mileage entry: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdateMileageEntry(
	ctx context.Context,
	entity *ifta.JurisdictionMileageEntry,
) (*ifta.JurisdictionMileageEntry, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.JurisdictionMileageEntryColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		if dberror.IsForeignKeyConstraintViolation(err) {
			return nil, errortypes.NewValidationError(
				"tractorId",
				errortypes.ErrInvalid,
				"The tractor, jurisdiction or move referenced by this entry does not exist",
			)
		}
		r.l.Error("failed to update ifta mileage entry", zap.Error(err))
		return nil, fmt.Errorf("update ifta mileage entry: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, "IFTA mileage entry", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) DeleteMileageEntry(
	ctx context.Context,
	req *repositories.DeleteMileageEntryRequest,
) error {
	cols := buncolgen.JurisdictionMileageEntryColumns
	result, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*ifta.JurisdictionMileageEntry)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.JurisdictionMileageEntryScopeTenantDelete(dq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID).
				Where(cols.Version.Eq(), req.Version)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to delete ifta mileage entry", zap.Error(err))
		return fmt.Errorf("delete ifta mileage entry: %w", err)
	}

	return dberror.CheckRowsAffected(result, "IFTA mileage entry", req.ID.String())
}
