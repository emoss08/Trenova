package reportquerytools

import (
	"testing"

	"github.com/emoss08/keel/analytics/export"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/turntaint"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunReport_AFreeTextColumnMarksTheCallThroughItsStatusValue(t *testing.T) {
	t.Parallel()

	service, _, tools := reportingTools(t)
	service.Settled = finishedRun(service, 3)
	service.Rows = map[pulid.ID]*export.Envelope{service.Run.ID: storedRows(t, 3)}
	tool := tools["report_run"]

	result, err := tool.Query(t.Context(), querytooltest.TestParams(map[string]any{
		"reportKey":  "ar_aging_by_customer",
		"parameters": map[string]any{"asOf": "2026-03-01"},
	}))
	require.NoError(t, err)

	policy := tool.Policy()
	marks := turntaint.Call(&policy, &llmports.ToolCall{ID: "call-1", Name: tool.Name()}, result, 10)

	require.Len(t, marks, 1)
	assert.Equal(t, agent.TaintSourceRecordNote, marks[0].Source)
	require.NotNil(t, marks[0].Ref)
	assert.Equal(t, agent.TaintEntityReport, marks[0].Ref.EntityType)
	assert.Equal(t, service.Run.ID.String(), marks[0].Ref.ID)
}
