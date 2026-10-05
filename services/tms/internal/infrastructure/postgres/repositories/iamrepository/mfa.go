package iamrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	mfaCols      = buncolgen.MFAAuthenticatorColumns
	recoveryCols = buncolgen.MFARecoveryCodeColumns
)

type MFARepositoryParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type mfaRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewMFARepository(p MFARepositoryParams) repositories.MFARepository {
	return &mfaRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.mfa-repository"),
	}
}

func (r *mfaRepository) GetTOTP(
	ctx context.Context,
	userID pulid.ID,
) (*iam.MFAAuthenticator, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*iam.MFAAuthenticator, error) {
		entity := new(iam.MFAAuthenticator)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(entity).
			Where(mfaCols.UserID.Eq(), userID).
			Where(mfaCols.Type.Eq(), iam.MFAAuthenticatorTypeTOTP).
			Limit(1).
			Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // no authenticator is a valid state
		}
		if err != nil {
			return nil, fmt.Errorf("get totp authenticator: %w", err)
		}

		return entity, nil
	})
}

func (r *mfaRepository) ReplacePendingTOTP(
	ctx context.Context,
	authenticator *iam.MFAAuthenticator,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.DBForContext(ctx)

		exists, err := db.NewSelect().
			Model((*iam.MFAAuthenticator)(nil)).
			Where(mfaCols.UserID.Eq(), authenticator.UserID).
			Where(mfaCols.Type.Eq(), iam.MFAAuthenticatorTypeTOTP).
			Where(mfaCols.Enabled.IsTrue()).
			Exists(ctx)
		if err != nil {
			return fmt.Errorf("check enabled totp authenticator: %w", err)
		}
		if exists {
			return errortypes.NewBusinessError(
				"Two-factor authentication is already turned on. Turn it off before enrolling a new authenticator.",
			)
		}

		if _, err = db.NewDelete().
			Model((*iam.MFAAuthenticator)(nil)).
			Where(mfaCols.UserID.Eq(), authenticator.UserID).
			Where(mfaCols.Type.Eq(), iam.MFAAuthenticatorTypeTOTP).
			Exec(ctx); err != nil {
			return fmt.Errorf("remove pending totp authenticator: %w", err)
		}

		if _, err = db.NewInsert().Model(authenticator).Exec(ctx); err != nil {
			return fmt.Errorf("insert totp authenticator: %w", err)
		}

		return nil
	})
}

func (r *mfaRepository) ActivateTOTP(
	ctx context.Context,
	req *repositories.ActivateTOTPRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.DBForContext(ctx)

		result, err := db.NewUpdate().
			Model((*iam.MFAAuthenticator)(nil)).
			Set(mfaCols.Enabled.Set(), true).
			Set(mfaCols.VerifiedAt.Set(), req.VerifiedAt).
			Set(mfaCols.LastUsedAt.Set(), req.VerifiedAt).
			Set(mfaCols.LastUsedStep.Set(), req.Step).
			Set(mfaCols.UpdatedAt.Set(), req.VerifiedAt).
			Where(mfaCols.ID.Eq(), req.AuthenticatorID).
			Where(mfaCols.UserID.Eq(), req.UserID).
			Where(mfaCols.Enabled.IsFalse()).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("activate totp authenticator: %w", err)
		}
		if affected, _ := result.RowsAffected(); affected == 0 {
			return errortypes.NewBusinessError(
				"This enrollment is no longer pending. Start the enrollment again.",
			)
		}

		return r.replaceRecoveryCodes(ctx, req.UserID, req.RecoveryCodes)
	})
}

func (r *mfaRepository) RecordTOTPUse(
	ctx context.Context,
	req repositories.RecordTOTPUseRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		result, err := r.db.DBForContext(ctx).NewUpdate().
			Model((*iam.MFAAuthenticator)(nil)).
			Set(mfaCols.LastUsedStep.Set(), req.Step).
			Set(mfaCols.LastUsedAt.Set(), req.UsedAt).
			Where(mfaCols.ID.Eq(), req.AuthenticatorID).
			Where(mfaCols.UserID.Eq(), req.UserID).
			Where(mfaCols.LastUsedStep.Lt(), req.Step).
			Exec(ctx)
		if err != nil {
			return false, fmt.Errorf("record totp use: %w", err)
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return false, fmt.Errorf("record totp use: %w", err)
		}

		return affected == 1, nil
	})
}

func (r *mfaRepository) DeleteAll(ctx context.Context, userID pulid.ID) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		db := r.db.DBForContext(ctx)

		if _, err := db.NewDelete().
			Model((*iam.MFARecoveryCode)(nil)).
			Where(recoveryCols.UserID.Eq(), userID).
			Exec(ctx); err != nil {
			return 0, fmt.Errorf("delete recovery codes: %w", err)
		}

		result, err := db.NewDelete().
			Model((*iam.MFAAuthenticator)(nil)).
			Where(mfaCols.UserID.Eq(), userID).
			Exec(ctx)
		if err != nil {
			return 0, fmt.Errorf("delete authenticators: %w", err)
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("delete authenticators: %w", err)
		}

		return int(affected), nil
	})
}

func (r *mfaRepository) ReplaceRecoveryCodes(
	ctx context.Context,
	req *repositories.ReplaceRecoveryCodesRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		return r.replaceRecoveryCodes(ctx, req.UserID, req.Codes)
	})
}

func (r *mfaRepository) replaceRecoveryCodes(
	ctx context.Context,
	userID pulid.ID,
	codes []*iam.MFARecoveryCode,
) error {
	db := r.db.DBForContext(ctx)

	if _, err := db.NewDelete().
		Model((*iam.MFARecoveryCode)(nil)).
		Where(recoveryCols.UserID.Eq(), userID).
		Exec(ctx); err != nil {
		return fmt.Errorf("delete recovery codes: %w", err)
	}

	if len(codes) == 0 {
		return nil
	}

	if _, err := db.NewInsert().Model(&codes).Exec(ctx); err != nil {
		return fmt.Errorf("insert recovery codes: %w", err)
	}

	return nil
}

func (r *mfaRepository) ConsumeRecoveryCode(
	ctx context.Context,
	req repositories.ConsumeRecoveryCodeRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		result, err := r.db.DBForContext(ctx).NewUpdate().
			Model((*iam.MFARecoveryCode)(nil)).
			Set(recoveryCols.UsedAt.Set(), req.UsedAt).
			Where(recoveryCols.UserID.Eq(), req.UserID).
			Where(recoveryCols.CodeHash.Eq(), req.CodeHash).
			Where(recoveryCols.UsedAt.IsNull()).
			Exec(ctx)
		if err != nil {
			return false, fmt.Errorf("consume recovery code: %w", err)
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return false, fmt.Errorf("consume recovery code: %w", err)
		}

		return affected == 1, nil
	})
}

func (r *mfaRepository) CountUnusedRecoveryCodes(
	ctx context.Context,
	userID pulid.ID,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		count, err := r.db.DBForContext(ctx).NewSelect().
			Model((*iam.MFARecoveryCode)(nil)).
			Where(recoveryCols.UserID.Eq(), userID).
			Where(recoveryCols.UsedAt.IsNull()).
			Count(ctx)
		if err != nil {
			return 0, fmt.Errorf("count recovery codes: %w", err)
		}

		return count, nil
	})
}
