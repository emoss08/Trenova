package services_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"

	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/pkg/testutils/factory"

	"github.com/emoss08/trenova/pkg/testutils"
)

func TestNewHazardousMaterialService(t *testing.T) {
	ctx := context.Background()
	s, cleanup := testutils.SetupTestServer(t)
	defer cleanup()

	service := services.NewHazardousMaterialService(s)
	org, err := factory.NewOrganizationFactory(s.DB).MustCreateOrganization(ctx)
	require.NoError(t, err)
	user, err := factory.NewUserFactory(s.DB).CreateOrGetUser(ctx)
	require.NoError(t, err)

	createHazardousMaterial := func(name string) *models.HazardousMaterial {
		return &models.HazardousMaterial{
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Status:         property.StatusActive,
			Name:           name,
			HazardClass:    "HazardClass1And1",
		}
	}

	t.Run("CreateAndGet", func(t *testing.T) {
		created, err := service.Create(ctx, createHazardousMaterial("TEST"), user.ID)
		require.NoError(t, err)
		assert.NotNil(t, created)
		assert.NotEqual(t, uuid.Nil, created.ID)

		// Get the created HazardousMaterial
		fetched, err := service.Get(ctx, created.ID, created.OrganizationID, created.BusinessUnitID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, fetched.ID)
		assert.Equal(t, created.Name, fetched.Name)
	})

	t.Run("GetAll", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			_, err = service.Create(ctx, createHazardousMaterial(fmt.Sprintf("code%d", i)), user.ID)
			require.NoError(t, err)
		}

		filter := &services.HazardousMaterialQueryFilter{
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Limit:          10,
			Offset:         0,
		}

		charges, count, err := service.GetAll(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 5)
		assert.GreaterOrEqual(t, len(charges), 5)
	})

	t.Run("Update", func(t *testing.T) {
		// Create a new HazardousMaterial
		newHazardousMaterial := createHazardousMaterial("TEST1")
		created, err := service.Create(ctx, newHazardousMaterial, user.ID)
		require.NoError(t, err)

		// Update the HazardousMaterial
		created.Description = "Test Description"
		updated, err := service.UpdateOne(ctx, created, user.ID)
		require.NoError(t, err)
		assert.Equal(t, "Test Description", updated.Description)

		// Fetch the updated HazardousMaterial
		fetched, err := service.Get(ctx, updated.ID, updated.OrganizationID, updated.BusinessUnitID)
		require.NoError(t, err)
		assert.Equal(t, "Test Description", fetched.Description)
	})

	t.Run("QueryFiltering", func(t *testing.T) {
		// Create HazardousMaterial with different codes
		names := []string{"TEST2", "TEST3", "TEST4"}
		for _, name := range names {
			entity := createHazardousMaterial(name)
			entity.Name = name
			_, err = service.Create(ctx, entity, user.ID)
			require.NoError(t, err)
		}

		// Query with a specific code
		filter := &services.HazardousMaterialQueryFilter{
			Query:          "TEST2",
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Limit:          10,
			Offset:         0,
		}

		results, count, err := service.GetAll(ctx, filter)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
		assert.Equal(t, "TEST2", results[0].Name)
	})
}
