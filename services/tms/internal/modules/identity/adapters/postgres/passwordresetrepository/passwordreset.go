package passwordresetrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const resolveTokenScopeReason = "resolve a password reset token before its tenant is known"

var tokenErrors = dberror.Mapping{Entity: "Password reset token"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) tenantports.PasswordResetTokenRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.password-reset-repository"),
	}
}

func tokens(ctx context.Context, preds ...keel.Pred) *keel.Query[*tenant.PasswordResetToken] {
	prt := tenantdb.PasswordResetTokens
	q := prt.Query()
	if dbscope.IsSystem(ctx) {
		q = q.AllTenants()
	}

	return q.Where(preds...)
}

func outstanding() keel.Pred {
	prt := tenantdb.PasswordResetTokens
	return keel.And(prt.UsedAt.IsNull(), prt.InvalidatedAt.IsNull())
}

func (r *repository) Create(ctx context.Context, token *tenant.PasswordResetToken) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		return tokenErrors.Map(
			tenantdb.PasswordResetTokens.Create(ctx, r.db.Keel(ctx), token),
			token.ID.String(),
		)
	})
}

func (r *repository) FindRedeemableByHash(
	ctx context.Context,
	tokenHash string,
	now int64,
) (*tenant.PasswordResetToken, error) {
	ctx = dbscope.WithSystem(ctx, resolveTokenScopeReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tenant.PasswordResetToken, error) {
		prt := tenantdb.PasswordResetTokens
		token, err := tokens(
			ctx,
			prt.TokenHash.Eq(tokenHash),
			outstanding(),
			prt.ExpiresAt.Gt(now),
		).
			Load(prt.User()).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, tokenErrors.Map(err, "")
		}
		return token, nil
	})
}

// MarkUsed puts the used_at guard in the UPDATE, not in a check before it: two
// tabs opened from the same email would otherwise both read an unused token and
// both reset the password. Whichever statement lands first updates a row; the
// second updates none.
func (r *repository) MarkUsed(
	ctx context.Context,
	tokenID pulid.ID,
	now int64,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		prt := tenantdb.PasswordResetTokens
		affected, err := tokens(ctx, prt.ID.Eq(tokenID), outstanding()).
			Update(ctx, r.db.Keel(ctx), prt.UsedAt.To(now))
		if err != nil {
			return false, err
		}
		return affected > 0, nil
	})
}

func (r *repository) InvalidateOutstanding(
	ctx context.Context,
	userID pulid.ID,
	now int64,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		prt := tenantdb.PasswordResetTokens
		_, err := tokens(ctx, prt.UserID.Eq(userID), outstanding()).
			Update(ctx, r.db.Keel(ctx), prt.InvalidatedAt.To(now))
		return err
	})
}

func (r *repository) CountSince(
	ctx context.Context,
	userID pulid.ID,
	since int64,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		prt := tenantdb.PasswordResetTokens
		n, err := tokens(
			ctx,
			prt.UserID.Eq(userID),
			prt.Purpose.Eq(tenant.PasswordResetPurposeReset),
			prt.CreatedAt.Gte(since),
		).Count(ctx, r.db.Keel(ctx))
		return int(n), err
	})
}
