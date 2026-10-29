package securitybriefingservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/iamportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeAccessSettings struct {
	settings *iam.AccessSettings
}

func (f *fakeAccessSettings) Get(
	_ context.Context,
	_ pagination.TenantInfo,
) (*iam.AccessSettings, error) {
	return f.settings, nil
}

func (f *fakeAccessSettings) Update(
	_ context.Context,
	_ *iamports.UpdateAccessSettingsRequest,
) (*iam.AccessSettings, error) {
	return f.settings, nil
}

type harness struct {
	repo    *iamportsmock.MockSecurityBriefingRepository
	iamRepo *iamportsmock.MockIAMRepository
	service *Service
	actor   *permissionports.RequestActor
}

func newHarness(
	t *testing.T,
	session securityconfig.SessionConfig,
	allowed ...permission.Resource,
) *harness {
	t.Helper()

	allowedSet := make(map[string]bool, len(allowed))
	for _, resource := range allowed {
		allowedSet[resource.String()] = true
	}

	engine := permissionportsmock.NewMockPermissionEngine(t)
	engine.EXPECT().
		CheckBatch(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *permissionports.BatchPermissionCheckRequest,
		) (*permissionports.BatchPermissionCheckResult, error) {
			results := make([]permissionports.PermissionCheckResult, 0, len(req.Checks))
			for _, check := range req.Checks {
				results = append(results, permissionports.PermissionCheckResult{
					Allowed: check.Operation == permission.OpRead && allowedSet[check.Resource],
				})
			}
			return &permissionports.BatchPermissionCheckResult{Results: results}, nil
		})

	repo := iamportsmock.NewMockSecurityBriefingRepository(t)
	iamRepo := iamportsmock.NewMockIAMRepository(t)
	service := New(Params{
		Repo: repo,
		IAM:  iamRepo,
		AccessSettings: &fakeAccessSettings{
			settings: &iam.AccessSettings{RequireRoleActivation: true},
		},
		Permissions:    engine,
		SecurityConfig: &securityconfig.Config{Session: session},
		Logger:         zap.NewNop(),
	})

	return &harness{
		repo:    repo,
		iamRepo: iamRepo,
		service: service,
		actor: &permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeUser,
			PrincipalID:    pulid.MustNew("usr_"),
			UserID:         pulid.MustNew("usr_"),
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
	}
}

func idleSession() securityconfig.SessionConfig {
	return securityconfig.SessionConfig{
		MaxAge:      12 * time.Hour,
		IdleTimeout: 30 * time.Minute,
		IdleWarning: 2 * time.Minute,
	}
}

func TestGet_LeavesEverySectionNullWithoutReadPermission(t *testing.T) {
	t.Parallel()

	h := newHarness(t, idleSession())

	briefing, err := h.service.Get(t.Context(), h.actor)
	require.NoError(t, err)

	assert.Nil(t, briefing.Members)
	assert.Nil(t, briefing.MFA)
	assert.Nil(t, briefing.SignIns)
	assert.Nil(t, briefing.APIKeys)
	assert.Nil(t, briefing.SSO)
	assert.Nil(t, briefing.Provisioning)
	assert.Nil(t, briefing.Policies)
	assert.Equal(t, iam.SecuritySessionPolicy{
		MaxAgeSeconds:      43_200,
		IdleTimeoutSeconds: 1_800,
		IdleWarningSeconds: 120,
	}, briefing.Session)
	assert.Equal(t, []iam.SecurityFinding{
		{Code: iam.SecurityFindingIdleTimeout, Severity: iam.SecurityFindingPass},
	}, briefing.Findings)
}

func TestGet_ReportsNoIdleTimeoutWhenTheDeploymentHasNone(t *testing.T) {
	t.Parallel()

	h := newHarness(
		t,
		securityconfig.SessionConfig{MaxAge: 12 * time.Hour, IdleWarning: 2 * time.Minute},
	)

	briefing, err := h.service.Get(t.Context(), h.actor)
	require.NoError(t, err)

	assert.Equal(t, iam.SecuritySessionPolicy{MaxAgeSeconds: 43_200}, briefing.Session)
	assert.Equal(t, []iam.SecurityFinding{
		{Code: iam.SecurityFindingIdleTimeout, Severity: iam.SecurityFindingWarning},
	}, briefing.Findings)
}

func TestGet_CountsMFAWithoutListingPeopleWhenUsersAreNotReadable(t *testing.T) {
	t.Parallel()

	h := newHarness(t, idleSession(), permission.ResourceMFAAuthenticator)
	h.repo.EXPECT().
		MFACoverage(mock.Anything, mock.MatchedBy(func(req iamports.MFACoverageRequest) bool {
			return !req.IncludeList &&
				req.Limit == iam.SecurityUnenrolledListLimit &&
				req.TenantInfo.OrgID == h.actor.OrganizationID &&
				req.TenantInfo.BuID == h.actor.BusinessUnitID
		})).
		Return(&iam.SecurityMFACoverage{Enrolled: 3, Total: 4}, nil)

	briefing, err := h.service.Get(t.Context(), h.actor)
	require.NoError(t, err)

	require.NotNil(t, briefing.MFA)
	assert.Nil(t, briefing.MFA.Unenrolled)
	assert.Nil(t, briefing.Members)
	assert.Equal(t, iam.SecurityFinding{
		Code:     iam.SecurityFindingMFACoverage,
		Severity: iam.SecurityFindingWarning,
		Count:    1,
		Total:    4,
	}, briefing.Findings[0])
}

