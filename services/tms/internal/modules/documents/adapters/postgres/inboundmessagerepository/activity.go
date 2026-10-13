//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package inboundmessagerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/buncolgen/triagecol"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type activityRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewActivityRepository(p Params) inboundmessageports.TriageActivityRepository {
	return &activityRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.triage-activity-repository"),
	}
}

func (r *activityRepository) Insert(ctx context.Context, activities []*triage.Activity) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(activities) == 0 {
			return nil
		}

		_, err := r.db.DBForContext(ctx).NewInsert().Model(&activities).Exec(ctx)

		return err
	})
}

func (r *activityRepository) ListByMessageIDs(
	ctx context.Context,
	req inboundmessageports.ListActivitiesRequest,
) (map[pulid.ID][]*triage.Activity, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID][]*triage.Activity, error) {
		grouped := make(map[pulid.ID][]*triage.Activity, len(req.MessageIDs))
		if len(req.MessageIDs) == 0 {
			return grouped, nil
		}

		entities := make([]*triage.Activity, 0, len(req.MessageIDs)*4)
		if err := listActivitiesQuery(
			r.db.DBForContext(ctx),
			req,
			&entities,
		).Scan(ctx); err != nil {
			return nil, err
		}
		for _, entity := range entities {
			grouped[entity.MessageID] = append(grouped[entity.MessageID], entity)
		}

		return grouped, nil
	})
}

const activityRankAlias = "ranked"

func listActivitiesQuery(
	db bun.IDB,
	req inboundmessageports.ListActivitiesRequest,
	entities *[]*triage.Activity,
) *bun.SelectQuery {
	cols := triagecol.ActivityColumns
	limit := req.Limit
	if limit <= 0 {
		limit = inboundmessageports.MaxActivitiesPerRead
	}

	ranked := db.NewSelect().
		Model((*triage.Activity)(nil)).
		ColumnExpr(cols.ID.Qualified()).
		ColumnExpr(
			"ROW_NUMBER() OVER (PARTITION BY ? ORDER BY ? DESC, ? DESC) AS activity_rank",
			bun.Safe(cols.MessageID.Qualified()),
			bun.Safe(cols.CreatedAt.Qualified()),
			bun.Safe(cols.ID.Qualified()),
		).
		Apply(triagecol.ActivityApplyTenant(req.TenantInfo)).
		Where(cols.MessageID.In(), bun.List(req.MessageIDs))
	kept := db.NewSelect().
		TableExpr("(?) AS ?", ranked, bun.Ident(activityRankAlias)).
		ColumnExpr(cols.ID.WithAlias(activityRankAlias).Qualified()).
		Where("activity_rank <= ?", limit)

	return db.NewSelect().
		Model(entities).
		Apply(triagecol.ActivityApplyTenant(req.TenantInfo)).
		Where(cols.MessageID.In(), bun.List(req.MessageIDs)).
		Where(cols.ID.In(), kept).
		Order(cols.CreatedAt.OrderAsc(), cols.ID.OrderAsc())
}
