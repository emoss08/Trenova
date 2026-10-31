package palettequestionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/palettequestion"
	"github.com/emoss08/trenova/internal/modules/agent/domain/palettequestion/palettequestiondb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/paletteports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxDeleteBatch = 1000

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

var _ paletteports.QuestionRepository = (*repository)(nil)

func New(p Params) paletteports.QuestionRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.palette-question-repository")}
}

func (r *repository) Create(ctx context.Context, question *palettequestion.Question) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := palettequestiondb.Questions.Create(ctx, r.db.Keel(ctx), question); err != nil {
			r.l.Error("failed to record a palette question", zap.Error(err))

			return err
		}

		return nil
	})
}

func (r *repository) Get(
	ctx context.Context,
	ref paletteports.AskRef,
) (*palettequestion.Question, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*palettequestion.Question, error) {
		q := palettequestiondb.Questions
		question, err := q.Where(q.ID.Eq(ref.AskID), q.UserID.Eq(ref.TenantInfo.UserID)).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "Question")
		}

		return question, nil
	})
}

func (r *repository) CountAsking(
	ctx context.Context,
	tenant pagination.TenantInfo,
	since int64,
) (int, error) {
	return r.count(ctx, tenant, since, palettequestiondb.Questions.Status.Eq(palettequestion.StatusAsking))
}

func (r *repository) CountUnkeptSince(
	ctx context.Context,
	tenant pagination.TenantInfo,
	since int64,
) (int, error) {
	return r.count(ctx, tenant, since, palettequestiondb.Questions.KeptThreadID.IsNull())
}

func (r *repository) count(
	ctx context.Context,
	tenant pagination.TenantInfo,
	since int64,
	also keel.Pred,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		q := palettequestiondb.Questions
		n, err := q.Where(q.UserID.Eq(tenant.UserID), q.CreatedAt.Gte(since), also).
			Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("count palette questions: %w", err)
		}

		return int(n), nil
	})
}

func (r *repository) Finish(
	ctx context.Context,
	ref paletteports.AskRef,
	status palettequestion.Status,
	at int64,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		q := palettequestiondb.Questions
		if _, err := q.Where(q.ID.Eq(ref.AskID), q.Status.Eq(palettequestion.StatusAsking)).
			Update(ctx, r.db.Keel(ctx), q.Status.To(status), q.FinishedAt.To(at)); err != nil {
			return fmt.Errorf("finish palette question: %w", err)
		}

		return nil
	})
}

func (r *repository) ClaimKeep(
	ctx context.Context,
	ref paletteports.AskRef,
	now int64,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		q := palettequestiondb.Questions
		n, err := q.Where(
			q.ID.Eq(ref.AskID),
			q.UserID.Eq(ref.TenantInfo.UserID),
			q.KeptThreadID.IsNull(),
			keel.Or(
				q.KeepClaimedAt.IsNull(),
				q.KeepClaimedAt.Lt(now-palettequestion.KeepClaimWindow),
			),
		).Update(ctx, r.db.Keel(ctx), q.KeepClaimedAt.To(now))
		if err != nil {
			return false, fmt.Errorf("claim palette question: %w", err)
		}

		return n == 1, nil
	})
}

func (r *repository) MarkKept(
	ctx context.Context,
	ref paletteports.AskRef,
	threadID pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		q := palettequestiondb.Questions
		if _, err := q.Where(q.ID.Eq(ref.AskID), q.UserID.Eq(ref.TenantInfo.UserID)).
			Update(ctx, r.db.Keel(ctx), q.KeptThreadID.To(threadID)); err != nil {
			return fmt.Errorf("mark palette question kept: %w", err)
		}

		return nil
	})
}

func (r *repository) ReleaseKeep(ctx context.Context, ref paletteports.AskRef) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		q := palettequestiondb.Questions
		if _, err := q.Where(q.ID.Eq(ref.AskID), q.KeptThreadID.IsNull()).
			Update(ctx, r.db.Keel(ctx), q.KeepClaimedAt.ToNull()); err != nil {
			return fmt.Errorf("release palette question: %w", err)
		}

		return nil
	})
}

func (r *repository) DeleteBefore(ctx context.Context, before int64, limit int) (int, error) {
	ctx = dbscope.WithSystem(ctx, "delete palette questions past retention across every organization")
	if before <= 0 {
		return 0, nil
	}
	if limit <= 0 || limit > maxDeleteBatch {
		limit = maxDeleteBatch
	}

	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		db := r.db.Keel(ctx)
		q := palettequestiondb.Questions
		stale, err := q.AllTenants().
			Pick(q.ID, q.OrganizationID, q.BusinessUnitID).
			Where(q.CreatedAt.Lt(before)).
			Order(q.CreatedAt.Asc()).
			Limit(limit).
			List(ctx, db)
		if err != nil || len(stale) == 0 {
			return 0, err
		}

		ids := make([]pulid.ID, len(stale))
		orgs := make([]pulid.ID, len(stale))
		bus := make([]pulid.ID, len(stale))
		for idx, question := range stale {
			ids[idx] = question.ID
			orgs[idx] = question.OrganizationID
			bus[idx] = question.BusinessUnitID
		}
		doomed := keel.Unnest(
			"doomed",
			[]string{"id", "organization_id", "business_unit_id"},
			ids, orgs, bus,
		)
		n, err := q.AllTenants().
			With(doomed).
			Join(doomed, keel.And(
				doomed.Col("id").EqCol(q.ID),
				doomed.Col("organization_id").EqCol(q.OrganizationID),
				doomed.Col("business_unit_id").EqCol(q.BusinessUnitID),
			)).
			Where(keel.True()).
			Delete(ctx, db)
		if err != nil {
			r.l.Error("failed to delete stale palette questions", zap.Error(err))

			return 0, err
		}

		return int(n), nil
	})
}
