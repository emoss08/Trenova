package invoicerunservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func runFixture() *invoicerun.InvoiceRun {
	return &invoicerun.InvoiceRun{
		ID:             pulid.MustNew("invrun_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		PeriodStart:    1_700_000_000,
		PeriodEnd:      1_702_592_000,
	}
}

func buildCandidate(
	customerID pulid.ID,
	pro string,
	amount string,
	mutate func(*repositories.ConsolidationCandidate),
) *repositories.ConsolidationCandidate {
	c := &repositories.ConsolidationCandidate{
		BillingQueueItemID: pulid.MustNew("bqi_"),
		ShipmentID:         pulid.MustNew("shp_"),
		CustomerID:         customerID,
		CustomerName:       "Acme Foods",
		ProNumber:          pro,
		CurrencyCode:       "USD",
		SplitBy:            customer.InvoiceSplitKeyCustomer,
		TotalChargeAmount:  decimal.NewNullDecimal(decimal.RequireFromString(amount)),
	}
	if mutate != nil {
		mutate(c)
	}

	return c
}

func serviceWithCandidates(
	t *testing.T,
	candidates []*repositories.ConsolidationCandidate,
) *Service {
	t.Helper()

	repo := mocks.NewMockBillingQueueRepository(t)
	repo.EXPECT().
		ListConsolidationCandidates(mock.Anything, mock.Anything).
		Return(candidates, nil)

	return &Service{l: zap.NewNop(), billingQueueRepo: repo}
}

func TestBuildGroupsOneGroupPerCustomer(t *testing.T) {
	t.Parallel()

	custA := pulid.MustNew("cus_")
	custB := pulid.MustNew("cus_")

	svc := serviceWithCandidates(t, []*repositories.ConsolidationCandidate{
		buildCandidate(custA, "PRO-1", "100.00", nil),
		buildCandidate(custB, "PRO-2", "250.00", nil),
		buildCandidate(custA, "PRO-3", "75.50", nil),
	})

	groups, err := svc.buildGroups(t.Context(), runFixture(), []pulid.ID{custA, custB})
	require.NoError(t, err)
	require.Len(t, groups, 2)

	// An invoice never spans customers.
	assert.Equal(t, custA, groups[0].CustomerID)
	assert.Equal(t, 2, groups[0].ItemCount)
	assert.True(t, groups[0].TotalAmount.Equal(decimal.RequireFromString("175.50")))

	assert.Equal(t, custB, groups[1].CustomerID)
	assert.Equal(t, 1, groups[1].ItemCount)
}

func TestBuildGroupsSplitsByPONumber(t *testing.T) {
	t.Parallel()

	cust := pulid.MustNew("cus_")
	withPO := func(pro, po string) *repositories.ConsolidationCandidate {
		return buildCandidate(cust, pro, "100.00", func(c *repositories.ConsolidationCandidate) {
			c.SplitBy = customer.InvoiceSplitKeyCustomerAndPONumber
			c.OrderPONumber = po
		})
	}

	svc := serviceWithCandidates(t, []*repositories.ConsolidationCandidate{
		withPO("PRO-1", "PO-1"),
		withPO("PRO-2", "PO-2"),
		withPO("PRO-3", "PO-1"),
		withPO("PRO-4", ""),
	})

	groups, err := svc.buildGroups(t.Context(), runFixture(), []pulid.ID{cust})
	require.NoError(t, err)
	require.Len(t, groups, 3)

	labels := []string{groups[0].GroupLabel, groups[1].GroupLabel, groups[2].GroupLabel}
	assert.Contains(t, labels, "PO PO-1")
	assert.Contains(t, labels, "PO PO-2")
	// The blank PO is its own invoice, not folded into the first real one.
	assert.Contains(t, labels, "No PO")

	assert.Equal(t, 2, groups[0].ItemCount)
}

func TestBuildGroupsHonoursTheShipmentCap(t *testing.T) {
	t.Parallel()

	cust := pulid.MustNew("cus_")
	candidates := make([]*repositories.ConsolidationCandidate, 0, 5)
	for _, pro := range []string{"PRO-1", "PRO-2", "PRO-3", "PRO-4", "PRO-5"} {
		candidates = append(
			candidates,
			buildCandidate(cust, pro, "10.00", func(c *repositories.ConsolidationCandidate) {
				c.MaxShipmentsPerInvoice = 2
			}),
		)
	}

	svc := serviceWithCandidates(t, candidates)

	groups, err := svc.buildGroups(t.Context(), runFixture(), []pulid.ID{cust})
	require.NoError(t, err)
	require.Len(t, groups, 3)

	assert.Equal(t, 2, groups[0].ItemCount)
	assert.Equal(t, 2, groups[1].ItemCount)
	assert.Equal(t, 1, groups[2].ItemCount)

	// The parts are distinguishable to an operator and unique to the index.
	assert.Equal(t, "Acme Foods (1 of 3)", groups[0].GroupLabel)
	assert.NotEqual(t, groups[0].GroupKey, groups[1].GroupKey)
}

func TestBuildGroupsCarriesShipmentIdentityOntoItems(t *testing.T) {
	t.Parallel()

	cust := pulid.MustNew("cus_")
	serviceDate := int64(1_700_500_000)
	orderID := pulid.MustNew("ord_")

	svc := serviceWithCandidates(t, []*repositories.ConsolidationCandidate{
		buildCandidate(cust, "PRO-1", "100.00", func(c *repositories.ConsolidationCandidate) {
			c.ShipmentBOL = "BOL-1"
			c.OrderPONumber = "PO-9"
			c.OrderID = orderID
			c.ServiceDate = &serviceDate
		}),
	})

	groups, err := svc.buildGroups(t.Context(), runFixture(), []pulid.ID{cust})
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Len(t, groups[0].Items, 1)

	item := groups[0].Items[0]
	assert.Equal(t, "PRO-1", item.ProNumber)
	assert.Equal(t, "BOL-1", item.BOL)
	assert.Equal(t, "PO-9", item.PONumber)
	assert.Equal(t, orderID, item.OrderID)
	require.NotNil(t, item.ServiceDate)
	assert.Equal(t, serviceDate, *item.ServiceDate)
}

func TestBuildGroupsWithNoCandidates(t *testing.T) {
	t.Parallel()

	svc := serviceWithCandidates(t, nil)

	groups, err := svc.buildGroups(t.Context(), runFixture(), []pulid.ID{pulid.MustNew("cus_")})
	require.NoError(t, err)
	assert.Empty(t, groups)
}
