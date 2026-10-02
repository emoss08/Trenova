package agentjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/proposalrecorder"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
)

func actorActivities(t *testing.T) (*Activities, *mocks.MockUserRepository) {
	t.Helper()

	users := mocks.NewMockUserRepository(t)

	return NewActivities(ActivitiesParams{Logger: zap.NewNop(), Users: users}), users
}

func TestUnattendedActor_NamesTheSystemUserAndStaysAnAgent(t *testing.T) {
	t.Parallel()

	activities, users := actorActivities(t)
	system := pulid.MustNew("usr_")
	users.On("GetSystemUser", mock.Anything, []string{"id"}).
		Return(&tenant.User{ID: system}, nil).
		Once()
	scope := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	definition := &agentdefinition.Definition{ID: pulid.MustNew("agdef_"), Name: "Dispatch Agent"}

	actor, err := activities.unattendedActor(t.Context(), scope, definition)
	require.NoError(t, err)

	assert.Equal(t, system, actor.UserID, "writes are attributed to the system user")
	assert.Equal(t, serviceports.PrincipalTypeAgent, actor.PrincipalType,
		"authorization stays the agent's")
	assert.Equal(t, definition.ID, actor.PrincipalID, "the principal names which agent runs")
	assert.True(t, actor.IsAgent())
	assert.False(t, actor.IsUser())
	assert.Equal(t, pulid.Nil, actor.PersonUserID(), "the system user is not a person")
	assert.Equal(t, scope.OrgID, actor.OrganizationID)
	assert.Equal(t, scope.BuID, actor.BusinessUnitID)

	assert.Equal(t, system, actor.ExecutorUserID(),
		"an automatic write is executed by the system user")

	audit := actor.AuditActor()
	assert.Equal(t, serviceports.PrincipalTypeAgent, audit.PrincipalType)
	assert.Equal(t, definition.ID, audit.PrincipalID)
	assert.Equal(t, system, audit.UserID, "the audit log names the system user")
}

func TestAgentActorFor_FallsBackToTheGenericAgentWithoutADefinition(t *testing.T) {
	t.Parallel()

	scope := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	assert.Equal(t, serviceports.AgentPrincipalID, agentActorFor(scope, nil).PrincipalID)
	assert.Equal(t, serviceports.AgentPrincipalID,
		agentActorFor(scope, &agentdefinition.Definition{}).PrincipalID)
	assert.True(t, agentActorFor(scope, nil).UserID.IsNil(),
		"only an unattended run's actor carries the system user")
}

func TestUnattendedActor_AMissingSystemUserFailsTheAttemptRetryably(t *testing.T) {
	t.Parallel()

	activities, users := actorActivities(t)
	users.On("GetSystemUser", mock.Anything, []string{"id"}).
		Return(nil, errors.New("connection refused")).
		Once()

	actor, err := activities.unattendedActor(
		t.Context(),
		pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		&agentdefinition.Definition{ID: pulid.MustNew("agdef_")},
	)

	require.Error(t, err)
	assert.Nil(t, actor)
	var applicationErr *temporal.ApplicationError
	require.ErrorAs(t, err, &applicationErr)
	assert.False(t, applicationErr.NonRetryable(), "the run is retried rather than run unnamed")
}

type noProposalsYet struct {
	repositories.AgentProposalRepository
}

func (noProposalsYet) ListByRun(
	context.Context,
	repositories.ListAgentProposalsByRunRequest,
) ([]*agent.AgentProposal, error) {
	return nil, nil
}

type createdProposals struct {
	created []*agent.AgentProposal
}

func (s *createdProposals) Create(
	_ context.Context,
	proposal *agent.AgentProposal,
) (*agent.AgentProposal, error) {
	s.created = append(s.created, proposal)

	return proposal, nil
}

func TestRecordProposals_AnAutomaticWriteIsExecutedByTheSystemUser(t *testing.T) {
	t.Parallel()

	users := mocks.NewMockUserRepository(t)
	system := pulid.MustNew("usr_")
	users.On("GetSystemUser", mock.Anything, []string{"id"}).
		Return(&tenant.User{ID: system}, nil).
		Once()
	store := &createdProposals{}
	activities := &Activities{
		logger:       zap.NewNop(),
		users:        users,
		proposalRepo: noProposalsYet{},
		recorder:     proposalrecorder.NewWithStores(zap.NewNop(), nil, store),
	}
	scope := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}

	_, err := activities.recordProposals(t.Context(), recordProposalsParams{
		Definition: &agentdefinition.Definition{ID: pulid.MustNew("agdef_"), Name: "Dispatch Agent"},
		Run:        &agent.AgentRun{ID: pulid.MustNew("ar_")},
		TenantInfo: scope,
		Actions: []serviceports.PendingAction{{
			ToolName:  "assign_move",
			Arguments: map[string]any{},
			Rationale: "cover the move",
			Tier:      agent.TierAutoExecute,
			Executed:  true,
		}},
	})
	require.NoError(t, err)

	require.Len(t, store.created, 1)
	assert.Equal(t, system, store.created[0].ExecutedByUserID,
		"an unattended run's automatic write names the system user as its executor")
}

func TestRecordProposals_NothingToFileReadsNoSystemUser(t *testing.T) {
	t.Parallel()

	activities := &Activities{
		logger:       zap.NewNop(),
		users:        mocks.NewMockUserRepository(t),
		proposalRepo: noProposalsYet{},
		recorder:     proposalrecorder.NewWithStores(zap.NewNop(), nil, &createdProposals{}),
	}

	_, err := activities.recordProposals(t.Context(), recordProposalsParams{
		Definition: &agentdefinition.Definition{ID: pulid.MustNew("agdef_")},
		Run:        &agent.AgentRun{ID: pulid.MustNew("ar_")},
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	})
	require.NoError(t, err)
}
