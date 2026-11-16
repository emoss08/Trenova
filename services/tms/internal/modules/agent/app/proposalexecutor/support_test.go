package proposalexecutor

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func supportActor(orgID, buID pulid.ID) *permissionports.RequestActor {
	principalID := pulid.MustNew(supportaccess.PrincipalIDPrefix)
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    principalID,
		UserID:         principalID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}
}

func TestExecute_TrenovaSupportNeverDecidesAWriteHeldForAPerson(t *testing.T) {
	t.Parallel()

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	tool := newEgressTool("receivable_post_customer_payment", agent.EgressMoney)
	repo := &fakeProposalRepo{}
	executor := newExecutor(tool, repo, &fakePermissions{allowed: true})

	err := executor.Execute(t.Context(), taintedProposal(tool.name, orgID, buID), nil, supportActor(orgID, buID))

	require.ErrorIs(t, err, ErrTaintedNeedsPerson,
		"a change held for a person of the organization is not Trenova staff's to approve")
	assert.Zero(t, tool.calls)
}

func TestAssertActorMayRun_TrenovaSupportIsNotThePersonASelfScopedChangeIsFor(t *testing.T) {
	t.Parallel()

	tool := selfScopedTool{&recordingTool{
		name:      "home_add_widget",
		resource:  permission.ResourceHomeLayoutPreset,
		operation: permission.OpUpdate,
	}}
	executor := newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true})

	err := executor.assertActorMayRun(t.Context(), tool, supportActor(pulid.MustNew("org_"), pulid.MustNew("bu_")))
	require.Error(t, err)
}

func TestAssertActorMayRun_TrenovaSupportApprovesOnlyWhatItsGrantHolds(t *testing.T) {
	t.Parallel()

	tool := &recordingTool{
		name:      "shipment_update",
		resource:  permission.ResourceShipment,
		operation: permission.OpUpdate,
	}
	actor := supportActor(pulid.MustNew("org_"), pulid.MustNew("bu_"))

	refused := newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: false})
	require.Error(t, refused.assertActorMayRun(t.Context(), tool, actor),
		"a read-only grant, or a resource on the deny list, refuses the approval")

	perms := &fakePermissions{allowed: true}
	allowed := newExecutor(tool, &fakeProposalRepo{}, perms)
	require.NoError(t, allowed.assertActorMayRun(t.Context(), tool, actor))
	require.NotNil(t, perms.lastReq)
	assert.Equal(t, actor.UserID, perms.lastReq.UserID,
		"the permission engine is asked about the support principal, which it answers from the grant")
}

func TestExecute_TrenovaSupportNeverApprovesASendOutsideTheOrganization(t *testing.T) {
	t.Parallel()

	tool := newEgressTool("customer_email_update", agent.EgressCustomerVisible)
	proposal := testProposal(tool.name, map[string]any{"message": "ETA moved"})
	executor := newExecutor(tool, &fakeProposalRepo{}, &fakePermissions{allowed: true})

	err := executor.Execute(t.Context(), proposal, nil,
		supportActor(proposal.OrganizationID, proposal.BusinessUnitID))
	require.Error(t, err)
	assert.Zero(t, tool.calls, "a full grant does not reach the customer's customers")

	err = executor.Execute(t.Context(), proposal, nil,
		testActor(proposal.OrganizationID, proposal.BusinessUnitID))
	require.NoError(t, err, "a person of the organization approves the same send")
	assert.Equal(t, 1, tool.calls)
}
