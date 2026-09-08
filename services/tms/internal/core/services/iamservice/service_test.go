package iamservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type accessPolicyServiceDeps struct {
	svc         *service
	repo        *mocks.MockIAMRepository
	policyCache *mocks.MockAccessPolicyCacheRepository
	tenant      pagination.TenantInfo
	lookup      repositories.IAMTenantPolicyLookupRequest
}

func setupAccessPolicyService(t *testing.T) accessPolicyServiceDeps {
	t.Helper()

	repo := mocks.NewMockIAMRepository(t)
	policyCache := mocks.NewMockAccessPolicyCacheRepository(t)
	tenant := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	return accessPolicyServiceDeps{
		svc: &service{
			repo:        repo,
			policyCache: policyCache,
			validator:   newValidator(&accessPolicyValidatorRepo{}, &accessPolicyUniquenessChecker{}),
			l:           zap.NewNop(),
		},
		repo:        repo,
		policyCache: policyCache,
		tenant:      tenant,
		lookup: repositories.IAMTenantPolicyLookupRequest{
			OrganizationID: tenant.OrgID,
			BusinessUnitID: tenant.BuID,
		},
	}
}

func validAccessPolicy() *iam.AccessPolicy {
	return &iam.AccessPolicy{
		Name:      "Deny shipment read",
		Resource:  "shipment",
		Operation: "read",
		Effect:    iam.PolicyEffectDeny,
		Priority:  10,
		Enabled:   true,
	}
}

func TestCreateAccessPolicy_InvalidatesCache(t *testing.T) {
	t.Parallel()

	deps := setupAccessPolicyService(t)
	ctx := t.Context()
	entity := validAccessPolicy()

	deps.repo.EXPECT().
		CreateAccessPolicy(ctx, mock.AnythingOfType("*iam.AccessPolicy")).
		Return(entity, nil).
		Once()
	deps.policyCache.EXPECT().Invalidate(ctx, deps.lookup).Return(nil).Once()

	created, err := deps.svc.CreateAccessPolicy(ctx, deps.tenant, entity)

	require.NoError(t, err)
	assert.Equal(t, deps.tenant.OrgID, created.OrganizationID)
	assert.Equal(t, deps.tenant.BuID, created.BusinessUnitID)
}

func TestCreateAccessPolicy_RepositoryFailureSkipsInvalidation(t *testing.T) {
	t.Parallel()

	deps := setupAccessPolicyService(t)
	ctx := t.Context()

	deps.repo.EXPECT().
		CreateAccessPolicy(ctx, mock.AnythingOfType("*iam.AccessPolicy")).
		Return(nil, errors.New("insert failed")).
		Once()

	_, err := deps.svc.CreateAccessPolicy(ctx, deps.tenant, validAccessPolicy())

	require.Error(t, err)
	deps.policyCache.AssertNotCalled(t, "Invalidate", mock.Anything, mock.Anything)
}

func TestUpdateAccessPolicy_InvalidatesCache(t *testing.T) {
	t.Parallel()

	deps := setupAccessPolicyService(t)
	ctx := t.Context()
	entity := validAccessPolicy()
	id := pulid.MustNew("ap_")

	deps.repo.EXPECT().
		UpdateAccessPolicy(ctx, mock.AnythingOfType("*iam.AccessPolicy")).
		Return(entity, nil).
		Once()
	deps.policyCache.EXPECT().Invalidate(ctx, deps.lookup).Return(nil).Once()

	updated, err := deps.svc.UpdateAccessPolicy(ctx, deps.tenant, id, entity)

	require.NoError(t, err)
	assert.Equal(t, id, updated.ID)
}

func TestDeleteAccessPolicy_InvalidatesCache(t *testing.T) {
	t.Parallel()

	deps := setupAccessPolicyService(t)
	ctx := t.Context()
	id := pulid.MustNew("ap_")

	deps.repo.EXPECT().DeleteAccessPolicy(ctx, deps.tenant, id).Return(nil).Once()
	deps.policyCache.EXPECT().Invalidate(ctx, deps.lookup).Return(nil).Once()

	require.NoError(t, deps.svc.DeleteAccessPolicy(ctx, deps.tenant, id))
}

func TestDeleteAccessPolicy_InvalidationFailureDoesNotFailMutation(t *testing.T) {
	t.Parallel()

	deps := setupAccessPolicyService(t)
	ctx := t.Context()
	id := pulid.MustNew("ap_")

	deps.repo.EXPECT().DeleteAccessPolicy(ctx, deps.tenant, id).Return(nil).Once()
	deps.policyCache.EXPECT().Invalidate(ctx, deps.lookup).Return(errors.New("redis down")).Once()

	require.NoError(t, deps.svc.DeleteAccessPolicy(ctx, deps.tenant, id))
}

func TestDeleteAccessPolicy_WithoutCacheStillDeletes(t *testing.T) {
	t.Parallel()

	deps := setupAccessPolicyService(t)
	deps.svc.policyCache = nil
	ctx := t.Context()
	id := pulid.MustNew("ap_")

	deps.repo.EXPECT().DeleteAccessPolicy(ctx, deps.tenant, id).Return(nil).Once()

	require.NoError(t, deps.svc.DeleteAccessPolicy(ctx, deps.tenant, id))
	deps.policyCache.AssertNotCalled(t, "Invalidate", mock.Anything, mock.Anything)
}
