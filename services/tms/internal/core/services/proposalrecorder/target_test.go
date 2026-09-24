package proposalrecorder

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturingStore struct{ created []*agent.AgentProposal }

func (s *capturingStore) Create(
	_ context.Context,
	p *agent.AgentProposal,
) (*agent.AgentProposal, error) {
	s.created = append(s.created, p)

	return p, nil
}

func recordOne(t *testing.T, action serviceports.PendingAction) *agent.AgentProposal {
	t.Helper()

	store := &capturingStore{}
	actor := &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	_, err := NewWithStores(nil, nil, store).Record(t.Context(), &RecordRequest{
		Actor:    actor,
		Run:      &agent.AgentRun{ID: pulid.MustNew("arun_")},
		Actions:  []serviceports.PendingAction{action},
		Evidence: messageEvidence,
	})
	require.NoError(t, err)
	require.Len(t, store.created, 1)

	return store.created[0]
}

// The runtime pins the record at proposal time; the recorder is what writes
// that pin down. Dropping it here would leave every proposal unchecked with
// nothing upstream any the wiser.
func TestRecord_KeepsThePinnedTargetOnTheProposal(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("shp_")
	proposal := recordOne(t, serviceports.PendingAction{
		ToolName:  "place_shipment_hold",
		Arguments: map[string]any{"shipmentId": id.String()},
		Rationale: "it is late",
		Tier:      agent.TierPropose,
		Target: &serviceports.ProposalTarget{
			Resource: permission.ResourceShipment,
			ID:       id,
			Version:  4,
		},
	})

	assert.Equal(t, string(permission.ResourceShipment), proposal.TargetResource)
	assert.Equal(t, id, proposal.TargetID)
	assert.Equal(t, int64(4), proposal.TargetVersion)
}

func TestRecord_LeavesAnUnpinnedActionUnpinned(t *testing.T) {
	t.Parallel()

	proposal := recordOne(t, serviceports.PendingAction{
		ToolName:  "raise_exception",
		Arguments: map[string]any{},
		Rationale: "something is off",
		Tier:      agent.TierPropose,
	})

	assert.Empty(t, proposal.TargetResource)
	assert.True(t, proposal.TargetID.IsNil())
	assert.Zero(t, proposal.TargetVersion)
}
