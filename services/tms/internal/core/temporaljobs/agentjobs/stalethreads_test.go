package agentjobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type staleThreads struct {
	repositories.ConversationRepository

	asked   []repositories.DeleteStaleThreadsRequest
	deleted map[conversation.ThreadOrigin]int
}

func (s *staleThreads) DeleteStaleThreads(
	_ context.Context,
	req repositories.DeleteStaleThreadsRequest,
) (int, error) {
	s.asked = append(s.asked, req)

	return s.deleted[req.Origin], nil
}

func TestDeleteStaleAskThreadsActivity_AlsoClearsFormulaConversationsNeverSaved(t *testing.T) {
	t.Parallel()

	threads := &staleThreads{deleted: map[conversation.ThreadOrigin]int{
		conversation.ThreadOriginAsk:     4,
		conversation.ThreadOriginFormula: 3,
	}}
	a := &Activities{logger: zap.NewNop(), conversations: threads}

	result, err := a.DeleteStaleAskThreadsActivity(t.Context(), &DeleteStaleAskThreadsInput{
		Before: 1_700_000_000,
	})
	require.NoError(t, err)

	assert.Equal(t, 7, result.Deleted)
	require.Len(t, threads.asked, 2)
	assert.Equal(t, conversation.ThreadOriginAsk, threads.asked[0].Origin)
	assert.False(t, threads.asked[0].SubjectlessOnly)
	assert.Equal(t, conversation.ThreadOriginFormula, threads.asked[1].Origin)
	assert.True(t, threads.asked[1].SubjectlessOnly,
		"a formula conversation about a saved template is kept with its template")
	for _, req := range threads.asked {
		assert.Equal(t, int64(1_700_000_000), req.Before)
		assert.Equal(t, deleteStaleAskBatch, req.Limit)
	}
}
