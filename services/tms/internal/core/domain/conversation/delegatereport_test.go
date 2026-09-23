package conversation

import (
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDelegateReport_BoundedKeepsTheRowSmall(t *testing.T) {
	t.Parallel()

	made := make([]DelegateWrite, 0, maxReportWrites+5)
	for range maxReportWrites + 5 {
		made = append(made, DelegateWrite{
			ToolName: "create_report",
			CallID:   "call_1",
			Tier:     agent.TierAutoExecute,
			Summary:  "On-time\nthis month " + strings.Repeat("s", 1000),
			Result: &agent.ToolExecutionResult{
				Action: "created",
				Kind:   "report",
				Name:   strings.Repeat("n", 1000),
				IDs:    map[string]string{"definitionId": "rd_1"},
				Record: &agent.RecordRef{EntityType: "report", ID: "rd_1"},
			},
			Error: strings.Repeat("e", 2000),
		})
	}
	published := make([]DelegateDocument, 0, maxReportDocuments+2)
	for range maxReportDocuments + 2 {
		published = append(published, DelegateDocument{
			ID:    pulid.MustNew("aart_"),
			Kind:  "document",
			Title: strings.Repeat("t", 1000),
		})
	}

	report := &DelegateReport{
		DelegateCallID: "call_1",
		AgentID:        pulid.MustNew("agdef_"),
		AgentName:      "Report\nBuilder",
		Icon:           "receipt",
		Accent:         "teal",
		Status:         DelegateStatusCompleted,
		Reply:          strings.Repeat("r", 10_000),
		Reason:         strings.Repeat("why ", 1000),
		Made:           made,
		Awaiting:       []DelegateWrite{{ToolName: "share_report", Tier: agent.TierPropose}},
		Published:      published,
		ToolCallsUsed:  26,
	}

	bounded := report.Bounded()

	require.NotNil(t, bounded)
	assert.Equal(t, "Report Builder", bounded.AgentName, "one line")
	assert.Equal(t, "receipt", bounded.Icon)
	assert.Equal(t, "teal", bounded.Accent)
	assert.Equal(t, DelegateStatusCompleted, bounded.Status)
	assert.Equal(t, 26, bounded.ToolCallsUsed)

	assert.True(t, strings.HasSuffix(bounded.Reply, "…"), "a cut answer says it was cut")
	assert.Len(t, []rune(bounded.Reply), maxReportReplyRunes+1)
	assert.True(t, strings.HasSuffix(bounded.Reason, "…"))

	assert.Len(t, bounded.Made, maxReportWrites)
	assert.Equal(t, 5, bounded.MoreMade, "what was left out is counted")
	assert.Len(t, bounded.Awaiting, 1)
	assert.Zero(t, bounded.MoreAwaiting)
	assert.Len(t, bounded.Published, maxReportDocuments)
	assert.Equal(t, 2, bounded.MorePublished)

	write := bounded.Made[0]
	assert.True(t, strings.HasPrefix(write.Summary, "On-time this month s"), "one line")
	assert.LessOrEqual(t, len([]rune(write.Summary)), maxWriteSummaryRunes+1)
	assert.LessOrEqual(t, len([]rune(write.Error)), maxWriteErrorRunes+1)
	require.NotNil(t, write.Result)
	assert.Less(t, len([]rune(write.Result.Name)), 1000, "the result is bounded as a proposal's")
	assert.Equal(t, &agent.RecordRef{EntityType: "report", ID: "rd_1"}, write.Result.Record)
	assert.LessOrEqual(t, len([]rune(bounded.Published[0].Title)), maxReportNameRunes)

	encoded, err := sonic.Marshal(bounded)
	require.NoError(t, err)
	assert.Less(t, len(encoded), 64*1024, "the saved account stays small")

	assert.Len(t, report.Made, maxReportWrites+5, "the account it was cut from is left alone")
}

func TestDelegateReport_BoundedKeepsTheShapeOfTheLiveAccount(t *testing.T) {
	t.Parallel()

	var missing *DelegateReport
	assert.Nil(t, missing.Bounded())

	bounded := (&DelegateReport{
		DelegateCallID: "call_1",
		Status:         DelegateStatusDeclined,
		Reason:         "Report Builder is disabled.",
	}).Bounded()

	require.NotNil(t, bounded)
	assert.NotNil(t, bounded.Made, "an empty list is sent as a list")
	assert.NotNil(t, bounded.Awaiting)
	assert.NotNil(t, bounded.Published)
	assert.Equal(t, "Report Builder is disabled.", bounded.Reason)
	assert.Empty(t, bounded.Reply)

	encoded, err := sonic.MarshalString(bounded)
	require.NoError(t, err)
	assert.Contains(t, encoded, `"made":[]`)
	assert.NotContains(t, encoded, "moreMade", "nothing was left out")
}
