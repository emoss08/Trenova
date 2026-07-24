package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

type approveOnlyTool struct{}

func (approveOnlyTool) Name() string                { return "approve_only" }
func (approveOnlyTool) Description() string         { return "" }
func (approveOnlyTool) ParamSchema() map[string]any { return map[string]any{} }
func (approveOnlyTool) Reversible() bool            { return false }
func (approveOnlyTool) PermissionResource() permission.Resource {
	return permission.ResourceBillingQueue
}
func (approveOnlyTool) PermissionOperation() permission.Operation { return permission.OpApprove }
func (approveOnlyTool) RequiresIdempotencyKey() bool              { return false }
func (approveOnlyTool) DefaultAutonomyTier() agent.AutonomyTier   { return agent.TierPropose }
func (approveOnlyTool) Execute(_ context.Context, _ serviceports.ToolExecuteParams) error {
	return nil
}

func TestGuardExecute_AgentCannotApprove(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	err := guardExecute(approveOnlyTool{}, serviceports.ToolExecuteParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Actor: &serviceports.RequestActor{
			PrincipalType:  serviceports.PrincipalTypeAgent,
			PrincipalID:    serviceports.AgentPrincipalID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
	})

	require.ErrorIs(t, err, ErrAgentCannotApprove)
}

func TestExecute_TenantMismatch_DoesNotCallPort(t *testing.T) {
	// The mock has no expectations configured; if the tool reaches the port,
	// testify panics, failing the test.
	billing := mocks.NewMockBillingQueueService(t)
	tool := newTransitionToInReviewTool(billing)

	actorOrg := pulid.MustNew("org_")
	actorBu := pulid.MustNew("bu_")

	err := tool.Execute(t.Context(), serviceports.ToolExecuteParams{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Actor: &serviceports.RequestActor{
			PrincipalType:  serviceports.PrincipalTypeUser,
			OrganizationID: actorOrg,
			BusinessUnitID: actorBu,
		},
		Params: map[string]any{"billingQueueItemId": pulid.MustNew("bqi_").String()},
	})

	require.ErrorIs(t, err, ErrTenantMismatch)
}
