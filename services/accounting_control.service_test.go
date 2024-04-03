package services_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/enttest"
	"github.com/emoss08/trenova/services"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountingControlOps_GetAccountingControl(t *testing.T) {
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

	// Create accounting control.
	_, err = client.AccountingControl.Create().
		SetOrganization(organization).
		SetBusinessUnit(bu).
		Save(context.Background())

	require.NoError(t, err)

	// Create an instance of AccountingControlOps with the mock client and logger.
	ops := &services.AccountingControlOps{
		Client: client,
	}

	// Call the GetAccountingControl method.
	accountingControl, err := ops.GetAccountingControl(context.Background(), organization.ID, bu.ID)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, accountingControl)
}

func TestAccountingControlOps_UpdateAccountingControl(t *testing.T) {
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

	// Create accounting control.
	ac, err := client.AccountingControl.Create().
		SetOrganization(organization).
		SetBusinessUnit(bu).
		Save(context.Background())

	require.NoError(t, err)

	// Create an instance of AccountingControlOps with the mock client and logger.
	ops := &services.AccountingControlOps{
		Client: client,
	}

	// Call the UpdateAccountingControl method.
	ac.RecThreshold = 10
	ac.RecThresholdAction = "Halt"
	ac.AutoCreateJournalEntries = true
	ac.JournalEntryCriteria = "OnShipmentBill"
	ac.RestrictManualJournalEntries = true
	ac.RequireJournalEntryApproval = true
	ac.EnableRecNotifications = true
	ac.HaltOnPendingRec = true

	updatedAC, err := ops.UpdateAccountingControl(context.Background(), *ac)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, updatedAC)
}
