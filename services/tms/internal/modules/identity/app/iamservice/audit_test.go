package iamservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks/iamportsmock"

	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/internal/testutil/securityaudittest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func auditedService(
	t *testing.T,
) (*service, *iamportsmock.MockIAMRepository, *securityaudittest.Recorder) {
	t.Helper()

	repo := iamportsmock.NewMockIAMRepository(t)
	recorder := &securityaudittest.Recorder{}
	return &service{
		repo:      repo,
		auditor:   recorder,
		l:         zap.NewNop(),
		db:        dbtest.NopConnection{},
		directory: &recordingDirectoryGrants{},
	}, repo, recorder
}

type recordingDirectoryGrants struct {
	changed []pagination.TenantInfo
	revoked []pulid.ID
	err     error
}

func (r *recordingDirectoryGrants) MappingsChanged(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
) error {
	r.changed = append(r.changed, tenantInfo)
	return r.err
}

func (r *recordingDirectoryGrants) RevokeDirectoryGrants(
	_ context.Context,
	_ pagination.TenantInfo,
	directoryID pulid.ID,
) error {
	r.revoked = append(r.revoked, directoryID)
	return r.err
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

	deps.repo.EXPECT().
		CreateAccessPolicy(ctx, mock.AnythingOfType("*iam.AccessPolicy")).
		Return(saved, nil).
		Once()
	deps.repo.EXPECT().DeleteAccessPolicy(ctx, deps.tenant, saved.ID).Return(nil).Once()

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

	repo.EXPECT().
		GetIdentityProvider(mock.Anything, tenant, existing.ID).
		Return(existing, nil).
		Once()
	repo.EXPECT().DeleteIdentityProvider(mock.Anything, tenant, existing.ID).Return(nil).Once()

	require.NoError(t, svc.DeleteIdentityProvider(t.Context(), tenant, existing.ID))

	change := recorder.Only(t)
	assert.Equal(t, permission.ResourceIdentityProvider, change.Resource)
	assert.Equal(t, permission.OpDelete, change.Operation)
	assert.Same(t, existing, change.Before)
	assert.Equal(t, tenant.UserID, change.Actor.UserID)
	assert.Equal(t, permissionports.PrincipalTypeUser, change.Actor.PrincipalType)
}

func TestFailedIdentityProviderDeletionIsNotRecorded(t *testing.T) {
	t.Parallel()

	svc, repo, recorder := auditedService(t)
	tenant := auditTenant()
	id := pulid.MustNew("idp_")

	repo.EXPECT().
		GetIdentityProvider(mock.Anything, tenant, id).
		Return(&iam.IdentityProvider{ID: id}, nil).
		Once()
	repo.EXPECT().
		DeleteIdentityProvider(mock.Anything, tenant, id).
		Return(errors.New("in use")).
		Once()

	require.Error(t, svc.DeleteIdentityProvider(t.Context(), tenant, id))
	assert.Empty(t, recorder.Changes())
}

