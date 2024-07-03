package models_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAccessorialCharge_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		ac := &models.AccessorialCharge{
			Status:         property.StatusActive,
			OrganizationID: uuid.New(),
			BusinessUnitID: uuid.New(),
			Code:           "CODE",
			Description:    "Test Accessorial Charge",
		}

		err := ac.Validate()
		require.NoError(t, err)
	})

	t.Run("invalid", func(t *testing.T) {
		ac := &models.AccessorialCharge{
			Status:         property.StatusActive,
			OrganizationID: uuid.New(),
			BusinessUnitID: uuid.New(),
		}

		err := ac.Validate()
		require.Error(t, err)
	})
}
