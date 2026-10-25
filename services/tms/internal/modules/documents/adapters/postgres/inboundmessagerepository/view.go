//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package inboundmessagerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage/triagedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

var viewErrors = dberror.Mapping{Entity: "View"}

type viewRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewViewRepository(p Params) inboundmessageports.TriageViewRepository {
	return &viewRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.triage-view-repository"),
	}
}

func (r *viewRepository) List(
	ctx context.Context,
	req inboundmessageports.ListViewsRequest,
) ([]*triage.View, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*triage.View, error) {
		v := triagedb.Views
		views, err := v.Where(v.UserID.Eq(req.UserID)).
			Order(v.Position.Asc(), v.CreatedAt.Asc(), v.ID.Asc()).
			Limit(triage.MaxViewsPerUser).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}
		if views == nil {
			views = []*triage.View{}
		}

		return views, nil
	})
}

func (r *viewRepository) Get(
	ctx context.Context,
	req inboundmessageports.GetViewRequest,
) (*triage.View, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triage.View, error) {
		v := triagedb.Views
		view, err := v.Where(v.ID.Eq(req.ID), v.UserID.Eq(req.UserID)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, viewErrors.Map(err, req.ID.String())
		}

		return view, nil
	})
}

func (r *viewRepository) Create(
	ctx context.Context,
	view *triage.View,
	limit int,
) (*triage.View, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*triage.View, error) {
		db := r.db.Keel(ctx)
		if err := keel.LockKey(
			ctx,
			db,
			"triage-views:"+view.OrganizationID.String()+":"+view.UserID.String(),
		); err != nil {
			return nil, err
		}

		type viewStats struct {
			Count        int64
			NextPosition int64
		}

		v := triagedb.Views
		stats, err := keel.IntoOne[viewStats](ctx, db, v.
			Pick(keel.Count(), keel.DynOf(v.Position).Max().Add(1).Coalesce(0)).
			Where(v.UserID.Eq(view.UserID)))
		if err != nil {
			return nil, err
		}
		if stats.Count >= int64(limit) {
			return nil, errortypes.NewBusinessError(
				"You can keep at most {0} saved views. Delete one to save another.", limit,
			)
		}

		view.Position = int(stats.NextPosition)
		if err = v.Create(ctx, db, view); err != nil {
			return nil, err
		}

		return view, nil
	})
}

func (r *viewRepository) Rename(ctx context.Context, view *triage.View) (*triage.View, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*triage.View, error) {
		v := triagedb.Views
		renamed, err := v.Where(v.ID.Eq(view.ID), v.UserID.Eq(view.UserID)).UpdateReturning(
			ctx,
			r.db.Keel(ctx),
			v.Name.To(stringutils.StripNUL(view.Name)),
			v.UpdatedAt.To(view.UpdatedAt),
		)
		if err != nil {
			return nil, err
		}
		if err = dberror.CheckAffected(int64(len(renamed)), "View", view.ID.String()); err != nil {
			return nil, err
		}

		return renamed[0], nil
	})
}

func (r *viewRepository) Delete(ctx context.Context, req inboundmessageports.GetViewRequest) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		v := triagedb.Views
		n, err := v.Where(v.ID.Eq(req.ID), v.UserID.Eq(req.UserID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return err
		}

		return dberror.CheckAffected(n, "View", req.ID.String())
	})
}

func (r *viewRepository) WaitingCounts(
	ctx context.Context,
	req inboundmessageports.ViewWaitingCountsRequest,
) (map[pulid.ID]int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]int, error) {
		counts := make(map[pulid.ID]int, len(req.ViewIDs))
		if len(req.ViewIDs) == 0 {
			return counts, nil
		}

		db := r.db.Keel(ctx)
		v := triagedb.Views
		views, err := v.Pick(v.ID, v.Query).
			Where(v.ID.In(req.ViewIDs...)).
			Limit(len(req.ViewIDs)).
			List(ctx, db)
		if err != nil {
			return nil, err
		}
		if len(views) == 0 {
			return counts, nil
		}

		type viewCount struct {
			ViewID pulid.ID
			Count  int64
		}

		rows, err := keel.Into[viewCount](ctx, db, waitingCountsQuery(views))
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			counts[row.ViewID] = int(row.Count)
		}

		return counts, nil
	})
}

func waitingCountsQuery(views []*triage.View) keel.Projection {
	first := waitingSearchCount(views[0])
	if len(views) == 1 {
		return first
	}

	counts := keel.UnionAll(first, waitingSearchCount(views[1]))
	for _, view := range views[2:] {
		counts = keel.UnionAll(counts, waitingSearchCount(view))
	}

	return counts
}

func waitingSearchCount(view *triage.View) *keel.Query[inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages
	q := m.Pick(keel.DynLit(view.ID.String()).Cast(keel.TypeString), keel.Count()).
		Where(m.Status.In(needsReviewStatuses...), m.Parked.IsNull())

	return searchPredicate(q, view.Query)
}
