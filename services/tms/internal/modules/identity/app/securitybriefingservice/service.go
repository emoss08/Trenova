package securitybriefingservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	mfaRequiredCondition = "mfaRequired"
	directoryListLimit   = 100
)

var briefingResources = []permission.Resource{
	permission.ResourceUser,
	permission.ResourceMFAAuthenticator,
	permission.ResourceAuthEvent,
	permission.ResourceRiskDecision,
	permission.ResourceAPIKey,
	permission.ResourceIdentityProvider,
	permission.ResourceSCIMDirectory,
	permission.ResourceAccessPolicy,
}

type Params struct {
	fx.In

	Repo           iamports.SecurityBriefingRepository
	IAM            iamports.IAMRepository
	AccessSettings iamports.AccessSettingsService
	Permissions    permissionports.PermissionEngine
	Config         *config.Config
	Logger         *zap.Logger
}

type Service struct {
	repo           iamports.SecurityBriefingRepository
	iam            iamports.IAMRepository
	accessSettings iamports.AccessSettingsService
	permissions    permissionports.PermissionEngine
	session        config.SessionConfig
	l              *zap.Logger
}

func New(p Params) *Service {
	return &Service{
		repo:           p.Repo,
		iam:            p.IAM,
		accessSettings: p.AccessSettings,
		permissions:    p.Permissions,
		session:        p.Config.Security.Session,
		l:              p.Logger.Named("service.security-briefing"),
	}
}

type readAccess map[permission.Resource]bool

func (s *Service) Get(
	ctx context.Context,
	actor *permissionports.RequestActor,
) (*iam.SecurityBriefing, error) {
	if actor == nil {
		return nil, errortypes.NewAuthorizationError("A signed-in actor is required")
	}

	access, err := s.readAccess(ctx, actor)
	if err != nil {
		return nil, err
	}

	tenant := actor.TenantInfo()
	briefing := iam.NewSecurityBriefing(timeutils.NowUnix())
	briefing.Session = s.sessionPolicy()

	g, gctx := errgroup.WithContext(ctx)
	if access[permission.ResourceUser] {
		g.Go(func() error {
			var loadErr error
			briefing.Members, loadErr = s.repo.MemberSummary(gctx, iamports.MemberSummaryRequest{
				TenantInfo:    tenant,
				Now:           briefing.GeneratedAt,
				DormantBefore: briefing.DormantBefore(),
			})
			return loadErr
		})
	}
	if access[permission.ResourceMFAAuthenticator] {
		g.Go(func() error {
			var loadErr error
			briefing.MFA, loadErr = s.repo.MFACoverage(gctx, iamports.MFACoverageRequest{
				TenantInfo:  tenant,
				Now:         briefing.GeneratedAt,
				Limit:       iam.SecurityUnenrolledListLimit,
				IncludeList: access[permission.ResourceUser],
			})
			return loadErr
		})
	}
	if access[permission.ResourceAuthEvent] {
		g.Go(func() error {
			var loadErr error
			briefing.SignIns, loadErr = s.repo.SignInSummary(gctx, iamports.SignInSummaryRequest{
				TenantInfo:  tenant,
				Since:       briefing.SignInSince(),
				IncludeRisk: access[permission.ResourceRiskDecision],
			})
			return loadErr
		})
	}
	if access[permission.ResourceAPIKey] {
		g.Go(func() error {
			var loadErr error
			briefing.APIKeys, loadErr = s.repo.APIKeySummary(gctx, iamports.APIKeySummaryRequest{
				TenantInfo:     tenant,
				Now:            briefing.GeneratedAt,
				UnusedBefore:   briefing.APIKeyUnusedBefore(),
				ExpiringBefore: briefing.APIKeyExpiringBefore(),
			})
			return loadErr
		})
	}
	if access[permission.ResourceIdentityProvider] {
		g.Go(func() error {
			var loadErr error
			briefing.SSO, loadErr = s.ssoSummary(gctx, tenant)
			return loadErr
		})
	}
	if access[permission.ResourceSCIMDirectory] {
		g.Go(func() error {
			var loadErr error
			briefing.Provisioning, loadErr = s.provisioningSummary(gctx, tenant)
			return loadErr
		})
	}
	if access[permission.ResourceAccessPolicy] {
		g.Go(func() error {
			var loadErr error
			briefing.Policies, loadErr = s.policySummary(gctx, tenant)
			return loadErr
		})
	}

	if err = g.Wait(); err != nil {
		s.l.Error("failed to build security briefing", zap.Error(err))
		return nil, err
	}

	briefing.Findings = iam.BuildSecurityFindings(briefing)

	return briefing, nil
}

