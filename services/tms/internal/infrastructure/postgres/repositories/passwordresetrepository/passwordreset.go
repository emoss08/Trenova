package passwordresetrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
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

func New(p Params) repositories.PasswordResetTokenRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.password-reset-repository"),
	}
}

func (r *repository) Create(ctx context.Context, token *tenant.PasswordResetToken) error {
	_, err := r.db.DB().NewInsert().Model(token).Exec(ctx)
	return err
}

func (r *repository) FindRedeemableByHash(
	ctx context.Context,
	tokenHash string,
	now int64,
) (*tenant.PasswordResetToken, error) {
	token := new(tenant.PasswordResetToken)
	if err := r.db.DB().NewSelect().
		Model(token).
		Relation("User").
		Where("prt.token_hash = ?", tokenHash).
		Where("prt.used_at IS NULL").
		Where("prt.invalidated_at IS NULL").
		Where("prt.expires_at > ?", now).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Password reset token")
	}
	return token, nil
}

func (r *repository) MarkUsed(
	ctx context.Context,
	tokenID pulid.ID,
	now int64,
) (bool, error) {
	// The used_at guard is part of the UPDATE, not a check before it: two tabs opened
	// from the same email would otherwise both read an unused token and both reset the
	// password. Whichever statement lands first updates a row; the second updates none.
	result, err := r.db.DB().NewUpdate().
		Model((*tenant.PasswordResetToken)(nil)).
		Set("used_at = ?", now).
		Where("id = ?", tokenID).
		Where("used_at IS NULL").
		Where("invalidated_at IS NULL").
		Exec(ctx)
	if err != nil {
		return false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *repository) InvalidateOutstanding(
	ctx context.Context,
	userID pulid.ID,
	now int64,
) error {
	_, err := r.db.DB().NewUpdate().
		Model((*tenant.PasswordResetToken)(nil)).
		Set("invalidated_at = ?", now).
		Where("user_id = ?", userID).
		Where("used_at IS NULL").
		Where("invalidated_at IS NULL").
		Exec(ctx)
	return err
}

func (r *repository) CountSince(
	ctx context.Context,
	userID pulid.ID,
	since int64,
) (int, error) {
	return r.db.DB().NewSelect().
		Model((*tenant.PasswordResetToken)(nil)).
		Where("prt.user_id = ?", userID).
		Where("prt.created_at >= ?", since).
		Count(ctx)
}
