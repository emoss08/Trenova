package agentjobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type staleThreads struct {
	conversationports.ConversationRepository

	asked   []conversationports.DeleteStaleThreadsRequest
	deleted map[conversation.ThreadOrigin]int
}

func (s *staleThreads) DeleteStaleThreads(
	_ context.Context,
	req conversationports.DeleteStaleThreadsRequest,
) (int, error) {
	s.asked = append(s.asked, req)

	return s.deleted[req.Origin], nil
}

func TestDeleteStaleFormulaDraftsActivity_ClearsOnlyConversationsAboutUnsavedTemplates(t *testing.T) {
	t.Parallel()

	threads := &staleThreads{deleted: map[conversation.ThreadOrigin]int{
		conversation.ThreadOriginFormula: 3,
	}}
	a := &Activities{logger: zap.NewNop(), conversations: threads}

	result, err := a.DeleteStaleFormulaDraftsActivity(t.Context(), &RetentionInput{
		Before: 1_700_000_000,
	})
	require.NoError(t, err)

	assert.Equal(t, 3, result.Deleted)
	require.Len(t, threads.asked, 1)
	req := threads.asked[0]
	assert.Equal(t, conversation.ThreadOriginFormula, req.Origin)
	assert.True(t, req.SubjectlessOnly,
		"a formula conversation about a saved template is kept with its template")
	assert.Equal(t, int64(1_700_000_000), req.Before)
	assert.Equal(t, deleteStaleDraftBatch, req.Limit)
}
