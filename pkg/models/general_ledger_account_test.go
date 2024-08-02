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

func TestGeneralLedgerAccount_Validate(t *testing.T) {
	ctx := context.Background()
	s, cleanup := testutils.SetupTestServer(t)
	defer cleanup()

	org, err := factory.NewOrganizationFactory(s.DB).MustCreateOrganization(ctx)
	require.NoError(t, err)

	glAccount := models.GeneralLedgerAccount{
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		Status:         property.StatusActive,
		AccountNumber:  "1000-00",
		AccountType:    property.GLAccountTypeExpense,
	}

	t.Run("ValidateAccountNumberFormat", func(t *testing.T) {
		glAccount.AccountNumber = "1234567"
		err = glAccount.Validate()
		require.Error(t, err)
		require.Equal(t, "accountNumber: Account number must be in the format ####-##. Please try again.", err.Error())
	})

	t.Run("ValidateAccountNumberLength", func(t *testing.T) {
		glAccount.AccountNumber = "1234-567"
		err = glAccount.Validate()
		require.Error(t, err)
		require.Equal(t, "accountNumber: Account number must be 7 characters. Please try again.", err.Error())
	})
}
