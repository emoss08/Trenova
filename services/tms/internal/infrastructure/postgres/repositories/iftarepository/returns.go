package iftarepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

var openReturnStatuses = []ifta.ReturnStatus{ifta.ReturnStatusDraft, ifta.ReturnStatusFinalized}

func openReturnConflict() error {
	return errortypes.NewConflictError(
		"A return for this period is already open; recompute, reopen or amend it instead",
	)
}

func (r *repository) applyReturnFilters(
	q *bun.SelectQuery,
	req *repositories.ListReturnsRequest,
) *bun.SelectQuery {
	cols := buncolgen.ReturnColumns
	if req.Year > 0 {
		q = q.Where(cols.Year.Eq(), req.Year)
	}
	if len(req.Statuses) > 0 {
		q = q.Where(cols.Status.In(), bun.In(req.Statuses))
	}
	return q
}

func (r *repository) ListReturns(
	ctx context.Context,
	req *repositories.ListReturnsRequest,
) (*pagination.CursorListResult[*ifta.Return], error) {
	dba := r.db.DBForContext(ctx)
	filter := filterOrEmpty(req.Filter)
	cols := buncolgen.ReturnColumns

	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.NewSelect().
			Model((*ifta.Return)(nil)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.ReturnScopeTenant(sq, filter.TenantInfo)
			}).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.ReturnTable.Alias,
					filter,
					(*ifta.Return)(nil),
				)
				return r.applyReturnFilters(sq, req)
			}).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count ifta returns", zap.Error(err))
			return nil, fmt.Errorf("count ifta returns: %w", err)
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*ifta.Return]{
		Filter:     filter,
		Cursor:     req.Cursor,
		TotalCount: totalCount,
		Query: func(items *[]*ifta.Return) *bun.SelectQuery {
			return dba.NewSelect().
				Model(items).
				ColumnExpr(buncolgen.ReturnTable.All()).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return buncolgen.ReturnScopeTenant(sq, filter.TenantInfo)
				})
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			sq, applyErr := querybuilder.ApplyCursorFilters(
				sq,
				buncolgen.ReturnTable.Alias,
				filter,
				req.Cursor,
				(*ifta.Return)(nil),
			)
			if applyErr != nil {
				return sq, applyErr
			}
			sq = r.applyReturnFilters(sq, req)
			if len(filter.Sort) == 0 {
				sq = sq.Order(cols.Year.OrderDesc()).
					Order(cols.Quarter.OrderDesc()).
					Order(cols.AmendmentNumber.OrderDesc())
			}
			return sq, nil
		},
	})
	if err != nil {
		r.l.Error("failed to list ifta returns", zap.Error(err))
		return nil, fmt.Errorf("list ifta returns: %w", err)
	}

	return result, nil
}

func (r *repository) GetReturnByID(
	ctx context.Context,
	req *repositories.GetReturnByIDRequest,
) (*ifta.Return, error) {
	rel := buncolgen.ReturnRelations
	entity := new(ifta.Return)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ReturnScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.ReturnColumns.ID.Eq(), req.ID)
		})
	if req.IncludeLines || req.IncludeJurisdictions {
		q = q.Relation(rel.Lines, func(lq *bun.SelectQuery) *bun.SelectQuery {
			return lq.Order(buncolgen.ReturnLineColumns.SortOrder.OrderAsc())
		})
	}
	if req.IncludeJurisdictions {
		q = q.Relation(buncolgen.Rel(rel.Lines, buncolgen.ReturnLineRelations.Jurisdiction))
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "IFTA return")
	}

	return entity, nil
}

func (r *repository) GetReturnsByIDs(
	ctx context.Context,
	req *repositories.GetReturnsByIDsRequest,
) ([]*ifta.Return, error) {
	entities := make([]*ifta.Return, 0, len(req.IDs))
	if len(req.IDs) == 0 {
		return entities, nil
	}

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ReturnScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.ReturnColumns.ID.In(), bun.List(req.IDs))
		}).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to get ifta returns by ids", zap.Error(err))
		return nil, fmt.Errorf("get ifta returns by ids: %w", err)
	}

	return entities, nil
}

func (r *repository) GetOpenReturnForPeriod(
	ctx context.Context,
	req *repositories.GetOpenReturnForPeriodRequest,
) (*ifta.Return, error) {
	cols := buncolgen.ReturnColumns
	entity := new(ifta.Return)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ReturnScopeTenant(sq, req.TenantInfo).
				Where(cols.Year.Eq(), req.Year).
				Where(cols.Quarter.Eq(), req.Quarter).
				Where(cols.Status.In(), bun.In(openReturnStatuses))
		}).
		Order(cols.AmendmentNumber.OrderDesc()).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil
		}
		r.l.Error("failed to read open ifta return", zap.Error(err))
		return nil, fmt.Errorf("get open ifta return: %w", err)
	}

	return entity, nil
}

