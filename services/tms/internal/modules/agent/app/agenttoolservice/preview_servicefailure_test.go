package agenttoolservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveServiceFailurePreview_ShowsTheResolutionAndTheCustomer214(t *testing.T) {
	t.Parallel()

	existing := openFailure()
	failures := &agenttoolstest.FakeFailureDecider{Existing: existing}
	tool := operationstools.NewResolveServiceFailureTool(failures)
	params := agenttoolstest.ExecuteParams(map[string]any{
		"serviceFailureId": existing.ID.String(),
		"reasonCodeId":     pulid.MustNew("sfrc_").String(),
		"notes":            "Shipper closed early; driver waited until opening.",
	})

	preview, err := tool.(agentports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, failures.Resolved, "a preview must not resolve")
	assert.Contains(t, preview.Summary, "SF-7")
	assert.Contains(t, preview.Summary, "LATE_SHIPPER (Shipper closed early)")

	change := agenttoolstest.FindChange(t, preview, agent.PreviewOperationUpdate)
	assert.Equal(t, permission.ResourceServiceFailure, change.Resource)
	assert.Equal(t, "Resolved", agenttoolstest.FindField(t, change, "status").After)
	reason := agenttoolstest.FindField(t, change, "reasonCodeId")
	require.NotNil(t, reason.AfterRef)
	assert.Equal(t, permission.ResourceServiceFailureReasonCode, reason.AfterRef.Resource)

	send := agenttoolstest.FindChange(t, preview, agent.PreviewOperationSend)
	assert.Equal(t, agent.MessageChannelEDI, send.Message.Channel)
	assert.Contains(t, send.Message.Body, "Shipper closed early; driver waited")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, failures.Resolved)
	assert.Equal(t, failures.Resolved.Notes, agenttoolstest.FindField(t, change, "internalNotes").After)
}
