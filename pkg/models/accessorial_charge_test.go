package models_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/testutils"
	"github.com/emoss08/trenova/pkg/testutils/factory"
	"github.com/stretchr/testify/require"
)

func TestAccessorialCharge_Validate(t *testing.T) {
	ctx := context.Background()
	s, cleanup := testutils.SetupTestServer(t)
	defer cleanup()

	org, err := factory.NewOrganizationFactory(s.DB).MustCreateOrganization(ctx)
	require.NoError(t, err)

	charge := models.AccessorialCharge{
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		Description:    "Test Description",
		Method:         "Distance",
		Code:           "Test",
	}

	t.Run("ValidateCodeLength", func(t *testing.T) {
		charge.Code = "TESTTESTTESTTEST"
		err = charge.Validate()
		require.Error(t, err)
		require.Equal(t, "code: Code must be between 1 and 10 characters. Please try again.", err.Error())
	})
}
