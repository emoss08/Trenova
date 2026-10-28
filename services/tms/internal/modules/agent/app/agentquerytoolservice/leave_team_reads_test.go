package agentquerytoolservice_test

import (
	"context"
	"slices"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/agentread"
	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/graphqlreadports"
	"github.com/emoss08/trenova/internal/modules/identity/app/orgstructureservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerleaveservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// teamLeaveGrant is a driver manager's grant: leave cases, held to the team.
type teamLeaveGrant struct {
	mocks.DenyResourcesPermissionEngine
}

func (g *teamLeaveGrant) Check(
	ctx context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	if req.Resource == permission.ResourceWorkerLeave.String() {
		return &permissionports.PermissionCheckResult{
			Allowed:   true,
			DataScope: permission.DataScopeTeam,
		}, nil
	}

	return g.DenyResourcesPermissionEngine.Check(ctx, req)
}

// teamStructure is a reporting line: the manager answers for members.
type teamStructure struct {
	workerports.OrgStructureRepository

	manager pulid.ID
	members map[pulid.ID]bool
}

func (s *teamStructure) ListDelegations(
	context.Context,
	*workerports.ListApprovalDelegationsRequest,
) ([]*worker.ApprovalDelegation, error) {
	return nil, nil
}

func (s *teamStructure) ManagesWorker(
	_ context.Context,
	_ pagination.TenantInfo,
	managerIDs []pulid.ID,
	workerID pulid.ID,
) (bool, error) {
	return slices.Contains(managerIDs, s.manager) && s.members[workerID], nil
}

// teamLeaveCases holds one case per worker and narrows to the team the way
// the repository's workerteam filter does.
type teamLeaveCases struct {
	workerports.WorkerLeaveRepository

	cases   []*worker.WorkerLeaveCase
	team    *teamStructure
	mu      sync.Mutex
	scopes  [][]pulid.ID
	workers []pulid.ID
}

func (r *teamLeaveCases) ListCases(
	_ context.Context,
	req *workerports.ListLeaveCasesRequest,
) ([]*worker.WorkerLeaveCase, error) {
	r.mu.Lock()
	r.scopes = append(r.scopes, req.ManagerIDs)
	r.workers = append(r.workers, req.WorkerID)
	r.mu.Unlock()

	out := make([]*worker.WorkerLeaveCase, 0, len(r.cases))
	for _, leave := range r.cases {
		if req.WorkerID.IsNotNil() && leave.WorkerID != req.WorkerID {
			continue
		}
		if len(req.ManagerIDs) > 0 && (!slices.Contains(req.ManagerIDs, r.team.manager) ||
			!r.team.members[leave.WorkerID]) {
			continue
		}
		out = append(out, leave)
	}

	return out, nil
}

func (r *teamLeaveCases) reads() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.scopes)
}

func leaveCaseFor(workerID pulid.ID) *worker.WorkerLeaveCase {
	return &worker.WorkerLeaveCase{
		ID:        pulid.MustNew("wlc_"),
		WorkerID:  workerID,
		LeaveType: worker.LeaveTypeFMLA,
		Status:    worker.LeaveCaseApproved,
		Frequency: worker.LeaveContinuous,
		StartsAt:  1_788_000_000,
		Entries:   []*worker.WorkerLeaveEntry{},
	}
}

// A driver manager's agent reads leave cases through the screen's resolver,
// which holds every worker's list to the manager's team, refuses one worker
// off it, and refuses an unattended agent, which answers for no one.
func TestLeaveCases_ATeamGrantReadsOnlyTheTeamThroughGraphQL(t *testing.T) {
	t.Parallel()

	person := agentreadtest.Person()
	member, stranger := pulid.MustNew("wrk_"), pulid.MustNew("wrk_")
	team := &teamStructure{manager: person.UserID, members: map[pulid.ID]bool{member: true}}
	mine, theirs := leaveCaseFor(member), leaveCaseFor(stranger)
	repo := &teamLeaveCases{cases: []*worker.WorkerLeaveCase{mine, theirs}, team: team}
	engine := &teamLeaveGrant{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{
		Engine: engine,
		Resolvers: agentread.ResolverParams{
			WorkerLeaveService: workerleaveservice.NewWithDeps(
				workerleaveservice.Deps{Repo: repo},
			),
			OrgStructureService: orgstructureservice.NewWithDeps(
				orgstructureservice.Deps{Repo: team},
			),
		},
	})
	tool, err := agentquerytoolservice.ConvertedListTool("worker_list_leave_cases", reader, engine)
	require.NoError(t, err)
	call := func(actor *permissionports.RequestActor, args map[string]any) (any, error) {
		return tool.Query(readscope.With(t.Context(), nil), &agenttoolports.QueryToolParams{
			OrganizationID: actor.OrganizationID,
			BusinessUnitID: actor.BusinessUnitID,
			Actor:          actor,
			Timezone:       "America/Chicago",
			Params:         args,
			RunID:          pulid.MustNew("ar_"),
		})
	}

	everyone, err := call(person, map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, []string{mine.ID.String()}, itemIDs(asMap(t, everyone)),
		"the organization's cases are narrowed to the manager's team")
	assert.Equal(t, []pulid.ID{person.UserID}, repo.scopes[0])

	one, err := call(person, map[string]any{"workerId": member.String()})
	require.NoError(t, err)
	assert.Equal(t, []string{mine.ID.String()}, itemIDs(asMap(t, one)))

	before := repo.reads()
	_, err = call(person, map[string]any{"workerId": stranger.String()})
	var refused *graphqlreadports.ReadRefusedError
	require.ErrorAs(t, err, &refused, "a worker off the team is refused")
	assert.Equal(t, before, repo.reads(), "nothing is read for a worker off the team")

	_, err = call(agentreadtest.Agent(person), map[string]any{})
	require.ErrorAs(t, err, &refused,
		"an unattended agent answers for no one, so a team grant reads nothing")
	assert.Equal(t, before, repo.reads(), "the agent's read never reaches the repository")
}
