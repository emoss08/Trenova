package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/pkg/auditctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestWithAuditSource_CreditsEveryWriteToTheRunThatMadeIt(t *testing.T) {
	t.Parallel()

	runID := pulid.MustNew("arun_")
	stepOwner := pulid.MustNew("aplan_")
	thread := pulid.MustNew("athr_")

	tests := []struct {
		name    string
		req     *agentruntimeports.RunRequest
		runID   string
		channel string
	}{
		{
			name:    "a person's conversation",
			req:     &agentruntimeports.RunRequest{RunID: runID, ThreadID: thread},
			runID:   runID.String(),
			channel: auditctx.ChannelAgentConversation,
		},
		{
			name:    "an unattended run with a thread",
			req:     &agentruntimeports.RunRequest{RunID: runID, ThreadID: thread, Unattended: true},
			runID:   runID.String(),
			channel: auditctx.ChannelAgentRun,
		},
		{
			name: "the step owner names the run when set",
			req: &agentruntimeports.RunRequest{
				RunID:     runID,
				StepOwner: agentruntimeports.RunStepOwner{ID: stepOwner},
			},
			runID:   stepOwner.String(),
			channel: auditctx.ChannelAgentRun,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			source := auditctx.From(withAuditSource(t.Context(), tt.req))
			assert.Equal(t, tt.runID, source.AIRunID)
			assert.Equal(t, tt.channel, source.Channel)
		})
	}
}

func TestWithAuditSource_LeavesTheContextAloneWithoutARun(t *testing.T) {
	t.Parallel()

	assert.Equal(t, auditctx.Source{}, auditctx.From(withAuditSource(t.Context(), nil)))
	assert.Equal(
		t,
		auditctx.Source{},
		auditctx.From(withAuditSource(t.Context(), &agentruntimeports.RunRequest{})),
	)
}
