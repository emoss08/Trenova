package invoicepostingservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/billingtest/invoiceservicetest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func factoringService(
	t *testing.T,
	cus *customer.Customer,
	factors *invoiceservicetest.FakeFactors,
) *Service {
	t.Helper()
	customers := customerportsmock.NewMockCustomerRepository(t)
	customers.EXPECT().GetByID(mock.Anything, mock.Anything).RunAndReturn(
		func(_ context.Context, req customerports.GetCustomerByIDRequest) (*customer.Customer, error) {
			assert.True(t, req.IncludeBillingProfile)
			return cus, nil
		}).
		Maybe()
	return &Service{
		l:             zap.NewNop(),
		customerRepo:  customers,
		factoringRepo: factors,
		rec: invoiceaction.NewRecorder(
			invoiceaction.RecorderParams{Logger: zap.NewNop()},
		),
	}
}

func TestAssignFactoringRecordsTheCustomersFactor(t *testing.T) {
	t.Parallel()

	factor := &factoringcompany.FactoringCompany{
		ID:     pulid.MustNew("fac_"),
		Name:   "RTS Financial",
		Status: domaintypes.StatusActive,
	}
	cus := invoiceservicetest.FactoredCustomer(&factor.ID)
	svc := factoringService(
		t,
		cus,
		&invoiceservicetest.FakeFactors{ByID: map[pulid.ID]*factoringcompany.FactoringCompany{
			factor.ID: factor,
		}},
	)
	entity := &invoice.Invoice{CustomerID: cus.ID}

	require.NoError(t, svc.assignFactoring(t.Context(), entity, pagination.TenantInfo{}))

	require.NotNil(t, entity.FactoringCompanyID)
	assert.Equal(t, factor.ID, *entity.FactoringCompanyID)
}

func TestAssignFactoringClearsItForACustomerNoLongerFactored(t *testing.T) {
	t.Parallel()

	stale := pulid.MustNew("fac_")
	cus := invoiceservicetest.FactoredCustomer(nil)
	svc := factoringService(t, cus, &invoiceservicetest.FakeFactors{})
	entity := &invoice.Invoice{CustomerID: cus.ID, FactoringCompanyID: &stale}

	require.NoError(t, svc.assignFactoring(t.Context(), entity, pagination.TenantInfo{}))

	assert.Nil(t, entity.FactoringCompanyID)
}

func TestAssignFactoringRefusesAnInactiveFactor(t *testing.T) {
	t.Parallel()

	factor := &factoringcompany.FactoringCompany{
		ID:     pulid.MustNew("fac_"),
		Name:   "RTS Financial",
		Status: domaintypes.StatusInactive,
	}
	cus := invoiceservicetest.FactoredCustomer(&factor.ID)
	svc := factoringService(
		t,
		cus,
		&invoiceservicetest.FakeFactors{ByID: map[pulid.ID]*factoringcompany.FactoringCompany{
			factor.ID: factor,
		}},
	)
	entity := &invoice.Invoice{CustomerID: cus.ID}

	err := svc.assignFactoring(t.Context(), entity, pagination.TenantInfo{})

	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Nil(t, entity.FactoringCompanyID)
}
