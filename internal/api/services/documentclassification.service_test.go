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

func TestNewDocumentClassificationService(t *testing.T) {
	ctx := context.Background()
	s, cleanup := testutils.SetupTestServer(t)
	defer cleanup()

	service := services.NewDocumentClassificationService(s)
	org, err := factory.NewOrganizationFactory(s.DB).MustCreateOrganization(ctx)
	require.NoError(t, err)
	user, err := factory.NewUserFactory(s.DB).CreateOrGetUser(ctx)
	require.NoError(t, err)

	createTestDocumentClassification := func(code string) *models.DocumentClassification {
		return &models.DocumentClassification{
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Status:         property.StatusActive,
			Code:           code,
			Description:    "Test Description",
		}
	}

	t.Run("CreateAndGet", func(t *testing.T) {
		created, err := service.Create(ctx, createTestDocumentClassification("OKAY"), user.ID)
		require.NoError(t, err)
		assert.NotNil(t, created)
		assert.NotEqual(t, uuid.Nil, created.ID)

		// Get the created DocumentClassification
		fetched, err := service.Get(ctx, created.ID, created.OrganizationID, created.BusinessUnitID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, fetched.ID)
		assert.Equal(t, created.Code, fetched.Code)
	})

	t.Run("GetAll", func(t *testing.T) {
		// Create multiple document classifications
		for i := 0; i < 5; i++ {
			_, err = service.Create(ctx, createTestDocumentClassification(fmt.Sprintf("COD%d", i)), user.ID)
			require.NoError(t, err)
		}

		// Query all document classifications
		filter := &services.DocumentClassificationQueryFilter{
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
		// Create a new DocumentClassification
		newDocumentClassification := createTestDocumentClassification("TES1")
		created, err := service.Create(ctx, newDocumentClassification, user.ID)
		require.NoError(t, err)

		// Update the DocumentClassification
		created.Description = "Testing update"
		updated, err := service.UpdateOne(ctx, created, user.ID)
		require.NoError(t, err)
		assert.Equal(t, "Testing update", updated.Description)

		// Fetch the updated DocumentClassification
		fetched, err := service.Get(ctx, updated.ID, updated.OrganizationID, updated.BusinessUnitID)
		require.NoError(t, err)
		assert.Equal(t, "Testing update", fetched.Description)
	})

	t.Run("QueryFiltering", func(t *testing.T) {
		// Create DocumentClassification with different codes
		codes := []string{"ABCI", "DEFI", "GHII"}
		for _, code := range codes {
			entity := createTestDocumentClassification(code)
			entity.Code = code
			_, err = service.Create(ctx, entity, user.ID)
			require.NoError(t, err)
		}

		// Query with a specific code
		filter := &services.DocumentClassificationQueryFilter{
			Query:          "ABCI",
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Limit:          10,
			Offset:         0,
		}

		results, count, err := service.GetAll(ctx, filter)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
		assert.Equal(t, "ABCI", results[0].Code)
	})
}
