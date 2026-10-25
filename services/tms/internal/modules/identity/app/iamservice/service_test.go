package iamservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/testutil/mocks/iamportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type accessPolicyServiceDeps struct {
	svc    *service
	repo   *iamportsmock.MockIAMRepository
	tenant pagination.TenantInfo
}

func setupAccessPolicyService(t *testing.T) accessPolicyServiceDeps {
	t.Helper()

	repo := iamportsmock.NewMockIAMRepository(t)
	tenant := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	return accessPolicyServiceDeps{
		svc: &service{
			repo: repo,
			validator: newValidator(
				&accessPolicyValidatorRepo{},
				&accessPolicyUniquenessChecker{},
			),
			l: zap.NewNop(),
		},
		repo:   repo,
		tenant: tenant,
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

func TestCreateAccessPolicy_StampsTheTenant(t *testing.T) {
	t.Parallel()

	deps := setupAccessPolicyService(t)
	ctx := t.Context()
	entity := validAccessPolicy()

	deps.repo.EXPECT().
		CreateAccessPolicy(ctx, mock.AnythingOfType("*iam.AccessPolicy")).
		Return(entity, nil).
		Once()

	created, err := deps.svc.CreateAccessPolicy(ctx, deps.tenant, entity)

	require.NoError(t, err)
	assert.Equal(t, deps.tenant.OrgID, created.OrganizationID)
	assert.Equal(t, deps.tenant.BuID, created.BusinessUnitID)
}

func TestCreateAccessPolicy_RepositoryFailurePropagates(t *testing.T) {
	t.Parallel()

	deps := setupAccessPolicyService(t)
	ctx := t.Context()
	insertErr := errors.New("insert failed")

	deps.repo.EXPECT().
		CreateAccessPolicy(ctx, mock.AnythingOfType("*iam.AccessPolicy")).
		Return(nil, insertErr).
		Once()

	_, err := deps.svc.CreateAccessPolicy(ctx, deps.tenant, validAccessPolicy())

	require.ErrorIs(t, err, insertErr)
}

func TestUpdateAccessPolicy_KeepsTheRequestedID(t *testing.T) {
	t.Parallel()

	deps := setupAccessPolicyService(t)
	ctx := t.Context()
	entity := validAccessPolicy()
	id := pulid.MustNew("ap_")

	deps.repo.EXPECT().
		UpdateAccessPolicy(ctx, mock.AnythingOfType("*iam.AccessPolicy")).
		Return(entity, nil).
		Once()

	updated, err := deps.svc.UpdateAccessPolicy(ctx, deps.tenant, id, entity)

	require.NoError(t, err)
	assert.Equal(t, id, updated.ID)
}

func TestDeleteAccessPolicy_DeletesThroughTheRepository(t *testing.T) {
	t.Parallel()

	deps := setupAccessPolicyService(t)
	ctx := t.Context()
	id := pulid.MustNew("ap_")

	deps.repo.EXPECT().DeleteAccessPolicy(ctx, deps.tenant, id).Return(nil).Once()

	require.NoError(t, deps.svc.DeleteAccessPolicy(ctx, deps.tenant, id))
}
