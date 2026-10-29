package tooldispatch

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/modules/agent/agenttest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthorize_TheRefusalNamesWhoseAccessFellShort(t *testing.T) {
	t.Parallel()

	denied := &agenttest.StubPermissions{Denied: map[string]bool{
		permission.ResourceWorker.String() + ":" + string(permission.OpUpdate): true,
	}}
	d := testDispatcher(DispatcherParams{Permissions: denied})

	person, refused := d.authorize(
		t.Context(),
		runtimetest.Actor(),
		"update_worker",
		permission.ResourceWorker,
		permission.OpUpdate,
	)
	require.True(t, refused)
	assert.Equal(t, aitrace.OutcomeDenied, person.Verdict)
	assert.Equal(t,
		`Tool "update_worker" is not permitted: the person you are working for `+
			"does not have update access to worker. Do not try it another way or propose it "+
			"instead. Tell the person plainly that their account does not have update access "+
			"to worker, that an administrator grants it under their role, and what you would "+
			"do once it is granted. Do everything else they asked that does not need it.",
		person.Content,
	)

	unattended, refused := d.authorize(
		t.Context(), runtimetest.SystemUserActor(), "update_worker", permission.ResourceWorker,
		permission.OpUpdate,
	)
	require.True(t, refused)
	assert.Equal(t, aitrace.OutcomeDenied, unattended.Verdict)
	assert.Equal(t,
		`Tool "update_worker" is not permitted: this agent's unattended access `+
			"does not include update on worker. Do not try it another way. Finish what the "+
			"run can do without it and say in your report that this step needs a person, or "+
			"someone to widen what this agent may do unattended in AI Control.",
		unattended.Content,
	)
	assert.NotContains(t, unattended.Content, "the person you are working for")
	assert.Equal(t, "lacks update access to worker", unattended.Reason)
}
