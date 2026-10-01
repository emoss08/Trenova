package iamservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/securityaudittest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func auditedService(t *testing.T) (*service, *mocks.MockIAMRepository, *securityaudittest.Recorder) {
	t.Helper()

	repo := mocks.NewMockIAMRepository(t)
	recorder := &securityaudittest.Recorder{}
	return &service{repo: repo, auditor: recorder, l: zap.NewNop()}, repo, recorder
}

func auditTenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func TestAccessPolicyChangesAreRecorded(t *testing.T) {
	t.Parallel()

	deps := setupAccessPolicyService(t)
	recorder := &securityaudittest.Recorder{}
	deps.svc.auditor = recorder
	ctx := t.Context()
	entity := validAccessPolicy()
	saved := validAccessPolicy()
	saved.ID = pulid.MustNew("iap_")

	deps.repo.EXPECT().CreateAccessPolicy(ctx, mock.AnythingOfType("*iam.AccessPolicy")).Return(saved, nil).Once()
	deps.repo.EXPECT().DeleteAccessPolicy(ctx, deps.tenant, saved.ID).Return(nil).Once()
	deps.policyCache.EXPECT().Invalidate(ctx, deps.lookup).Return(nil).Twice()

	_, err := deps.svc.CreateAccessPolicy(ctx, deps.tenant, entity)
	require.NoError(t, err)
	require.NoError(t, deps.svc.DeleteAccessPolicy(ctx, deps.tenant, saved.ID))

	changes := recorder.Changes()
	require.Len(t, changes, 2)
	assert.Equal(t, permission.ResourceAccessPolicy, changes[0].Resource)
	assert.Equal(t, permission.OpCreate, changes[0].Operation)
	assert.Equal(t, permission.OpDelete, changes[1].Operation)
	assert.Equal(t, saved.ID.String(), changes[0].ResourceID)
	assert.Equal(t, saved.ID.String(), changes[1].ResourceID)
}

func TestIdentityProviderDeletionRecordsWhatWasRemoved(t *testing.T) {
	t.Parallel()

	svc, repo, recorder := auditedService(t)
	tenant := auditTenant()
	existing := &iam.IdentityProvider{ID: pulid.MustNew("idp_"), Name: "Okta"}

	repo.EXPECT().GetIdentityProvider(mock.Anything, tenant, existing.ID).Return(existing, nil).Once()
	repo.EXPECT().DeleteIdentityProvider(mock.Anything, tenant, existing.ID).Return(nil).Once()

	require.NoError(t, svc.DeleteIdentityProvider(t.Context(), tenant, existing.ID))

	change := recorder.Only(t)
	assert.Equal(t, permission.ResourceIdentityProvider, change.Resource)
	assert.Equal(t, permission.OpDelete, change.Operation)
	assert.Same(t, existing, change.Before)
	assert.Equal(t, tenant.UserID, change.Actor.UserID)
	assert.Equal(t, services.PrincipalTypeUser, change.Actor.PrincipalType)
}

func TestFailedIdentityProviderDeletionIsNotRecorded(t *testing.T) {
	t.Parallel()

	svc, repo, recorder := auditedService(t)
	tenant := auditTenant()
	id := pulid.MustNew("idp_")

	repo.EXPECT().GetIdentityProvider(mock.Anything, tenant, id).Return(&iam.IdentityProvider{ID: id}, nil).Once()
	repo.EXPECT().DeleteIdentityProvider(mock.Anything, tenant, id).Return(errors.New("in use")).Once()

	require.Error(t, svc.DeleteIdentityProvider(t.Context(), tenant, id))
	assert.Empty(t, recorder.Changes())
}

func TestSCIMTokenLifecycleIsRecordedWithoutTheSecret(t *testing.T) {
	t.Parallel()

	svc, repo, recorder := auditedService(t)
	orgID, directoryID := pulid.MustNew("org_"), pulid.MustNew("scd_")
	saved := &iam.SCIMToken{ID: pulid.MustNew("sct_"), DirectoryID: directoryID, Prefix: "scim_abc"}

	repo.EXPECT().CreateSCIMToken(mock.Anything, mock.AnythingOfType("*iam.SCIMToken")).Return(saved, nil).Once()
	repo.EXPECT().RevokeSCIMToken(mock.Anything, orgID, saved.ID).Return(saved, nil).Once()

	created, err := svc.CreateSCIMToken(t.Context(), orgID, directoryID, &services.SCIMTokenCreateRequest{Name: "Okta sync"})
	require.NoError(t, err)
	_, err = svc.RevokeSCIMToken(t.Context(), orgID, saved.ID)
	require.NoError(t, err)

	changes := recorder.Changes()
	require.Len(t, changes, 2)
	assert.Equal(t, "SCIM token created", changes[0].Comment)
	assert.Equal(t, "SCIM token revoked", changes[1].Comment)
	for _, change := range changes {
		assert.Equal(t, directoryID.String(), change.ResourceID)
		assert.Equal(t, "scim_abc", change.Metadata["tokenPrefix"])
		assert.NotContains(t, change.Metadata, "token")
		assert.NotEqual(t, created.Token, change.Metadata["tokenPrefix"])
	}
}

func TestSCIMDirectoryUpdateRequiresTheDirectoryInTheTenant(t *testing.T) {
	t.Parallel()

	svc, repo, recorder := auditedService(t)
	tenant := auditTenant()
	id := pulid.MustNew("scd_")

	repo.EXPECT().GetSCIMDirectory(mock.Anything, repositories.GetSCIMDirectoryRequest{ID: id, TenantInfo: tenant}).
		Return(nil, errors.New("not found")).Once()

	_, err := svc.UpdateSCIMDirectory(t.Context(), tenant, id, &iam.SCIMDirectory{TenantSlug: "acme"})
	require.Error(t, err)
	assert.Empty(t, recorder.Changes())
}
