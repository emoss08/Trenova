package benchrun

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/deskbench"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranscriptTools_PairsCallsWithTheirResults(t *testing.T) {
	t.Parallel()

	transcript := &agent.RunTranscript{Messages: []agent.TranscriptMessage{
		{Role: "assistant", ToolCalls: []agent.TranscriptToolCall{
			{ID: "c1", Name: "billing_get_queue_item", Arguments: map[string]any{"id": "bqi_1"}},
			{ID: "c2", Name: "shipment_place_hold"},
		}},
		{Role: "tool", ToolCallID: "c1", ToolName: "billing_get_queue_item", ToolVerdict: "ran"},
		{Role: "tool", ToolCallID: "c2", ToolName: "shipment_place_hold", ToolVerdict: "proposed"},
	}}

	tools := transcriptTools(transcript)
	require.Len(t, tools, 2)
	assert.Equal(t, "ran", tools[0].Verdict)
	assert.True(t, tools[0].Finished)
	assert.True(t, tools[1].Proposed)
}

type runRecordsFake struct {
	agentruntimeports.AgentRunRepository

	run        *agent.AgentRun
	transcript *agent.RunTranscript
}

func (f *runRecordsFake) GetByID(
	_ context.Context,
	req agentruntimeports.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	if req.ID != f.run.ID {
		return nil, assert.AnError
	}
	run := *f.run

	return &run, nil
}

func (f *runRecordsFake) ListTranscriptsByIDs(
	_ context.Context,
	req *agentruntimeports.ListAgentRunsByIDsRequest,
) ([]*agent.AgentRun, error) {
	runs := make([]*agent.AgentRun, 0, len(req.IDs))
	for _, id := range req.IDs {
		if id == f.run.ID {
			runs = append(runs, &agent.AgentRun{ID: id, Transcript: f.transcript})
		}
	}

	return runs, nil
}

func TestAwaitLoop_EndsOnTheStatusTheRunRecords(t *testing.T) {
	t.Parallel()

	runID := pulid.MustNew("arun_")
	records := &runRecordsFake{
		run: &agent.AgentRun{
			ID:      runID,
			Status:  agent.RunStatusAwaitingDecision,
			Summary: "Filed an exception for a person.",
		},
		transcript: &agent.RunTranscript{
			Messages: []agent.TranscriptMessage{{Role: "Assistant", Content: "Done."}},
		},
	}
	runner := &Runner{runRecords: records}
	session := &deskbench.Session{
		Tenant: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	}
	fired := &firedRun{run: &agent.AgentRun{ID: runID}}

	ctx, cancel := context.WithTimeout(t.Context(), runPollInterval/2)
	defer cancel()

	run, err := runner.awaitLoop(ctx, session, fired)
	require.NoError(t, err, "a run already awaiting a decision ends the wait on the first read")
	assert.Equal(t, agent.RunStatusAwaitingDecision, run.Status)
	assert.Equal(t, "Filed an exception for a person.", run.Summary)

	transcript, err := runner.loadTranscript(t.Context(), session, runID)
	require.NoError(t, err)
	assert.Equal(t, records.transcript, transcript)
}
