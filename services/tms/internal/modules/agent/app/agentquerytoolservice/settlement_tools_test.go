package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCarrierSettlements struct {
	settlements []*carriersettlement.CarrierSettlement
	matches     []*carriersettlement.InvoiceMatch
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
