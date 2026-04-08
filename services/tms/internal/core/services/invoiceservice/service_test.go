package invoiceservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestResolvePaymentTerm(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		customer *customer.Customer
		control  *tenant.BillingControl
		expected string
	}{
		{
			name: "customer override wins",
			customer: &customer.Customer{
				BillingProfile: &customer.CustomerBillingProfile{
					PaymentTerm: customer.PaymentTermNet15,
				},
			},
			control: &tenant.BillingControl{
				PaymentTerm: tenant.PaymentTermNet60,
			},
			expected: "Net15",
		},
		{
			name: "tenant fallback used when customer term missing",
			customer: &customer.Customer{
				BillingProfile: &customer.CustomerBillingProfile{},
			},
			control: &tenant.BillingControl{
				PaymentTerm: tenant.PaymentTermNet45,
			},
			expected: "Net45",
		},
		{
			name:     "empty when neither source has a term",
			customer: &customer.Customer{},
			control:  nil,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, string(resolvePaymentTerm(tc.customer, tc.control)))
		})
	}
}

func TestBuildInvoiceEntityUsesTenantFallbackAndSignsCreditMemoAmounts(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	item := &billingqueue.BillingQueueItem{
		ID:             pulid.MustNew("bqi_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ShipmentID:     pulid.MustNew("shp_"),
		BillType:       billingqueue.BillTypeCreditMemo,
		Number:         "CM-1001",
	}
	shp := &shipment.Shipment{
		ID:                  item.ShipmentID,
		FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(100)),
		OtherChargeAmount:   decimal.NewNullDecimal(decimal.NewFromInt(35)),
		TotalChargeAmount:   decimal.NewNullDecimal(decimal.NewFromInt(135)),
		ProNumber:           "PRO123",
		BOL:                 "BOL123",
		ActualDeliveryDate:  int64Ptr(1_700_000_500),
		AdditionalCharges: []*shipment.AdditionalCharge{
			{
				Unit:   2,
				Amount: decimal.NewFromInt(20),
			},
			nil,
		},
	}
	cus := &customer.Customer{
		ID:           pulid.MustNew("cus_"),
		Name:         "Acme Logistics",
		Code:         "ACME",
		AddressLine1: "100 Main",
		City:         "Nashville",
		PostalCode:   "37201",
		BillingProfile: &customer.CustomerBillingProfile{
			BillingCurrency: "CAD",
		},
	}
	control := &tenant.BillingControl{
		PaymentTerm: tenant.PaymentTermNet45,
	}

	svc := &Service{l: zap.NewNop()}

	entity := svc.buildInvoiceEntity(item, shp, cus, control)

	require.NotNil(t, entity)
	assert.Equal(t, item.Number, entity.Number)
	assert.Equal(t, "Net45", string(entity.PaymentTerm))
	assert.Equal(t, "CAD", entity.CurrencyCode)
	assert.Equal(t, decimal.NewFromInt(-100), entity.SubtotalAmount)
	assert.Equal(t, decimal.NewFromInt(-35), entity.OtherAmount)
	assert.Equal(t, decimal.NewFromInt(-135), entity.TotalAmount)
	assert.Len(t, entity.Lines, 2)
	assert.Equal(t, decimal.NewFromInt(-100), entity.Lines[0].Amount)
	assert.Equal(t, decimal.NewFromInt(-20), entity.Lines[1].Amount)
	assert.True(t, entity.Lines[1].UnitPrice.Equal(decimal.NewFromInt(-10)))
	assert.Equal(t, shp.ActualDeliveryDate, entity.ServiceDate)
}

func TestShouldAutoPost(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")

	t.Run("tenant auto bill wins", func(t *testing.T) {
		t.Parallel()

		billingRepo := mocks.NewMockBillingControlRepository(t)
		billingRepo.EXPECT().GetByOrgID(t.Context(), orgID).Return(&tenant.BillingControl{
			AutoBill: true,
		}, nil)

		svc := &Service{
			l:           zap.NewNop(),
			billingRepo: billingRepo,
		}

		autoPost := svc.shouldAutoPost(t.Context(), orgID, &customer.Customer{
			BillingProfile: &customer.CustomerBillingProfile{AutoBill: false},
		})

		assert.True(t, autoPost)
	})

	t.Run("customer auto bill applies when tenant is off", func(t *testing.T) {
		t.Parallel()

		billingRepo := mocks.NewMockBillingControlRepository(t)
		billingRepo.EXPECT().GetByOrgID(t.Context(), orgID).Return(&tenant.BillingControl{
			AutoBill: false,
		}, nil)

		svc := &Service{
			l:           zap.NewNop(),
			billingRepo: billingRepo,
		}

		autoPost := svc.shouldAutoPost(t.Context(), orgID, &customer.Customer{
			BillingProfile: &customer.CustomerBillingProfile{AutoBill: true},
		})

		assert.True(t, autoPost)
	})

	t.Run("customer fallback still works on tenant not found", func(t *testing.T) {
		t.Parallel()

		billingRepo := mocks.NewMockBillingControlRepository(t)
		billingRepo.EXPECT().GetByOrgID(t.Context(), orgID).
			Return(nil, errortypes.NewNotFoundError("billing control not found"))

		svc := &Service{
			l:           zap.NewNop(),
			billingRepo: billingRepo,
		}

		autoPost := svc.shouldAutoPost(t.Context(), orgID, &customer.Customer{
			BillingProfile: &customer.CustomerBillingProfile{AutoBill: true},
		})

		assert.True(t, autoPost)
	})
}

func int64Ptr(v int64) *int64 {
	return &v
}
