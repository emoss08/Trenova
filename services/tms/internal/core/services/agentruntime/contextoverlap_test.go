package agentruntime

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const meetWait = 2 * time.Second

func meet(ctx context.Context, ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	case <-time.After(meetWait):
		return false
	case <-ctx.Done():
		return false
	}
}

type meeting struct {
	orgStarted    chan struct{}
	vectorStarted chan struct{}

	orgSawVector atomic.Bool
	vectorSawOrg atomic.Bool
}

type meetingOrganizations struct {
	repositories.OrganizationRepository

	meeting *meeting
	org     *tenant.Organization
}

func (m *meetingOrganizations) GetByID(
	ctx context.Context,
	_ repositories.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	close(m.meeting.orgStarted)
	m.meeting.orgSawVector.Store(meet(ctx, m.meeting.vectorStarted))

	return m.org, nil
}

type meetingVectorizer struct {
	serviceports.QueryVectorizer

	meeting *meeting
}

func (m *meetingVectorizer) Vectorize(
	ctx context.Context,
	_ *serviceports.QueryVectorRequest,
) (serviceports.QueryVector, error) {
	close(m.meeting.vectorStarted)
	m.meeting.vectorSawOrg.Store(meet(ctx, m.meeting.orgStarted))

	return serviceports.QueryVector{}, nil
}

/*
The context a turn opens with is read from several places that do not depend
on one another — the organization, the person, the agents it may hand work
to, and the memories, which wait on an embedding of the question. They were
read one after the other, so the embedding call waited behind the organization
read and the person waited for both.
*/
func TestContextBuilder_ReadsTheOrganizationWhileTheQuestionIsEmbedded(t *testing.T) {
	t.Parallel()

	m := &meeting{orgStarted: make(chan struct{}), vectorStarted: make(chan struct{})}
	memories := &fakeMemories{answer: &serviceports.MemoryContext{}}
	builder := &ContextBuilder{
		logger: zap.NewNop(),
		organizations: &meetingOrganizations{
			meeting: m,
			org:     &tenant.Organization{Timezone: "America/Chicago"},
		},
		users:      &stubUsers{user: &tenant.User{Name: "Dana", Timezone: "UTC"}},
		vectorizer: &meetingVectorizer{meeting: m},
		memories:   memories,
		runtime: newRuntime(
			&scriptedCompletion{},
			&stubQueryRegistry{},
			&stubActionRegistry{},
			nil,
		),
	}
	actor := testActor()
	definition := testDefinition()
	definition.ContextProviders = []agentdefinition.ContextProvider{
		agentdefinition.ContextUser,
		agentdefinition.ContextMemory,
	}

	rc, err := builder.Build(t.Context(), &serviceports.RuntimeContextRequest{
		Definition: definition,
		Actor:      actor,
		Trigger:    agent.RunTriggerChat,
		Query: serviceports.QueryVectorRequest{
			TenantInfo: actor.TenantInfo(),
			Text:       "where is S1",
		},
	})

	require.NoError(t, err)
	assert.True(t, m.orgSawVector.Load(), "the organization read waited for the embedding")
	assert.True(t, m.vectorSawOrg.Load(), "the embedding waited for the organization read")
	assert.Equal(t, "America/Chicago", rc.Timezone, "the organization's clock still wins")
	require.NotNil(t, rc.User)
	assert.Equal(t, "Dana", rc.User.Name)
	require.NotNil(t, memories.asked, "memories were still read")
}
