package services_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/enttest"
	"github.com/emoss08/trenova/services"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

type MockLogger struct {
	mock.Mock
	*logrus.Entry
}

func TestAccessorialChargeOps_GetAccessorialCharges(t *testing.T) {
	mockLogger := &MockLogger{
		Entry: logrus.NewEntry(logrus.StandardLogger()), // Initialize the embedded Entry
	}

	// Create an in-memory database for testing
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1",
		enttest.WithOptions(ent.Log(t.Log)),
	)
	defer client.Close()
	ctx := context.Background()
	// Run the auto migration tool.
	if err := client.Schema.Create(ctx); err != nil {
		t.Fatalf("failed creating schema resources: %v", err)
	}

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

	// Create accessorial charge.
	charge, err := client.AccessorialCharge.Create().
		SetOrganizationID(organization.ID).
		SetBusinessUnitID(bu.ID).
		SetStatus("A").
		SetCode("TEST").
		SetMethod("Distance").
		Save(context.Background())

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, charge)

	// Create an instance of AccessorialChargeOps with the mock client and logger.
	ops := &services.AccessorialChargeOps{
		Client: client,
		Logger: mockLogger.Logger,
	}

	// Call the GetAccessorialCharges method.
	charges, count, err := ops.GetAccessorialCharges(
		context.Background(), 10, 0, organization.ID, bu.ID,
	)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, charges)
	assert.Equal(t, 1, count)
}
