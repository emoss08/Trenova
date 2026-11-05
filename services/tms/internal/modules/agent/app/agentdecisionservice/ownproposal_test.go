package agentdecisionservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssertOwnProposal_HoldsAProposalToTheAgentThatRaisedIt(t *testing.T) {
	t.Parallel()

	t.Run("an agent open to everyone", func(t *testing.T) {
		t.Parallel()

		h := newWindowHarness(t, 1)

		require.NoError(t, h.service.AssertOwnProposal(t.Context(), h.ids[0], h.tenant, h.actor))
	})

	t.Run("an agent restricted to roles the person holds", func(t *testing.T) {
		t.Parallel()

		h := newWindowHarness(t, 1)
		agentID := h.service.runRepo.(windowRuns).agentID
		h.service.definitions = windowDefinitions{mode: agentdefinition.AccessRoles}
		h.service.permissions = &agenttest.StubPermissions{GrantedAgents: []pulid.ID{agentID}}

		require.NoError(t, h.service.AssertOwnProposal(t.Context(), h.ids[0], h.tenant, h.actor))
	})

	t.Run("an agent the person may no longer use", func(t *testing.T) {
		t.Parallel()

		h := newWindowHarness(t, 1)
		h.service.definitions = windowDefinitions{mode: agentdefinition.AccessRoles}

		err := h.service.AssertOwnProposal(t.Context(), h.ids[0], h.tenant, h.actor)

		require.Error(t, err)
		assert.True(t, errortypes.IsAuthorizationError(err))
	})

	t.Run("an agent that no longer exists", func(t *testing.T) {
		t.Parallel()

		h := newWindowHarness(t, 1)
		h.service.definitions = windowDefinitions{
			err: errortypes.NewNotFoundError("Agent definition not found"),
		}

		err := h.service.AssertOwnProposal(t.Context(), h.ids[0], h.tenant, h.actor)

		require.Error(t, err)
		assert.True(t, errortypes.IsBusinessError(err))
	})
}
