package assistanttest

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationchapter"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantchapterports"
)

type StubChapters struct {
	assistantchapterports.ChapterRepository

	Chapters []*conversationchapter.Chapter
	Err      error
}

func (s *StubChapters) List(
	context.Context,
	assistantchapterports.ListChaptersRequest,
) ([]*conversationchapter.Chapter, error) {
	return s.Chapters, s.Err
}

func SpokenStretch(from, to int) []conversation.Message {
	messages := make([]conversation.Message, 0, to-from+1)
	for sequence := from; sequence <= to; sequence++ {
		role := conversation.RoleUser
		if sequence%2 == 1 {
			role = conversation.RoleAssistant
		}
		messages = append(messages, conversation.Message{
			Sequence: sequence,
			Role:     role,
			Kind:     conversation.MessageKindMessage,
			Content:  fmt.Sprintf("m%d", sequence),
		})
	}

	return messages
}

func EarlierSummary(sequence, through int) conversation.Message {
	return conversation.Message{
		Sequence:   sequence,
		Role:       conversation.RoleUser,
		Kind:       conversation.MessageKindCompaction,
		Content:    "- Looked at the late loads.",
		Compaction: &conversation.Compaction{Through: through},
	}
}

func StartedAt(sequence int64) *int64 { return &sequence }
