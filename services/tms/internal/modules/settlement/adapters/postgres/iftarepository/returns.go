package iftarepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta/iftadb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/iftaports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const returnEntity = "IFTA return"

var (
	openReturnStatuses = []ifta.ReturnStatus{ifta.ReturnStatusDraft, ifta.ReturnStatusFinalized}
	returnErrors       = dberror.Mapping{Entity: returnEntity}
)

func openReturnConflict() error {
	return errortypes.NewConflictError(
		"A return for this period is already open; recompute, reopen or amend it instead",
	)
}

func returnFilters(req *iftaports.ListReturnsRequest) []keel.Pred {
	rt := iftadb.Returns

	return []keel.Pred{
		when(req.Year > 0, rt.Year.Eq(int64(req.Year))),
		when(len(req.Statuses) > 0, rt.Status.In(req.Statuses...)),
	}
}

func (r *repository) ListReturns(
	ctx context.Context,
	req *iftaports.ListReturnsRequest,
) (*pagination.CursorListResult[*ifta.Return], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*ifta.Return], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				iftadb.Returns.Where(returnFilters(req)...),
				&listquery.Request[ifta.Return]{
					Filter: withDefaultSort(
						req.Filter,
						desc("year"),
						desc("quarter"),
						desc("amendmentNumber"),
					),
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				r.l.Error("failed to list ifta returns", zap.Error(err))
				return nil, fmt.Errorf("list ifta returns: %w", err)
			}

			return result, nil
		},
	)
}

func (r *repository) GetReturnByID(
	ctx context.Context,
	req *iftaports.GetReturnByIDRequest,
) (*ifta.Return, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ifta.Return, error) {
		rt := iftadb.Returns
		q := rt.Where(rt.ID.Eq(req.ID))
		if req.IncludeLines || req.IncludeJurisdictions {
			lines := rt.Lines()
			ordered := lines.Order(lines.SortOrder.Asc())
			if req.IncludeJurisdictions {
				ordered = ordered.Load(lines.Jurisdiction())
			}
			q = q.Load(ordered)
		}

		entity, err := q.One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, returnErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetReturnsByIDs(
	ctx context.Context,
	req *iftaports.GetReturnsByIDsRequest,
) ([]*ifta.Return, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*ifta.Return, error) {
		if len(req.IDs) == 0 {
			return []*ifta.Return{}, nil
		}

		rt := iftadb.Returns
		entities, err := rt.Where(rt.ID.In(req.IDs...)).
			Limit(len(req.IDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get ifta returns by ids", zap.Error(err))
			return nil, fmt.Errorf("get ifta returns by ids: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) GetOpenReturnForPeriod(
	ctx context.Context,
	req *iftaports.GetOpenReturnForPeriodRequest,
) (*ifta.Return, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ifta.Return, error) {
		rt := iftadb.Returns
		entity, err := rt.Where(
			rt.Year.Eq(int64(req.Year)),
			rt.Quarter.Eq(int64(req.Quarter)),
			rt.Status.In(openReturnStatuses...),
		).Order(rt.AmendmentNumber.Desc()).First(ctx, r.db.Keel(ctx))
		if dberror.IsNotFoundError(err) {
			return nil, nil
		}
		if err != nil {
			r.l.Error("failed to read open ifta return", zap.Error(err))
			return nil, fmt.Errorf("get open ifta return: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) GetLatestReturnForPeriod(
	ctx context.Context,
	req *iftaports.GetLatestReturnForPeriodRequest,
) (*ifta.Return, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*ifta.Return, error) {
		rt := iftadb.Returns
		entity, err := rt.Where(
			rt.Year.Eq(int64(req.Year)),
			rt.Quarter.Eq(int64(req.Quarter)),
		).Order(rt.AmendmentNumber.Desc(), rt.CreatedAt.Desc()).First(ctx, r.db.Keel(ctx))
		if dberror.IsNotFoundError(err) {
			return nil, nil
		}
		if err != nil {
			r.l.Error("failed to read latest ifta return", zap.Error(err))
			return nil, fmt.Errorf("get latest ifta return: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) CreateReturn(ctx context.Context, entity *ifta.Return) (*ifta.Return, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ifta.Return, error) {
		if err := iftadb.Returns.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			if dberror.IsUniqueConstraintViolation(err) {
				return nil, openReturnConflict()
			}
			r.l.Error("failed to create ifta return", zap.Error(err))
			return nil, returnErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateReturn(ctx context.Context, entity *ifta.Return) (*ifta.Return, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ifta.Return, error) {
		if err := r.saveReturn(ctx, entity); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) saveReturn(ctx context.Context, entity *ifta.Return) error {
	if err := iftadb.Returns.Save(ctx, r.db.Keel(ctx), entity); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return openReturnConflict()
		}
		r.l.Error("failed to update ifta return", zap.Error(err))
		return returnErrors.Map(err, entity.ID.String())
	}

	return nil
}

func (r *repository) ReplaceReturnLines(
	ctx context.Context,
	ret *ifta.Return,
	lines []*ifta.ReturnLine,
) (*ifta.Return, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*ifta.Return, error) {
		db := r.db.Keel(ctx)
		rl := iftadb.ReturnLines
		if _, err := rl.Where(rl.ReturnID.Eq(ret.ID)).Delete(ctx, db); err != nil {
			r.l.Error("failed to replace ifta return lines", zap.Error(err))
			return nil, fmt.Errorf("delete return lines: %w", err)
		}

		for i, line := range lines {
			line.ReturnID = ret.ID
			line.OrganizationID = ret.OrganizationID
			line.BusinessUnitID = ret.BusinessUnitID
			line.SortOrder = i
		}
		if err := rl.CreateMany(ctx, db, lines); err != nil {
			r.l.Error("failed to replace ifta return lines", zap.Error(err))
			return nil, fmt.Errorf("insert return lines: %w", err)
		}

		now := timeutils.NowUnix()
		ret.ComputedAt = &now
		if err := r.saveReturn(ctx, ret); err != nil {
			return nil, err
		}

		if lines == nil {
			lines = []*ifta.ReturnLine{}
		}
		ret.Lines = lines

		return ret, nil
	})
}

func (r *repository) DeleteReturn(
	ctx context.Context,
	req *iftaports.DeleteReturnRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		rl := iftadb.ReturnLines
		if _, err := rl.Where(rl.ReturnID.Eq(req.ID)).Delete(ctx, db); err != nil {
			r.l.Error("failed to delete ifta return", zap.Error(err))
			return fmt.Errorf("delete ifta return lines: %w", err)
		}

		rt := iftadb.Returns
		n, err := rt.Where(
			rt.ID.Eq(req.ID),
			rt.Version.Eq(req.Version),
			rt.Status.Eq(ifta.ReturnStatusDraft),
		).Delete(ctx, db)
		if err != nil {
			r.l.Error("failed to delete ifta return", zap.Error(err))
			return fmt.Errorf("delete ifta return: %w", err)
		}

		return dberror.CheckAffected(n, returnEntity, req.ID.String())
	})
}
