package conversation

import (
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/shared/sliceutils"
)

func RunTranscriptOf(messages []Message) *agent.RunTranscript {
	if len(messages) == 0 {
		return nil
	}

	entries := make([]agent.TranscriptMessage, len(messages))
	weights := make([]int, len(messages))
	for idx := range messages {
		entries[idx], weights[idx] = boundedTranscriptMessage(&messages[idx])
	}

	head, tail := sliceutils.KeepEnds(weights, agent.TranscriptBytes)
	omitted := len(entries) - head - tail
	if omitted == 0 {
		return &agent.RunTranscript{Messages: entries}
	}

	kept := make([]agent.TranscriptMessage, 0, head+tail)
	kept = append(kept, entries[:head]...)
	kept = append(kept, entries[len(entries)-tail:]...)

	return &agent.RunTranscript{
		Messages:        kept,
		OmittedMessages: omitted,
		OmittedAt:       head,
	}
}

func boundedTranscriptMessage(message *Message) (entry agent.TranscriptMessage, size int) {
	entry = transcriptMessageOf(message)
	size = encodedSize(&entry)
	if size <= agent.TranscriptMessageBytes {
		return entry, size
	}

	entry = entry.WithoutBody()
	size = encodedSize(&entry)

	return entry, size
}

func transcriptMessageOf(message *Message) agent.TranscriptMessage {
	entry := agent.TranscriptMessage{
		Role:              string(message.Role),
		Kind:              string(message.Kind),
		Content:           message.Content,
		ToolCallID:        message.ToolCallID,
		ToolName:          message.ToolName,
		ToolFailed:        message.ToolFailed,
		ToolVerdict:       message.ToolVerdict,
		ToolSummary:       message.ToolSummary,
		AgentDefinitionID: message.AgentDefinitionID,
		DelegateCallID:    message.DelegateCallID,
		CreatedAt:         message.CreatedAt,
	}
	if message.Reasoning.Readable() {
		entry.Reasoning = message.Reasoning.Text
	}
	if len(message.ToolCalls) > 0 {
		entry.ToolCalls = make([]agent.TranscriptToolCall, len(message.ToolCalls))
		for idx, call := range message.ToolCalls {
			entry.ToolCalls[idx] = agent.TranscriptToolCall{
				ID:        call.ID,
				Name:      call.Name,
				Arguments: call.Arguments,
			}
		}
	}

	return entry
}

func encodedSize(entry *agent.TranscriptMessage) int {
	encoded, err := sonic.Marshal(entry)
	if err != nil {
		return agent.TranscriptMessageBytes + 1
	}

	return len(encoded)
}

func MessageOfTranscript(entry *agent.TranscriptMessage) Message {
	message := Message{
		Role:              Role(entry.Role),
		Kind:              MessageKind(entry.Kind),
		Content:           entry.Content,
		ToolCallID:        entry.ToolCallID,
		ToolName:          entry.ToolName,
		ToolFailed:        entry.ToolFailed,
		ToolVerdict:       entry.ToolVerdict,
		ToolSummary:       entry.ToolSummary,
		AgentDefinitionID: entry.AgentDefinitionID,
		DelegateCallID:    entry.DelegateCallID,
		CreatedAt:         entry.CreatedAt,
	}
	if message.Kind == "" {
		message.Kind = MessageKindMessage
	}
	if entry.Reasoning != "" {
		message.Reasoning = &ReasoningTrace{Text: entry.Reasoning}
	}
	if len(entry.ToolCalls) > 0 {
		message.ToolCalls = make([]ToolCallRecord, len(entry.ToolCalls))
		for idx, call := range entry.ToolCalls {
			message.ToolCalls[idx] = ToolCallRecord{
				ID:        call.ID,
				Name:      call.Name,
				Arguments: call.Arguments,
			}
		}
	}

	return message
}
