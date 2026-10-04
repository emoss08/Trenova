package cloudsignuprepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/cloudsignup"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	entityName = "Signup"

	createScopeReason          = "record a cloud signup request, which belongs to no organization yet"
	tokenLookupScopeReason     = "lock a cloud signup by its verification token before any organization exists"
	emailLookupScopeReason     = "find the pending cloud signup for an email address before any organization exists"
	refreshScopeReason         = "replace the token on a pending cloud signup before any organization exists"
	touchScopeReason           = "reissue the verification token of a pending cloud signup"
	attemptsScopeReason        = "count verification attempts against a pending cloud signup"
	markProvisionedScopeReason = "record the organization a verified cloud signup provisioned"
	rejectScopeReason          = "reject a pending cloud signup"
	expireScopeReason          = "expire unverified cloud signups past their token lifetime"
	countProvisionedReason     = "count cloud signups provisioned since a time for the daily signup cap"
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

func New(p Params) repositories.CloudSignupRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.cloud-signup-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	entity *cloudsignup.CloudSignup,
) (*cloudsignup.CloudSignup, error) {
	ctx = dbscope.WithSystem(ctx, createScopeReason)

	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*cloudsignup.CloudSignup, error) {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
			if !dberror.IsUniqueConstraintViolation(err) {
				r.l.Error("failed to create cloud signup", zap.Error(err))
			}
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) GetPendingByTokenHash(
	ctx context.Context,
	tokenHash string,
) (*cloudsignup.CloudSignup, error) {
	ctx = dbscope.WithSystem(ctx, tokenLookupScopeReason)

	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*cloudsignup.CloudSignup, error) {
		cols := buncolgen.CloudSignupColumns
		entity := new(cloudsignup.CloudSignup)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Where(cols.TokenHash.Eq(), tokenHash).
			Where(cols.Status.Eq(), cloudsignup.StatusPending).
			For("UPDATE").
			Limit(1).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, entityName)
		}

		return entity, nil
	})
}

func (r *repository) GetPendingByEmail(
	ctx context.Context,
	emailNormalized string,
) (*cloudsignup.CloudSignup, error) {
	ctx = dbscope.WithSystem(ctx, emailLookupScopeReason)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*cloudsignup.CloudSignup, error) {
		cols := buncolgen.CloudSignupColumns
		entity := new(cloudsignup.CloudSignup)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Where(cols.EmailNormalized.Eq(), emailNormalized).
			Where(cols.Status.Eq(), cloudsignup.StatusPending).
			Limit(1).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, entityName)
		}

		return entity, nil
	})
}

func (r *repository) Refresh(
	ctx context.Context,
	req *repositories.RefreshCloudSignupRequest,
) (*cloudsignup.CloudSignup, error) {
	ctx = dbscope.WithSystem(ctx, refreshScopeReason)

	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*cloudsignup.CloudSignup, error) {
		cols := buncolgen.CloudSignupColumns
		entity := new(cloudsignup.CloudSignup)

		result, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			Where(cols.ID.Eq(), req.ID).
			Where(cols.Status.Eq(), cloudsignup.StatusPending).
			Set(cols.Name.Set(), req.Name).
			Set(cols.EmailAddress.Set(), req.EmailAddress).
			Set(cols.CompanyName.Set(), req.CompanyName).
			Set(cols.PasswordHash.Set(), req.PasswordHash).
			Set(cols.TokenHash.Set(), req.TokenHash).
			Set(cols.ExpiresAt.Set(), req.ExpiresAt).
			Set(cols.ClientIP.Set(), bun.NullZero(req.ClientIP)).
			Set(cols.UserAgent.Set(), bun.NullZero(req.UserAgent)).
			Set(cols.Attempts.Inc(1)).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Returning("*").
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to refresh cloud signup", zap.String("signupId", req.ID.String()), zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckFound(result, entityName); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) Touch(
	ctx context.Context,
	req *repositories.TouchCloudSignupRequest,
) (*cloudsignup.CloudSignup, error) {
	ctx = dbscope.WithSystem(ctx, touchScopeReason)

	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*cloudsignup.CloudSignup, error) {
		cols := buncolgen.CloudSignupColumns
		entity := new(cloudsignup.CloudSignup)

		q := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			Where(cols.ID.Eq(), req.ID).
			Where(cols.Status.Eq(), cloudsignup.StatusPending).
			Set(cols.Attempts.Inc(1)).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix())
		if req.TokenHash != "" {
			q = q.Set(cols.TokenHash.Set(), req.TokenHash)
		}
		if req.ExpiresAt > 0 {
			q = q.Set(cols.ExpiresAt.Set(), req.ExpiresAt)
		}
		if req.ClientIP != "" {
			q = q.Set(cols.ClientIP.Set(), req.ClientIP)
		}
		if req.UserAgent != "" {
			q = q.Set(cols.UserAgent.Set(), req.UserAgent)
		}

		result, err := q.Returning("*").Exec(ctx)
		if err != nil {
			r.l.Error("failed to touch cloud signup", zap.String("signupId", req.ID.String()), zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckFound(result, entityName); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) IncrementAttempts(ctx context.Context, id pulid.ID) (int, error) {
	ctx = dbscope.WithSystem(ctx, attemptsScopeReason)

	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		cols := buncolgen.CloudSignupColumns
		var attempts int

		err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*cloudsignup.CloudSignup)(nil)).
			Where(cols.ID.Eq(), id).
			Set(cols.Attempts.Inc(1)).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Returning(cols.Attempts.Bare()).
			Scan(ctx, &attempts)
		if err != nil {
			return 0, dberror.HandleNotFoundError(err, entityName)
		}

		return attempts, nil
	})
}

