package rateconfirmationrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation/rateconfirmationdb"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateconfirmationports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const defaultTokenPurgeLimit = 500

var (
	confirmationErrors = dberror.Mapping{Entity: "Rate confirmation"}
	tokenErrors        = dberror.Mapping{Entity: "Rate confirmation token"}
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

func New(p Params) rateconfirmationports.RateConfirmationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.rate-confirmation-repository"),
	}
}

func (r *repository) GetByID(
	ctx context.Context,
	req *rateconfirmationports.GetRateConfirmationByIDRequest,
) (*rateconfirmation.RateConfirmation, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*rateconfirmation.RateConfirmation, error) {
			rc := rateconfirmationdb.RateConfirmations
			entity, err := rc.Where(rc.ID.Eq(req.RateConfirmationID)).
				Load(rc.Carrier(), rc.CarrierAssignment()).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to get rate confirmation", zap.Error(err))
				return nil, confirmationErrors.Map(err, req.RateConfirmationID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetActiveByAssignmentID(
	ctx context.Context,
	_ pagination.TenantInfo,
	assignmentID pulid.ID,
) (*rateconfirmation.RateConfirmation, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*rateconfirmation.RateConfirmation, error) {
			rc := rateconfirmationdb.RateConfirmations
			entity, err := rc.Where(
				rc.CarrierAssignmentID.Eq(assignmentID),
				rc.Status.Ne(rateconfirmation.StatusVoided),
			).One(ctx, r.db.Keel(ctx))
			if err != nil {
				if dberror.IsNotFoundError(err) {
					return nil, nil //nolint:nilnil // nil result represents an optional absence in this API
				}
				r.l.Error("failed to get active rate confirmation", zap.Error(err))
				return nil, err
			}

			return entity, nil
		},
	)
}

func (r *repository) ListByMoveID(
	ctx context.Context,
	req *rateconfirmationports.ListRateConfirmationsByMoveRequest,
) ([]*rateconfirmation.RateConfirmation, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*rateconfirmation.RateConfirmation, error) {
			rc := rateconfirmationdb.RateConfirmations
			entities, err := rc.Where(rc.ShipmentMoveID.Eq(req.ShipmentMoveID)).
				Load(rc.Carrier()).
				Order(rc.Revision.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list rate confirmations", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func maxRevisionQuery(assignmentID pulid.ID) *keel.Query[rateconfirmation.RateConfirmation] {
	rc := rateconfirmationdb.RateConfirmations

	return rc.Pick(rc.Revision.Max()).Where(rc.CarrierAssignmentID.Eq(assignmentID))
}

func (r *repository) MaxRevisionForAssignment(
	ctx context.Context,
	_ pagination.TenantInfo,
	assignmentID pulid.ID,
) (int64, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int64, error) {
		maxRevision, err := keel.IntoOne[*int64](
			ctx,
			r.db.Keel(ctx),
			maxRevisionQuery(assignmentID),
		)
		if err != nil {
			r.l.Error("failed to resolve max rate confirmation revision", zap.Error(err))
			return 0, err
		}

		if maxRevision == nil {
			return 0, nil
		}

		return *maxRevision, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *rateconfirmation.RateConfirmation,
) (*rateconfirmation.RateConfirmation, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*rateconfirmation.RateConfirmation, error) {
			err := rateconfirmationdb.RateConfirmations.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to create rate confirmation", zap.Error(err))
				return nil, confirmationErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *rateconfirmation.RateConfirmation,
) (*rateconfirmation.RateConfirmation, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*rateconfirmation.RateConfirmation, error) {
			err := rateconfirmationdb.RateConfirmations.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to update rate confirmation", zap.Error(err))
				return nil, confirmationErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) CreateToken(
	ctx context.Context,
	entity *rateconfirmation.RateConfirmationToken,
) (*rateconfirmation.RateConfirmationToken, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*rateconfirmation.RateConfirmationToken, error) {
			err := rateconfirmationdb.RateConfirmationTokens.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to create rate confirmation token", zap.Error(err))
				return nil, tokenErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

// GetTokenByHash resolves a public token. It is deliberately not
// tenant-scoped: the caller only holds the opaque token, and the row itself
// carries the tenant that every follow-up query is scoped to.
func (r *repository) GetTokenByHash(
	ctx context.Context,
	tokenHash string,
) (*rateconfirmation.RateConfirmationToken, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"resolve a public rate confirmation link before its tenant is known",
	)
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*rateconfirmation.RateConfirmationToken, error) {
			t := rateconfirmationdb.RateConfirmationTokens
			entity, err := t.AllTenants().Where(t.TokenHash.Eq(tokenHash)).One(ctx, r.db.Keel(ctx))
			if err != nil {
				if dberror.IsNotFoundError(err) {
					return nil, nil //nolint:nilnil // nil token represents an optional absence
				}
				r.l.Error("failed to get rate confirmation token", zap.Error(err))
				return nil, err
			}

			return entity, nil
		},
	)
}

// MarkTokenUsed is the single-use gate for public sign links: the conditional
// update only wins for the first caller.
func (r *repository) MarkTokenUsed(
	ctx context.Context,
	req *rateconfirmationports.MarkRateConfirmationTokenUsedRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		t := rateconfirmationdb.RateConfirmationTokens
		affected, err := t.Where(
			t.ID.Eq(req.TokenID),
			t.UsedAt.IsNull(),
			t.RevokedAt.IsNull(),
		).Update(ctx, r.db.Keel(ctx), t.UsedAt.To(req.UsedAt))
		if err != nil {
			r.l.Error("failed to mark rate confirmation token used", zap.Error(err))
			return false, err
		}

		return affected > 0, nil
	})
}

func (r *repository) RevokeTokensForRateConfirmation(
	ctx context.Context,
	_ pagination.TenantInfo,
	rateConfirmationID pulid.ID,
	revokedAt int64,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		t := rateconfirmationdb.RateConfirmationTokens
		_, err := t.Where(
			t.RateConfirmationID.Eq(rateConfirmationID),
			t.RevokedAt.IsNull(),
			t.UsedAt.IsNull(),
		).Update(ctx, r.db.Keel(ctx), t.RevokedAt.To(revokedAt))
		if err != nil {
			r.l.Error("failed to revoke rate confirmation tokens", zap.Error(err))
		}

		return err
	})
}

func deadTokensQuery(
	req tenderports.PurgeDeadTokensRequest,
	limit int,
) *keel.Query[rateconfirmation.RateConfirmationToken] {
	t := rateconfirmationdb.RateConfirmationTokens
	dead := t.As("dead")

	return t.AllTenants().Where(t.ID.InQuery(dead.AllTenants().
		Pick(dead.ID).
		Where(keel.DynOf(dead.UsedAt).Coalesce(dead.RevokedAt, dead.ExpiresAt).Lt(req.DeadBefore)).
		Limit(limit)))
}

// PurgeDeadSignTokens deletes signing links that can no longer authorize
// anything — used, revoked, or expired — once the retention window past their
// terminal moment has elapsed. A live link always coalesces to a future
// expires_at, so it can never match.
func (r *repository) PurgeDeadSignTokens(
	ctx context.Context,
	req tenderports.PurgeDeadTokensRequest,
) (int64, error) {
	ctx = dbscope.WithSystem(ctx, "purge dead rate confirmation tokens across every organization")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		limit := req.Limit
		if limit <= 0 {
			limit = defaultTokenPurgeLimit
		}

		purged, err := deadTokensQuery(req, limit).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to purge dead rate confirmation tokens", zap.Error(err))
			return 0, err
		}

		return purged, nil
	})
}
