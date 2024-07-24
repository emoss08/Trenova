package services_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/testutils"
	"github.com/emoss08/trenova/pkg/testutils/factory"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommodityService(t *testing.T) {
	ctx := context.Background()
	s := testutils.SetupTestServer(t)
	service := services.NewCommodityService(s)
	org, err := factory.NewOrganizationFactory(s.DB).MustCreateOrganization(ctx)
	require.NoError(t, err)

	createTestCommodity := func(name string) *models.Commodity {
		return &models.Commodity{
			Name:                name,
			Status:              property.StatusActive,
			OrganizationID:      org.ID,
			BusinessUnitID:      org.BusinessUnitID,
			HazardousMaterialID: uuid.Nil,
		}
	}

	t.Run("CreateAndGet", func(t *testing.T) {
		created, err := service.Create(ctx, createTestCommodity("OKAYOKAY"))
		require.NoError(t, err)
		assert.NotNil(t, created)
		assert.NotEqual(t, uuid.Nil, created.ID)

		// Get the created Commodity
		fetched, err := service.Get(ctx, created.ID, created.OrganizationID, created.BusinessUnitID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, fetched.ID)
		assert.Equal(t, created.Name, fetched.Name)
		assert.Equal(t, created.Status, fetched.Status)
	})
}