func TestGet_ListsUnenrolledPeopleWhenUsersAreReadable(t *testing.T) {
	t.Parallel()

	h := newHarness(t, idleSession(), permission.ResourceMFAAuthenticator, permission.ResourceUser)
	h.repo.EXPECT().
		MemberSummary(mock.Anything, mock.MatchedBy(func(req iamports.MemberSummaryRequest) bool {
			return req.DormantBefore == req.Now-iam.SecurityDormantAfterDays*24*60*60
		})).
		Return(&iam.SecurityMemberSummary{Active: 4}, nil)
	h.repo.EXPECT().
		MFACoverage(mock.Anything, mock.MatchedBy(func(req iamports.MFACoverageRequest) bool {
			return req.IncludeList
		})).
		Return(&iam.SecurityMFACoverage{
			Enrolled:   3,
			Total:      4,
			Unenrolled: []*iam.SecurityMemberRef{{Name: "Pat Doe"}},
		}, nil)

	briefing, err := h.service.Get(t.Context(), h.actor)
	require.NoError(t, err)

	require.NotNil(t, briefing.Members)
	require.Len(t, briefing.MFA.Unenrolled, 1)
	assert.Equal(t, "Pat Doe", briefing.MFA.Unenrolled[0].Name)
}

func TestGet_AsksForRiskDenialsOnlyWhenRiskDecisionsAreReadable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		allowed []permission.Resource
		risk    bool
	}{
		{name: "sign-ins only", allowed: []permission.Resource{permission.ResourceAuthEvent}},
		{
			name: "sign-ins and risk",
			allowed: []permission.Resource{
				permission.ResourceAuthEvent,
				permission.ResourceRiskDecision,
			},
			risk: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t, idleSession(), tc.allowed...)
			h.repo.EXPECT().
				SignInSummary(mock.Anything, mock.MatchedBy(func(req iamports.SignInSummaryRequest) bool {
					return req.IncludeRisk == tc.risk
				})).
				Return(&iam.SecuritySignInSummary{Succeeded: 12}, nil)

			briefing, err := h.service.Get(t.Context(), h.actor)
			require.NoError(t, err)
			require.NotNil(t, briefing.SignIns)
		})
	}
}

func TestGet_SummarizesSingleSignOnDirectoriesAndPolicies(t *testing.T) {
	t.Parallel()

	h := newHarness(
		t,
		idleSession(),
		permission.ResourceIdentityProvider,
		permission.ResourceSCIMDirectory,
		permission.ResourceAccessPolicy,
	)
	h.iamRepo.EXPECT().
		ListIdentityProviders(mock.Anything, mock.Anything).
		Return([]*iam.IdentityProvider{
			{Name: "Okta", Enabled: false, EnforceSSO: true},
			{Name: "Entra ID", Enabled: true, EnforceSSO: true, AllowFederatedMFA: true},
			{Name: "Google", Enabled: true},
		}, nil)
	sameTenant := func(req *iamports.ListSCIMDirectoryRequest) bool {
		return req.Filter.TenantInfo.OrgID == h.actor.OrganizationID
	}
	h.iamRepo.EXPECT().
		ListSCIMDirectories(mock.Anything, mock.MatchedBy(sameTenant)).
		Return(&pagination.ListResult[*iam.SCIMDirectory]{Items: []*iam.SCIMDirectory{
			{Enabled: true},
			{Enabled: false},
		}}, nil)
	h.iamRepo.EXPECT().
		ListAccessPolicies(mock.Anything, mock.Anything).
		Return([]*iam.AccessPolicy{
			{Enabled: true, Conditions: map[string]string{"mfaRequired": "TRUE"}},
			{Enabled: true, Conditions: map[string]string{"mfaRequired": "false"}},
			{Enabled: false, Conditions: map[string]string{"mfaRequired": "true"}},
		}, nil)

	briefing, err := h.service.Get(t.Context(), h.actor)
	require.NoError(t, err)

	assert.Equal(t, &iam.SecuritySSOSummary{
		EnabledProviders:     2,
		Enforced:             true,
		EnforcedProviderName: "Entra ID",
		TrustsFederatedMFA:   true,
	}, briefing.SSO)
	assert.Equal(t, &iam.SecurityProvisioningSummary{ActiveDirectories: 1}, briefing.Provisioning)
	assert.Equal(t, &iam.SecurityPolicySummary{
		Enabled:                2,
		RequiringMFA:           1,
		RoleActivationRequired: true,
	}, briefing.Policies)
}

func TestGet_RefusesWithoutAnActor(t *testing.T) {
	t.Parallel()

	service := New(Params{
		SecurityConfig: &securityconfig.Config{},
		Logger:         zap.NewNop(),
	})

	_, err := service.Get(t.Context(), nil)
	require.Error(t, err)
}
