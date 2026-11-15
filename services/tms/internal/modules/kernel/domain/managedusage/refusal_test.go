package managedusage_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/stretchr/testify/assert"
)

func TestDecideAtUsesTheLiveLevel(t *testing.T) {
	t.Parallel()

	storage := managedusage.Meter{
		Meter:     platformcatalog.MeterDocumentStorageBytes,
		Quantity:  1000,
		Allowance: 1000,
	}

	d := managedusage.DecideAt(state(false, storage), storage.Meter, 200, 800)
	assert.True(t, d.Allowed)
	assert.Equal(t, int64(200), d.Used)

	d = managedusage.DecideAt(state(false, storage), storage.Meter, 200, 801)
	assert.False(t, d.Allowed)
	assert.Equal(t, managedusage.ReasonAllowance, d.Reason)

	d = managedusage.DecideAt(state(true, storage), storage.Meter, 200, 5000)
	assert.True(t, d.Allowed, "overage on: uploads continue past the allowance")
}

func TestOrganizationsAreAHardCapEvenWithOverage(t *testing.T) {
	t.Parallel()

	orgs := managedusage.Meter{
		Meter:     platformcatalog.MeterOrganizationsTotal,
		Allowance: 1,
		HardCap:   true,
	}

	assert.True(t, managedusage.DecideAt(state(true, orgs), orgs.Meter, 0, 1).Allowed)
	assert.False(t, managedusage.DecideAt(state(true, orgs), orgs.Meter, 1, 1).Allowed)
}

func TestRefusalWordsAStorageLimitWithBothWaysOut(t *testing.T) {
	t.Parallel()

	err := managedusage.Refusal(platformcatalog.MeterDocumentStorageBytes, 1000, 900, 200, "managed")

	assert.Equal(t, string(platformcatalog.MeterDocumentStorageBytes), err.Meter)
	assert.Contains(t, err.Message, "Turn on overage from your Trenova account")
	assert.Contains(t, err.Message, "delete documents")
	assert.NotContains(t, err.Message, "$")
}

func TestRefusalWordsTheOrganizationLimitWithThePlanAllowance(t *testing.T) {
	t.Parallel()

	err := managedusage.Refusal(platformcatalog.MeterOrganizationsTotal, 5, 5, 1, "managed")

	assert.Equal(t, "organizations.total", err.Meter)
	assert.Equal(t, "Your plan includes 5 organization(s), and this instance has them all. "+
		"Add organizations to your plan from your Trenova account to create another.", err.Error())
}

func TestRefusalKeepsTheGeneralWordingWhenTheAllowanceIsNotWhatStopped(t *testing.T) {
	t.Parallel()

	err := managedusage.Refusal(platformcatalog.MeterDocumentStorageBytes, 1000, 100, 200, "managed")

	assert.Equal(t, "This organization has reached the limit of its plan", err.Message)
}

func TestNearStorageAllowanceAtFourFifths(t *testing.T) {
	t.Parallel()

	assert.False(t, managedusage.NearStorageAllowance(managedusage.Meter{Quantity: 799, Allowance: 1000}))
	assert.True(t, managedusage.NearStorageAllowance(managedusage.Meter{Quantity: 800, Allowance: 1000}))
	assert.False(t, managedusage.NearStorageAllowance(managedusage.Meter{Quantity: 5, Allowance: 0}))
}
