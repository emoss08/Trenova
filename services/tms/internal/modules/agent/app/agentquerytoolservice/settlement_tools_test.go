package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCarrierSettlements struct {
	settlements []*carriersettlement.CarrierSettlement
	matches     []*carriersettlement.InvoiceMatch
	invoices    []*edi.CarrierInvoice
	suggested   *carrier.Carrier
	listedEDI   *ediports.ListEDICarrierInvoicesRequest
}

func (f *fakeCarrierSettlements) ListUnmatchedEDIInvoices(
	_ context.Context,
	req *ediports.ListEDICarrierInvoicesRequest,
) (*pagination.ListResult[*edi.CarrierInvoice], error) {
	f.listedEDI = req

	return &pagination.ListResult[*edi.CarrierInvoice]{Items: f.invoices}, nil
}

func (f *fakeCarrierSettlements) SuggestCarrierForInvoice(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*carrier.Carrier, error) {
	return f.suggested, nil
}

func (f *fakeCarrierSettlements) ListInvoiceMatches(
	context.Context,
	*carriersettlementports.ListCarrierInvoiceMatchesRequest,
) (*pagination.ListResult[*carriersettlement.InvoiceMatch], error) {
	return &pagination.ListResult[*carriersettlement.InvoiceMatch]{Items: f.matches}, nil
}

func TestListCarrierInvoiceMatches_MarksOnlyInvoicesACarrierSentOverEDI(t *testing.T) {
	t.Parallel()

	ediInvoice := pulid.MustNew("ecinv_")
	tool := newListCarrierInvoiceMatchesTool(&fakeCarrierSettlements{
		matches: []*carriersettlement.InvoiceMatch{
			{ID: pulid.MustNew("cim_"), EDICarrierInvoiceID: &ediInvoice, VarianceMinor: 1500},
			{ID: pulid.MustNew("cim_")},
		},
	}, &fakePermissions{})

	result, err := tool.Query(t.Context(), agentParams(map[string]any{}, ""))
	require.NoError(t, err)

	outcome := result.(*gatedOutcome)
	rows := outcome.Items.([]invoiceMatchRow)
	require.Len(t, rows, 2)
	assert.Equal(t, "EDI", rows[0].Source)
	assert.Equal(t, "Manual", rows[1].Source)
	assert.Empty(t, rows[0].Variance)
	assert.Equal(t, []agent.RecordRef{{
		EntityType: carrierInvoiceEntity,
		ID:         ediInvoice.String(),
	}}, outcome.TaintedRecords())
}

func TestListEDICarrierInvoices_SuggestsACarrierAndMarksTheCarriersText(t *testing.T) {
	t.Parallel()

	linked := pulid.MustNew("car_")
	suggested := &carrier.Carrier{ID: pulid.MustNew("car_"), Name: "Blue Line Freight"}
	fake := &fakeCarrierSettlements{
		suggested: suggested,
		invoices: []*edi.CarrierInvoice{
			{
				ID:                   pulid.MustNew("ecinv_"),
				InvoiceNumber:        "BL-1",
				ReconciliationStatus: edi.CarrierInvoiceReconciliationStatusMappingRequired,
				TotalAmount:          decimal.NewNullDecimal(decimal.RequireFromString("1250")),
			},
			{
				ID:                   pulid.MustNew("ecinv_"),
				InvoiceNumber:        "BL-2",
				CarrierID:            linked,
				ReconciliationStatus: edi.CarrierInvoiceReconciliationStatusUnmatched,
			},
		},
	}
	tool := newListEDICarrierInvoicesTool(fake, &fakePermissions{})

	result, err := tool.Query(t.Context(), agentParams(map[string]any{
		"status": "MappingRequired",
	}, permission.SensitivityRestricted))
	require.NoError(t, err)

	assert.Equal(t, edi.CarrierInvoiceReconciliationStatusMappingRequired,
		fake.listedEDI.ReconciliationStatus)
	outcome := result.(*gatedOutcome)
	rows := outcome.Items.([]ediCarrierInvoiceRow)
	require.Len(t, rows, 2)
	assert.Equal(t, suggested.ID.String(), rows[0].SuggestedCarrierID)
	assert.Equal(t, "1250", rows[0].Total)
	assert.Equal(t, linked.String(), rows[1].CarrierID)
	assert.Empty(t, rows[1].SuggestedCarrierID)
	assert.Len(t, outcome.TaintedRecords(), 2)

	internal, err := tool.Query(t.Context(), agentParams(map[string]any{}, ""))
	require.NoError(t, err)
	assert.Empty(t, internal.(*gatedOutcome).Items.([]ediCarrierInvoiceRow)[0].Total)

	_, err = tool.Query(t.Context(), agentParams(map[string]any{"status": "Paid"}, ""))
	require.Error(t, err)
}
