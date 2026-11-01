package customerservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/parties/app/customerservice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGetShipmentBillingRules_ReadsEveryPayerOnceAndCarriesOnlyTheFormRules(t *testing.T) {
	t.Parallel()

	strict := pulid.MustNew("cus_")
	lenient := pulid.MustNew("cus_")
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	repo := customerportsmock.NewMockCustomerRepository(t)
	repo.EXPECT().
		GetShipmentBillingProfiles(mock.Anything, customerports.GetShipmentBillingRulesRequest{
			TenantInfo:  tenant,
			CustomerIDs: []pulid.ID{strict, lenient},
		}).
		Return([]*customer.CustomerBillingProfile{
			{
				CustomerID:                strict,
				EnforceCustomerBillingReq: true,
				RequireBOLNumber:          true,
				CreditStatus:              customer.CreditStatusHold,
				CreditHoldReason:          "Over limit",
				EnforceCreditLimit:        true,
				CreditHoldAutomatic:       true,
			},
			{
				CustomerID:       lenient,
				RequireBOLNumber: true,
				CreditStatus:     customer.CreditStatusActive,
			},
		}, nil).
		Once()

	svc := customerservice.New(customerservice.Params{Logger: zap.NewNop(), Repo: repo})
	rules, err := svc.GetShipmentBillingRules(t.Context(), customerports.GetShipmentBillingRulesRequest{
		TenantInfo:  tenant,
		CustomerIDs: []pulid.ID{strict, lenient, strict},
	})
	require.NoError(t, err)

	assert.Equal(t, []*customerports.ShipmentBillingRules{
		{
			CustomerID:          strict,
			BOLRequired:         true,
			CreditStatus:        customer.CreditStatusHold,
			CreditHoldReason:    "Over limit",
			EnforceCreditLimit:  true,
			CreditHoldAutomatic: true,
		},
		{
			CustomerID:   lenient,
			BOLRequired:  false,
			CreditStatus: customer.CreditStatusActive,
		},
	}, rules)
}

func TestGetShipmentBillingRules_RefusesMoreCustomersThanAShipmentBills(t *testing.T) {
	t.Parallel()

	ids := make([]pulid.ID, customerports.MaxShipmentBillingRulesCustomers+1)
	for i := range ids {
		ids[i] = pulid.MustNew("cus_")
	}
	repo := customerportsmock.NewMockCustomerRepository(t)

	svc := customerservice.New(customerservice.Params{Logger: zap.NewNop(), Repo: repo})
	_, err := svc.GetShipmentBillingRules(t.Context(), customerports.GetShipmentBillingRulesRequest{
		CustomerIDs: ids,
	})

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "customerIds", validation.Field)
	repo.AssertNotCalled(t, "GetShipmentBillingProfiles", mock.Anything, mock.Anything)
}