func TestSCIMTokenLifecycleIsRecordedWithoutTheSecret(t *testing.T) {
	t.Parallel()

	svc, repo, recorder := auditedService(t)
	orgID, directoryID := pulid.MustNew("org_"), pulid.MustNew("scd_")
	saved := &iam.SCIMToken{ID: pulid.MustNew("sct_"), DirectoryID: directoryID, Prefix: "scim_abc"}

	repo.EXPECT().GetSCIMDirectory(mock.Anything, mock.Anything).
		Return(&iam.SCIMDirectory{ID: directoryID, OrganizationID: orgID}, nil).Once()
	repo.EXPECT().
		CreateSCIMToken(mock.Anything, mock.AnythingOfType("*iam.SCIMToken")).
		Return(saved, nil).
		Once()
	repo.EXPECT().RevokeSCIMToken(mock.Anything, orgID, saved.ID).Return(saved, nil).Once()

	created, err := svc.CreateSCIMToken(
		t.Context(),
		orgID,
		directoryID,
		&iamports.SCIMTokenCreateRequest{Name: "Okta sync"},
	)
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

type fixedRoleGrants struct {
	err     error
	checkOf []pulid.ID
}

func (f *fixedRoleGrants) RequireCanGrantRole(
	_ context.Context,
	_ pagination.TenantInfo,
	roleID pulid.ID,
) error {
	f.checkOf = append(f.checkOf, roleID)
	return f.err
}

func expectDirectory(
	repo *iamportsmock.MockIAMRepository,
	id, ownerOrgID pulid.ID,
) {
	repo.EXPECT().GetSCIMDirectory(mock.Anything, mock.Anything).
		Return(&iam.SCIMDirectory{ID: id, OrganizationID: ownerOrgID}, nil).Once()
}

func TestSCIMTokenRefusedForAnotherOrganizationsDirectory(t *testing.T) {
	t.Parallel()

	svc, repo, recorder := auditedService(t)
	orgID, directoryID := pulid.MustNew("org_"), pulid.MustNew("scd_")
	expectDirectory(repo, directoryID, pulid.MustNew("org_"))

	_, err := svc.CreateSCIMToken(
		t.Context(),
		orgID,
		directoryID,
		&iamports.SCIMTokenCreateRequest{Name: "Okta sync"},
	)

	require.True(t, errortypes.IsNotFoundError(err))
	assert.Empty(t, recorder.Changes())
}

func TestSCIMMappingRefusedForAnotherOrganizationsDirectory(t *testing.T) {
	t.Parallel()

	svc, repo, recorder := auditedService(t)
	svc.plans = plantest.Allowing(t, platformplan.CapabilitySSO)
	grants := &fixedRoleGrants{}
	svc.grants = grants
	tenant := auditTenant()
	directoryID := pulid.MustNew("scd_")
	expectDirectory(repo, directoryID, pulid.MustNew("org_"))

	_, err := svc.CreateSCIMGroupRoleMapping(
		t.Context(),
		tenant,
		directoryID,
		&iam.SCIMGroupRoleMapping{RoleID: pulid.MustNew("rol_")},
	)

	require.True(t, errortypes.IsNotFoundError(err))
	assert.Empty(t, grants.checkOf)
	assert.Empty(t, recorder.Changes())
}

func TestSCIMMappingRefusedForARoleTheActorCannotGrant(t *testing.T) {
	t.Parallel()

	svc, repo, recorder := auditedService(t)
	svc.plans = plantest.Allowing(t, platformplan.CapabilitySSO)
	svc.grants = &fixedRoleGrants{err: errortypes.NewAuthorizationError("beyond your access")}
	tenant := auditTenant()
	directoryID := pulid.MustNew("scd_")

	expectDirectory(repo, directoryID, tenant.OrgID)
	_, err := svc.CreateSCIMGroupRoleMapping(
		t.Context(),
		tenant,
		directoryID,
		&iam.SCIMGroupRoleMapping{RoleID: pulid.MustNew("rol_")},
	)
	require.True(t, errortypes.IsAuthorizationError(err))

	expectDirectory(repo, directoryID, tenant.OrgID)
	_, err = svc.UpdateSCIMGroupRoleMapping(
		t.Context(),
		tenant,
		pulid.MustNew("sgr_"),
		&iam.SCIMGroupRoleMapping{DirectoryID: directoryID, RoleID: pulid.MustNew("rol_")},
	)
	require.True(t, errortypes.IsAuthorizationError(err))

	assert.Empty(t, recorder.Changes())
}

func TestSCIMMappingSavedForARoleTheActorCanGrant(t *testing.T) {
	t.Parallel()

	svc, repo, recorder := auditedService(t)
	svc.plans = plantest.Allowing(t, platformplan.CapabilitySSO)
	grants := &fixedRoleGrants{}
	svc.grants = grants
	tenant := auditTenant()
	directoryID, roleID := pulid.MustNew("scd_"), pulid.MustNew("rol_")
	expectDirectory(repo, directoryID, tenant.OrgID)
	repo.EXPECT().
		CreateSCIMGroupRoleMapping(mock.Anything, mock.AnythingOfType("*iam.SCIMGroupRoleMapping")).
		Return(&iam.SCIMGroupRoleMapping{ID: pulid.MustNew("sgr_")}, nil).
		Once()

	_, err := svc.CreateSCIMGroupRoleMapping(
		t.Context(),
		tenant,
		directoryID,
		&iam.SCIMGroupRoleMapping{RoleID: roleID},
	)

	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{roleID}, grants.checkOf)
	assert.Len(t, recorder.Changes(), 1)
}

