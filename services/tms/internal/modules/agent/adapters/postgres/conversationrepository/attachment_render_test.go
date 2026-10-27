package conversationrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestCarriesAttachmentAsksForThePersonsMessageInThatThread(t *testing.T) {
	t.Parallel()

	threadID := pulid.MustNew("athr_")
	documentID := pulid.MustNew("doc_")
	sql, args := keeltest.Render(t, carriesAttachmentQuery(&agentports.CarriesAttachmentRequest{
		ThreadID:   threadID,
		DocumentID: documentID,
	}))

	assert.Contains(t, sql, `am."thread_id" = $3`, sql)
	assert.Contains(t, sql, `am."role" = 'User'`, sql)
	assert.Contains(t, sql, `am."attachments" @> $4`, sql)
	assert.Equal(t, []any{
		threadID.String(),
		[]map[string]string{{"documentId": documentID.String()}},
	}, args[2:4])
}

func TestMessagesSinceCompactionReadAfterTheLatestSummary(t *testing.T) {
	t.Parallel()

	threadID := pulid.MustNew("athr_")
	sql, args := keeltest.Render(t, messagesQuery(&agentports.ListMessagesRequest{
		ThreadID:        threadID,
		SinceCompaction: true,
	}))

	assert.Contains(t, sql, `am."sequence" > coalesce((SELECT CAST((compacted."compaction" #>> $4) AS bigint)`, sql)
	assert.Contains(t, sql, `compacted."thread_id" = $3 AND compacted."kind" = 'Compaction'`, sql)
	assert.Contains(t, sql, `ORDER BY compacted."sequence" DESC LIMIT 1), $5)`, sql)
	assert.NotContains(t, args, string(conversation.MessageKindCompaction))
	assert.Equal(t, -1, args[4])
}
