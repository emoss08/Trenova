package agentruntime

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An agent holding a web tool used to read the organization's extensions
// twice as its turn opened: once to say what the tools are, and again to
// narrow what the person may use. One read answers both.
func TestOpenTurn_ReadsTheOrganizationsExtensionsOnce(t *testing.T) {
	t.Parallel()

	gate := webGate(agentextension.AvailabilitySelectedAgents)
	rt := newRuntime(
		&scriptedCompletion{},
		&stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
			queryTool(agentextension.ToolWebSearch, map[string]any{}, nil),
		}},
		&stubActionRegistry{},
		nil,
	)
	rewire(rt, func(p *Params) { p.Extensions = gate })

	turn := rt.OpenTurn(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition(agentextension.ToolWebSearch),
		Actor:      runtimetest.Actor(),
		Input:      "What changed in the ELD rules?",
	})

	require.NotNil(t, turn)
	assert.Equal(t, 1, gate.calls)
}

// meetingPermissions holds each check until the other distinct check has
// started, and counts how many found the other already under way.
type meetingPermissions struct {
	*agenttest.StubPermissions

	started chan struct{}
	first   atomic.Bool
	met     atomic.Int32
}

func (m *meetingPermissions) Check(
	ctx context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	if m.first.CompareAndSwap(false, true) {
		if meet(ctx, m.started) {
			m.met.Add(1)
		}
	} else {
		close(m.started)
		m.met.Add(1)
	}

	return m.StubPermissions.Check(ctx, req)
}

/*
Which of an agent's tools the person may use is one permission check per
resource and operation, and an agent with a wide toolbox holds dozens. They
were asked one after another, each a round trip to the permission cache, all
before the model was called. They are asked side by side and answer the same.
*/
func TestPermittedTools_ChecksDistinctPermissionsTogetherAndAnswersTheSame(t *testing.T) {
	t.Parallel()

	permissions := &meetingPermissions{
		StubPermissions: &agenttest.StubPermissions{Denied: map[string]bool{
			permission.ResourceCustomer.String() + ":" + string(permission.OpRead): true,
		}},
		started: make(chan struct{}),
	}
	rt := newRuntime(
		&scriptedCompletion{},
		&stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{
			queryTool("shipment_get", map[string]any{}, nil),
			queryTool("shipment_list", map[string]any{}, nil),
			&agenttest.StubQueryTool{
				ToolName: "customer_get",
				Result:   map[string]any{},
				Resource: permission.ResourceCustomer,
			},
		}},
		&stubActionRegistry{},
		nil,
	)
	rewire(rt, func(p *Params) { p.Permissions = permissions })

	permitted := rt.access.PermittedTools(t.Context(), runtimetest.Actor(), []string{
		"shipment_get", "customer_get", "shipment_list",
	})

	assert.Equal(t, []string{"shipment_get", "shipment_list"}, permitted)
	assert.EqualValues(t, 2, permissions.met.Load(), "the checks ran one after the other")
	assert.Len(t, permissions.Requests, 2, "one check per resource and operation")
}