func (s *Service) readAccess(
	ctx context.Context,
	actor *permissionports.RequestActor,
) (readAccess, error) {
	checks := make([]permissionports.ResourceOperationCheck, 0, len(briefingResources))
	for _, resource := range briefingResources {
		checks = append(checks, permissionports.ResourceOperationCheck{
			Resource:  resource.String(),
			Operation: permission.OpRead,
		})
	}

	result, err := s.permissions.CheckBatch(ctx, &permissionports.BatchPermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Checks:         checks,
	})
	if err != nil {
		return nil, fmt.Errorf("check which security records the caller may read: %w", err)
	}

	access := make(readAccess, len(briefingResources))
	for i, resource := range briefingResources {
		access[resource] = i < len(result.Results) && result.Results[i].Allowed
	}

	return access, nil
}

func (s *Service) sessionPolicy() iam.SecuritySessionPolicy {
	policy := iam.SecuritySessionPolicy{MaxAgeSeconds: int64(s.session.MaxAge / time.Second)}
	if s.session.IdleTimeoutEnabled() {
		policy.IdleTimeoutSeconds = int64(s.session.IdleTimeout / time.Second)
		policy.IdleWarningSeconds = int64(s.session.IdleWarning / time.Second)
	}

	return policy
}

func (s *Service) ssoSummary(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*iam.SecuritySSOSummary, error) {
	providers, err := s.iam.ListIdentityProviders(ctx, iamports.ListIAMRequest{TenantInfo: tenant})
	if err != nil {
		return nil, fmt.Errorf("list identity providers: %w", err)
	}

	summary := new(iam.SecuritySSOSummary)
	for _, provider := range providers {
		if provider == nil || !provider.Enabled {
			continue
		}
		summary.EnabledProviders++
		if provider.EnforceSSO && !summary.Enforced {
			summary.Enforced = true
			summary.EnforcedProviderName = provider.Name
			summary.TrustsFederatedMFA = provider.AllowFederatedMFA
		}
	}

	return summary, nil
}

func (s *Service) provisioningSummary(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*iam.SecurityProvisioningSummary, error) {
	directories, err := s.iam.ListSCIMDirectories(ctx, &iamports.ListSCIMDirectoryRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: directoryListLimit},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("list scim directories: %w", err)
	}

	summary := new(iam.SecurityProvisioningSummary)
	if directories == nil {
		return summary, nil
	}
	for _, directory := range directories.Items {
		if directory != nil && directory.Enabled {
			summary.ActiveDirectories++
		}
	}

	return summary, nil
}

func (s *Service) policySummary(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*iam.SecurityPolicySummary, error) {
	policies, err := s.iam.ListAccessPolicies(ctx, iamports.ListIAMRequest{TenantInfo: tenant})
	if err != nil {
		return nil, fmt.Errorf("list access policies: %w", err)
	}

	settings, err := s.accessSettings.Get(ctx, tenant)
	if err != nil {
		return nil, fmt.Errorf("read access settings: %w", err)
	}

	summary := &iam.SecurityPolicySummary{
		RoleActivationRequired: settings != nil && settings.RequireRoleActivation,
	}
	for _, policy := range policies {
		if policy == nil || !policy.Enabled {
			continue
		}
		summary.Enabled++
		if strings.EqualFold(strings.TrimSpace(policy.Conditions[mfaRequiredCondition]), "true") {
			summary.RequiringMFA++
		}
	}

	return summary, nil
}
