package briefingrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/briefing"
	"github.com/emoss08/trenova/internal/modules/agent/domain/briefing/briefingdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/briefingports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultListLimit = 14
	maxListLimit     = 90
	maxDeleteBatch   = 1000
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

func New(p Params) briefingports.BriefingRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.briefing-repository"),
	}
}

func (r *repository) Upsert(
	ctx context.Context,
	entity *briefing.Briefing,
) (*briefing.Briefing, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*briefing.Briefing, error) {
		b := briefingdb.Briefings
		inserted, err := b.Upsert(
			ctx,
			r.db.Keel(ctx),
			[]*briefing.Briefing{entity},
			upsertConflict(),
		)
		if err != nil {
			r.l.Error("failed to upsert briefing",
				zap.String("role", string(entity.RoleKey)),
				zap.String("date", entity.BriefingDate),
				zap.Error(err))

			return nil, fmt.Errorf("upsert briefing: %w", err)
		}
		if len(inserted) > 0 && inserted[0] {
			return entity, nil
		}

		stored, err := forDay(entity.RoleKey, entity.BriefingDate, entity.UserID).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read upserted briefing: %w", err)
		}

		return stored, nil
	})
}

func upsertConflict() keel.Conflict {
	b := briefingdb.Briefings
	return keel.OnConflict(
		b.OrganizationID,
		b.BusinessUnitID,
		b.RoleKey,
		b.BriefingDate,
		b.UserID.Coalesce(keel.ConstText("")),
	).Update(
		b.RunID,
		b.Status,
		b.Headline,
		b.Sections,
		b.Facts,
		b.Narrated,
		b.ModelIdentifier,
		b.ProviderID,
		b.FailureReason,
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req briefingports.GetBriefingByIDRequest,
) (*briefing.Briefing, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*briefing.Briefing, error) {
		entity, err := briefingdb.Briefings.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, errortypes.NewNotFoundError("Briefing not found")
			}

			return nil, fmt.Errorf("get briefing: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) GetForDay(
	ctx context.Context,
	req briefingports.GetBriefingForDayRequest,
) (*briefing.Briefing, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*briefing.Briefing, error) {
		entity, err := forDay(req.RoleKey, req.BriefingDate, req.UserID).One(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil
			}

			return nil, fmt.Errorf("get briefing for day: %w", err)
		}

		return entity, nil
	})
}

func forDay(
	roleKey briefing.RoleKey,
	briefingDate string,
	userID *pulid.ID,
) *keel.Query[*briefing.Briefing] {
	b := briefingdb.Briefings
	return b.Where(
		b.RoleKey.Eq(roleKey),
		b.BriefingDate.Eq(briefingDate),
		readerPred(userID),
	)
}

func (r *repository) List(
	ctx context.Context,
	req briefingports.ListBriefingsQuery,
) ([]*briefing.Briefing, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*briefing.Briefing, error) {
		entities, err := listQuery(&req).List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list briefings: %w", err)
		}

		return entities, nil
	})
}

func listQuery(req *briefingports.ListBriefingsQuery) *keel.Query[*briefing.Briefing] {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	b := briefingdb.Briefings
	return b.Where(
		keel.When(req.RoleKey != "", b.RoleKey.Eq(req.RoleKey)),
		readerPred(req.UserID),
	).
		Order(b.BriefingDate.Desc()).
		Limit(limit)
}

func (r *repository) MarkRead(
	ctx context.Context,
	req briefingports.GetBriefingByIDRequest,
	readAt int64,
) (*briefing.Briefing, error) {
	return r.stamp(ctx, req, briefingdb.Briefings.ReadAt, readAt)
}

func (r *repository) MarkEmailed(
	ctx context.Context,
	req briefingports.GetBriefingByIDRequest,
	emailedAt int64,
) (*briefing.Briefing, error) {
	return r.stamp(ctx, req, briefingdb.Briefings.EmailedAt, emailedAt)
}

func (r *repository) DeleteBefore(
	ctx context.Context,
	req briefingports.DeleteBriefingsBeforeRequest,
) (int, error) {
	ctx = dbscope.WithSystem(ctx, "delete briefings past retention across every organization")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		limit := req.Limit
		if limit <= 0 || limit > maxDeleteBatch {
			limit = maxDeleteBatch
		}

		b := briefingdb.Briefings
		n, err := b.AllTenants().
			Where(staleBatch(req.BeforeDate, limit)).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("delete briefings: %w", err)
		}

		return int(n), nil
	})
}

func staleBatch(beforeDate string, limit int) keel.Pred {
	b := briefingdb.Briefings
	stale := b.As("stale")
	return b.ID.InQuery(stale.AllTenants().
		Pick(stale.ID).
		Where(stale.BriefingDate.Lt(beforeDate)).
		Order(stale.BriefingDate.Asc()).
		Limit(limit))
}

func (r *repository) stamp(
	ctx context.Context,
	req briefingports.GetBriefingByIDRequest,
	column keel.Int64Col,
	at int64,
) (*briefing.Briefing, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*briefing.Briefing, error) {
		b := briefingdb.Briefings
		rows, err := b.Where(b.ID.Eq(req.ID)).UpdateReturning(ctx, r.db.Keel(ctx), column.To(at))
		if err != nil {
			return nil, fmt.Errorf("stamp briefing: %w", err)
		}
		if err = dberror.CheckAffected(int64(len(rows)), "Briefing", req.ID.String()); err != nil {
			return nil, err
		}

		return rows[0], nil
	})
}

func readerPred(userID *pulid.ID) keel.Pred {
	b := briefingdb.Briefings
	if userID == nil || userID.IsNil() {
		return b.UserID.IsNull()
	}

	return b.UserID.Eq(*userID)
}
