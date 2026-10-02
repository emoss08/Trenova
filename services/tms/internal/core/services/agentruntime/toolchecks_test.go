package agentruntime

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentextension"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
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
		&stubQueryRegistry{Tools: []serviceports.AgentQueryTool{
			queryTool(agentextension.ToolWebSearch, map[string]any{}, nil),
		}},
		&stubActionRegistry{},
		nil,
	)
	rt.extensions = gate

	turn := rt.OpenTurn(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition(agentextension.ToolWebSearch),
		Actor:      testActor(),
		Input:      "What changed in the ELD rules?",
	})

	require.NotNil(t, turn)
	assert.Equal(t, 1, gate.calls)
}

// meetingPermissions holds each check until the other distinct check has
// started, and counts how many found the other already under way.
type meetingPermissions struct {
	*agentruntimetest.StubPermissions

	started chan struct{}
	first   atomic.Bool
	met     atomic.Int32
}

func (m *meetingPermissions) Check(
	ctx context.Context,
	req *serviceports.PermissionCheckRequest,
) (*serviceports.PermissionCheckResult, error) {
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
		StubPermissions: &agentruntimetest.StubPermissions{Denied: map[string]bool{
			permission.ResourceCustomer.String() + ":" + string(permission.OpRead): true,
		}},
		started: make(chan struct{}),
	}
	rt := newRuntime(
		&scriptedCompletion{},
		&stubQueryRegistry{Tools: []serviceports.AgentQueryTool{
			queryTool("get_shipment", map[string]any{}, nil),
			queryTool("list_shipments", map[string]any{}, nil),
			&agentruntimetest.StubQueryTool{
				ToolName: "get_customer",
				Result:   map[string]any{},
				Resource: permission.ResourceCustomer,
			},
		}},
		&stubActionRegistry{},
		nil,
	)
	rt.permissions = permissions

	permitted := rt.permittedTools(t.Context(), testActor(), []string{
		"get_shipment", "get_customer", "list_shipments",
	})

	assert.Equal(t, []string{"get_shipment", "list_shipments"}, permitted)
	assert.EqualValues(t, 2, permissions.met.Load(), "the checks ran one after the other")
	assert.Len(t, permissions.Requests, 2, "one check per resource and operation")
}
