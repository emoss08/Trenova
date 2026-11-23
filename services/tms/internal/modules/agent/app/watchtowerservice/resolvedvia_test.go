package watchtowerservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/stretchr/testify/assert"
)

func TestResolvedVia_OnlyAChangeThatResolvesRecordsAResolution(t *testing.T) {
	t.Parallel()

	resolvedAt := int64(1_700_000_000)
	open := &watchtower.Item{Status: watchtower.StatusOpen}
	resolved := &watchtower.Item{Status: watchtower.StatusResolved, ResolvedAt: &resolvedAt}
	person := &permissionports.RequestActor{PrincipalType: permissionports.PrincipalTypeUser}
	agent := &permissionports.RequestActor{PrincipalType: permissionports.PrincipalTypeAgent}

	assert.Equal(t, watchtower.ResolvedViaPerson, resolvedVia(open, resolved, person))
	assert.Equal(t, watchtower.ResolvedViaAgent, resolvedVia(open, resolved, agent))
	assert.Empty(t, resolvedVia(open, open, person), "a change that leaves it open records nothing")
	assert.Empty(
		t,
		resolvedVia(resolved, resolved, person),
		"an already resolved item records nothing",
	)
}
