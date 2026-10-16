package iamrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/apikey"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/apikeycol"
	"github.com/emoss08/trenova/pkg/buncolgen/iamcol"
	"github.com/emoss08/trenova/pkg/buncolgen/tenantcol"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const mfaCoverageScopeReason = "count an organization's members' two-factor enrolment, which is recorded under the organization each person enrolled from"

type securityBriefingRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSecurityBriefingRepository(p Params) iamports.SecurityBriefingRepository {
	return &securityBriefingRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.security-briefing-repository"),
	}
}

type memberScope struct {
	TenantInfo pagination.TenantInfo
	Now        int64
}

func memberQuery(db bun.IDB, scope memberScope) *bun.SelectQuery {
	user := tenantcol.UserColumns
	membership := tenantcol.OrganizationMembershipColumns
	membershipTable := tenantcol.OrganizationMembershipTable

	return db.NewSelect().
		Model((*tenant.User)(nil)).
		Join("JOIN "+membershipTable.As(membershipTable.Alias)+" ON "+
			membership.UserID.EqColumn(user.ID)).
		Where(membership.OrganizationID.Eq(), scope.TenantInfo.OrgID).
		Where(membership.BusinessUnitID.Eq(), scope.TenantInfo.BuID).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where(membership.ExpiresAt.IsNull()).
				WhereOr(membership.ExpiresAt.Gt(), scope.Now)
		}).
		Where(user.Status.Eq(), domaintypes.StatusActive).
		Where(user.Username.NotEq(), tenant.SystemUsername)
}

func activeTOTPQuery(db bun.IDB) *bun.SelectQuery {
	mfa := iamcol.MFAAuthenticatorColumns

	return db.NewSelect().
		Model((*iam.MFAAuthenticator)(nil)).
		ColumnExpr("1").
		Where(mfa.UserID.EqColumn(tenantcol.UserColumns.ID)).
		Where(mfa.Type.Eq(), iam.MFAAuthenticatorTypeTOTP).
		Where(mfa.Enabled.IsTrue()).
		Where(mfa.VerifiedAt.Expr("COALESCE({}, 0) > 0"))
}

func (r *securityBriefingRepository) MemberSummary(
	ctx context.Context,
	req iamports.MemberSummaryRequest,
) (*iam.SecurityMemberSummary, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*iam.SecurityMemberSummary, error) {
		user := tenantcol.UserColumns
		membership := tenantcol.OrganizationMembershipColumns
		summary := new(iam.SecurityMemberSummary)

		err := memberQuery(r.db.DBForContext(ctx), memberScope{
			TenantInfo: req.TenantInfo,
			Now:        req.Now,
		}).
			ColumnExpr(buncolgen.Count("active")).
			ColumnExpr(buncolgen.CountFilter("locked", user.IsLocked.IsTrue())).
			ColumnExpr(buncolgen.CountFilter(
				"never_signed_in",
				user.LastLoginAt.Expr("COALESCE({}, 0) = 0"),
			)).
			ColumnExpr(
				buncolgen.CountFilter(
					"dormant",
					buncolgen.Expr(
						"COALESCE(NULLIF({0}, 0), {1}) < ?",
						user.LastLoginAt,
						membership.JoinedAt,
					),
				),
				req.DormantBefore,
			).
			ColumnExpr(buncolgen.CountFilter(
				"must_change_password",
				user.MustChangePassword.IsTrue(),
			)).
			Scan(ctx, summary)
		if err != nil {
			r.l.Error("failed to summarize organization members", zap.Error(err))
			return nil, fmt.Errorf("summarize organization members: %w", err)
		}

		return summary, nil
	})
}

type mfaCoverageCounts struct {
	Total    int64 `bun:"total"`
	Enrolled int64 `bun:"enrolled"`
}

func (r *securityBriefingRepository) MFACoverage(
	ctx context.Context,
	req iamports.MFACoverageRequest,
) (*iam.SecurityMFACoverage, error) {
	ctx = dbscope.WithSystem(ctx, mfaCoverageScopeReason)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*iam.SecurityMFACoverage, error) {
		db := r.db.DBForContext(ctx)
		scope := memberScope{TenantInfo: req.TenantInfo, Now: req.Now}
		counts := new(mfaCoverageCounts)

		if err := memberQuery(db, scope).
			ColumnExpr(buncolgen.Count("total")).
			ColumnExpr(
				buncolgen.CountFilter("enrolled", "EXISTS (?)"),
				activeTOTPQuery(db),
			).
			Scan(ctx, counts); err != nil {
			r.l.Error("failed to count two-factor enrolment", zap.Error(err))
			return nil, fmt.Errorf("count two-factor enrolment: %w", err)
		}

		coverage := &iam.SecurityMFACoverage{Enrolled: counts.Enrolled, Total: counts.Total}
		if !req.IncludeList {
			return coverage, nil
		}

		user := tenantcol.UserColumns
		coverage.Unenrolled = make([]*iam.SecurityMemberRef, 0, max(req.Limit, 0))
		if req.Limit <= 0 || counts.Total == counts.Enrolled {
			return coverage, nil
		}

		if err := memberQuery(db, scope).
			ColumnExpr(user.ID.As("user_id")).
			ColumnExpr(user.Name.As("name")).
			ColumnExpr(user.EmailAddress.As("email_address")).
			ColumnExpr(user.LastLoginAt.Expr("NULLIF({}, 0) AS last_login_at")).
			Where("NOT EXISTS (?)", activeTOTPQuery(db)).
			OrderExpr(user.LastLoginAt.Expr("NULLIF({}, 0) DESC NULLS LAST")).
			Order(user.Name.OrderAsc()).
			Limit(int64(req.Limit)).
			Scan(ctx, &coverage.Unenrolled); err != nil {
			r.l.Error("failed to list members without two-factor", zap.Error(err))
			return nil, fmt.Errorf("list members without two-factor: %w", err)
		}

		return coverage, nil
	})
}

