package aicorrectionrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultPurgeLimit    = 1000
	maxPurgeLimit        = 10000
	defaultAccuracyLimit = 2000
	maxAccuracyLimit     = 10000
	correctionEntity     = "AICorrection"
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

func New(p Params) repositories.AICorrectionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.aicorrection-repository"),
	}
}

func uniqueSource() string {
	cols := buncolgen.CorrectionColumns

	return "CONFLICT (" + strings.Join([]string{
		cols.OrganizationID.Bare(),
		cols.BusinessUnitID.Bare(),
		cols.SourceType.Bare(),
		cols.SourceID.Bare(),
	}, ", ") + ") DO UPDATE"
}

func (r *repository) Upsert(
	ctx context.Context,
	entity *aicorrection.Correction,
) (*aicorrection.Correction, error) {
	cols := buncolgen.CorrectionColumns

	_, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		On(uniqueSource()).
		Set(cols.Task.SetExcluded()).
		Set(cols.DocumentID.SetExcluded()).
		Set(cols.SubjectType.SetExcluded()).
		Set(cols.SubjectID.SetExcluded()).
		Set(cols.CapturedByID.SetExcluded()).
		Set(cols.DocumentKind.SetExcluded()).
		Set(cols.DocumentFingerprint.SetExcluded()).
		Set(cols.ExtractionModel.SetExcluded()).
		Set(cols.ExtractionProviderID.SetExcluded()).
		Set(cols.PredictedConfidence.SetExcluded()).
		Set(cols.Predicted.SetExcluded()).
		Set(cols.Confirmed.SetExcluded()).
		Set(cols.FieldResults.SetExcluded()).
		Set(cols.ScoredCount.SetExcluded()).
		Set(cols.CorrectCount.SetExcluded()).
		Set(cols.CorrectedCount.SetExcluded()).
		Set(cols.MissedCount.SetExcluded()).
		Set(cols.UnconfirmedCount.SetExcluded()).
		Set(cols.UnscoredCount.SetExcluded()).
		Set(cols.CapturedAt.SetExcluded()).
		Set(cols.Version.IncConflict(1)).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to upsert ai correction",
			zap.String("sourceType", entity.SourceType.String()),
			zap.String("sourceId", entity.SourceID.String()),
			zap.Error(err),
		)

		return nil, fmt.Errorf("upsert ai correction: %w", err)
	}

	return entity, nil
}

func (r *repository) PurgeBefore(
	ctx context.Context,
	req repositories.PurgeAICorrectionsRequest,
) (int64, error) {
	cols := buncolgen.CorrectionColumns
	limit := req.Limit
	if limit <= 0 {
		limit = defaultPurgeLimit
	}
	if limit > maxPurgeLimit {
		limit = maxPurgeLimit
	}

	dba := r.db.DBForContext(ctx)
	expired := dba.NewSelect().
		Model((*aicorrection.Correction)(nil)).
		Column(cols.ID.Bare()).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CorrectionScopeTenant(sq, req.TenantInfo).
				Where(cols.CreatedAt.Lt(), req.Before)
		}).
		OrderExpr(cols.CreatedAt.OrderAsc()).
		Limit(limit)

	res, err := dba.NewDelete().
		Model((*aicorrection.Correction)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.CorrectionScopeTenantDelete(dq, req.TenantInfo).
				Where(cols.ID.In(), expired)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to purge expired ai corrections", zap.Error(err))

		return 0, fmt.Errorf("purge expired ai corrections: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("purge expired ai corrections rows: %w", err)
	}

	return rows, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAICorrectionRequest,
) (*aicorrection.Correction, error) {
	entity := new(aicorrection.Correction)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CorrectionScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.CorrectionColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, correctionEntity)
	}

	return entity, nil
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListAICorrectionConnectionRequest,
) (*pagination.CursorListResult[*aicorrection.Correction], error) {
	dba := r.db.DBForContext(ctx)

	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.NewSelect().
			Model((*aicorrection.Correction)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.CorrectionTable.Alias,
					req.Filter,
					(*aicorrection.Correction)(nil),
				)

				return sq.Apply(buncolgen.CorrectionApplyTenant(req.Filter.TenantInfo))
			}).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count ai corrections", zap.Error(err))

			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*aicorrection.Correction]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: totalCount,
		Query: func(entities *[]*aicorrection.Correction) *bun.SelectQuery {
			q := dba.NewSelect().Model(entities)
			if len(req.Columns) > 0 {
				q = q.Column(req.Columns...)
			}

			return q
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			return querybuilder.ApplyCursorFilters(
				sq,
				buncolgen.CorrectionTable.Alias,
				req.Filter,
				req.Cursor,
				(*aicorrection.Correction)(nil),
			)
		},
	})
	if err != nil {
		r.l.Error("failed to list ai corrections", zap.Error(err))

		return nil, err
	}

	return result, nil
}

func (r *repository) ListForAccuracy(
	ctx context.Context,
	req repositories.ListAICorrectionsForAccuracyRequest,
) ([]*aicorrection.Correction, error) {
	cols := buncolgen.CorrectionColumns
	limit := req.Limit
	if limit <= 0 {
		limit = defaultAccuracyLimit
	}
	limit = min(limit, maxAccuracyLimit)

	entities := make([]*aicorrection.Correction, 0, min(limit, defaultAccuracyLimit))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Column(
			cols.ID.Bare(),
			cols.DocumentKind.Bare(),
			cols.ExtractionModel.Bare(),
			cols.ExtractionProviderID.Bare(),
			cols.FieldResults.Bare(),
			cols.ScoredCount.Bare(),
			cols.CorrectCount.Bare(),
			cols.CorrectedCount.Bare(),
			cols.MissedCount.Bare(),
			cols.UnconfirmedCount.Bare(),
			cols.UnscoredCount.Bare(),
			cols.CapturedAt.Bare(),
		).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CorrectionScopeTenant(sq, req.TenantInfo).
				Where(cols.Task.Eq(), req.Task).
				Where(cols.CapturedAt.Gte(), req.Since)
		}).
		Order(cols.CapturedAt.OrderDesc(), cols.ID.OrderAsc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list ai corrections for accuracy", zap.Error(err))

		return nil, fmt.Errorf("list ai corrections for accuracy: %w", err)
	}

	return entities, nil
}
