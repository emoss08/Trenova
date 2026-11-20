package customerservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/onplan"
	"github.com/emoss08/trenova/internal/modules/identity/app/planfields"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func onPlan(gate platformcatalogports.FeatureGate) *planfields.Keeper {
	registry := permissioncatalog.Default()
	return planfields.New(registry, onplan.New(registry, gate))
}

// storedCustomer is a customer whose billing profile holds credit terms, a
// factor and EDI invoicing, and who asked to be told about their freight.
func storedCustomer(entity *customer.Customer) *customer.Customer {
	stored := *entity
	factor := pulid.MustNew("fc_")
	stored.StatusUpdatePreference = customer.StatusUpdateArrivals
	stored.StatusUpdateRecipients = "ops@acme.test"
	stored.BrokerVettingEnabled = true
	stored.DOTNumber = "1234567"
	profile := customer.NewDefaultBillingProfile(
		entity.OrganizationID, entity.BusinessUnitID, entity.ID,
	)
	profile.CreditLimit = decimal.NewNullDecimal(decimal.NewFromInt(25000))
	profile.CreditStatus = customer.CreditStatusHold
	profile.EnforceCreditLimit = true
	profile.ApplyLateCharges = true
	profile.GracePeriodDays = 10
	profile.FactoringCompanyID = &factor
	profile.EDIInvoiceEnabled = true
	stored.BillingProfile = profile

	return &stored
}

// sentBack is the same customer as a base-only form sends it: every withheld
// field read null and came back as its zero value, and the name changed.
func sentBack(entity *customer.Customer) *customer.Customer {
	sent := *entity
	sent.Name = "Acme Freight"
	sent.StatusUpdatePreference = ""
	sent.StatusUpdateRecipients = ""
	sent.BrokerVettingEnabled = false
	sent.BillingProfile = customer.NewDefaultBillingProfile(
		entity.OrganizationID, entity.BusinessUnitID, entity.ID,
	)
	sent.BillingProfile.SendPaymentReminders = false
	sent.BillingProfile.SendStatements = false
	sent.BillingProfile.CreditStatus = ""

	return &sent
}

func TestPlanUpdate_ABaseOnlyUpdateKeepsTheStoredAddOnSettings(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	deps.svc.planFields = onPlan(plantest.BaseOnly())
	stored := storedCustomer(newTestEntity())
	entity := sentBack(stored)
	deps.formatter.On("Format", mock.Anything, entity).Return(nil)
	deps.repo.EXPECT().GetByID(
		mock.Anything,
		mock.MatchedBy(func(req customerports.GetCustomerByIDRequest) bool {
			return req.IncludeBillingProfile
		}),
	).Return(stored, nil)

	change, err := deps.svc.PlanUpdate(t.Context(), entity)
	require.NoError(t, err)

	after := change.After
	assert.Equal(t, "Acme Freight", after.Name, "the base change is written")
	assert.Equal(t, customer.StatusUpdateArrivals, after.StatusUpdatePreference)
	assert.Equal(t, "ops@acme.test", after.StatusUpdateRecipients)
	assert.True(t, after.BrokerVettingEnabled)
	profile := after.BillingProfile
	assert.True(t, profile.CreditLimit.Decimal.Equal(decimal.NewFromInt(25000)))
	assert.Equal(t, customer.CreditStatusHold, profile.CreditStatus)
	assert.True(t, profile.EnforceCreditLimit)
	assert.True(t, profile.ApplyLateCharges)
	assert.Equal(t, int16(10), profile.GracePeriodDays)
	assert.True(t, profile.SendPaymentReminders)
	assert.True(t, profile.SendStatements)
	assert.Equal(t, stored.BillingProfile.FactoringCompanyID, profile.FactoringCompanyID)
	assert.True(t, profile.EDIInvoiceEnabled)
}

func TestPlanUpdate_WithTheAddOnsTheUpdateWritesTheSentSettings(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	deps.svc.planFields = onPlan(plantest.AllFeatures())
	stored := storedCustomer(newTestEntity())
	entity := sentBack(stored)
	entity.BillingProfile.CreditStatus = customer.CreditStatusActive
	deps.formatter.On("Format", mock.Anything, entity).Return(nil)
	deps.repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(stored, nil)

	change, err := deps.svc.PlanUpdate(t.Context(), entity)
	require.NoError(t, err)

	after := change.After
	assert.NotEqual(t, customer.StatusUpdateArrivals, after.StatusUpdatePreference)
	assert.False(t, after.BrokerVettingEnabled)
	profile := after.BillingProfile
	assert.False(t, profile.CreditLimit.Valid)
	assert.Equal(t, customer.CreditStatusActive, profile.CreditStatus)
	assert.False(t, profile.SendPaymentReminders)
	assert.Nil(t, profile.FactoringCompanyID)
	assert.False(t, profile.EDIInvoiceEnabled)
}

func TestPlanCreate_ABaseOnlyCreateTakesTheBillingProfileDefaults(t *testing.T) {
	t.Parallel()

	deps := setupTest(t)
	deps.svc.planFields = onPlan(plantest.BaseOnly())
	entity := newCreateEntity()
	factor := pulid.MustNew("fc_")
	entity.StatusUpdatePreference = customer.StatusUpdateArrivals
	entity.BillingProfile = customer.NewDefaultBillingProfile(
		entity.OrganizationID, entity.BusinessUnitID, entity.ID,
	)
	entity.BillingProfile.CreditStatus = customer.CreditStatusHold
	entity.BillingProfile.FactoringCompanyID = &factor
	entity.BillingProfile.SendStatements = false
	deps.formatter.On("Format", mock.Anything, entity).Return(nil)

	planned, err := deps.svc.PlanCreate(t.Context(), entity)
	require.NoError(t, err)

	assert.NotEqual(t, customer.StatusUpdateArrivals, planned.StatusUpdatePreference)
	assert.Equal(t, customer.CreditStatusActive, planned.BillingProfile.CreditStatus)
	assert.Nil(t, planned.BillingProfile.FactoringCompanyID)
	assert.True(t, planned.BillingProfile.SendStatements, "a new profile's default")
}
