package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/orgstructureservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/teamscope"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type teamLeaveGrant struct {
	permissionports.PermissionEngine
}

func (teamLeaveGrant) Check(
	_ context.Context,
	_ *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	return &permissionports.PermissionCheckResult{
		Allowed:   true,
		DataScope: permission.DataScopeTeam,
	}, nil
}

type leaveManager struct {
	id      pulid.ID
	members map[pulid.ID]bool
}

func (m leaveManager) CanActFor(
	_ context.Context,
	req *orgstructureservice.ScopeRequest,
) (*orgstructureservice.ScopeResult, error) {
	return &orgstructureservice.ScopeResult{Allowed: m.members[req.WorkerID]}, nil
}

func (m leaveManager) ManagerIDs(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	_ worker.ApprovalScope,
	_ int64,
) ([]pulid.ID, map[pulid.ID]pulid.ID, error) {
	return []pulid.ID{m.id}, nil, nil
}

func TestListLeaveCases_ATeamGrantListsOnlyTheTeam(t *testing.T) {
	t.Parallel()

	member, stranger := pulid.MustNew("wrk_"), pulid.MustNew("wrk_")
	manager := leaveManager{id: pulid.MustNew("usr_"), members: map[pulid.ID]bool{member: true}}
	reads := &fakeWorkforceReads{}
	tool := newListWorkerLeaveCasesTool(reads, teamLeaveGrant{}, manager)

	_, err := tool.Query(t.Context(), testParams(map[string]any{}))
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{manager.id}, reads.casesReq.ManagerIDs,
		"the organization's cases are narrowed to the manager's team")

	_, err = tool.Query(t.Context(), testParams(map[string]any{paramWorkerID: member.String()}))
	require.NoError(t, err)

	reads.casesReq = nil
	_, err = tool.Query(t.Context(), testParams(map[string]any{paramWorkerID: stranger.String()}))
	require.ErrorIs(t, err, teamscope.ErrOutsideTeam)
	assert.Nil(t, reads.casesReq, "nothing is read for a worker off the team")
}
