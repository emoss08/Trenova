package ratequoteservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/internal/testutil/mocks/ratequoteportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestQuote_RatesTheRecordsTheShipmentNamesNotTheOnesItCarries(t *testing.T) {
	t.Parallel()

	store := ratingtest.NewRelationStore()
	acme := &customer.Customer{ID: pulid.MustNew("cus_"), Code: "ACME"}
	standard := &servicetype.ServiceType{ID: pulid.MustNew("svt_"), Code: "STD"}
	store.Customers[acme.ID] = acme
	store.ServiceTypes[standard.ID] = standard

	var rated *shipment.Shipment
	engine := ratequoteportsmock.NewMockRateEngine(t)
	engine.EXPECT().
		RateShipment(mock.Anything, mock.AnythingOfType("*ratequoteports.RateShipmentRequest")).
		RunAndReturn(func(
			_ context.Context,
			req *ratequoteports.RateShipmentRequest,
		) (*ratequoteports.RatedShipment, error) {
			rated = req.Shipment
			return &ratequoteports.RatedShipment{}, nil
		}).
		Once()
	billing := tenantportsmock.NewMockBillingControlRepository(t)
	billing.EXPECT().GetByOrgID(mock.Anything, mock.Anything).
		Return(nil, errors.New("not configured")).
		Maybe()

	svc := New(Params{
		Logger:             zap.NewNop(),
		Engine:             engine,
		BillingControlRepo: billing,
		Hydrator:           ratingtest.NewHydrator(t, store),
	})

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	forged := &shipment.Shipment{
		CustomerID:          acme.ID,
		Customer:            &customer.Customer{ID: acme.ID, Code: "VIP"},
		ServiceTypeID:       standard.ID,
		ServiceType:         &servicetype.ServiceType{ID: standard.ID, Code: "EXP"},
		FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromInt(9_999)),
		TotalChargeAmount:   decimal.NewNullDecimal(decimal.NewFromInt(9_999)),
		RateLocked:          true,
	}

	_, err := svc.Quote(t.Context(), &QuoteRequest{TenantInfo: tenantInfo, Shipment: forged})
	require.NoError(t, err)

	require.NotNil(t, rated)
	assert.Equal(t, "ACME", rated.Customer.Code)
	assert.Equal(t, "STD", rated.ServiceType.Code)
	assert.False(t, rated.FreightChargeAmount.Valid)
	assert.False(t, rated.TotalChargeAmount.Valid)
	assert.False(t, rated.RateLocked)
	require.Len(t, store.Tenants("customer"), 1)
	assert.Equal(t, tenantInfo.OrgID, store.Tenants("customer")[0].OrgID)
}
