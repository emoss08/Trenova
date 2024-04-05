package services_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/enttest"
	"github.com/emoss08/trenova/services"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocationOps_GetLocations(t *testing.T) {
	mockLogger := &MockLogger{
		Entry: logrus.NewEntry(logrus.StandardLogger()), // Initialize the embedded Entry
	}

	// Create an in-memory database for testing
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1",
		enttest.WithOptions(ent.Log(t.Log)),
	)
	defer client.Close()

	// Populate the database with test data.
	bu, err := client.BusinessUnit.Create().
		SetName("Trenova Transportation").
		SetEntityKey("TREN").
		SetPhoneNumber("123-456-7890").
		SetAddress("1234 Main St").
		Save(context.Background())

	require.NoError(t, err)

	organization, err := client.Organization.Create().
		SetName("Trenova Transportation").
		SetBusinessUnit(bu).
		SetBusinessUnitID(bu.ID).
		SetScacCode("TREN").
		SetDotNumber("1234567").
		Save(context.Background())

	require.NoError(t, err)

	// Create location.
	_, err = client.Location.Create().
		SetOrganization(organization).
		SetBusinessUnit(bu).
		Save(context.Background())

	require.NoError(t, err)

	// Create an instance of AccountingControlOps with the mock client and logger.
	ops := &services.LocationOps{
		Client: client,
		Logger: mockLogger.Logger,
	}

	// Call the GetAccountingControl method.
	locations, count, err := ops.GetLocations(
		context.Background(), 10, 0, organization.ID, bu.ID)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, locations)
	assert.Equal(t, 1, count)
}
