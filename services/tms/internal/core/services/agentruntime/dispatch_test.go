package agentruntime

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type echoPreviews struct {
	serviceports.ProposalPreviewService

	preview *agent.ToolPreview
}

func (p echoPreviews) Baseline(
	context.Context,
	*serviceports.ProposalBaselineRequest,
) *serviceports.ProposalBaselineResult {
	return &serviceports.ProposalBaselineResult{Preview: p.preview}
}

func filedContent(
	t *testing.T,
	previews serviceports.ProposalPreviewService,
	schema map[string]any,
	args map[string]any,
) string {
	t.Helper()

	tool := &agentruntimetest.StubActionTool{
		ToolName: "create_shipment",
		Tier:     agent.TierPropose,
		Schema:   schema,
	}
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("create_shipment", args),
		textTurn("I have proposed the shipment."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{}, &stubActionRegistry{
		Tools: []serviceports.AgentTool{tool},
	}, nil)
	if previews != nil {
		rt.previews = previews
	}

	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition("create_shipment"),
		Actor:      testActor(),
		Input:      "copy PRO-100 for Tuesday",
	})
	require.NoError(t, err)
	require.Len(t, result.Actions, 1)

	for _, message := range result.Messages {
		if message.ToolName == "create_shipment" {
			return message.Content
		}
	}
	require.FailNow(t, "the filing left no tool result")

	return ""
}

func shipmentSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"customerId": map[string]any{"type": "string"},
			"commodities": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "object"},
			},
			"notes": map[string]any{"type": "string"},
		},
		"required": []string{"customerId"},
	}
}

func TestDispatch_FilingEchoesThePreviewTheCardShows(t *testing.T) {
	t.Parallel()

	content := filedContent(t, echoPreviews{preview: &agent.ToolPreview{
		Summary: "Creates a shipment for Acme from Dallas to Tulsa, rated at $1,250.00.",
		Warnings: []agent.PreviewWarning{{
			Code:    agent.PreviewWarningCode("rate_coverage"),
			Message: "No rate agreement covers this lane; the formula template rates it.",
		}},
	}}, shipmentSchema(), map[string]any{"customerId": "cus_1"})

	assert.Contains(t, content, "Recorded a proposal to run \"create_shipment\"")
	assert.Contains(t, content,
		"Creates a shipment for Acme from Dallas to Tulsa, rated at $1,250.00.")
	assert.Contains(t, content,
		"No rate agreement covers this lane; the formula template rates it.")
	assert.Contains(t, content, "only what")
	assert.NotContains(t, content, "\"customerId\"",
		"with a preview the arguments are not repeated")
}

func TestDispatch_FilingWithoutAPreviewEchoesTheFiledArguments(t *testing.T) {
	t.Parallel()

	content := filedContent(t, nil, shipmentSchema(), map[string]any{
		"notes":      "Tuesday copy",
		"customerId": "cus_1",
	})

	assert.Contains(t, content, `{"customerId":"cus_1","notes":"Tuesday copy"}`,
		"the arguments are echoed as canonical JSON, keys sorted")
	assert.NotContains(t, content, "commodities",
		"what was not filed is not named")
}

func TestDispatch_FilingEchoIsBoundedToTwoKiB(t *testing.T) {
	t.Parallel()

	commodities := make([]any, 0, 60)
	for range 60 {
		commodities = append(commodities, map[string]any{
			"commodityId": "com_" + strings.Repeat("x", 30),
			"pieces":      12,
		})
	}
	content := filedContent(t, nil, shipmentSchema(), map[string]any{
		"customerId":  "cus_1",
		"notes":       strings.Repeat("n", 3000),
		"commodities": commodities,
	})

	_, echo, found := strings.Cut(content, filedArgumentsLead)
	require.True(t, found, content)
	assert.LessOrEqual(t, len(echo), maxFilingEchoBytes+len(filingEchoCutNote)+16)
	assert.Contains(t, echo, `customerId: "cus_1"`)
	assert.Contains(t, echo, "commodities: 60 items")
	assert.Contains(t, echo, filingEchoCutNote)
}

func TestDispatch_FilingSaysWhichArgumentsWereRenamed(t *testing.T) {
	t.Parallel()

	content := filedContent(t, nil, shipmentSchema(), map[string]any{"customer_id": "cus_1"})

	assert.Contains(t, content, "customer_id was read as customerId")
	assert.Contains(t, content, `{"customerId":"cus_1"}`)
}

func TestDispatch_AFilingWithNoRenameSaysNothingAboutNames(t *testing.T) {
	t.Parallel()

	content := filedContent(t, nil, shipmentSchema(), map[string]any{"customerId": "cus_1"})

	assert.NotContains(t, content, "renamed")
}
