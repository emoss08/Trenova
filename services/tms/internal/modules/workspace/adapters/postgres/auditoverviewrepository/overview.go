package auditoverviewrepository

import (
	"context"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/rollupread"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/auditrollup"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit/auditdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
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

func New(p Params) auditports.OverviewRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.audit-overview-repository")}
}

func (r *repository) OrganizationTimezone(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (string, error) {
		org := tenantdb.Organizations
		var zone string
		err := keel.Batch(ctx, r.db.Keel(ctx), keel.IntoValue(
			org.Pick(org.Timezone).
				Where(org.ID.Eq(tenant.OrgID)),
			&zone,
		))
		if err != nil {
			r.l.Error("failed to read the organization's time zone", zap.Error(err))
			return "", err
		}

		return zone, nil
	})
}

type activityRow struct {
	At           time.Time
	People       int64
	AgentsAsked  int64
	AgentsOwn    int64
	Integrations int64
	Jobs         int64
}

type actorKindRow struct {
	PrincipalType string
	Principals    int64
}

func (r *repository) Overview(
	ctx context.Context,
	req *auditports.OverviewRequest,
) (*auditports.OverviewRollup, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*auditports.OverviewRollup, error) {
		out := &auditports.OverviewRollup{}
		figures := auditrollup.DailyTable
		if req.Hourly {
			figures = auditrollup.HourlyTable
		}

		var activity []activityRow
		var kinds []actorKindRow
		var openCount int64
		items := []keel.BatchItem{
			keel.IntoValue(totals(figures, req.Access, req.Current), &out.Totals),
			keel.IntoValue(
				keel.RowsOf(figures.Ref).
					Pick(figures.Count(auditrollup.Entries)).
					Where(window(figures, req.Access, req.Previous)...),
				&out.PreviousEntries,
			),
			keel.IntoSlice(activityOf(figures, req.Access, req.Current), &activity),
			keel.IntoSlice(actorKinds(req.Access, req.Current), &kinds),
			keel.IntoSlice(topActors(req.Access, req.Current, req.ActorLimit), &out.TopActors),
		}
		if req.IncludeReviews {
			rv := auditdb.Reviews
			open := rv.Where(rv.ReviewedAt.IsNull())
			items = append(
				items,
				keel.ListInto(
					open.Clone().Order(rv.OccurredAt.Desc(), rv.ID.Desc()).Limit(req.ReviewLimit),
					&out.OpenReviews,
				),
				keel.CountInto(open.Clone(), &openCount),
			)
		}

		if err := keel.Batch(ctx, r.db.Keel(ctx), items...); err != nil {
			r.l.Error("failed to read the audit overview", zap.Error(err))
			return nil, err
		}

		out.Activity = make([]auditports.ActivityBucket, 0, len(activity))
		for _, row := range activity {
			out.Activity = append(out.Activity, auditports.ActivityBucket{
				At:           row.At,
				People:       row.People,
				Agents:       row.AgentsAsked + row.AgentsOwn,
				Integrations: row.Integrations,
				Jobs:         row.Jobs,
			})
		}
		out.Actors = actorCounts(kinds)
		out.OpenReviewCount = int(openCount)

		if err := r.nameActors(ctx, req, out.TopActors); err != nil {
			return nil, err
		}

		return out, nil
	})
}

func (r *repository) nameActors(
	ctx context.Context,
	req *auditports.OverviewRequest,
	actors []auditports.ActorStat,
) error {
	if len(actors) == 0 {
		return nil
	}

	e := auditdb.Entries
	names := make([][]string, len(actors))
	items := make([]keel.BatchItem, 0, len(actors))
	for i := range actors {
		items = append(items, keel.IntoSlice(
			e.Pick(e.ActorName).
				Where(
					e.PrincipalID.Eq(actors[i].PrincipalID),
					e.Timestamp.Gte(req.Current.From),
					e.Timestamp.Lte(req.Current.To),
					e.ActorName.IsNotNull(),
				).
				Order(e.Timestamp.Desc()).
				Limit(1),
			&names[i],
		))
	}
	if err := keel.Batch(ctx, r.db.Keel(ctx), items...); err != nil {
		r.l.Error("failed to name the audit overview's actors", zap.Error(err))
		return err
	}

	for i := range actors {
		if len(names[i]) > 0 {
			actors[i].Name = names[i][0]
		}
	}

	return nil
}

