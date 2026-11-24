package assistantartifacts

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListSize_CountsTheWholeListAsTheAgentWhoPublishesTheBoard(t *testing.T) {
	t.Parallel()

	f := newRowsFixture(t, &mocks.AllowAllPermissionEngine{})
	f.svc.conversations = &assistanttest.StubConversationRepo{
		GetErr: errortypes.NewNotFoundError("Thread not found within your organization"),
	}
	agent := &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeAgent,
		PrincipalID:    pulid.MustNew("agd_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	size, err := f.svc.ListSize(t.Context(), &assistantports.ListSizeRequest{
		Actor:      agent,
		ThreadID:   f.thread.ID,
		ArtifactID: f.artifact.ID,
		Agent:      &assistantports.BoardAgentRead{DataAccess: permission.SensitivityInternal},
	})

	require.NoError(t, err, "the thread is the agent's to read through its board")
	assert.Equal(t, 128, size.Total)
	assert.Equal(t, "shipment", size.Kind)
	require.Len(t, f.tool.calls, 1)
	call := f.tool.calls[0]
	assert.Equal(t, map[string]any{"status": []any{"InTransit"}, "limit": 1}, call.Params,
		"the list's own filters, one row, its total")
	assert.True(t, call.IncludeTotal)
	assert.Same(t, agent, call.Actor)
	assert.Equal(t, permission.SensitivityInternal, call.DataAccessCeiling)

	_, err = f.svc.ListSize(t.Context(), &assistantports.ListSizeRequest{
		Actor: agent, ThreadID: pulid.MustNew("athr_"), ArtifactID: f.artifact.ID,
	})
	assert.Error(t, err, "a table of another conversation is not counted")
}
