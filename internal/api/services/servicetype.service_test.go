package services_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/testutils/factory"

	"github.com/emoss08/trenova/pkg/testutils"
)

func TestNewServiceTypeService(t *testing.T) {
	ctx := context.Background()
	s, cleanup := testutils.SetupTestServer(t)
	defer cleanup()

	service := services.NewServiceTypeService(s)
	org, err := factory.NewOrganizationFactory(s.DB).MustCreateOrganization(ctx)
	require.NoError(t, err)
	user, err := factory.NewUserFactory(s.DB).CreateOrGetUser(ctx)
	require.NoError(t, err)

	createServiceType := func(code string) *models.ServiceType {
		return &models.ServiceType{
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Status:         property.StatusActive,
			Code:           code,
			Description:    "Test Description",
		}
	}

	t.Run("CreateAndGet", func(t *testing.T) {
		created, err := service.Create(ctx, createServiceType("TEST"), user.ID)
		require.NoError(t, err)
		assert.NotNil(t, created)
		assert.NotEqual(t, uuid.Nil, created.ID)

		// Get the created ServiceType
		fetched, err := service.Get(ctx, created.ID, created.OrganizationID, created.BusinessUnitID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, fetched.ID)
		assert.Equal(t, created.Code, fetched.Code)
	})

	t.Run("GetAll", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			_, err = service.Create(ctx, createServiceType(fmt.Sprintf("cod%d", i)), user.ID)
			require.NoError(t, err)
		}

		filter := &services.ServiceTypeQueryFilter{
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Limit:          10,
			Offset:         0,
		}

		entities, count, err := service.GetAll(ctx, filter)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 5)
		assert.GreaterOrEqual(t, len(entities), 5)
	})

	t.Run("Update", func(t *testing.T) {
		// Create a new ServiceType
		newServiceType := createServiceType("TES1")
		created, err := service.Create(ctx, newServiceType, user.ID)
		require.NoError(t, err)

		// Update the ServiceType
		created.Description = "Test Description 2"
		updated, err := service.UpdateOne(ctx, created, user.ID)
		require.NoError(t, err)
		assert.Equal(t, "Test Description 2", updated.Description)

		// Fetch the updated ServiceType
		fetched, err := service.Get(ctx, updated.ID, updated.OrganizationID, updated.BusinessUnitID)
		require.NoError(t, err)
		assert.Equal(t, "Test Description 2", fetched.Description)
	})

	t.Run("QueryFiltering", func(t *testing.T) {
		// Create ServiceType with different codes
		codes := []string{"TES2", "TES3", "TES4"}
		for _, code := range codes {
			entity := createServiceType(code)
			entity.Code = code
			_, err = service.Create(ctx, entity, user.ID)
			require.NoError(t, err)
		}

		// Query with a specific code
		filter := &services.ServiceTypeQueryFilter{
			Query:          "TES2",
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Limit:          10,
			Offset:         0,
		}

		results, count, err := service.GetAll(ctx, filter)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
		assert.Equal(t, "TES2", results[0].Code)
	})
}
