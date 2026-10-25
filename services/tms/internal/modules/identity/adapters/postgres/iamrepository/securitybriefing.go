package iamrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/apikey"
	"github.com/emoss08/trenova/internal/modules/identity/domain/apikey/apikeydb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam/iamdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
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

func memberQuery(ctx context.Context, scope memberScope) *keel.Query[*tenant.User] {
	u := tenantdb.Users
	m := tenantdb.OrganizationMemberships

	return forTenant(
		ctx,
		u.Join(m, m.UserID.EqCol(u.ID)),
		m.OrganizationID.Eq(scope.TenantInfo.OrgID),
		m.BusinessUnitID.Eq(scope.TenantInfo.BuID),
		keel.Or(m.ExpiresAt.IsNull(), m.ExpiresAt.Gt(scope.Now)),
		u.Status.Eq(domaintypes.StatusActive),
		u.Username.Ne(tenant.SystemUsername),
	)
}

func activeTOTP(ctx context.Context) keel.Pred {
	mfa := iamdb.MFAAuthenticators
	u := tenantdb.Users

	return keel.Exists(forTenant(
		ctx,
		mfa.Query(),
		mfa.UserID.EqCol(u.ID),
		mfa.Type.Eq(iam.MFAAuthenticatorTypeTOTP),
		mfa.Enabled.IsTrue(),
		keel.DynOf(mfa.VerifiedAt).Coalesce(0).Gt(0),
	))
}

func (r *securityBriefingRepository) MemberSummary(
	ctx context.Context,
	req iamports.MemberSummaryRequest,
) (*iam.SecurityMemberSummary, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*iam.SecurityMemberSummary, error) {
		u := tenantdb.Users
		m := tenantdb.OrganizationMemberships
		summary, err := keel.IntoOne[iam.SecurityMemberSummary](
			ctx,
			r.db.Keel(ctx),
			memberQuery(ctx, memberScope{TenantInfo: req.TenantInfo, Now: req.Now}).Pick(
				keel.Count(),
				keel.Count().Filter(u.IsLocked.IsTrue()),
				keel.Count().Filter(keel.DynOf(u.LastLoginAt).Coalesce(0).Eq(0)),
				keel.Count().Filter(
					keel.DynOf(u.LastLoginAt).NullIf(0).Coalesce(m.JoinedAt).Lt(req.DormantBefore),
				),
				keel.Count().Filter(u.MustChangePassword.IsTrue()),
			),
		)
		if err != nil {
			r.l.Error("failed to summarize organization members", zap.Error(err))
			return nil, fmt.Errorf("summarize organization members: %w", err)
		}

		return &summary, nil
	})
}

type mfaCoverageCounts struct {
	Total    int64
	Enrolled int64
}

func (r *securityBriefingRepository) MFACoverage(
	ctx context.Context,
	req iamports.MFACoverageRequest,
) (*iam.SecurityMFACoverage, error) {
	ctx = dbscope.WithSystem(ctx, mfaCoverageScopeReason)

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*iam.SecurityMFACoverage, error) {
		db := r.db.Keel(ctx)
		scope := memberScope{TenantInfo: req.TenantInfo, Now: req.Now}

		counts, err := keel.IntoOne[mfaCoverageCounts](
			ctx,
			db,
			memberQuery(ctx, scope).Pick(keel.Count(), keel.Count().Filter(activeTOTP(ctx))),
		)
		if err != nil {
			r.l.Error("failed to count two-factor enrolment", zap.Error(err))
			return nil, fmt.Errorf("count two-factor enrolment: %w", err)
		}

		coverage := &iam.SecurityMFACoverage{Enrolled: counts.Enrolled, Total: counts.Total}
		if !req.IncludeList {
			return coverage, nil
		}

		coverage.Unenrolled = make([]*iam.SecurityMemberRef, 0, max(req.Limit, 0))
		if req.Limit <= 0 || counts.Total == counts.Enrolled {
			return coverage, nil
		}

		u := tenantdb.Users
		lastSignIn := keel.DynOf(u.LastLoginAt).NullIf(0)
		members, err := keel.Into[iam.SecurityMemberRef](
			ctx,
			db,
			memberQuery(ctx, scope).
				Pick(u.ID, u.Name, u.EmailAddress, lastSignIn).
				Where(keel.Not(activeTOTP(ctx))).
				Order(keel.Desc(lastSignIn).NullsLast(), u.Name.Asc()).
				Limit(req.Limit),
		)
		if err != nil {
			r.l.Error("failed to list members without two-factor", zap.Error(err))
			return nil, fmt.Errorf("list members without two-factor: %w", err)
		}
		for i := range members {
			coverage.Unenrolled = append(coverage.Unenrolled, &members[i])
		}

		return coverage, nil
	})
}

type signInCounts struct {
	Succeeded int64
	Failed    int64
	Denied    int64
	Throttled int64
}

func (r *securityBriefingRepository) SignInSummary(
	ctx context.Context,
	req iamports.SignInSummaryRequest,
) (*iam.SecuritySignInSummary, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*iam.SecuritySignInSummary, error) {
		db := r.db.Keel(ctx)
		ae := iamdb.AuthEvents
		throttled := iamports.AuthEventProviderLoginThrottled
		outcome := func(o iam.AuthEventOutcome) keel.Agg {
			return keel.Count().Filter(keel.And(ae.Outcome.Eq(o), ae.Provider.Ne(throttled)))
		}

		counts, err := keel.IntoOne[signInCounts](ctx, db, ae.Where(
			ae.OrganizationID.Eq(req.TenantInfo.OrgID),
			ae.BusinessUnitID.Eq(req.TenantInfo.BuID),
			ae.OccurredAt.Gte(req.Since),
			ae.Provider.NotIn(iamports.NonSignInAuthEventProviders()...),
		).Pick(
			outcome(iam.AuthEventOutcomeSuccess),
			outcome(iam.AuthEventOutcomeFailed),
			outcome(iam.AuthEventOutcomeDenied),
			keel.Count().Filter(ae.Provider.Eq(throttled)),
		))
		if err != nil {
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

		rd := iamdb.RiskDecisions
		denied, err := rd.Where(
			rd.OrganizationID.Eq(req.TenantInfo.OrgID),
			rd.BusinessUnitID.Eq(req.TenantInfo.BuID),
			rd.Outcome.Eq(iam.RiskOutcomeDeny),
			rd.CreatedAt.Gte(req.Since),
		).Count(ctx, db)
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
		k := apikeydb.Keys
		summary, err := keel.IntoOne[iam.SecurityAPIKeySummary](ctx, r.db.Keel(ctx), k.Where(
			k.Status.Eq(apikey.StatusActive),
			keel.DynOf(k.ExpiresAt).Coalesce(0).Eq(0).Or(keel.DynOf(k.ExpiresAt).Gt(req.Now)),
		).Pick(
			keel.Count(),
			keel.Count().Filter(keel.DynOf(k.ExpiresAt).Coalesce(0).Eq(0)),
			keel.Count().Filter(
				keel.DynOf(k.LastUsedAt).NullIf(0).Coalesce(k.CreatedAt).Lt(req.UnusedBefore),
			),
			keel.Count().Filter(
				keel.DynOf(k.ExpiresAt).
					Gt(req.Now).
					And(keel.DynOf(k.ExpiresAt).Lte(req.ExpiringBefore)),
			),
		))
		if err != nil {
			r.l.Error("failed to summarize api keys", zap.Error(err))
			return nil, fmt.Errorf("summarize api keys: %w", err)
		}

		return &summary, nil
	})
}
