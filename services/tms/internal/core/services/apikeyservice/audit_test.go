package apikeyservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/apikey"
	permissiondomain "github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/securityaudittest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func auditTenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func TestCreateAPIKeyRecordsTheCreation(t *testing.T) {
	t.Parallel()

	svc, repo := newTestService(t)
	recorder := &securityaudittest.Recorder{}
	svc.auditor = recorder
	tenantInfo := auditTenant()
	creator := pulid.MustNew("usr_")

	repo.EXPECT().CountActiveByCreator(mock.Anything, tenantInfo, creator).Return(0, nil)
	repo.EXPECT().
		CreateWithPermissions(mock.Anything, mock.AnythingOfType("*apikey.Key"), mock.AnythingOfType("[]*apikey.Permission")).
		Return(nil)

	result, err := svc.CreateAPIKey(t.Context(), tenantInfo, &services.CreateAPIKeyRequest{
		Name:        "Customer Sync",
		Permissions: []services.APIKeyPermissionInput{newAllowedPermission()},
	}, creator)
	require.NoError(t, err)

	change := recorder.Only(t)
	assert.Equal(t, permissiondomain.ResourceAPIKey, change.Resource)
	assert.Equal(t, permissiondomain.OpCreate, change.Operation)
	assert.Equal(t, creator, change.Actor.UserID)
	assert.Equal(t, tenantInfo.OrgID, change.OrganizationID)
	assert.Nil(t, change.Before)
	after, ok := change.After.(services.APIKeyResponse)
	require.True(t, ok)
	assert.Equal(t, result.KeyPrefix, after.KeyPrefix)
	assert.NotContains(t, change.Metadata, "token")
}

func TestRotateAndRevokeAPIKeyRecordTheChange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		comment string
		call    func(*Service, pagination.TenantInfo, pulid.ID) error
	}{
		{
			name:    "rotate",
			comment: "API key secret rotated",
			call: func(svc *Service, tenantInfo pagination.TenantInfo, id pulid.ID) error {
				_, err := svc.RotateAPIKey(t.Context(), tenantInfo, id)
				return err
			},
		},
		{
			name:    "revoke",
			comment: "API key revoked",
			call: func(svc *Service, tenantInfo pagination.TenantInfo, id pulid.ID) error {
				_, err := svc.RevokeAPIKey(t.Context(), tenantInfo, id, tenantInfo.UserID)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, repo := newTestService(t)
			recorder := &securityaudittest.Recorder{}
			svc.auditor = recorder
			tenantInfo := auditTenant()
			key := &apikey.Key{
				ID:             pulid.MustNew("ak_"),
				OrganizationID: tenantInfo.OrgID,
				BusinessUnitID: tenantInfo.BuID,
				KeyPrefix:      "trv_old",
				Status:         apikey.StatusActive,
			}

			repo.EXPECT().GetByID(mock.Anything, tenantInfo, key.ID).Return(key, nil)
			repo.EXPECT().Update(mock.Anything, mock.AnythingOfType("*apikey.Key")).Return(nil)

			require.NoError(t, tt.call(svc, tenantInfo, key.ID))

			change := recorder.Only(t)
			assert.Equal(t, tt.comment, change.Comment)
			assert.Equal(t, key.ID.String(), change.ResourceID)
			assert.Equal(t, tenantInfo.UserID, change.Actor.UserID)
			before, ok := change.Before.(services.APIKeyResponse)
			require.True(t, ok)
			assert.Equal(t, string(apikey.StatusActive), before.Status)
			assert.NotNil(t, change.After)
		})
	}
}