type signInCounts struct {
	Succeeded int64 `bun:"succeeded"`
	Failed    int64 `bun:"failed"`
	Denied    int64 `bun:"denied"`
	Throttled int64 `bun:"throttled"`
}

func (r *securityBriefingRepository) SignInSummary(
	ctx context.Context,
	req iamports.SignInSummaryRequest,
) (*iam.SecuritySignInSummary, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*iam.SecuritySignInSummary, error) {
		db := r.db.DBForContext(ctx)
		events := iamcol.AuthEventColumns
		counts := new(signInCounts)
		throttled := iamports.AuthEventProviderLoginThrottled

		if err := db.NewSelect().
			Model((*iam.AuthEvent)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				return iamcol.AuthEventScopeTenant(sq, req.TenantInfo)
			}).
			Where(events.OccurredAt.Gte(), req.Since).
			Where(events.Provider.NotIn(), bun.List(iamports.NonSignInAuthEventProviders())).
			ColumnExpr(
				buncolgen.CountFilter("succeeded", events.Outcome.Eq(), events.Provider.NotEq()),
				iam.AuthEventOutcomeSuccess,
				throttled,
			).
			ColumnExpr(
				buncolgen.CountFilter("failed", events.Outcome.Eq(), events.Provider.NotEq()),
				iam.AuthEventOutcomeFailed,
				throttled,
			).
			ColumnExpr(
				buncolgen.CountFilter("denied", events.Outcome.Eq(), events.Provider.NotEq()),
				iam.AuthEventOutcomeDenied,
				throttled,
			).
			ColumnExpr(buncolgen.CountFilter("throttled", events.Provider.Eq()), throttled).
			Scan(ctx, counts); err != nil {
			r.l.Error("failed to summarize sign-ins", zap.Error(err))
			return nil, fmt.Errorf("summarize sign-ins: %w", err)
		}

		summary := &iam.SecuritySignInSummary{
			Succeeded: counts.Succeeded,
			Failed:    counts.Failed,
			Denied:    counts.Denied,
			Throttled: counts.Throttled,
		}
		if !req.IncludeRisk {
			return summary, nil
		}

		decisions := iamcol.RiskDecisionColumns
		denied, err := db.NewSelect().
			Model((*iam.RiskDecision)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				return iamcol.RiskDecisionScopeTenant(sq, req.TenantInfo)
			}).
			Where(decisions.Outcome.Eq(), iam.RiskOutcomeDeny).
			Where(decisions.CreatedAt.Gte(), req.Since).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count risk denials", zap.Error(err))
			return nil, fmt.Errorf("count risk denials: %w", err)
		}

		summary.RiskDenied = &denied

		return summary, nil
	})
}

func (r *securityBriefingRepository) APIKeySummary(
	ctx context.Context,
	req iamports.APIKeySummaryRequest,
) (*iam.SecurityAPIKeySummary, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*iam.SecurityAPIKeySummary, error) {
		keys := apikeycol.KeyColumns
		summary := new(iam.SecurityAPIKeySummary)

		err := r.db.DBForContext(ctx).NewSelect().
			Model((*apikey.Key)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				return apikeycol.KeyScopeTenant(sq, req.TenantInfo)
			}).
			Where(keys.Status.Eq(), apikey.StatusActive).
			Where(buncolgen.Expr("(COALESCE({0}, 0) = 0 OR {0} > ?)", keys.ExpiresAt), req.Now).
			ColumnExpr(buncolgen.Count("active")).
			ColumnExpr(buncolgen.CountFilter(
				"without_expiry",
				keys.ExpiresAt.Expr("COALESCE({}, 0) = 0"),
			)).
			ColumnExpr(
				buncolgen.CountFilter(
					"unused",
					buncolgen.Expr(
						"COALESCE(NULLIF({0}, 0), {1}) < ?",
						keys.LastUsedAt,
						keys.CreatedAt,
					),
				),
				req.UnusedBefore,
			).
			ColumnExpr(
				buncolgen.CountFilter("expiring_soon", keys.ExpiresAt.Gt(), keys.ExpiresAt.Lte()),
				req.Now,
				req.ExpiringBefore,
			).
			Scan(ctx, summary)
		if err != nil {
			r.l.Error("failed to summarize api keys", zap.Error(err))
			return nil, fmt.Errorf("summarize api keys: %w", err)
		}

		return summary, nil
	})
}
