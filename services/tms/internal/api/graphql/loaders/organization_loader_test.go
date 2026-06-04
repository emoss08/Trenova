package loaders

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestOrganizationBatchFunc_DelegatesToOrganizationService(t *testing.T) {
	t.Parallel()

	firstID := pulid.MustNew("org_")
	secondID := pulid.MustNew("org_")
	missingID := pulid.MustNew("org_")
	tenantInfo := pagination.TenantInfo{
		OrgID:  firstID,
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	organizationService := mocks.NewMockOrganizationService(t)
	organizationService.EXPECT().
		GetByIDs(mock.Anything, services.GetOrganizationsByIDsRequest{
			TenantInfo:      tenantInfo,
			OrganizationIDs: []pulid.ID{secondID, firstID, missingID},
			IncludeState:    true,
			IncludeBU:       true,
		}).
		Return([]*tenant.Organization{
			{ID: firstID, Name: "First"},
			{ID: secondID, Name: "Second"},
		}, nil).
		Once()
	factory := &OrganizationByIDLoaderFactory{organizationService: organizationService}

	results := factory.batchFunc(tenantInfo)(t.Context(), []string{
		secondID.String(),
		"bad",
		firstID.String(),
		missingID.String(),
		secondID.String(),
	})

	require.Len(t, results, 5)
	require.NoError(t, results[0].Error)
	assert.Equal(t, "Second", results[0].Data.Name)
	require.Error(t, results[1].Error)
	require.NoError(t, results[2].Error)
	assert.Equal(t, "First", results[2].Data.Name)
	require.Error(t, results[3].Error)
	require.NoError(t, results[4].Error)
	assert.Equal(t, "Second", results[4].Data.Name)
}

func TestOrganizationBatchFunc_ServiceErrorFillsValidResults(t *testing.T) {
	t.Parallel()

	organizationID := pulid.MustNew("org_")
	serviceErr := assert.AnError
	organizationService := mocks.NewMockOrganizationService(t)
	organizationService.EXPECT().
		GetByIDs(mock.Anything, services.GetOrganizationsByIDsRequest{
			OrganizationIDs: []pulid.ID{organizationID},
			IncludeState:    true,
			IncludeBU:       true,
		}).
		Return(nil, serviceErr).
		Once()
	factory := &OrganizationByIDLoaderFactory{organizationService: organizationService}

	results := factory.batchFunc(pagination.TenantInfo{})(context.Background(), []string{
		"bad",
		organizationID.String(),
	})

	require.Len(t, results, 2)
	require.Error(t, results[0].Error)
	require.ErrorIs(t, results[1].Error, serviceErr)
}
