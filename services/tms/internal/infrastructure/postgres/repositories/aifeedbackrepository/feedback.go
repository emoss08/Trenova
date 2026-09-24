package aifeedbackrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aifeedback"
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
	defaultNegativeLimit = 5000
	maxNegativeLimit     = 20000
	defaultWorstLimit    = 10
	maxWorstLimit        = 101
	defaultPurgeLimit    = 1000
	maxPurgeLimit        = 10000
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

func New(p Params) repositories.AIFeedbackRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.aifeedback-repository"),
	}
}

func uniqueTarget() string {
	cols := buncolgen.FeedbackColumns

	return "CONFLICT (" + strings.Join([]string{
		cols.OrganizationID.Bare(),
		cols.BusinessUnitID.Bare(),
		cols.UserID.Bare(),
		cols.TargetType.Bare(),
		cols.TargetID.Bare(),
		cols.TargetPart.Bare(),
	}, ", ") + ") DO UPDATE"
}

func (r *repository) Upsert(
	ctx context.Context,
	entity *aifeedback.Feedback,
) (*aifeedback.Feedback, error) {
	cols := buncolgen.FeedbackColumns

	_, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		On(uniqueTarget()).
		Set(cols.ThreadID.SetExcluded()).
		Set(cols.TurnID.SetExcluded()).
		Set(cols.RunID.SetExcluded()).
		Set(cols.AgentDefinitionID.SetExcluded()).
		Set(cols.DefinitionVersion.SetExcluded()).
		Set(cols.DetectorKey.SetExcluded()).
		Set(cols.Task.SetExcluded()).
		Set(cols.Model.SetExcluded()).
		Set(cols.ProviderID.SetExcluded()).
		Set(cols.PromptHash.SetExcluded()).
		Set(cols.ToolSpecHash.SetExcluded()).
		Set(cols.FingerprintSource.SetExcluded()).
		Set(cols.Rating.SetExcluded()).
		Set(cols.Reasons.SetExcluded()).
		Set(cols.Comment.SetExcluded()).
		Set(cols.TurnSnapshot.SetExcluded()).
		Set(cols.PatternKey.SetExcluded()).
		Set(cols.Version.IncConflict(1)).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to upsert ai feedback",
			zap.String("targetType", entity.TargetType.String()),
			zap.Error(err),
		)

		return nil, fmt.Errorf("upsert ai feedback: %w", err)
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	req repositories.DeleteAIFeedbackRequest,
) (bool, error) {
	cols := buncolgen.FeedbackColumns

	res, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*aifeedback.Feedback)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.FeedbackScopeTenantDelete(dq, req.TenantInfo).
				Where(cols.UserID.Eq(), req.UserID).
				Where(cols.TargetType.Eq(), req.Target.TargetType).
				Where(cols.TargetID.Eq(), req.Target.TargetID).
				Where(cols.TargetPart.Eq(), req.Target.TargetPart)
		}).
		Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("delete ai feedback: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete ai feedback rows: %w", err)
	}

	return rows > 0, nil
}

func (r *repository) ListForTargets(
	ctx context.Context,
	req repositories.ListAIFeedbackForTargetsRequest,
) ([]*aifeedback.Feedback, error) {
	if len(req.Targets) == 0 {
		return []*aifeedback.Feedback{}, nil
	}

	cols := buncolgen.FeedbackColumns
	rows := make([]*aifeedback.Feedback, 0, len(req.Targets))

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&rows).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FeedbackScopeTenant(sq, req.TenantInfo).
				Where(cols.UserID.Eq(), req.UserID).
				WhereGroup(" AND ", func(targets *bun.SelectQuery) *bun.SelectQuery {
					for _, target := range req.Targets {
						targets = targets.WhereGroup(" OR ", func(one *bun.SelectQuery) *bun.SelectQuery {
							return one.Where(cols.TargetType.Eq(), target.TargetType).
								Where(cols.TargetID.Eq(), target.TargetID).
								Where(cols.TargetPart.Eq(), target.TargetPart)
						})
					}

					return targets
				})
		}).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list ai feedback for targets", zap.Error(err))

		return nil, fmt.Errorf("list ai feedback for targets: %w", err)
	}

	return rows, nil
}

func (r *repository) ListByIDs(
	ctx context.Context,
	req repositories.ListAIFeedbackByIDsRequest,
) ([]*aifeedback.Feedback, error) {
	if len(req.IDs) == 0 {
		return []*aifeedback.Feedback{}, nil
	}

	cols := buncolgen.FeedbackColumns
	rows := make([]*aifeedback.Feedback, 0, len(req.IDs))

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&rows).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FeedbackScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.In(), bun.List(req.IDs))
		}).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list ai feedback by ids: %w", err)
	}

	return rows, nil
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListAIFeedbackConnectionRequest,
) (*pagination.CursorListResult[*aifeedback.Feedback], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))
	dba := r.db.DBForContext(ctx)

	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*aifeedback.Feedback)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.FeedbackTable.Alias,
					req.Filter,
					(*aifeedback.Feedback)(nil),
				)

				return sq.Apply(buncolgen.FeedbackApplyTenant(req.Filter.TenantInfo))
			}).
			Count(ctx)
		if err != nil {
			log.Error("failed to count ai feedback", zap.Error(err))

			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*aifeedback.Feedback]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(entities *[]*aifeedback.Feedback) *bun.SelectQuery {
				q := dba.NewSelect().Model(entities)
				if len(req.Columns) > 0 {
					q = q.Column(req.Columns...)
				}

				return q
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.FeedbackTable.Alias,
					req.Filter,
					req.Cursor,
					(*aifeedback.Feedback)(nil),
				)
			},
		})
	if err != nil {
		log.Error("failed to list ai feedback", zap.Error(err))

		return nil, err
	}

	return result, nil
}

