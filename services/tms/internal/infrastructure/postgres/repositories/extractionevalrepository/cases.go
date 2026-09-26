package extractionevalrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/extractioneval"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	caseEntity        = "ExtractionEvalCase"
	defaultCaseLimit  = extractioneval.DefaultCaseLimit
	maxActiveCaseRead = extractioneval.MaxCaseLimit
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type caseRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewCases(p Params) repositories.ExtractionEvalCaseRepository {
	return &caseRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.extractioneval-case-repository"),
	}
}

func (r *caseRepository) Create(
	ctx context.Context,
	entity *extractioneval.ExtractionCase,
) (*extractioneval.ExtractionCase, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, errortypes.NewBusinessError(
				"This document is already in the evaluation set",
			).WithInternal(err)
		}
		r.l.Error("failed to create extraction eval case", zap.Error(err))

		return nil, fmt.Errorf("create extraction eval case: %w", err)
	}

	return entity, nil
}

func (r *caseRepository) GetByID(
	ctx context.Context,
	req repositories.GetExtractionEvalCaseRequest,
) (*extractioneval.ExtractionCase, error) {
	entity := new(extractioneval.ExtractionCase)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ExtractionCaseScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.ExtractionCaseColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, caseEntity)
	}

	return entity, nil
}

func (r *caseRepository) GetBySourceCorrection(
	ctx context.Context,
	tenant pagination.TenantInfo,
	correctionID pulid.ID,
) (*extractioneval.ExtractionCase, error) {
	entity := new(extractioneval.ExtractionCase)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ExtractionCaseScopeTenant(sq, tenant).
				Where(buncolgen.ExtractionCaseColumns.SourceCorrectionID.Eq(), correctionID)
		}).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, caseEntity)
	}

	return entity, nil
}

func (r *caseRepository) Update(
	ctx context.Context,
	entity *extractioneval.ExtractionCase,
) (*extractioneval.ExtractionCase, error) {
	cols := buncolgen.ExtractionCaseColumns
	previous := entity.Version

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ExtractionCaseScopeTenantUpdate(uq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).
				Where(cols.ID.Eq(), entity.ID).
				Where(cols.Version.Eq(), previous)
		}).
		Set(cols.Status.Set(), entity.Status).
		Set(cols.Title.Set(), entity.Title).
		Set(cols.Notes.Set(), entity.Notes).
		Set(cols.Expected.Set(), entity.Expected).
		Set(cols.ExpectedFieldCount.Set(), entity.ExpectedFieldCount).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Set(cols.Version.Inc(1)).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update extraction eval case", zap.Error(err))

		return nil, fmt.Errorf("update extraction eval case: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, caseEntity, entity.ID.String()); err != nil {
		return nil, err
	}
	entity.Version = previous + 1

	return entity, nil
}

func (r *caseRepository) Delete(
	ctx context.Context,
	req repositories.GetExtractionEvalCaseRequest,
) error {
	res, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*extractioneval.ExtractionCase)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.ExtractionCaseScopeTenantDelete(dq, req.TenantInfo).
				Where(buncolgen.ExtractionCaseColumns.ID.Eq(), req.ID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to delete extraction eval case", zap.Error(err))

		return fmt.Errorf("delete extraction eval case: %w", err)
	}

	return dberror.CheckRowsAffected(res, caseEntity, req.ID.String())
}

func (r *caseRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListExtractionEvalCaseConnectionRequest,
) (*pagination.CursorListResult[*extractioneval.ExtractionCase], error) {
	dba := r.db.DBForContext(ctx)

	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.NewSelect().
			Model((*extractioneval.ExtractionCase)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.ExtractionCaseTable.Alias,
					req.Filter,
					(*extractioneval.ExtractionCase)(nil),
				)

				return sq.Apply(buncolgen.ExtractionCaseApplyTenant(req.Filter.TenantInfo))
			}).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count extraction eval cases", zap.Error(err))

			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*extractioneval.ExtractionCase]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: totalCount,
		Query: func(entities *[]*extractioneval.ExtractionCase) *bun.SelectQuery {
			q := dba.NewSelect().Model(entities)
			if len(req.Columns) > 0 {
				q = q.Column(req.Columns...)
			}

			return q
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			return querybuilder.ApplyCursorFilters(
				sq,
				buncolgen.ExtractionCaseTable.Alias,
				req.Filter,
				req.Cursor,
				(*extractioneval.ExtractionCase)(nil),
			)
		},
	})
	if err != nil {
		r.l.Error("failed to list extraction eval cases", zap.Error(err))

		return nil, err
	}

	return result, nil
}

func (r *caseRepository) ListActive(
	ctx context.Context,
	req repositories.ListActiveExtractionEvalCasesRequest,
) ([]*extractioneval.ExtractionCase, error) {
	cols := buncolgen.ExtractionCaseColumns
	limit := req.Limit
	if limit <= 0 {
		limit = defaultCaseLimit
	}
	limit = min(limit, maxActiveCaseRead)

	entities := make([]*extractioneval.ExtractionCase, 0, limit)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Column(cols.ID.Bare(), cols.Title.Bare(), cols.OrganizationID.Bare(), cols.BusinessUnitID.Bare()).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ExtractionCaseScopeTenant(sq, req.TenantInfo).
				Where(cols.Task.Eq(), req.Task).
				Where(cols.Status.Eq(), extractioneval.CaseStatusActive)
		}).
		Order(cols.CreatedAt.OrderDesc(), cols.ID.OrderAsc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list active extraction eval cases", zap.Error(err))

		return nil, fmt.Errorf("list active extraction eval cases: %w", err)
	}

	return entities, nil
}

func (r *caseRepository) Counts(
	ctx context.Context,
	req repositories.CountExtractionEvalCasesRequest,
) (*repositories.ExtractionEvalCaseCounts, error) {
	cols := buncolgen.ExtractionCaseColumns
	var row struct {
		Candidate int `bun:"candidate"`
		Active    int `bun:"active"`
		Retired   int `bun:"retired"`
	}

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*extractioneval.ExtractionCase)(nil)).
		ColumnExpr(buncolgen.CountFilter("candidate", cols.Status.Eq()), extractioneval.CaseStatusCandidate).
		ColumnExpr(buncolgen.CountFilter("active", cols.Status.Eq()), extractioneval.CaseStatusActive).
		ColumnExpr(buncolgen.CountFilter("retired", cols.Status.Eq()), extractioneval.CaseStatusRetired).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ExtractionCaseScopeTenant(sq, req.TenantInfo).
				Where(cols.Task.Eq(), req.Task)
		}).
		Scan(ctx, &row)
	if err != nil {
		r.l.Error("failed to count extraction eval cases", zap.Error(err))

		return nil, fmt.Errorf("count extraction eval cases: %w", err)
	}

	return &repositories.ExtractionEvalCaseCounts{
		Candidate: row.Candidate,
		Active:    row.Active,
		Retired:   row.Retired,
	}, nil
}
