// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

package services_test

import (
	"context"
	"fmt"
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

func TestCommentTypeService(t *testing.T) {
	ctx := context.Background()
	s := testutils.SetupTestServer(t)
	service := services.NewCommentTypeService(s)
	org, err := factory.NewOrganizationFactory(s.DB).MustCreateOrganization(ctx)
	require.NoError(t, err)

	createTestCommentType := func(name string) *models.CommentType {
		return &models.CommentType{
			Status:         property.StatusActive,
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Name:           name,
			Description:    "Test Accessorial Charge",
			Severity:       property.SeverityHigh,
		}
	}

	t.Run("CreateAndGet", func(t *testing.T) {
		created, err := service.Create(ctx, createTestCommentType("OKAYOKAY"))
		require.NoError(t, err)
		assert.NotNil(t, created)
		assert.NotEqual(t, uuid.Nil, created.ID)

		// Get the created AccessorialCharge
		fetched, err := service.Get(ctx, created.ID, created.OrganizationID, created.BusinessUnitID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, fetched.ID)
		assert.Equal(t, created.Name, fetched.Name)
		assert.Equal(t, created.Description, fetched.Description)
	})

	t.Run("GetAll", func(t *testing.T) {
		// Create multiple AccessorialCharges
		for i := 0; i < 5; i++ {
			_, err = service.Create(ctx, createTestCommentType(fmt.Sprintf("CODE%d", i)))
			require.NoError(t, err)
		}

		// Query all AccessorialCharges
		filter := &services.CommentTypeQueryFilter{
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
		// Create a new AccessorialCharge
		newCharge := createTestCommentType("WEIRD")
		created, err := service.Create(ctx, newCharge)
		require.NoError(t, err)

		// Update the AccessorialCharge
		created.Description = "Updated Description"
		updated, err := service.UpdateOne(ctx, created)
		require.NoError(t, err)
		assert.Equal(t, "Updated Description", updated.Description)

		// Fetch the updated AccessorialCharge
		fetched, err := service.Get(ctx, updated.ID, updated.OrganizationID, updated.BusinessUnitID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Description", fetched.Description)
	})

	t.Run("QueryFiltering", func(t *testing.T) {
		// Create AccessorialCharges with different codes
		codes := []string{"ABC", "DEF", "GHI"}
		for _, code := range codes {
			charge := createTestCommentType(code)
			charge.Name = code
			_, err = service.Create(ctx, charge)
			require.NoError(t, err)
		}

		// Query with a specific code
		filter := &services.CommentTypeQueryFilter{
			Query:          "ABC",
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Limit:          10,
			Offset:         0,
		}
		charges, count, err := service.GetAll(ctx, filter)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
		assert.Equal(t, "ABC", charges[0].Name)
	})
}
