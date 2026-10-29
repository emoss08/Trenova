package assistantthreads

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func subjectService(
	definition *agentdefinition.Definition,
	allowed ...string,
) (*Service, *assistanttest.CreatingConversations, *assistanttest.SubjectPermissions) {
	permissions := &assistanttest.SubjectPermissions{
		Allowed: map[string]bool{"assistant:create": true},
	}
	for _, key := range allowed {
		permissions.Allowed[key] = true
	}
	conversations := &assistanttest.CreatingConversations{}

	return &Service{
		logger:        zap.NewNop(),
		conversations: conversations,
		definitions:   &assistanttest.StubDefinitions{Definition: definition},
		permissions:   permissions,
	}, conversations, permissions
}

func workerThread() *assistantports.StartThreadRequest {
	return &assistantports.StartThreadRequest{
		AgentDefinitionID: pulid.MustNew("agdef_"),
		SubjectType:       agent.SubjectWorker,
		SubjectID:         pulid.MustNew("wrk_"),
	}
}

/*
A conversation about a record reads that record into the prompt on every turn.
Opening one about a record the person cannot read used to be allowed: the
subject was stored as sent and described by tenant alone, so a person with
the assistant and nothing else could have a driver's file read to them.
*/
func TestStartThread_RefusesASubjectThePersonCannotRead(t *testing.T) {
	t.Parallel()

	svc, conversations, permissions := subjectService(assistanttest.ChatDefinition("Dispatch", ""))

	_, err := startAs(t, svc, workerThread(), assistanttest.Actor())

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, errortypes.ErrForbidden, validation.Code)
	assert.Nil(t, conversations.Created, "nothing is stored for a refused subject")
	assert.Equal(t, []string{"worker:read"}, permissions.Asked)
}

func TestStartThread_OpensAConversationAboutARecordThePersonMayRead(t *testing.T) {
	t.Parallel()

	svc, conversations, _ := subjectService(
		assistanttest.ChatDefinition("Dispatch", ""),
		"worker:read",
	)

	thread, err := startAs(t, svc, workerThread(), assistanttest.Actor())

	require.NoError(t, err)
	require.NotNil(t, conversations.Created)
	assert.Equal(t, agent.SubjectWorker, thread.SubjectType)
}

// A conversation about the organization, or with no subject, names no record
// and needs no grant beyond the assistant itself.
func TestStartThread_NeedsNoGrantWithoutARecord(t *testing.T) {
	t.Parallel()

	svc, _, permissions := subjectService(assistanttest.ChatDefinition("Dispatch", ""))

	_, err := startAs(t, svc, &assistantports.StartThreadRequest{
		AgentDefinitionID: pulid.MustNew("agdef_"),
	}, assistanttest.Actor())

	require.NoError(t, err)
	assert.Empty(t, permissions.Asked)
}

// A desk's autonomy was earned on its own work. A person talking to it would
// borrow that autonomy for writes of their own choosing.
func TestStartThread_RefusesAnAgentThatRunsOnItsOwn(t *testing.T) {
	t.Parallel()

	desk := assistanttest.ChatDefinition("Intake desk", agenttemplate.TemplateIntakeDesk)
	desk.TriggerMode = agentdefinition.TriggerEvent
	svc, conversations, _ := subjectService(desk)

	_, err := startAs(t, svc, &assistantports.StartThreadRequest{
		AgentDefinitionID: desk.ID,
	}, assistanttest.Actor())

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Nil(t, conversations.Created)
}
