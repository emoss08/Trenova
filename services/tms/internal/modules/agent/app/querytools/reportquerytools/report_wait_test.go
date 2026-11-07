package reportquerytools

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentwait"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordedRunWaits struct {
	asked []*agentwaitports.WaitForReportRunRequest
	err   error
}

func (r *recordedRunWaits) WaitForReportRun(
	_ context.Context,
	req *agentwaitports.WaitForReportRunRequest,
) (*agentwait.Wait, error) {
	r.asked = append(r.asked, req)
	if r.err != nil {
		return nil, r.err
	}

	return &agentwait.Wait{ID: pulid.MustNew(agentwait.IDPrefix)}, nil
}

func runWithWaits(
	t *testing.T,
	waits *recordedRunWaits,
	params *agenttoolports.QueryToolParams,
) reportRunStatus {
	t.Helper()

	_, _, tools := reportingTools(t)
	runner, ok := tools["report_run"].(*runReportTool)
	require.True(t, ok)
	runner.waits = waits

	result, err := runner.Query(t.Context(), params)
	require.NoError(t, err)
	status, ok := result.(reportRunStatus)
	require.True(t, ok)

	return status
}

/*
"Run AR aging and tell me who is past 90 days" ended at "it has started": the
card filled in when the run finished, but the agent never heard, so the
question the person asked was never answered.
*/
func TestRunReport_ARunStillGoingWithANextStepPicksTheConversationUpWhenItFinishes(t *testing.T) {
	t.Parallel()

	waits := &recordedRunWaits{}
	params := querytooltest.TestParams(map[string]any{
		"reportKey":  "ar_aging_by_customer",
		"parameters": map[string]any{"asOf": "2026-03-01"},
		"then":       "name the customers over 90 days and their balances",
	})
	params.ThreadID = pulid.MustNew("athr_")
	params.AgentDefinitionID = pulid.MustNew("agdef_")

	status := runWithWaits(t, waits, params)

	require.Len(t, waits.asked, 1)
	asked := waits.asked[0]
	assert.Equal(t, status.RunID, asked.ReportRunID.String())
	assert.Equal(t, params.ThreadID, asked.ThreadID)
	assert.Equal(t, params.AgentDefinitionID, asked.DefinitionID)
	assert.Equal(t, "AR Aging by Customer", asked.ReportName)
	assert.Equal(t, "name the customers over 90 days and their balances", asked.Then)
	assert.Contains(t, status.Note, "picks itself up again when the run finishes")
}

func TestRunReport_ARunAskedForOnItsOwnSetsNoWait(t *testing.T) {
	t.Parallel()

	waits := &recordedRunWaits{}
	status := runWithWaits(t, waits, querytooltest.TestParams(map[string]any{
		"reportKey":  "ar_aging_by_customer",
		"parameters": map[string]any{"asOf": "2026-03-01"},
	}))

	assert.Empty(t, waits.asked, "the card shows the result; nothing to pick up")
	assert.NotContains(t, status.Note, "picks itself up")
}

func TestRunReport_ADelegatedTaskNeverParksTheConversation(t *testing.T) {
	t.Parallel()

	waits := &recordedRunWaits{}
	params := querytooltest.TestParams(map[string]any{
		"reportKey":  "ar_aging_by_customer",
		"parameters": map[string]any{"asOf": "2026-03-01"},
		"then":       "total it",
	})
	params.Delegated = true

	runWithWaits(t, waits, params)

	assert.Empty(t, waits.asked)
}

func TestRunReport_AWaitThatCannotBeSetStillAnswersAsStarted(t *testing.T) {
	t.Parallel()

	waits := &recordedRunWaits{err: errors.New("There are already 10 open waits here.")}
	status := runWithWaits(t, waits, querytooltest.TestParams(map[string]any{
		"reportKey":  "ar_aging_by_customer",
		"parameters": map[string]any{"asOf": "2026-03-01"},
		"then":       "total it",
	}))

	assert.False(t, status.Finished)
	assert.Contains(t, status.Note, "could not be set to pick itself up")
	assert.Contains(t, status.Note, "already 10 open waits")
}
