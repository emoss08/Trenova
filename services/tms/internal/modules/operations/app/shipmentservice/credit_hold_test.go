package shipmentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func heldPayer(id pulid.ID, automatic bool) *customer.Customer {
	return &customer.Customer{
		ID:   id,
		Name: "Acme Logistics",
		BillingProfile: &customer.CustomerBillingProfile{
			EnforceCreditLimit:  true,
			CreditStatus:        customer.CreditStatusHold,
			CreditHoldAutomatic: automatic,
			CreditHoldReason:    "Open receivables of 15000.00 USD are over the credit limit of 10000.00 USD",
		},
	}
}

func newShipmentFor(customerID pulid.ID) *shipment.Shipment {
	return &shipment.Shipment{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		CustomerID:     customerID,
	}
}

func TestRefuseCreditHold_RefusesANewShipmentForAHeldPayer(t *testing.T) {
	t.Parallel()

	payerID := pulid.MustNew("cus_")
	repo := customerportsmock.NewMockCustomerRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.MatchedBy(func(req customerports.GetCustomerByIDRequest) bool {
		return req.ID == payerID && req.IncludeBillingProfile
	})).Return(heldPayer(payerID, true), nil).Once()
	svc := &service{customerRepo: repo}

	err := svc.refuseCreditHold(t.Context(), newShipmentFor(payerID), nil)

	require.Error(t, err)
	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "customerId", fieldErr.Field)
	assert.Equal(t,
		"Acme Logistics is on credit hold: Open receivables of 15000.00 USD are over the credit "+
			"limit of 10000.00 USD. Accounts receivable has to lift the hold before new shipments "+
			"are booked for them.",
		fieldErr.Message)
}

func TestRefuseCreditHold_NamesTheBillToWhenAThirdPartyPays(t *testing.T) {
	t.Parallel()

	billTo := pulid.MustNew("cus_")
	entity := newShipmentFor(pulid.MustNew("cus_"))
	entity.BillToCustomerID = &billTo
	repo := customerportsmock.NewMockCustomerRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.MatchedBy(func(req customerports.GetCustomerByIDRequest) bool {
		return req.ID == billTo
	})).Return(heldPayer(billTo, false), nil).Once()

	err := (&service{customerRepo: repo}).refuseCreditHold(t.Context(), entity, nil)

	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "billToCustomerId", fieldErr.Field)
}

func TestRefuseCreditHold_LetsAnExistingShipmentForTheSamePayerBeWorked(t *testing.T) {
	t.Parallel()

	payerID := pulid.MustNew("cus_")
	repo := customerportsmock.NewMockCustomerRepository(t)

	err := (&service{customerRepo: repo}).refuseCreditHold(
		t.Context(), newShipmentFor(payerID), newShipmentFor(payerID),
	)

	require.NoError(t, err)
	repo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}

func TestRefuseCreditHold_AllowsAPayerInGoodStanding(t *testing.T) {
	t.Parallel()

	payerID := pulid.MustNew("cus_")
	payer := heldPayer(payerID, false)
	payer.BillingProfile.CreditStatus = customer.CreditStatusWarning
	repo := customerportsmock.NewMockCustomerRepository(t)
	repo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(payer, nil).Once()

	require.NoError(t, (&service{customerRepo: repo}).refuseCreditHold(
		t.Context(), newShipmentFor(payerID), nil,
	))
}
