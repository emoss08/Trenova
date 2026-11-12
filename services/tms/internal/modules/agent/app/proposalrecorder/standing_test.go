package proposalrecorder

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type waitingProposals struct {
	waiting []*agent.AgentProposal
	asked   *agentproposalports.ListStandingByDefinitionRequest
}

func (w *waitingProposals) ListStandingByDefinition(
	_ context.Context,
	req *agentproposalports.ListStandingByDefinitionRequest,
) ([]*agent.AgentProposal, error) {
	w.asked = req

	return w.waiting, nil
}

func recordBeside(
	t *testing.T,
	standing *waitingProposals,
	trigger agent.RunTrigger,
	definitionID pulid.ID,
	actions ...agentruntimeports.PendingAction,
) *RecordResult {
	t.Helper()

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	svc := recorderOn(nil, &capturingStore{})
	svc.standing = standing
	result, err := svc.Record(t.Context(), &RecordRequest{
		Actor:      &permissionports.RequestActor{OrganizationID: orgID, BusinessUnitID: buID},
		Definition: &agentdefinition.Definition{ID: definitionID, Name: "Load monitor"},
		Run: &agent.AgentRun{
			ID:             pulid.MustNew("arun_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Trigger:        trigger,
		},
		Actions:  actions,
		Evidence: messageEvidence,
	})
	require.NoError(t, err)

	return result
}

func evaluate(shipment pulid.ID) agentruntimeports.PendingAction {
	action := actionOn("exception_evaluate_service_failures", permission.ResourceShipment, shipment)
	action.Arguments = map[string]any{"shipmentId": shipment.String()}

	return action
}

/*
A scheduled run that reaches the write its agent already has waiting on a
person adds nothing to the queue: the repeat is recorded as superseded, and a
write about another record, or with other values, is filed as usual.
*/
func TestRecord_AnUnattendedRunDoesNotRepeatWhatItsAgentHasWaiting(t *testing.T) {
	t.Parallel()

	late, other := pulid.MustNew("shp_"), pulid.MustNew("shp_")
	definitionID := pulid.MustNew("agdef_")
	standing := &waitingProposals{waiting: []*agent.AgentProposal{{
		ID:             pulid.MustNew("ap_"),
		ToolName:       "exception_evaluate_service_failures",
		ToolParams:     map[string]any{"shipmentId": late.String()},
		TargetResource: string(permission.ResourceShipment),
		TargetID:       late,
		Status:         agent.ProposalStatusPending,
	}}}

	result := recordBeside(t, standing, agent.RunTriggerScheduled, definitionID,
		evaluate(late), evaluate(other))

	require.NotNil(t, standing.asked)
	assert.Equal(t, definitionID, standing.asked.DefinitionID)
	assert.Equal(t, []string{"exception_evaluate_service_failures"}, standing.asked.ToolNames)
	require.Len(t, result.Proposals, 2)
	assert.Equal(t, agent.ProposalStatusSuperseded, result.Proposals[0].Status)
	assert.Equal(t, agent.ProposalStatusPending, result.Proposals[1].Status)
	assert.Nil(t, result.Plan, "one write left to decide is a proposal, not a plan")
}

// What a person asks for in a conversation is theirs to see filed, whatever
// the agent already has waiting.
func TestRecord_AConversationFilesWhatItIsAskedEvenWhenItRepeats(t *testing.T) {
	t.Parallel()

	late := pulid.MustNew("shp_")
	standing := &waitingProposals{waiting: []*agent.AgentProposal{{
		ToolName:       "exception_evaluate_service_failures",
		ToolParams:     map[string]any{"shipmentId": late.String()},
		TargetResource: string(permission.ResourceShipment),
		TargetID:       late,
		Status:         agent.ProposalStatusPending,
	}}}

	result := recordBeside(t, standing, agent.RunTriggerChat, pulid.MustNew("agdef_"),
		evaluate(late))

	assert.Nil(t, standing.asked)
	require.Len(t, result.Proposals, 1)
	assert.Equal(t, agent.ProposalStatusPending, result.Proposals[0].Status)
}
