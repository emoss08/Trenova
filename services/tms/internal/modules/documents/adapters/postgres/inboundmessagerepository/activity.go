//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package inboundmessagerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage/triagedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const activityRankCTE = "ranked_activities"

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

		return triagedb.Activities.CreateMany(ctx, r.db.Keel(ctx), activities)
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

		entities, err := listActivitiesQuery(req).All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}

		for _, entity := range entities {
			grouped[entity.MessageID] = append(grouped[entity.MessageID], entity)
		}

		return grouped, nil
	})
}

func listActivitiesQuery(
	req inboundmessageports.ListActivitiesRequest,
) *keel.Query[triage.Activity] {
	limit := req.Limit
	if limit <= 0 {
		limit = inboundmessageports.MaxActivitiesPerRead
	}

	a := triagedb.Activities
	ra := triagedb.Activities.As("ranked_activity")
	ranked := keel.CTE(activityRankCTE, []string{"id", "activity_rank"}, ra.
		Pick(ra.ID, keel.RowNumber().Over([]keel.Expr{ra.MessageID}, ra.CreatedAt.Desc(), ra.ID.Desc())).
		Where(ra.MessageID.In(req.MessageIDs...)))

	return a.Where(
		a.MessageID.In(req.MessageIDs...),
		a.ID.InQuery(ranked.Query().
			With(ranked).
			Pick(ranked.Col("id")).
			Where(ranked.Col("activity_rank").Lte(limit))),
	).Order(a.CreatedAt.Asc(), a.ID.Asc())
}