func (r *repository) MarkProvisioned(
	ctx context.Context,
	req *repositories.MarkCloudSignupProvisionedRequest,
) error {
	ctx = dbscope.WithSystem(ctx, markProvisionedScopeReason)

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := buncolgen.CloudSignupColumns

		result, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*cloudsignup.CloudSignup)(nil)).
			Where(cols.ID.Eq(), req.ID).
			Where(cols.Status.Eq(), cloudsignup.StatusPending).
			Set(cols.Status.Set(), cloudsignup.StatusProvisioned).
			Set(cols.ProvisionedOrganizationID.Set(), req.OrganizationID).
			Set(cols.ProvisionedBusinessUnitID.Set(), req.BusinessUnitID).
			Set(cols.ProvisionedUserID.Set(), req.UserID).
			Set(cols.VerifiedAt.Set(), req.VerifiedAt).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to mark cloud signup provisioned",
				zap.String("signupId", req.ID.String()),
				zap.Error(err),
			)
			return err
		}

		return dberror.CheckFound(result, entityName)
	})
}

func (r *repository) Reject(ctx context.Context, req *repositories.RejectCloudSignupRequest) error {
	ctx = dbscope.WithSystem(ctx, rejectScopeReason)

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := buncolgen.CloudSignupColumns

		result, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*cloudsignup.CloudSignup)(nil)).
			Where(cols.ID.Eq(), req.ID).
			Where(cols.Status.Eq(), cloudsignup.StatusPending).
			Set(cols.Status.Set(), cloudsignup.StatusRejected).
			Set(cols.RejectionReason.Set(), bun.NullZero(req.Reason)).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to reject cloud signup", zap.String("signupId", req.ID.String()), zap.Error(err))
			return err
		}

		return dberror.CheckFound(result, entityName)
	})
}

func (r *repository) Expire(ctx context.Context, now int64) (int64, error) {
	ctx = dbscope.WithSystem(ctx, expireScopeReason)

	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		cols := buncolgen.CloudSignupColumns

		result, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*cloudsignup.CloudSignup)(nil)).
			Where(cols.Status.Eq(), cloudsignup.StatusPending).
			Where(cols.ExpiresAt.Lte(), now).
			Set(cols.Status.Set(), cloudsignup.StatusExpired).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to expire cloud signups", zap.Error(err))
			return 0, fmt.Errorf("expire cloud signups: %w", err)
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("expire cloud signups: %w", err)
		}

		return affected, nil
	})
}

func (r *repository) CountProvisionedSince(ctx context.Context, since int64) (int, error) {
	ctx = dbscope.WithSystem(ctx, countProvisionedReason)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		cols := buncolgen.CloudSignupColumns

		count, err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*cloudsignup.CloudSignup)(nil)).
			Where(cols.Status.Eq(), cloudsignup.StatusProvisioned).
			Where(cols.VerifiedAt.Gte(), since).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count provisioned cloud signups", zap.Error(err))
			return 0, fmt.Errorf("count provisioned cloud signups: %w", err)
		}

		return count, nil
	})
}
