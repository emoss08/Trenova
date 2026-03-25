package apikeyservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/apikey"
	permissiondomain "github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestService(t *testing.T) (*Service, *mocks.MockAPIKeyRepository) {
	t.Helper()

	repo := mocks.NewMockAPIKeyRepository(t)
	svc := &Service{
		repo:     repo,
		registry: permissiondomain.NewRegistry(),
		cfg: &config.Config{
			Security: config.SecurityConfig{
				APIToken: config.APITokenConfig{
					DefaultExpiry:    2 * time.Hour,
					MaxExpiry:        24 * time.Hour,
					MaxTokensPerUser: 100,
				},
			},
		},
		l: zap.NewNop(),
	}

	return svc, repo
}

func newAllowedPermission() services.APIKeyPermissionInput {
	return services.APIKeyPermissionInput{
		Resource:   permissiondomain.ResourceCustomer.String(),
		Operations: []permissiondomain.Operation{permissiondomain.OpRead},
		DataScope:  permissiondomain.DataScopeOrganization,
	}
}

func TestCreateAPIKeyRejectsDisallowedResource(t *testing.T) {
	t.Parallel()

	svc, repo := newTestService(t)
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	userID := pulid.MustNew("usr_")
	repo.EXPECT().CountActiveByCreator(mock.Anything, tenantInfo, userID).Return(0, nil)

	_, err := svc.CreateAPIKey(
		t.Context(),
		tenantInfo,
		&services.CreateAPIKeyRequest{
			Name: "Unsafe key",
			Permissions: []services.APIKeyPermissionInput{
				{
					Resource:   permissiondomain.ResourceUser.String(),
					Operations: []permissiondomain.Operation{permissiondomain.OpRead},
					DataScope:  permissiondomain.DataScopeOrganization,
				},
			},
		},
		userID,
	)

	require.Error(t, err)
}

func TestCreateAPIKeyAppliesDefaultExpiry(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)

	expiresAt, err := svc.resolveCreateExpiry(0)
	require.NoError(t, err)
	require.GreaterOrEqual(t, expiresAt, timeutils.NowUnix()+int64((2*time.Hour).Seconds())-2)
}

func TestCreateAPIKeyRejectsExpiryBeyondMax(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)

	_, err := svc.resolveCreateExpiry(timeutils.NowUnix() + int64((48 * time.Hour).Seconds()))
	require.Error(t, err)
}

func TestCreateAPIKeyEnforcesCreatorLimit(t *testing.T) {
	t.Parallel()

	svc, repo := newTestService(t)
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	userID := pulid.MustNew("usr_")

	repo.EXPECT().CountActiveByCreator(mock.Anything, tenantInfo, userID).Return(100, nil)

	_, err := svc.CreateAPIKey(
		t.Context(),
		tenantInfo,
		&services.CreateAPIKeyRequest{
			Name:        "Rate Limited Key",
			Permissions: []services.APIKeyPermissionInput{newAllowedPermission()},
		},
		userID,
	)

	require.Error(t, err)
}

func TestCreateAPIKeySuccessPersistsPermissions(t *testing.T) {
	t.Parallel()

	svc, repo := newTestService(t)
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	userID := pulid.MustNew("usr_")
	req := &services.CreateAPIKeyRequest{
		Name:        "Customer Sync",
		Description: "Reads customer data",
		Permissions: []services.APIKeyPermissionInput{newAllowedPermission()},
	}

	repo.EXPECT().CountActiveByCreator(mock.Anything, tenantInfo, userID).Return(0, nil)
	repo.EXPECT().
		CreateWithPermissions(mock.Anything, mock.AnythingOfType("*apikey.Key"), mock.AnythingOfType("[]*apikey.Permission")).
		Run(func(_ context.Context, key *apikey.Key, perms []*apikey.Permission) {
			key.ID = pulid.MustNew("ak_")
			key.CreatedAt = timeutils.NowUnix()
			key.UpdatedAt = key.CreatedAt
			require.Len(t, perms, 1)
			assert.Equal(t, tenantInfo.OrgID, key.OrganizationID)
			assert.Equal(t, tenantInfo.BuID, key.BusinessUnitID)
			assert.Equal(t, userID, key.CreatedByID)
			assert.Equal(t, permissiondomain.ResourceCustomer.String(), perms[0].Resource)
			assert.Equal(t, key.OrganizationID, perms[0].OrganizationID)
			assert.Equal(t, key.BusinessUnitID, perms[0].BusinessUnitID)
		}).
		Return(nil)

	result, err := svc.CreateAPIKey(t.Context(), tenantInfo, req, userID)

	require.NoError(t, err)
	require.NotEmpty(t, result.Token)
	assert.Equal(t, "Customer Sync", result.Name)
	assert.Equal(t, string(apikey.StatusActive), result.Status)
	assert.Len(t, result.Permissions, 1)
}

