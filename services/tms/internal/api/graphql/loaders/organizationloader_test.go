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

	values, errs := factory.batchFunc(tenantInfo)(t.Context(), []string{
		secondID.String(),
		"bad",
		firstID.String(),
		missingID.String(),
		secondID.String(),
	})

	require.Len(t, values, 5)
	require.NoError(t, errs[0])
	assert.Equal(t, "Second", values[0].Name)
	require.Error(t, errs[1])
	require.NoError(t, errs[2])
	assert.Equal(t, "First", values[2].Name)
	require.Error(t, errs[3])
	require.NoError(t, errs[4])
	assert.Equal(t, "Second", values[4].Name)
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

	values, errs := factory.batchFunc(pagination.TenantInfo{})(context.Background(), []string{
		"bad",
		organizationID.String(),
	})

	require.Len(t, values, 2)
	require.Error(t, errs[0])
	require.ErrorIs(t, errs[1], serviceErr)
}
