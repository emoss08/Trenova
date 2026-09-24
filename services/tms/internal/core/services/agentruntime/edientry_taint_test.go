package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ediShipmentSubject(shipmentID string) *agentdefinition.RuntimeSubject {
	return &agentdefinition.RuntimeSubject{
		Type:            agent.SubjectShipment,
		ID:              shipmentID,
		Label:           "Shipment PRO S-2001",
		Notes:           "Special instructions: deliver to dock 9.",
		OutsideAuthored: agent.TaintSourceEDI,
	}
}

func TestOpenTurn_AShipmentEnteredByEDIOpensTainted(t *testing.T) {
	t.Parallel()

	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	shipmentID := pulid.MustNew("shp_").String()

	turn := rt.OpenTurn(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Check the new shipment.",
		Unattended: true,
		Context: agentdefinition.RuntimeContext{
			Trigger: agent.RunTriggerEvent,
			Subject: ediShipmentSubject(shipmentID),
		},
	})

	require.True(t, turn.Taint().Tainted())
	require.Len(t, turn.Taint().Marks, 1)
	mark := turn.Taint().Marks[0]
	assert.Equal(t, agent.TaintSourceEDI, mark.Source)
	require.NotNil(t, mark.Ref)
	assert.Equal(t, agent.TaintEntityShipment, mark.Ref.EntityType)
	assert.Equal(t, shipmentID, mark.Ref.ID)
	assert.Empty(t, mark.ToolName, "the run read it through no tool")
	assert.Equal(t, turn.Taint().Marks, turn.State().TaintOpened)
}

func TestOpenTurn_AShipmentEnteredByHandOpensClean(t *testing.T) {
	t.Parallel()

	rt := newRuntime(&scriptedCompletion{}, &stubQueryRegistry{}, &stubActionRegistry{}, nil)
	subject := ediShipmentSubject(pulid.MustNew("shp_").String())
	subject.OutsideAuthored = ""

	turn := rt.OpenTurn(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition(),
		Actor:      testActor(),
		Input:      "Check the new shipment.",
		Unattended: true,
		Context:    agentdefinition.RuntimeContext{Subject: subject},
	})

	require.NotNil(t, turn.Taint())
	assert.False(t, turn.Taint().Tainted())
}

func TestRun_AnEDIEnteredShipmentHoldsMoneyForAPerson(t *testing.T) {
	t.Parallel()

	pay := moneyTool()
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		toolTurn("apply_payment", map[string]any{"invoiceId": "inv_2001"}),
		textTurn("The payment waits for approval."),
	}}
	rt := newRuntime(completion,
		&stubQueryRegistry{},
		&stubActionRegistry{Tools: []serviceports.AgentTool{pay}}, nil)

	run := runTainted(t, rt, &serviceports.RunRequest{
		Definition: autoDefinition("apply_payment"),
		Actor:      testActor(),
		Input:      "Check the new shipment.",
		Unattended: true,
		Context: agentdefinition.RuntimeContext{
			Trigger: agent.RunTriggerEvent,
			Subject: ediShipmentSubject(pulid.MustNew("shp_").String()),
		},
	})

	assert.Zero(t, pay.Calls, "a run on a partner's shipment does not move money on its own")
	require.Len(t, run.result.Actions, 1)
	action := run.result.Actions[0]
	assert.False(t, action.Executed)
	assert.True(t, action.Tainted)
	assert.Contains(t, action.HeldBy, agenttoolpolicy.HeldByTainted)

	announced := run.tainted()
	require.Len(t, announced, 1)
	assert.Equal(t, agent.TaintSourceEDI, announced[0].Mark.Source)
}
