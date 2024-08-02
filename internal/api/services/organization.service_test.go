package services_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/pkg/testutils/factory"

	"github.com/emoss08/trenova/pkg/testutils"
)

func TestNewOrganizationService(t *testing.T) {
	ctx := context.Background()
	s, cleanup := testutils.SetupTestServer(t)
	defer cleanup()

	service := services.NewOrganizationService(s)
	org, err := factory.NewOrganizationFactory(s.DB).MustCreateOrganization(ctx)
	require.NoError(t, err)

	t.Run("GetOrganization", func(t *testing.T) {
		organization, err := service.GetOrganization(ctx, org.BusinessUnitID, org.ID)
		require.NoError(t, err)

		assert.NotNil(t, organization)
		assert.Equal(t, org.ID, organization.ID)
	})

	t.Run("UpdateOrganization", func(t *testing.T) {
		organization, err := service.GetOrganization(ctx, org.BusinessUnitID, org.ID)
		require.NoError(t, err)

		organization.Name = "Updated Name"

		updated, err := service.UpdateOrganization(ctx, organization)
		require.NoError(t, err)

		assert.NotNil(t, updated)
		assert.Equal(t, organization.ID, updated.ID)
		assert.Equal(t, "Updated Name", updated.Name)
	})
}
