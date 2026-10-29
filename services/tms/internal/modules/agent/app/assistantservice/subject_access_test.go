package assistantservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/assistanttest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingSubjects struct {
	described int
}

func (r *recordingSubjects) Describe(
	_ context.Context,
	_ pagination.TenantInfo,
	subjectType agent.SubjectType,
	subjectID pulid.ID,
) (*agentturn.RuntimeSubject, error) {
	r.described++

	return &agentturn.RuntimeSubject{
		Type:  subjectType,
		ID:    subjectID.String(),
		Label: "Driver Ada Lovelace",
		Notes: `{"status":"Active"}`,
	}, nil
}

// Access is checked on every turn: a person who lost access keeps the
// conversation, but the record is no longer read into it.
func TestDescribeSubject_WithholdsARecordThePersonCanNoLongerRead(t *testing.T) {
	t.Parallel()

	subjects := &recordingSubjects{}
	svc := &Service{
		logger: zap.NewNop(),
		permissions: &assistanttest.SubjectPermissions{
			Allowed: map[string]bool{"assistant:create": true},
		},
		subjects: subjects,
	}
	thread := &conversation.Thread{
		ID:          pulid.MustNew("athr_"),
		SubjectType: agent.SubjectWorker,
		SubjectID:   pulid.MustNew("wrk_"),
	}

	subject := svc.describeSubject(
		t.Context(),
		thread,
		assistanttest.Actor(),
		pagination.TenantInfo{},
	)

	require.NotNil(t, subject)
	assert.Empty(t, subject.Notes, "only the kind and id are named")
	assert.Zero(t, subjects.described, "the record is never loaded")

	svc.permissions = &assistanttest.SubjectPermissions{Allowed: map[string]bool{
		"assistant:create": true,
		"worker:read":      true,
	}}
	subject = svc.describeSubject(
		t.Context(),
		thread,
		assistanttest.Actor(),
		pagination.TenantInfo{},
	)
	require.NotNil(t, subject)
	assert.NotEmpty(t, subject.Notes)
	assert.Equal(t, 1, subjects.described)
}

var _ conversationports.ConversationRepository = (*assistanttest.CreatingConversations)(nil)
