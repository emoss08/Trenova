package teamscope_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/orgstructureservice"
	"github.com/emoss08/trenova/internal/core/services/teamscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type grant struct {
	allowed bool
	scope   permission.DataScope
	asked   *services.PermissionCheckRequest
}

func (g *grant) Check(
	_ context.Context,
	req *services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	g.asked = req

	return &services.PermissionCheckResult{Allowed: g.allowed, DataScope: g.scope}, nil
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
		Actor:     &services.RequestActor{UserID: pulid.MustNew("usr_")},
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
