package models_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/pkg/models/property"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/testutils"
	"github.com/emoss08/trenova/pkg/testutils/factory"
	"github.com/stretchr/testify/require"
)

func TestCommodity_Validate(t *testing.T) {
	ctx := context.Background()
	s, cleanup := testutils.SetupTestServer(t)
	defer cleanup()

	org, err := factory.NewOrganizationFactory(s.DB).MustCreateOrganization(ctx)
	require.NoError(t, err)

	commodity := models.Commodity{
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		Status:         property.StatusActive,
		Name:           "TEST",
		IsHazmat:       false,
		UnitOfMeasure:  "Pallet",
	}

	t.Run("ValidateHazardousMaterialID", func(t *testing.T) {
		commodity.IsHazmat = true
		err = commodity.Validate()
		require.Error(t, err)
		require.Equal(t, "hazardousMaterialId: Hazardous Material ID is required when isHazmat is true. Please try again.", err.Error())
	})

	t.Run("ValidateNameLength", func(t *testing.T) {
		commodity.IsHazmat = false
		commodity.Name = "TESTTESTTESTTESTTESTTESTTESTTESTTESTTESTTESTTESTTEST"
		err = commodity.Validate()
		require.Error(t, err)
		require.Equal(t, "name: Name must be between 1 and 50 characters. Please try again.", err.Error())
	})
}
