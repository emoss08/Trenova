package toolsummary_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolsummary"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestFiledSummary_ReadsBackTheSummaryFilingEchoWrote(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("ap_")
	result := "Recorded a proposal to run \"invoice_post\" (proposal " + id.String() + ")." +
		toolsummary.FilingEcho(&agentruntimeports.ProposalBaselineResult{
			Preview: &agentpreview.ToolPreview{Summary: "Would post INV-1 for 10.00 USD."},
		}, nil)

	assert.Equal(t, "Would post INV-1 for 10.00 USD.", toolsummary.FiledSummary(result, id))
	assert.Empty(t, toolsummary.FiledSummary(result, pulid.MustNew("ap_")))
	assert.Empty(t, toolsummary.FiledSummary(
		"Recorded a proposal to run \"x\" (proposal "+id.String()+").", id),
		"a proposal filed without a card has no summary")
}
