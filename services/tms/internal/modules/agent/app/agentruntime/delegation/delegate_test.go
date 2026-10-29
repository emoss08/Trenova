package delegation

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDelegateReport_HandsBackWhatTheLookupsReadWordForWord(t *testing.T) {
	t.Parallel()

	delegate := runtimetest.ReportBuilder()
	read := `{"invoices":[{"number":"INV-104","daysOverdue":12}]}`
	run := Run{
		Definition: &agentdefinition.Definition{ID: delegate.ID, Name: delegate.Name},
		Result: &agentruntimeports.RunResult{
			Reply: "One invoice is overdue.",
			Messages: []conversation.Message{
				{Role: conversation.RoleTool, ToolCallID: "call_d1", ToolName: "invoice_list",
					Content: toolresult.Fence("invoice_list", read)},
				{Role: conversation.RoleTool, ToolCallID: "call_d2", ToolName: "invoice_list",
					Content: "Tool failed.", ToolFailed: true},
				{
					Role:       conversation.RoleTool,
					ToolCallID: "call_d3",
					ToolName:   runtimetools.FindToolsName,
					Content: toolresult.Fence(
						runtimetools.FindToolsName,
						"These tools are now callable",
					),
				},
			},
		},
	}

	report := Report(&delegate, "call_1", &run)

	require.Len(t, report.Findings, 1)
	assert.Equal(t, "invoice_list", report.Findings[0].ToolName)
	assert.Equal(t, "call_d1", report.Findings[0].CallID)
	assert.Equal(t, read, report.Findings[0].Content)
	assert.Equal(t, "One invoice is overdue.", report.Reply)

	outcome := Outcome(&report, true)
	assert.Contains(t, outcome.Content, "INV-104")
	assert.Contains(t, outcome.Content, "changed nothing")
}

func TestDelegateOutcome_KeepsTheReportWholeWithinOneToolResult(t *testing.T) {
	t.Parallel()

	report := conversation.DelegateReport{
		AgentName: "Billing helper",
		Status:    assistantstreamports.DelegateStatusCompleted,
		Reply:     strings.Repeat("r", 1900),
	}
	for idx := range 6 {
		report.Findings = append(report.Findings, conversation.DelegateFinding{
			ToolName: "invoice_list",
			CallID:   fmt.Sprintf("call_%d", idx),
			Content:  `{"rows":["` + strings.Repeat("\"", 900) + `"]}`,
		})
	}

	outcome := Outcome(&report, true)

	_, payload, fenced := toolresult.Unfence(
		outcome.Content[:strings.Index(outcome.Content, "\n\n[")],
	)
	require.True(t, fenced)
	assert.LessOrEqual(t, len(payload), toolresult.MaxBytes)
	assert.True(t, sonic.ValidString(payload), "the report is never cut mid-document")
	assert.NotContains(t, payload, "was cut off")
	assert.Positive(t, report.MoreFindings, "every finding left out is counted")
	assert.Equal(t, 6, len(report.Findings)+report.MoreFindings)
	assert.Contains(t, outcome.Content, "were left out of findings")
}

func TestDelegateReport_CountsFindingsPastWhatOneAnswerCarries(t *testing.T) {
	t.Parallel()

	delegate := runtimetest.ReportBuilder()
	messages := make([]conversation.Message, 0, 8)
	for idx := range 8 {
		messages = append(messages, conversation.Message{
			Role:       conversation.RoleTool,
			ToolCallID: fmt.Sprintf("call_%d", idx),
			ToolName:   "invoice_list",
			Content:    toolresult.Fence("invoice_list", strings.Repeat("x", 3<<10)),
		})
	}
	run := Run{Result: &agentruntimeports.RunResult{Messages: messages}}

	report := Report(&delegate, "call_1", &run)

	assert.Len(t, report.Findings, 3)
	assert.Equal(t, 5, report.MoreFindings)
	assert.Contains(t, Note(&report, true), "were left out of findings")
	assert.LessOrEqual(t, len(report.Bounded().Findings[0].Content), 2000)
}
