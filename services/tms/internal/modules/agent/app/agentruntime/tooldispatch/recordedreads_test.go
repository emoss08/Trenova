package tooldispatch

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentquality"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func recordedRequest() *agentruntimeports.RunRequest {
	return &agentruntimeports.RunRequest{
		Definition: &agentdefinition.Definition{ID: pulid.MustNew("agd_")},
		Actor: &permissionports.RequestActor{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		},
	}
}

func shipmentRecording(t *testing.T, fixtures ...agentquality.ToolFixture) *agentquality.Recording {
	t.Helper()

	recording := (&agentquality.EvalCase{ToolFixtures: fixtures}).Recording()
	require.NotNil(t, recording)

	return recording
}

func liveShipment() *agenttest.StubQueryTool {
	return &agenttest.StubQueryTool{
		ToolName: "shipment_get",
		Result:   map[string]any{"proNumber": "S-100", "status": "Delivered"},
	}
}

func TestRunQueryTool_ARecordedReadIsAnsweredWithWhatTheOriginalWasGiven(t *testing.T) {
	t.Parallel()

	dispatcher := &Dispatcher{logger: zap.NewNop(), labels: &askedLabeler{}}
	tool := liveShipment()
	args := map[string]any{"id": "shp_1"}
	recording := shipmentRecording(t, agentquality.ToolFixture{
		Tool:   "shipment_get",
		Args:   args,
		Result: map[string]any{"proNumber": "S-100", "status": "InTransit"},
	})

	outcome := dispatcher.runQueryToolWith(
		t.Context(),
		recordedRequest(),
		tool,
		&llmports.ToolCall{ID: "call_1", Name: "shipment_get", Arguments: args},
		&queryExtras{recorded: recording, asked: args},
	)

	assert.False(t, outcome.Failed)
	assert.Equal(t, aitrace.OutcomeRan, outcome.Verdict)
	assert.Contains(t, outcome.Content, "InTransit")
	assert.NotContains(t, outcome.Content, "Delivered",
		"the shipment has since been delivered, which the person who decided never saw")
	assert.Zero(t, tool.Calls, "the records are not read for a read that was recorded")
}

func TestRunQueryTool_AReadTheOriginalNeverMadeIsMadeAgainstTheRecords(t *testing.T) {
	t.Parallel()

	dispatcher := &Dispatcher{logger: zap.NewNop(), labels: &askedLabeler{}}
	tool := liveShipment()
	recording := shipmentRecording(t, agentquality.ToolFixture{
		Tool:   "shipment_get",
		Args:   map[string]any{"id": "shp_1"},
		Result: map[string]any{"status": "InTransit"},
	})
	other := map[string]any{"id": "shp_2"}

	outcome := dispatcher.runQueryToolWith(
		t.Context(),
		recordedRequest(),
		tool,
		&llmports.ToolCall{ID: "call_1", Name: "shipment_get", Arguments: other},
		&queryExtras{recorded: recording, asked: other},
	)

	assert.False(t, outcome.Failed)
	assert.Contains(t, outcome.Content, "Delivered")
	assert.Equal(t, 1, tool.Calls)
}

func TestRunQueryTool_ARecordedReadMatchesTheArgumentsTheModelSent(t *testing.T) {
	t.Parallel()

	dispatcher := &Dispatcher{logger: zap.NewNop(), labels: &askedLabeler{}}
	tool := liveShipment()
	sent := map[string]any{"id": "shp_1", "limit": "5"}
	recording := shipmentRecording(t, agentquality.ToolFixture{
		Tool:   "shipment_get",
		Args:   sent,
		Result: map[string]any{"status": "InTransit"},
	})

	outcome := dispatcher.runQueryToolWith(
		t.Context(),
		recordedRequest(),
		tool,
		&llmports.ToolCall{
			ID:        "call_1",
			Name:      "shipment_get",
			Arguments: map[string]any{"id": "shp_1", "limit": float64(5)},
		},
		&queryExtras{recorded: recording, asked: sent},
	)

	assert.Contains(t, outcome.Content, "InTransit",
		"a conversation keeps the call as the model wrote it, before its arguments were corrected")
	assert.Zero(t, tool.Calls)
}

func TestRunQueryTool_ARecordedFailureFailsAgain(t *testing.T) {
	t.Parallel()

	dispatcher := &Dispatcher{logger: zap.NewNop(), labels: &askedLabeler{}}
	tool := liveShipment()
	args := map[string]any{"id": "shp_9"}
	recording := shipmentRecording(t, agentquality.ToolFixture{
		Tool:   "shipment_get",
		Args:   args,
		Result: "Shipment not found",
		Failed: true,
	})

	outcome := dispatcher.runQueryToolWith(
		t.Context(),
		recordedRequest(),
		tool,
		&llmports.ToolCall{ID: "call_1", Name: "shipment_get", Arguments: args},
		&queryExtras{recorded: recording, asked: args},
	)

	assert.True(t, outcome.Failed)
	assert.Equal(t, aitrace.OutcomeFailed, outcome.Verdict)
	assert.Contains(t, outcome.Content, "Shipment not found")
	assert.Zero(t, tool.Calls)
}

func TestRunQueryTool_WithNothingRecordedTheRecordsAreRead(t *testing.T) {
	t.Parallel()

	dispatcher := &Dispatcher{logger: zap.NewNop(), labels: &askedLabeler{}}
	tool := liveShipment()
	args := map[string]any{"id": "shp_1"}

	outcome := dispatcher.runQueryToolWith(
		t.Context(),
		recordedRequest(),
		tool,
		&llmports.ToolCall{ID: "call_1", Name: "shipment_get", Arguments: args},
		&queryExtras{},
	)

	assert.Contains(t, outcome.Content, "Delivered")
	assert.Equal(t, 1, tool.Calls)
}

func TestRunQueryTool_WordsAResultsDatesAgainstTheTurnsClock(t *testing.T) {
	t.Parallel()

	asked := time.Date(2026, time.March, 3, 9, 30, 0, 0, time.UTC).Unix()
	eta := asked + 2*24*60*60
	dispatcher := &Dispatcher{logger: zap.NewNop(), labels: &askedLabeler{}}
	tool := &agenttest.StubQueryTool{
		ToolName: "shipment_get",
		Result:   map[string]any{"proNumber": "S-100", "eta": eta},
	}
	req := recordedRequest()
	req.Context.Now = asked
	req.Context.Timezone = "UTC"
	args := map[string]any{"id": "shp_1"}

	outcome := dispatcher.runQueryToolWith(
		t.Context(),
		req,
		tool,
		&llmports.ToolCall{ID: "call_1", Name: "shipment_get", Arguments: args},
		&queryExtras{},
	)

	assert.Contains(t, outcome.Content, timeutils.DescribeUnixInstantIn(eta, asked, "UTC"))
	assert.Contains(t, outcome.Content, "(in 2 days)",
		"the result is worded against the day the prompt named, not the day the replay ran")
}

func TestOrganizationNow_IsTheTurnsClockInTheOrganizationsZone(t *testing.T) {
	t.Parallel()

	asked := time.Date(2026, time.March, 3, 4, 30, 0, 0, time.UTC).Unix()
	req := recordedRequest()
	req.Context.Now = asked
	req.Context.Timezone = "America/Chicago"

	now := organizationNow(req)

	assert.Equal(t, asked, now.Unix())
	assert.Equal(t, 2, now.Day(),
		"half past four in the morning UTC is still the evening before in Chicago, which is "+
			"the day a call's \"today\" has to mean")
	assert.Equal(t, time.March, now.Month())
}
