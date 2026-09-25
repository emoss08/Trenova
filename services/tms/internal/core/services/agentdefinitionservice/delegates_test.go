package agentdefinitionservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// tenantDefinitions answers ListByIDs from one tenant's agents, as the
// repository does: an agent of another tenant is simply not found.
type tenantDefinitions struct {
	repositories.AgentDefinitionRepository

	agents []*agentdefinition.Definition
}

func (r *tenantDefinitions) ListByIDs(
	_ context.Context,
	req repositories.ListAgentDefinitionsByIDsRequest,
) ([]*agentdefinition.Definition, error) {
	found := make([]*agentdefinition.Definition, 0, len(req.IDs))
	for _, definition := range r.agents {
		if definition.OrganizationID != req.TenantInfo.OrgID ||
			definition.BusinessUnitID != req.TenantInfo.BuID {
			continue
		}
		for _, id := range req.IDs {
			if id == definition.ID {
				found = append(found, definition)
			}
		}
	}

	return found, nil
}

func delegateCandidate(
	org, bu pulid.ID,
	name string,
	change func(*agentdefinition.Definition),
) *agentdefinition.Definition {
	definition := &agentdefinition.Definition{
		ID:              pulid.MustNew("agdef_"),
		OrganizationID:  org,
		BusinessUnitID:  bu,
		Name:            name,
		AutonomyCeiling: agent.TierPropose,
		Enabled:         true,
	}
	definition.ApplyDefaults()
	if change != nil {
		change(definition)
	}

	return definition
}

func delegateMessages(t *testing.T, s *Service, definition, previous *agentdefinition.Definition) map[string]string {
	t.Helper()

	multiErr := errortypes.NewMultiError()
	require.NoError(t, s.validateDelegates(t.Context(), definition, previous, multiErr))

	messages := make(map[string]string, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		messages[e.Field] = e.Message
	}

	return messages
}

// Each delegate must be an agent in the same tenant that a person can talk
// to, and one added now must be enabled. One already on the list that has
// since been disabled does not make the agent unsaveable.
func TestValidateDelegates_ChecksEachAgentAgainstTheTenant(t *testing.T) {
	t.Parallel()

	org, bu := pulid.MustNew("org_"), pulid.MustNew("bu_")
	usable := delegateCandidate(org, bu, "Report Builder", nil)
	disabled := delegateCandidate(org, bu, "Billing Desk", func(d *agentdefinition.Definition) {
		d.Enabled = false
	})
	background := delegateCandidate(org, bu, "Load Monitor", func(d *agentdefinition.Definition) {
		d.TriggerMode = agentdefinition.TriggerEvent
	})
	foreign := delegateCandidate(pulid.MustNew("org_"), bu, "Elsewhere", nil)
	page := delegateCandidate(org, bu, "Shipment import assistant",
		func(d *agentdefinition.Definition) {
			d.SystemKey = agentdefinition.SystemKeyImportAssistant
		})

	s := &Service{
		l: zap.NewNop(),
		repo: &tenantDefinitions{agents: []*agentdefinition.Definition{
			usable, disabled, background, foreign, page,
		}},
	}
	definition := delegateCandidate(org, bu, "Homepage Widget Builder", nil)
	definition.DelegateIDs = []pulid.ID{
		usable.ID, disabled.ID, background.ID, foreign.ID, page.ID,
	}

	messages := delegateMessages(t, s, definition, nil)
	assert.NotContains(t, messages, "delegateIds[0]")
	assert.Contains(t, messages["delegateIds[1]"], "Billing Desk is disabled")
	assert.Contains(t, messages["delegateIds[2]"], "Load Monitor runs on its own")
	assert.Contains(t, messages["delegateIds[3]"], "does not exist in this organization")
	assert.Contains(t, messages["delegateIds[4]"], "works on its own page")

	previous := *definition
	previous.DelegateIDs = []pulid.ID{disabled.ID}
	kept := delegateMessages(t, s, definition, &previous)
	assert.NotContains(t, kept, "delegateIds[1]",
		"a delegate disabled since it was added stays, and is refused when asked")
}
