package tendertools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenderToRoutingGuidePreview_ShowsTheOffersInOrder(t *testing.T) {
	t.Parallel()

	tenders := &agenttoolstest.FakeTenders{}
	tool := NewTenderToRoutingGuideTool(tenders)
	params := agenttoolstest.ExecuteParams(
		map[string]any{"shipmentMoveId": pulid.MustNew("smv_").String()},
	)
	params.IdempotencyKey = "idem-1"

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, tenders.Waterfall, "a preview must not start a tender")
	assert.Contains(t, preview.Summary, "Dallas outbound")
	assert.Contains(t, preview.Summary, "Lapsed Freight (insurance lapsed)")

	created := agenttoolstest.FindChange(t, preview, agent.PreviewOperationCreate)
	assert.Equal(t, permission.ResourceTender, created.Resource)
	require.NotNil(t, created.Money)
	require.Len(t, created.Money.Lines, 2)
	assert.Equal(t, "1. Acme Trucking (flat)", created.Money.Lines[0].Label)
	assert.False(t, created.Money.TotalAfter.Valid, "offers are alternatives, not a sum")

	var sends []agent.RecordChange
	for i := range preview.Changes {
		if preview.Changes[i].Operation == agent.PreviewOperationSend {
			sends = append(sends, preview.Changes[i])
		}
	}
	require.Len(t, sends, 2)
	assert.Equal(t, agent.MessageChannelEmail, sends[0].Message.Channel)
	assert.Equal(t, []string{"dispatch@acme.example"}, sends[0].Message.To)
	assert.Equal(t, agent.MessageChannelEDI, sends[1].Message.Channel)
	assert.Contains(t, sends[1].Message.Body, "only if every carrier before it declines")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, tenders.Waterfall)
}

func TestTenderToCarriersPreview_WarnsOfTheRefusalWithoutAnOverride(t *testing.T) {
	t.Parallel()

	tenders := &agenttoolstest.FakeTenders{}
	tool := NewTenderToCarriersTool(tenders)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"shipmentMoveId": pulid.MustNew("smv_").String(),
		"mode":           "SpotBroadcast",
		"lines": []any{
			map[string]any{"carrierId": pulid.MustNew("carr_").String(), "rate": "1450.00"},
		},
	})
	params.IdempotencyKey = "idem-1"

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, tenders.Spot)
	assert.Contains(t, preview.Summary, "all at once")
	assert.Contains(t, preview.Summary, "insurance expires")
	require.Len(t, preview.Warnings, 1)
	assert.Equal(t, agent.PreviewWarningWouldFail, preview.Warnings[0].Code)
}