func access(t *rollupread.Table, a auditports.EntryAccess) []keel.Pred {
	preds := make([]keel.Pred, 0, 2)
	if !a.AllResources {
		names := make([]string, 0, len(a.Resources))
		for _, resource := range a.Resources {
			names = append(names, resource.String())
		}
		preds = append(preds, keel.StringOf(t.Col(auditrollup.Resource)).In(names...))
	}
	if !a.IncludeSecurity {
		preds = append(
			preds,
			keel.StringOf(t.Col(auditrollup.Category)).Ne(string(audit.CategorySecurity)),
		)
	}

	return preds
}

func window(
	t *rollupread.Table,
	a auditports.EntryAccess,
	w auditports.OverviewWindow,
) []keel.Pred {
	return append([]keel.Pred{
		t.Bucket().Gte(time.Unix(w.From, 0)),
		t.Bucket().Lte(time.Unix(w.To, 0)),
	}, access(t, a)...)
}

func totals(
	t *rollupread.Table,
	a auditports.EntryAccess,
	w auditports.OverviewWindow,
) keel.Projection {
	picks := make([]keel.Expr, 0, len(auditrollup.FigureColumns))
	for _, name := range auditrollup.FigureColumns {
		picks = append(picks, t.Count(name))
	}

	return keel.RowsOf(t.Ref).Pick(picks...).Where(window(t, a, w)...)
}

func activityOf(
	t *rollupread.Table,
	a auditports.EntryAccess,
	w auditports.OverviewWindow,
) keel.Projection {
	bucket := t.Bucket()
	return keel.RowsOf(t.Ref).
		Group(bucket).
		Pick(
			bucket,
			t.Count(auditrollup.People),
			t.Count(auditrollup.AgentsAsked),
			t.Count(auditrollup.AgentsOwn),
			t.Count(auditrollup.Integrations),
			t.Count(auditrollup.Jobs),
		).
		Where(window(t, a, w)...).
		Order(bucket.Asc())
}

func actorKinds(a auditports.EntryAccess, w auditports.OverviewWindow) keel.Projection {
	t := auditrollup.ActorDailyTable
	principalType := t.Col(auditrollup.PrincipalType)
	return keel.RowsOf(t.Ref).
		Group(principalType).
		Pick(principalType, keel.CastInt64(t.Col(auditrollup.PrincipalID).CountDistinct())).
		Where(window(t, a, w)...)
}

func topActors(a auditports.EntryAccess, w auditports.OverviewWindow, limit int) keel.Projection {
	t := auditrollup.ActorDailyTable
	id := t.Col(auditrollup.PrincipalID)
	principalType := t.Col(auditrollup.PrincipalType)
	entries := t.Count(auditrollup.Entries)

	return keel.RowsOf(t.Ref).
		Group(id, principalType).
		Pick(id, principalType, keel.Text(""), entries, t.Count(auditrollup.Sensitive)).
		Where(window(t, a, w)...).
		Order(keel.Desc(entries), id.Asc()).
		Limit(limit)
}

func actorCounts(rows []actorKindRow) auditports.ActorCounts {
	var out auditports.ActorCounts
	for _, row := range rows {
		n := int(row.Principals)
		switch audit.ActorKindOfPrincipal(row.PrincipalType) {
		case audit.ActorAgents:
			out.Agents += n
		case audit.ActorIntegrations:
			out.Integrations += n
		case audit.ActorJobs:
			out.Jobs += n
		default:
			out.People += n
		}
	}

	return out
}