func (r *repository) DailySatisfaction(
	ctx context.Context,
	req repositories.AIFeedbackWindowRequest,
) ([]*repositories.AIFeedbackDay, error) {
	cols := buncolgen.FeedbackColumns
	rows := make([]*repositories.AIFeedbackDay, 0)
	day := cols.CreatedAt.Expr("to_char(to_timestamp({}) AT TIME ZONE ?, 'YYYY-MM-DD')")

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*aifeedback.Feedback)(nil)).
		ColumnExpr(day+" AS day", timezoneOf(req.Timezone)).
		ColumnExpr(buncolgen.CountFilter("positive", cols.Rating.Eq()), aifeedback.RatingPositive).
		ColumnExpr(buncolgen.CountFilter("negative", cols.Rating.Eq()), aifeedback.RatingNegative).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return windowScope(sq, req)
		}).
		GroupExpr("day").
		OrderExpr("day ASC").
		Scan(ctx, &rows)
	if err != nil {
		r.l.Error("failed to read ai feedback satisfaction by day", zap.Error(err))

		return nil, fmt.Errorf("ai feedback satisfaction by day: %w", err)
	}

	return rows, nil
}

func (r *repository) WorstRated(
	ctx context.Context,
	req repositories.AIFeedbackWindowRequest,
) ([]*repositories.AIFeedbackTargetScore, error) {
	cols := buncolgen.FeedbackColumns
	limit := req.Limit
	if limit <= 0 {
		limit = defaultWorstLimit
	}
	if limit > maxWorstLimit {
		limit = maxWorstLimit
	}

	countRating := cols.Rating.Expr("COUNT(*) FILTER (WHERE {} = ?)")
	rows := make([]*repositories.AIFeedbackTargetScore, 0, limit)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*aifeedback.Feedback)(nil)).
		ColumnExpr(cols.TargetType.As("target_type")).
		ColumnExpr(cols.TargetID.As("target_id")).
		ColumnExpr(cols.TargetPart.As("target_part")).
		ColumnExpr(countRating+" AS positive", aifeedback.RatingPositive).
		ColumnExpr(countRating+" AS negative", aifeedback.RatingNegative).
		ColumnExpr(buncolgen.Max(cols.CreatedAt, "last_rated_at")).
		ColumnExpr(buncolgen.Expr(
			"(ARRAY_AGG({0} ORDER BY {1} ASC, {2} DESC))[1] AS sample_id",
			cols.ID,
			cols.Rating,
			cols.CreatedAt,
		)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return windowScope(sq, req)
		}).
		GroupExpr(cols.TargetType.Qualified()).
		GroupExpr(cols.TargetID.Qualified()).
		GroupExpr(cols.TargetPart.Qualified()).
		Having(countRating+" > 0", aifeedback.RatingNegative).
		OrderExpr(
			"("+countRating+" - "+countRating+") DESC",
			aifeedback.RatingNegative,
			aifeedback.RatingPositive,
		).
		OrderExpr(countRating+" DESC", aifeedback.RatingNegative).
		OrderExpr(buncolgen.Expr("MAX({}) DESC", cols.CreatedAt)).
		OrderExpr(cols.TargetType.OrderAsc()).
		OrderExpr(cols.TargetID.OrderAsc()).
		OrderExpr(cols.TargetPart.OrderAsc()).
		Limit(limit).
		Offset(max(req.Offset, 0)).
		Scan(ctx, &rows)
	if err != nil {
		r.l.Error("failed to read worst-rated ai output", zap.Error(err))

		return nil, fmt.Errorf("worst-rated ai output: %w", err)
	}

	return rows, nil
}

func windowScope(sq *bun.SelectQuery, req repositories.AIFeedbackWindowRequest) *bun.SelectQuery {
	cols := buncolgen.FeedbackColumns

	sq = buncolgen.FeedbackScopeTenant(sq, req.TenantInfo).
		Where(cols.CreatedAt.Gte(), req.Since)
	if req.AgentDefinitionID.IsNotNil() {
		return sq.Where(cols.AgentDefinitionID.Eq(), req.AgentDefinitionID)
	}

	return sq.Where(cols.AgentDefinitionID.IsNotNull())
}

