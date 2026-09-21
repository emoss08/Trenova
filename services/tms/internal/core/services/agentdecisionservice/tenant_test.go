package agentdecisionservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The tenant on the request and the decider's own are asserted to agree
// before anything is read, rather than trusted to have come from the same
// auth context in every caller.
func TestDecideWithOutcome_RefusesATenantThatIsNotTheDeciders(t *testing.T) {
	t.Parallel()

	actor := &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	_, err := (&Service{}).DecideWithOutcome(t.Context(), &services.DecideAgentProposalRequest{
		ProposalID: pulid.MustNew("ap_"),
		Decision:   agent.DecisionAccepted,
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: actor.BusinessUnitID},
	}, actor)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "own organization")
}