func TestSCIMDirectoryUpdateRequiresTheDirectoryInTheTenant(t *testing.T) {
	t.Parallel()

	svc, repo, recorder := auditedService(t)
	tenant := auditTenant()
	id := pulid.MustNew("scd_")

	repo.EXPECT().
		GetSCIMDirectory(mock.Anything, iamports.GetSCIMDirectoryRequest{ID: id, TenantInfo: tenant}).
		Return(nil, errors.New("not found")).
		Once()

	_, err := svc.UpdateSCIMDirectory(
		t.Context(),
		tenant,
		id,
		&iam.SCIMDirectory{TenantSlug: "acme"},
	)
	require.Error(t, err)
	assert.Empty(t, recorder.Changes())
}

func TestSCIMMappingRecordsWhoSavedItAndReconcilesTheDirectory(t *testing.T) {
	t.Parallel()

	svc, repo, _ := auditedService(t)
	svc.plans = plantest.Allowing(t, platformplan.CapabilitySSO)
	svc.grants = &fixedRoleGrants{}
	directory := &recordingDirectoryGrants{}
	svc.directory = directory
	tenant := auditTenant()
	directoryID := pulid.MustNew("scd_")
	expectDirectory(repo, directoryID, tenant.OrgID)

	var saved *iam.SCIMGroupRoleMapping
	repo.EXPECT().
		CreateSCIMGroupRoleMapping(mock.Anything, mock.AnythingOfType("*iam.SCIMGroupRoleMapping")).
		RunAndReturn(func(_ context.Context, entity *iam.SCIMGroupRoleMapping) (*iam.SCIMGroupRoleMapping, error) {
			saved = entity
			entity.ID = pulid.MustNew("sgr_")
			return entity, nil
		}).
		Once()

	_, err := svc.CreateSCIMGroupRoleMapping(
		t.Context(),
		tenant,
		directoryID,
		&iam.SCIMGroupRoleMapping{RoleID: pulid.MustNew("rol_"), SavedByID: pulid.MustNew("usr_")},
	)

	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(
		t,
		tenant.UserID,
		saved.SavedByID,
		"the saver is the person saving, never the body",
	)
	assert.Equal(t, []pagination.TenantInfo{tenant}, directory.changed)
}

func TestSCIMMappingIsNotSavedWhenItsGrantsCannotBeReconciled(t *testing.T) {
	t.Parallel()

	svc, repo, recorder := auditedService(t)
	svc.plans = plantest.Allowing(t, platformplan.CapabilitySSO)
	svc.grants = &fixedRoleGrants{}
	svc.directory = &recordingDirectoryGrants{err: errors.New("database unavailable")}
	tenant := auditTenant()
	directoryID := pulid.MustNew("scd_")
	expectDirectory(repo, directoryID, tenant.OrgID)
	repo.EXPECT().
		CreateSCIMGroupRoleMapping(mock.Anything, mock.AnythingOfType("*iam.SCIMGroupRoleMapping")).
		Return(&iam.SCIMGroupRoleMapping{ID: pulid.MustNew("sgr_")}, nil).
		Once()

	_, err := svc.CreateSCIMGroupRoleMapping(
		t.Context(),
		tenant,
		directoryID,
		&iam.SCIMGroupRoleMapping{RoleID: pulid.MustNew("rol_")},
	)

	require.Error(t, err)
	assert.Empty(t, recorder.Changes())
}

func TestSCIMDirectoryDeletionRevokesWhatItGrantedFirst(t *testing.T) {
	t.Parallel()

	svc, repo, recorder := auditedService(t)
	directory := &recordingDirectoryGrants{}
	svc.directory = directory
	tenant := auditTenant()
	directoryID := pulid.MustNew("scd_")
	expectDirectory(repo, directoryID, tenant.OrgID)
	repo.EXPECT().DeleteSCIMDirectory(mock.Anything, tenant, directoryID).Return(nil).Once()

	require.NoError(t, svc.DeleteSCIMDirectory(t.Context(), tenant, directoryID))
	assert.Equal(t, []pulid.ID{directoryID}, directory.revoked)
	assert.Len(t, recorder.Changes(), 1)
}

func TestSCIMDirectoryIsKeptWhenItsGrantsCannotBeRevoked(t *testing.T) {
	t.Parallel()

	svc, repo, recorder := auditedService(t)
	svc.directory = &recordingDirectoryGrants{err: errors.New("database unavailable")}
	tenant := auditTenant()
	directoryID := pulid.MustNew("scd_")
	expectDirectory(repo, directoryID, tenant.OrgID)

	require.Error(t, svc.DeleteSCIMDirectory(t.Context(), tenant, directoryID))
	assert.Empty(t, recorder.Changes())
}
