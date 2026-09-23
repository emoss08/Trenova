package assistantservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type subjectPermissions struct {
	serviceports.PermissionEngine

	allowed map[string]bool
	asked   []string
}

func (p *subjectPermissions) Check(
	_ context.Context,
	req *serviceports.PermissionCheckRequest,
) (*serviceports.PermissionCheckResult, error) {
	key := req.Resource + ":" + string(req.Operation)
	p.asked = append(p.asked, key)

	return &serviceports.PermissionCheckResult{Allowed: p.allowed[key]}, nil
}

type recordingSubjects struct {
	described int
}

func (r *recordingSubjects) Describe(
	_ context.Context,
	_ pagination.TenantInfo,
	subjectType agent.SubjectType,
	subjectID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	r.described++

	return &agentdefinition.RuntimeSubject{
		Type:  subjectType,
		ID:    subjectID.String(),
		Label: "Driver Ada Lovelace",
		Notes: `{"status":"Active"}`,
	}, nil
}

func subjectService(
	definition *agentdefinition.Definition,
	allowed ...string,
) (*Service, *creatingConversations, *subjectPermissions) {
	permissions := &subjectPermissions{allowed: map[string]bool{}}
	for _, key := range allowed {
		permissions.allowed[key] = true
	}
	conversations := &creatingConversations{}

	return &Service{
		logger:        zap.NewNop(),
		conversations: conversations,
		definitions:   &stubDefinitions{definition: definition},
		permissions:   permissions,
		subjects:      &recordingSubjects{},
	}, conversations, permissions
}

func workerThread() *serviceports.StartThreadRequest {
	return &serviceports.StartThreadRequest{
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

	svc, conversations, permissions := subjectService(chatDefinition("Dispatch", ""))

	_, err := svc.StartThread(t.Context(), workerThread(), testActor())

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, errortypes.ErrForbidden, validation.Code)
	assert.Nil(t, conversations.created, "nothing is stored for a refused subject")
	assert.Equal(t, []string{"worker:read"}, permissions.asked)
}

func TestStartThread_OpensAConversationAboutARecordThePersonMayRead(t *testing.T) {
	t.Parallel()

	svc, conversations, _ := subjectService(chatDefinition("Dispatch", ""), "worker:read")

	thread, err := svc.StartThread(t.Context(), workerThread(), testActor())

	require.NoError(t, err)
	require.NotNil(t, conversations.created)
	assert.Equal(t, agent.SubjectWorker, thread.SubjectType)
}

// A conversation about the organization, or with no subject, names no record
// and needs no grant beyond the assistant itself.
func TestStartThread_NeedsNoGrantWithoutARecord(t *testing.T) {
	t.Parallel()

	svc, _, permissions := subjectService(chatDefinition("Dispatch", ""))

	_, err := svc.StartThread(t.Context(), &serviceports.StartThreadRequest{
		AgentDefinitionID: pulid.MustNew("agdef_"),
	}, testActor())

	require.NoError(t, err)
	assert.Empty(t, permissions.asked)
}

// A desk's autonomy was earned on its own work. A person talking to it would
// borrow that autonomy for writes of their own choosing.
func TestStartThread_RefusesAnAgentThatRunsOnItsOwn(t *testing.T) {
	t.Parallel()

	desk := chatDefinition("Intake desk", agentdefinition.TemplateIntakeDesk)
	desk.TriggerMode = agentdefinition.TriggerEvent
	svc, conversations, _ := subjectService(desk)

	_, err := svc.StartThread(t.Context(), &serviceports.StartThreadRequest{
		AgentDefinitionID: desk.ID,
	}, testActor())

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Nil(t, conversations.created)
}

// Access is checked on every turn: a person who lost access keeps the
// conversation, but the record is no longer read into it.
func TestDescribeSubject_WithholdsARecordThePersonCanNoLongerRead(t *testing.T) {
	t.Parallel()

	svc, _, _ := subjectService(chatDefinition("Dispatch", ""))
	subjects := &recordingSubjects{}
	svc.subjects = subjects
	thread := &conversation.Thread{
		ID:          pulid.MustNew("athr_"),
		SubjectType: agent.SubjectWorker,
		SubjectID:   pulid.MustNew("wrk_"),
	}

	subject := svc.describeSubject(t.Context(), thread, testActor(), pagination.TenantInfo{})

	require.NotNil(t, subject)
	assert.Empty(t, subject.Notes, "only the kind and id are named")
	assert.Zero(t, subjects.described, "the record is never loaded")

	svc.permissions = &subjectPermissions{allowed: map[string]bool{"worker:read": true}}
	subject = svc.describeSubject(t.Context(), thread, testActor(), pagination.TenantInfo{})
	require.NotNil(t, subject)
	assert.NotEmpty(t, subject.Notes)
	assert.Equal(t, 1, subjects.described)
}

var _ repositories.ConversationRepository = (*creatingConversations)(nil)
