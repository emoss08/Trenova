package proposalexecutor

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type selfScopedTool struct {
	*recordingTool
}

func (selfScopedTool) SelfScoped() bool { return true }

/*
A change to the person's own home page is approved without a role grant, as
the home page's editor lets them make it — but only by a person. The tool
itself checks the approver is the person it was proposed for.
*/
func TestAssertActorMayRun_ASelfScopedToolNeedsAPersonNotAGrant(t *testing.T) {
	t.Parallel()

	tool := selfScopedTool{&recordingTool{
		name:      "add_home_widget",
		resource:  permission.ResourceHomeLayoutPreset,
		operation: permission.OpUpdate,
	}}
	perms := &fakePermissions{allowed: false}
	executor := newExecutor(tool, &fakeProposalRepo{}, perms)

	person := &services.RequestActor{
		PrincipalType: services.PrincipalTypeUser,
		PrincipalID:   pulid.MustNew("usr_"),
		UserID:        pulid.MustNew("usr_"),
	}
	require.NoError(t, executor.assertActorMayRun(t.Context(), tool, person))
	assert.Nil(t, perms.lastReq, "no grant is asked for")

	agentActor := &services.RequestActor{
		PrincipalType: services.PrincipalTypeAgent,
		PrincipalID:   pulid.MustNew("agdef_"),
	}
	require.Error(t, executor.assertActorMayRun(t.Context(), tool, agentActor))
}