func (r *repository) TotalsByAgent(
	ctx context.Context,
	req repositories.AIFeedbackAgentTotalsRequest,
) ([]*repositories.AIFeedbackAgentTotals, error) {
	cols := buncolgen.FeedbackColumns
	rows := make([]*repositories.AIFeedbackAgentTotals, 0, len(req.AgentDefinitionIDs))

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*aifeedback.Feedback)(nil)).
		ColumnExpr(cols.AgentDefinitionID.As("agent_definition_id")).
		ColumnExpr(buncolgen.CountFilter("positive", cols.Rating.Eq()), aifeedback.RatingPositive).
		ColumnExpr(buncolgen.CountFilter("negative", cols.Rating.Eq()), aifeedback.RatingNegative).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.FeedbackScopeTenant(sq, req.TenantInfo).
				Where(cols.AgentDefinitionID.IsNotNull()).
				Where(cols.CreatedAt.Gte(), req.Since)
			if req.Until > 0 {
				sq = sq.Where(cols.CreatedAt.Lt(), req.Until)
			}
			if len(req.AgentDefinitionIDs) > 0 {
				sq = sq.Where(cols.AgentDefinitionID.In(), bun.List(req.AgentDefinitionIDs))
			}

			return sq
		}).
		GroupExpr(cols.AgentDefinitionID.Qualified()).
		Scan(ctx, &rows)
	if err != nil {
		r.l.Error("failed to count ai feedback by agent", zap.Error(err))

		return nil, fmt.Errorf("ai feedback totals by agent: %w", err)
	}

	return rows, nil
}

func timezoneOf(timezone string) string {
	if strings.TrimSpace(timezone) == "" {
		return "UTC"
	}

	return timezone
}

func (r *repository) ListNegativeSince(
	ctx context.Context,
	req repositories.ListNegativeAIFeedbackRequest,
) ([]*aifeedback.Feedback, error) {
	cols := buncolgen.FeedbackColumns
	limit := req.Limit
	if limit <= 0 {
		limit = defaultNegativeLimit
	}
	if limit > maxNegativeLimit {
		limit = maxNegativeLimit
	}
	rows := make([]*aifeedback.Feedback, 0, min(limit, defaultNegativeLimit))

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&rows).
		Column(
			cols.ID.Bare(),
			cols.BusinessUnitID.Bare(),
			cols.OrganizationID.Bare(),
			cols.UserID.Bare(),
			cols.TargetType.Bare(),
			cols.TargetID.Bare(),
			cols.TargetPart.Bare(),
			cols.ThreadID.Bare(),
			cols.AgentDefinitionID.Bare(),
			cols.Rating.Bare(),
			cols.Reasons.Bare(),
			cols.Comment.Bare(),
			cols.PatternKey.Bare(),
			cols.CreatedAt.Bare(),
		).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FeedbackScopeTenant(sq, req.TenantInfo).
				Where(cols.Rating.Eq(), aifeedback.RatingNegative).
				Where(cols.AgentDefinitionID.IsNotNull()).
				Where(cols.CreatedAt.Gte(), req.Since)
		}).
		OrderExpr(cols.CreatedAt.OrderDesc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list negative ai feedback", zap.Error(err))

		return nil, fmt.Errorf("list negative ai feedback: %w", err)
	}

	return rows, nil
}

func (r *repository) PurgeBefore(
	ctx context.Context,
	req repositories.PurgeAIFeedbackRequest,
) (int64, error) {
	cols := buncolgen.FeedbackColumns
	limit := req.Limit
	if limit <= 0 {
		limit = defaultPurgeLimit
	}
	if limit > maxPurgeLimit {
		limit = maxPurgeLimit
	}

	dba := r.db.DBForContext(ctx)
	expired := dba.NewSelect().
		Model((*aifeedback.Feedback)(nil)).
		Column(cols.ID.Bare()).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FeedbackScopeTenant(sq, req.TenantInfo).
				Where(cols.CreatedAt.Lt(), req.Before)
		}).
		OrderExpr(cols.CreatedAt.OrderAsc()).
		Limit(limit)

	res, err := dba.NewDelete().
		Model((*aifeedback.Feedback)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.FeedbackScopeTenantDelete(dq, req.TenantInfo).
				Where(cols.ID.In(), expired)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to purge expired ai feedback", zap.Error(err))

		return 0, fmt.Errorf("purge expired ai feedback: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("purge expired ai feedback rows: %w", err)
	}

	return rows, nil
}

func (r *repository) LinkEvalCase(
	ctx context.Context,
	req repositories.LinkAIFeedbackEvalCaseRequest,
) error {
	cols := buncolgen.FeedbackColumns

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*aifeedback.Feedback)(nil)).
		Set(cols.EvalCaseID.Set(), req.EvalCaseID).
		Set(cols.Version.Inc(1)).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.FeedbackScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.FeedbackID)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to link ai feedback to an evaluation case", zap.Error(err))

		return fmt.Errorf("link ai feedback to evaluation case: %w", err)
	}

	return dberror.CheckRowsAffected(res, "AIFeedback", req.FeedbackID.String())
}
