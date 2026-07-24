package permission_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/stretchr/testify/require"
)

func TestIsAgentAllowed_DeniesBillingApproval(t *testing.T) {
	require.False(
		t,
		permission.IsAgentAllowed(permission.ResourceBillingQueue, permission.OpApprove),
		"the agent permission set must exclude approving billing queue items",
	)
	require.False(
		t,
		permission.IsAgentAllowed(permission.ResourceBillingQueue, permission.OpUpdate),
		"the agent permission set must exclude updating billing queue items",
	)
}

func TestIsAgentAllowed_AllowsReadAndPropose(t *testing.T) {
	require.True(t, permission.IsAgentAllowed(permission.ResourceBillingQueue, permission.OpRead))
	require.True(t, permission.IsAgentAllowed(permission.ResourceAgentProposal, permission.OpCreate))
	require.True(t, permission.IsAgentAllowed(permission.ResourceAgentException, permission.OpCreate))
}

func TestIsAgentAllowed_DeniesUnknownResource(t *testing.T) {
	require.False(t, permission.IsAgentAllowed(permission.ResourceInvoice, permission.OpApprove))
	require.False(t, permission.IsAgentAllowed(permission.ResourceUser, permission.OpRead))
}
