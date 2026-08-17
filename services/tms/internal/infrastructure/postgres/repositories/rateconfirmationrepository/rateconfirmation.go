package rateconfirmationrepository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

const defaultTokenPurgeLimit = 500

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.RateConfirmationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.rate-confirmation-repository"),
	}
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetRateConfirmationByIDRequest,
) (*rateconfirmation.RateConfirmation, error) {
	cols := buncolgen.RateConfirmationColumns
	entity := new(rateconfirmation.RateConfirmation)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("Carrier").
		Relation("CarrierAssignment").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateConfirmationScopeTenant(sq, req.TenantInfo).
				Where(cols.ID.Eq(), req.RateConfirmationID)
		}).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to get rate confirmation", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Rate confirmation")
	}

	return entity, nil
}

func (r *repository) GetActiveByAssignmentID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	assignmentID pulid.ID,
) (*rateconfirmation.RateConfirmation, error) {
	cols := buncolgen.RateConfirmationColumns
	entity := new(rateconfirmation.RateConfirmation)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateConfirmationScopeTenant(sq, tenantInfo).
				Where(cols.CarrierAssignmentID.Eq(), assignmentID).
				Where(cols.Status.Ne(), rateconfirmation.StatusVoided)
		}).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // nil result represents an optional absence in this API
		}
		r.l.Error("failed to get active rate confirmation", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) ListByMoveID(
	ctx context.Context,
	req *repositories.ListRateConfirmationsByMoveRequest,
) ([]*rateconfirmation.RateConfirmation, error) {
	cols := buncolgen.RateConfirmationColumns
	entities := make([]*rateconfirmation.RateConfirmation, 0, 4)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation("Carrier").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateConfirmationScopeTenant(sq, req.TenantInfo).
				Where(cols.ShipmentMoveID.Eq(), req.ShipmentMoveID)
		}).
		Order(cols.Revision.OrderDesc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list rate confirmations", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) MaxRevisionForAssignment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	assignmentID pulid.ID,
) (int64, error) {
	cols := buncolgen.RateConfirmationColumns
	var maxRevision sql.NullInt64
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*rateconfirmation.RateConfirmation)(nil)).
		ColumnExpr("MAX(revision)").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.RateConfirmationScopeTenant(sq, tenantInfo).
				Where(cols.CarrierAssignmentID.Eq(), assignmentID)
		}).
		Scan(ctx, &maxRevision)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		r.l.Error("failed to resolve max rate confirmation revision", zap.Error(err))
		return 0, err
	}

	if !maxRevision.Valid {
		return 0, nil
	}
	return maxRevision.Int64, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *rateconfirmation.RateConfirmation,
) (*rateconfirmation.RateConfirmation, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create rate confirmation", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *rateconfirmation.RateConfirmation,
) (*rateconfirmation.RateConfirmation, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update rate confirmation", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(
		results,
		"Rate confirmation",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) CreateToken(
	ctx context.Context,
	entity *rateconfirmation.RateConfirmationToken,
) (*rateconfirmation.RateConfirmationToken, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to create rate confirmation token", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

// GetTokenByHash resolves a public token. It is deliberately not
// tenant-scoped: the caller only holds the opaque token, and the row itself
// carries the tenant that every follow-up query is scoped to.
func (r *repository) GetTokenByHash(
	ctx context.Context,
	tokenHash string,
) (*rateconfirmation.RateConfirmationToken, error) {
	cols := buncolgen.RateConfirmationTokenColumns
	entity := new(rateconfirmation.RateConfirmationToken)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.TokenHash.Eq(), tokenHash).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // nil token represents an optional absence
		}
		r.l.Error("failed to get rate confirmation token", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

// MarkTokenUsed is the single-use gate for public sign links: the conditional
// update only wins for the first caller.
func (r *repository) MarkTokenUsed(
	ctx context.Context,
	req *repositories.MarkRateConfirmationTokenUsedRequest,
) (bool, error) {
	cols := buncolgen.RateConfirmationTokenColumns
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*rateconfirmation.RateConfirmationToken)(nil)).
		Set("used_at = ?", req.UsedAt).
		Set("updated_at = ?", timeutils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.RateConfirmationTokenScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.TokenID).
				Where(cols.UsedAt.IsNull()).
				Where(cols.RevokedAt.IsNull())
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to mark rate confirmation token used", zap.Error(err))
		return false, err
	}

	affected, err := results.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

func (r *repository) RevokeTokensForRateConfirmation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	rateConfirmationID pulid.ID,
	revokedAt int64,
) error {
	cols := buncolgen.RateConfirmationTokenColumns
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*rateconfirmation.RateConfirmationToken)(nil)).
		Set("revoked_at = ?", revokedAt).
		Set("updated_at = ?", timeutils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.RateConfirmationTokenScopeTenantUpdate(uq, tenantInfo).
				Where(cols.RateConfirmationID.Eq(), rateConfirmationID).
				Where(cols.RevokedAt.IsNull()).
				Where(cols.UsedAt.IsNull())
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to revoke rate confirmation tokens", zap.Error(err))
	}
	return err
}

// PurgeDeadSignTokens deletes signing links that can no longer authorize
// anything — used, revoked, or expired — once the retention window past their
// terminal moment has elapsed. A live link always coalesces to a future
// expires_at, so it can never match.
func (r *repository) PurgeDeadSignTokens(
	ctx context.Context,
	req repositories.PurgeDeadTokensRequest,
) (int64, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultTokenPurgeLimit
	}

	cols := buncolgen.RateConfirmationTokenColumns
	deadAt := buncolgen.Expr(
		"COALESCE({0}, {1}, {2})", cols.UsedAt, cols.RevokedAt, cols.ExpiresAt,
	)
	dba := r.db.DBForContext(ctx)

	dead := dba.NewSelect().
		Model((*rateconfirmation.RateConfirmationToken)(nil)).
		Column(cols.ID.Bare()).
		Where(deadAt+" < ?", req.DeadBefore).
		Limit(limit)

	result, err := dba.NewDelete().
		Model((*rateconfirmation.RateConfirmationToken)(nil)).
		Where(cols.ID.Expr("{} IN (?)"), dead).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to purge dead rate confirmation tokens", zap.Error(err))
		return 0, err
	}

	return result.RowsAffected()
}
