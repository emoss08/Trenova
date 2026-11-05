package tooldispatch

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sessionSeeingPermissions struct {
	*agenttest.StubPermissions

	mu       sync.Mutex
	sessions []authctx.SessionAttributes
}

func (p *sessionSeeingPermissions) Check(
	ctx context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	if session, ok := authctx.GetSessionAttributes(ctx); ok {
		p.mu.Lock()
		p.sessions = append(p.sessions, session)
		p.mu.Unlock()
	}

	return p.StubPermissions.Check(ctx, req)
}

func TestDispatch_ChecksPermissionAsAnUnprovenSessionWithoutOne(t *testing.T) {
	t.Parallel()

	tool := &agenttest.StubActionTool{ToolName: "dispatch_assign_move", Tier: agent.TierAutoExecute}
	permissions := &sessionSeeingPermissions{StubPermissions: &agenttest.StubPermissions{}}
	d := testDispatcher(DispatcherParams{
		ActionTools: &agenttest.StubActionRegistry{Tools: []agenttoolports.AgentTool{tool}},
		Permissions: permissions,
	})
	actor := runtimetest.Actor()

	d.dispatch(t.Context(), &DispatchParams{
		req: &agentruntimeports.RunRequest{
			Definition: runtimetest.Definition(tool.ToolName),
			Actor:      actor,
		},
		call:  llmports.ToolCall{ID: "call-1", Name: tool.ToolName},
		taint: &agent.RunTaint{},
	})

	require.NotEmpty(t, permissions.sessions)
	assert.Equal(t, actor.UserID, permissions.sessions[0].UserID)
	assert.Equal(t, authctx.UnprovenAssurance, permissions.sessions[0].AuthenticatorAAL)
}
