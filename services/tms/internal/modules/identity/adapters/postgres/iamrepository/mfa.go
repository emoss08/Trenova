package iamrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam/iamdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
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

func NewMFARepository(p MFARepositoryParams) iamports.MFARepository {
	return &mfaRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.mfa-repository"),
	}
}

func userTOTP(ctx context.Context, userID pulid.ID) *keel.Query[iam.MFAAuthenticator] {
	mfa := iamdb.MFAAuthenticators
	return forTenant(
		ctx,
		mfa.Query(),
		mfa.UserID.Eq(userID),
		mfa.Type.Eq(iam.MFAAuthenticatorTypeTOTP),
	)
}

func userRecoveryCodes(ctx context.Context, userID pulid.ID) *keel.Query[iam.MFARecoveryCode] {
	rc := iamdb.MFARecoveryCodes
	return forTenant(ctx, rc.Query(), rc.UserID.Eq(userID))
}

func (r *mfaRepository) GetTOTP(
	ctx context.Context,
	userID pulid.ID,
) (*iam.MFAAuthenticator, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*iam.MFAAuthenticator, error) {
		entity, err := userTOTP(ctx, userID).First(ctx, r.db.Keel(ctx))
		if dberror.IsNotFoundError(err) {
			return nil, nil
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
		db := r.db.Keel(ctx)
		mfa := iamdb.MFAAuthenticators

		exists, err := userTOTP(ctx, authenticator.UserID).
			Where(mfa.Enabled.IsTrue()).
			Exists(ctx, db)
		if err != nil {
			return fmt.Errorf("check enabled totp authenticator: %w", err)
		}
		if exists {
			return errortypes.NewBusinessError(
				"Two-factor authentication is already turned on. Turn it off before enrolling a new authenticator.",
			)
		}

		if _, err = userTOTP(ctx, authenticator.UserID).Delete(ctx, db); err != nil {
			return fmt.Errorf("remove pending totp authenticator: %w", err)
		}

		if err = mfa.Create(ctx, db, authenticator); err != nil {
			return fmt.Errorf("insert totp authenticator: %w", err)
		}

		return nil
	})
}

func (r *mfaRepository) ActivateTOTP(
	ctx context.Context,
	req *iamports.ActivateTOTPRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		mfa := iamdb.MFAAuthenticators
		affected, err := forTenant(
			ctx,
			mfa.Query(),
			mfa.ID.Eq(req.AuthenticatorID),
			mfa.UserID.Eq(req.UserID),
			mfa.Enabled.IsFalse(),
		).Update(
			ctx,
			r.db.Keel(ctx),
			mfa.Enabled.To(true),
			mfa.VerifiedAt.To(req.VerifiedAt),
			mfa.LastUsedAt.To(req.VerifiedAt),
			mfa.LastUsedStep.To(req.Step),
			mfa.UpdatedAt.To(req.VerifiedAt),
		)
		if err != nil {
			return fmt.Errorf("activate totp authenticator: %w", err)
		}
		if affected == 0 {
			return errortypes.NewBusinessError(
				"This enrollment is no longer pending. Start the enrollment again.",
			)
		}

		return r.replaceRecoveryCodes(ctx, req.UserID, req.RecoveryCodes)
	})
}

func (r *mfaRepository) RecordTOTPUse(
	ctx context.Context,
	req iamports.RecordTOTPUseRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		mfa := iamdb.MFAAuthenticators
		affected, err := forTenant(
			ctx,
			mfa.Query(),
			mfa.ID.Eq(req.AuthenticatorID),
			mfa.UserID.Eq(req.UserID),
			mfa.LastUsedStep.Lt(req.Step),
		).Update(
			ctx,
			r.db.Keel(ctx),
			mfa.LastUsedStep.To(req.Step),
			mfa.LastUsedAt.To(req.UsedAt),
			mfa.UpdatedAt.ToExpr(mfa.UpdatedAt),
		)
		if err != nil {
			return false, fmt.Errorf("record totp use: %w", err)
		}

		return affected == 1, nil
	})
}

func (r *mfaRepository) DeleteAll(ctx context.Context, userID pulid.ID) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		db := r.db.Keel(ctx)
		if _, err := userRecoveryCodes(ctx, userID).Delete(ctx, db); err != nil {
			return 0, fmt.Errorf("delete recovery codes: %w", err)
		}

		mfa := iamdb.MFAAuthenticators
		affected, err := forTenant(ctx, mfa.Query(), mfa.UserID.Eq(userID)).Delete(ctx, db)
		if err != nil {
			return 0, fmt.Errorf("delete authenticators: %w", err)
		}

		return int(affected), nil
	})
}

func (r *mfaRepository) ReplaceRecoveryCodes(
	ctx context.Context,
	req *iamports.ReplaceRecoveryCodesRequest,
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
	db := r.db.Keel(ctx)
	if _, err := userRecoveryCodes(ctx, userID).Delete(ctx, db); err != nil {
		return fmt.Errorf("delete recovery codes: %w", err)
	}

	if len(codes) == 0 {
		return nil
	}

	if err := iamdb.MFARecoveryCodes.CreateMany(ctx, db, codes); err != nil {
		return fmt.Errorf("insert recovery codes: %w", err)
	}

	return nil
}

func (r *mfaRepository) ConsumeRecoveryCode(
	ctx context.Context,
	req iamports.ConsumeRecoveryCodeRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		rc := iamdb.MFARecoveryCodes
		affected, err := userRecoveryCodes(ctx, req.UserID).
			Where(
				rc.UsedAt.IsNull(),
				keel.Or(
					keel.And(
						rc.HashVersion.Eq(iam.RecoveryCodeHashHMAC),
						rc.CodeHash.Eq(req.CodeHash),
					),
					keel.And(
						rc.HashVersion.Eq(iam.RecoveryCodeHashSHA256),
						rc.CodeHash.Eq(req.LegacyCodeHash),
					),
				),
			).
			Update(ctx, r.db.Keel(ctx), rc.UsedAt.To(req.UsedAt))
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
		rc := iamdb.MFARecoveryCodes
		count, err := userRecoveryCodes(ctx, userID).
			Where(rc.UsedAt.IsNull()).
			Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("count recovery codes: %w", err)
		}

		return int(count), nil
	})
}

func (r *mfaRepository) CountLegacyRecoveryCodes(
	ctx context.Context,
	userID pulid.ID,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		rc := iamdb.MFARecoveryCodes
		count, err := userRecoveryCodes(ctx, userID).
			Where(rc.UsedAt.IsNull(), rc.HashVersion.Eq(iam.RecoveryCodeHashSHA256)).
			Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("count legacy recovery codes: %w", err)
		}

		return int(count), nil
	})
}