func (r *repository) GetLatestReturnForPeriod(
	ctx context.Context,
	req *repositories.GetLatestReturnForPeriodRequest,
) (*ifta.Return, error) {
	cols := buncolgen.ReturnColumns
	entity := new(ifta.Return)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ReturnScopeTenant(sq, req.TenantInfo).
				Where(cols.Year.Eq(), req.Year).
				Where(cols.Quarter.Eq(), req.Quarter)
		}).
		Order(cols.AmendmentNumber.OrderDesc()).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil
		}
		r.l.Error("failed to read latest ifta return", zap.Error(err))
		return nil, fmt.Errorf("get latest ifta return: %w", err)
	}

	return entity, nil
}

func (r *repository) CreateReturn(ctx context.Context, entity *ifta.Return) (*ifta.Return, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, openReturnConflict()
		}
		r.l.Error("failed to create ifta return", zap.Error(err))
		return nil, fmt.Errorf("create ifta return: %w", err)
	}

	return entity, nil
}

func (r *repository) UpdateReturn(ctx context.Context, entity *ifta.Return) (*ifta.Return, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(buncolgen.ReturnColumns.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, openReturnConflict()
		}
		r.l.Error("failed to update ifta return", zap.Error(err))
		return nil, fmt.Errorf("update ifta return: %w", err)
	}
	if err = dberror.CheckRowsAffected(results, "IFTA return", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) ReplaceReturnLines(
	ctx context.Context,
	ret *ifta.Return,
	lines []*ifta.ReturnLine,
) (*ifta.Return, error) {
	tenantInfo := pagination.TenantInfo{OrgID: ret.OrganizationID, BuID: ret.BusinessUnitID}
	lineCols := buncolgen.ReturnLineColumns
	ov := ret.Version
	ret.Version++
	now := timeutils.NowUnix()
	ret.ComputedAt = &now

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		if _, dErr := tx.NewDelete().
			Model((*ifta.ReturnLine)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.ReturnLineScopeTenantDelete(dq, tenantInfo).
					Where(lineCols.ReturnID.Eq(), ret.ID)
			}).
			Exec(txCtx); dErr != nil {
			return fmt.Errorf("delete return lines: %w", dErr)
		}

		if len(lines) > 0 {
			for i, line := range lines {
				line.ReturnID = ret.ID
				line.OrganizationID = ret.OrganizationID
				line.BusinessUnitID = ret.BusinessUnitID
				line.SortOrder = i
			}
			if _, iErr := tx.NewInsert().Model(&lines).Returning("*").Exec(txCtx); iErr != nil {
				return fmt.Errorf("insert return lines: %w", iErr)
			}
		}

		results, uErr := tx.NewUpdate().
			Model(ret).
			WherePK().
			Where(buncolgen.ReturnColumns.Version.Eq(), ov).
			Returning("*").
			Exec(txCtx)
		if uErr != nil {
			return fmt.Errorf("update return totals: %w", uErr)
		}
		return dberror.CheckRowsAffected(results, "IFTA return", ret.ID.String())
	})
	if err != nil {
		r.l.Error("failed to replace ifta return lines", zap.Error(err))
		return nil, err
	}

	if lines == nil {
		lines = []*ifta.ReturnLine{}
	}
	ret.Lines = lines

	return ret, nil
}

func (r *repository) DeleteReturn(ctx context.Context, req *repositories.DeleteReturnRequest) error {
	cols := buncolgen.ReturnColumns
	lineCols := buncolgen.ReturnLineColumns

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
		if _, lErr := tx.NewDelete().
			Model((*ifta.ReturnLine)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.ReturnLineScopeTenantDelete(dq, req.TenantInfo).
					Where(lineCols.ReturnID.Eq(), req.ID)
			}).
			Exec(txCtx); lErr != nil {
			return fmt.Errorf("delete ifta return lines: %w", lErr)
		}

		result, dErr := tx.NewDelete().
			Model((*ifta.Return)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.ReturnScopeTenantDelete(dq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID).
					Where(cols.Version.Eq(), req.Version).
					Where(cols.Status.Eq(), ifta.ReturnStatusDraft)
			}).
			Exec(txCtx)
		if dErr != nil {
			return fmt.Errorf("delete ifta return: %w", dErr)
		}
		return dberror.CheckRowsAffected(result, "IFTA return", req.ID.String())
	})
	if err != nil {
		r.l.Error("failed to delete ifta return", zap.Error(err))
		return err
	}

	return nil
}