func TestRotateAPIKeyResetsUsageState(t *testing.T) {
	t.Parallel()

	svc, repo := newTestService(t)
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	keyID := pulid.MustNew("ak_")
	existing := &apikey.Key{
		ID:                keyID,
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		Name:              "Rotating Key",
		KeyPrefix:         "trv_old",
		Status:            apikey.StatusRevoked,
		LastUsedAt:        123,
		LastUsedIP:        "192.0.2.1",
		LastUsedUserAgent: "integration-test",
		RevokedAt:         456,
		RevokedByID:       pulid.MustNew("usr_"),
	}

	repo.EXPECT().GetByID(mock.Anything, tenantInfo, keyID).Return(existing, nil)
	repo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*apikey.Key")).
		Run(func(_ context.Context, key *apikey.Key) {
			assert.Equal(t, apikey.StatusActive, key.Status)
			assert.Zero(t, key.LastUsedAt)
			assert.Empty(t, key.LastUsedIP)
			assert.Empty(t, key.LastUsedUserAgent)
			assert.Zero(t, key.RevokedAt)
			assert.True(t, key.RevokedByID.IsNil())
			assert.NotEqual(t, "trv_old", key.KeyPrefix)
		}).
		Return(nil)

	result, err := svc.RotateAPIKey(t.Context(), tenantInfo, keyID)

	require.NoError(t, err)
	require.NotEmpty(t, result.Token)
	assert.Equal(t, string(apikey.StatusActive), result.Status)
}

func TestRevokeAPIKeySetsRevokedFields(t *testing.T) {
	t.Parallel()

	svc, repo := newTestService(t)
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	keyID := pulid.MustNew("ak_")
	userID := pulid.MustNew("usr_")
	existing := &apikey.Key{
		ID:             keyID,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Name:           "Revoked Key",
		Status:         apikey.StatusActive,
	}

	repo.EXPECT().GetByID(mock.Anything, tenantInfo, keyID).Return(existing, nil)
	repo.EXPECT().
		Update(mock.Anything, mock.AnythingOfType("*apikey.Key")).
		Run(func(_ context.Context, key *apikey.Key) {
			assert.Equal(t, apikey.StatusRevoked, key.Status)
			assert.Equal(t, userID, key.RevokedByID)
			assert.NotZero(t, key.RevokedAt)
		}).
		Return(nil)

	result, err := svc.RevokeAPIKey(t.Context(), tenantInfo, keyID, userID)

	require.NoError(t, err)
	assert.Equal(t, string(apikey.StatusRevoked), result.Status)
}

func TestCreateAPIKeyNormalizesPermissionsBeforePersisting(t *testing.T) {
	t.Parallel()

	svc, repo := newTestService(t)
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	userID := pulid.MustNew("usr_")

	repo.EXPECT().CountActiveByCreator(mock.Anything, tenantInfo, userID).Return(0, nil)
	repo.EXPECT().
		CreateWithPermissions(mock.Anything, mock.AnythingOfType("*apikey.Key"), mock.AnythingOfType("[]*apikey.Permission")).
		Run(func(_ context.Context, key *apikey.Key, perms []*apikey.Permission) {
			key.ID = pulid.MustNew("ak_")
			key.CreatedAt = timeutils.NowUnix()
			key.UpdatedAt = key.CreatedAt
			require.Len(t, perms, 1)
			assert.Equal(t, permissiondomain.ResourceCustomer.String(), perms[0].Resource)
			assert.Equal(t, permissiondomain.DataScopeOrganization, perms[0].DataScope)
			assert.Equal(
				t,
				[]permissiondomain.Operation{permissiondomain.OpRead, permissiondomain.OpUpdate},
				perms[0].Operations,
			)
		}).
		Return(nil)

	result, err := svc.CreateAPIKey(t.Context(), tenantInfo, &services.CreateAPIKeyRequest{
		Name: "Normalized",
		Permissions: []services.APIKeyPermissionInput{
			{
				Resource: "  " + permissiondomain.ResourceCustomer.String() + "  ",
				Operations: []permissiondomain.Operation{
					permissiondomain.OpUpdate,
					permissiondomain.OpRead,
					permissiondomain.OpUpdate,
				},
				DataScope: "",
			},
		},
	}, userID)

	require.NoError(t, err)
	require.Len(t, result.Permissions, 1)
	assert.Equal(t, permissiondomain.ResourceCustomer.String(), result.Permissions[0].Resource)
	assert.Equal(t, permissiondomain.DataScopeOrganization, result.Permissions[0].DataScope)
	assert.Equal(
		t,
		[]permissiondomain.Operation{permissiondomain.OpRead, permissiondomain.OpUpdate},
		result.Permissions[0].Operations,
	)
}
