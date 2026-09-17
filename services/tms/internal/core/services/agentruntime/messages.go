package agentruntime

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/stringutils"
)

const maxRationaleChars = 600

func toToolCallRecords(calls []serviceports.ToolCall) []conversation.ToolCallRecord {
	if len(calls) == 0 {
		return nil
	}

	records := make([]conversation.ToolCallRecord, 0, len(calls))
	for _, call := range calls {
		records = append(records, conversation.ToolCallRecord{
			ID:        call.ID,
			Name:      call.Name,
			Arguments: call.Arguments,
		})
	}

	return records
}

func fromToolCallRecords(records []conversation.ToolCallRecord) []serviceports.ToolCall {
	if len(records) == 0 {
		return nil
	}

	calls := make([]serviceports.ToolCall, 0, len(records))
	for _, record := range records {
		calls = append(calls, serviceports.ToolCall{
			ID:        record.ID,
			Name:      record.Name,
			Arguments: record.Arguments,
		})
	}

	return calls
}

func toAdapterMessages(history []conversation.Message) []serviceports.Message {
	messages := make([]serviceports.Message, 0, len(history)+1)

	for _, msg := range history {
		switch msg.Role {
		case conversation.RoleAssistant:
			if msg.Refused {
				continue
			}
			messages = append(messages, serviceports.Message{
				Role:      serviceports.RoleAssistant,
				Content:   msg.Content,
				ToolCalls: fromToolCallRecords(msg.ToolCalls),
			})
		case conversation.RoleTool:
			messages = append(messages, serviceports.Message{
				Role:       serviceports.RoleTool,
				Content:    msg.Content,
				ToolCallID: msg.ToolCallID,
				ToolName:   msg.ToolName,
				IsError:    msg.ToolFailed,
			})
		default:
			if msg.Refused {
				continue
			}
			messages = append(messages, serviceports.Message{
				Role:    serviceports.RoleUser,
				Content: msg.Content,
			})
		}
	}

	return messages
}

func proposalRationale(completionText, toolName string) string {
	trimmed := strings.TrimSpace(completionText)
	if trimmed == "" {
		return fmt.Sprintf(
			"The agent asked to run %s without explaining why.",
			toolName,
		)
	}

	return stringutils.Ellipsize(trimmed, maxRationaleChars)
}
