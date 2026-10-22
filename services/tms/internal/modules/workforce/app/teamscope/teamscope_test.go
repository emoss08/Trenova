package teamscope_test

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

type grant struct {
	allowed bool
	scope   permission.DataScope
	asked   *permissionports.PermissionCheckRequest
}

func (g *grant) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	g.asked = req

	return &permissionports.PermissionCheckResult{Allowed: g.allowed, DataScope: g.scope}, nil
}

type team map[pulid.ID]bool

func (t team) CanActFor(
	_ context.Context,
	req *orgstructureservice.ScopeRequest,
) (*orgstructureservice.ScopeResult, error) {
	return &orgstructureservice.ScopeResult{Allowed: t[req.WorkerID]}, nil
}

func request(workerID pulid.ID) *teamscope.Request {
	return &teamscope.Request{
		Actor:     &permissionports.RequestActor{UserID: pulid.MustNew("usr_")},
		Resource:  permission.ResourceWorkerSchedule,
		Operation: permission.OpAssign,
		WorkerID:  workerID,
	}
}

func TestRequire_AnOrganizationGrantReachesEveryWorker(t *testing.T) {
	t.Parallel()

	permissions := &grant{allowed: true, scope: permission.DataScopeOrganization}
	guard := teamscope.Guard{Permissions: permissions}

	require.NoError(t, guard.Require(t.Context(), request(pulid.MustNew("wrk_"))))
	assert.Equal(t, permission.ResourceWorkerSchedule.String(), permissions.asked.Resource)
	assert.Equal(t, permission.OpAssign, permissions.asked.Operation)
}

func TestRequire_ATeamGrantReachesOnlyTheTeam(t *testing.T) {
	t.Parallel()

	member, stranger := pulid.MustNew("wrk_"), pulid.MustNew("wrk_")
	guard := teamscope.Guard{
		Permissions: &grant{allowed: true, scope: permission.DataScopeTeam},
		Teams:       team{member: true},
	}

	require.NoError(t, guard.Require(t.Context(), request(member)))
	require.ErrorIs(t, guard.Require(t.Context(), request(stranger)), teamscope.ErrOutsideTeam)

	guard.Teams = nil
	require.ErrorIs(t, guard.Require(t.Context(), request(member)), teamscope.ErrOutsideTeam)
}

func TestRequire_NoGrantOrNoCheckerIsRefused(t *testing.T) {
	t.Parallel()

	refused := teamscope.Guard{Permissions: &grant{allowed: false}}
	require.Error(t, refused.Require(t.Context(), request(pulid.MustNew("wrk_"))))

	require.ErrorIs(t, teamscope.Guard{}.Require(t.Context(), request(pulid.MustNew("wrk_"))),
		teamscope.ErrUnchecked)
}

type managers struct {
	ids   []pulid.ID
	scope worker.ApprovalScope
}

func (m *managers) ManagerIDs(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	scope worker.ApprovalScope,
	_ int64,
) ([]pulid.ID, map[pulid.ID]pulid.ID, error) {
	m.scope = scope
	return m.ids, nil, nil
}

func TestNarrow_ATeamGrantNamesTheManagersAndAWiderOneNone(t *testing.T) {
	t.Parallel()

	manager := pulid.MustNew("usr_")
	team := &managers{ids: []pulid.ID{manager}}
	req := request(pulid.Nil)
	req.Resource = permission.ResourceWorkerLeave
	req.Operation = permission.OpRead
	req.ApprovalScope = worker.ApprovalScopeTimeOff

	narrowed, err := teamscope.ListGuard{
		Permissions: &grant{allowed: true, scope: permission.DataScopeTeam},
		Managers:    team,
	}.Narrow(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{manager}, narrowed)
	assert.Equal(t, worker.ApprovalScopeTimeOff, team.scope)

	whole, err := teamscope.ListGuard{
		Permissions: &grant{allowed: true, scope: permission.DataScopeOrganization},
		Managers:    team,
	}.Narrow(t.Context(), req)
	require.NoError(t, err)
	assert.Nil(t, whole, "an organization grant lists everyone")

	_, err = teamscope.ListGuard{
		Permissions: &grant{allowed: true, scope: permission.DataScopeTeam},
	}.Narrow(t.Context(), req)
	require.ErrorIs(t, err, teamscope.ErrOutsideTeam,
		"a team grant with nobody to say who is on the team lists nobody")

	_, err = teamscope.ListGuard{Permissions: &grant{allowed: false}}.Narrow(t.Context(), req)
	require.Error(t, err)
}
