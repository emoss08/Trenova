package carriersettlementresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/settlementloaders"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type matchesByMove struct {
	carriersettlementports.CarrierInvoiceMatchRepository

	matches  map[pulid.ID][]*carriersettlement.InvoiceMatch
	requests []carriersettlementports.ListLiveCarrierInvoiceMatchesByMoveIDsRequest
}

func (m *matchesByMove) ListLiveByMoveIDs(
	_ context.Context,
	req carriersettlementports.ListLiveCarrierInvoiceMatchesByMoveIDsRequest,
) (map[pulid.ID][]*carriersettlement.InvoiceMatch, error) {
	m.requests = append(m.requests, req)
	return m.matches, nil
}

func ref(id pulid.ID) *pulid.ID {
	return &id
}

func TestCostSummaryIsNullWhenTheLinesWereNotRead(t *testing.T) {
	t.Parallel()

	resolver := &CarrierSettlementResolver{&Deps{}}
	unread, err := resolver.CostSummary(t.Context(), &carriersettlement.CarrierSettlement{})
	require.NoError(t, err)
	assert.Nil(t, unread)

	invoices, err := resolver.MoveInvoices(t.Context(), &carriersettlement.CarrierSettlement{})
	require.NoError(t, err)
	assert.Nil(t, invoices)

	groups, err := resolver.CostSummary(t.Context(), &carriersettlement.CarrierSettlement{
		Lines: []*carriersettlement.CarrierSettlementLine{
			{
				EventType:   carriersettlement.CostEventTypeFuelSurcharge,
				Description: "Fuel surcharge",
				AmountMinor: 100,
			},
			{
				EventType:   carriersettlement.CostEventTypeFuelSurcharge,
				Description: "Fuel surcharge",
				AmountMinor: 250,
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, int64(350), groups[0].AmountMinor)
}

func TestMoveInvoicesReadsEveryMovesInvoicesInOneBatch(t *testing.T) {
	t.Parallel()

	invoiced := pulid.MustNew("smv_")
	waiting := pulid.MustNew("smv_")
	match := &carriersettlement.InvoiceMatch{
		ID:            pulid.MustNew("cim_"),
		InvoiceNumber: "88412",
		Status:        carriersettlement.InvoiceMatchStatusVariance,
		VarianceMinor: 15000,
	}
	repo := &matchesByMove{matches: map[pulid.ID][]*carriersettlement.InvoiceMatch{
		invoiced: {match},
	}}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	ctx := settlementloaders.NewCarrierInvoiceMatchesByMoveIDLoaderFactory(
		settlementloaders.CarrierInvoiceMatchesByMoveIDLoaderFactoryParams{
			CarrierInvoiceMatchRepo: repo,
		},
	).Attach(t.Context(), tenant)

	invoices, err := (&CarrierSettlementResolver{&Deps{}}).MoveInvoices(
		ctx,
		&carriersettlement.CarrierSettlement{Lines: []*carriersettlement.CarrierSettlementLine{
			{MoveID: ref(invoiced), ProNumber: "S-240177", AmountMinor: 230000},
			{MoveID: ref(waiting), ProNumber: "S-240190", AmountMinor: 218000},
			{AmountMinor: -35000},
		}},
	)
	require.NoError(t, err)
	require.Len(t, repo.requests, 1)
	assert.ElementsMatch(t, []pulid.ID{invoiced, waiting}, repo.requests[0].MoveIDs)
	assert.Equal(t, tenant, repo.requests[0].TenantInfo)
	require.Len(t, invoices, 2)
	assert.Same(t, match, invoices[0].Match)
	assert.Equal(t, "S-240177", invoices[0].ProNumber)
	assert.Nil(t, invoices[1].Match)
	assert.Equal(t, int64(218000), invoices[1].CostMinor)
}
