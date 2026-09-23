package agentevalcaseservice

import (
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
)

const wholeResult = "result"

type sensitivityReader interface {
	GetFieldSensitivity(resource, field string) permission.FieldSensitivity
}

type Redactor struct {
	sensitivity sensitivityReader
	queryTools  serviceports.AgentQueryToolRegistry
	actionTools serviceports.AgentToolRegistry
}

func NewRedactor(
	sensitivity sensitivityReader,
	queryTools serviceports.AgentQueryToolRegistry,
	actionTools serviceports.AgentToolRegistry,
) *Redactor {
	return &Redactor{sensitivity: sensitivity, queryTools: queryTools, actionTools: actionTools}
}

type redaction struct {
	fields []agentquality.RedactedField
}

func (r *redaction) record(tool, path string, sensitivity permission.FieldSensitivity) {
	r.fields = append(r.fields, agentquality.RedactedField{
		Tool:        tool,
		Path:        path,
		Sensitivity: sensitivity.String(),
	})
}

func (r *Redactor) resourceOf(tool string) (permission.Resource, bool) {
	if r.queryTools != nil {
		if query, ok := r.queryTools.Get(tool); ok {
			return query.PermissionResource(), true
		}
	}
	if r.actionTools != nil {
		if action, ok := r.actionTools.Get(tool); ok {
			return action.PermissionResource(), true
		}
	}

	return "", false
}

func (r *Redactor) value(tool string, value any, out *redaction) any {
	if value == nil || agentquality.IsRuntimeTool(tool) {
		return value
	}

	resource, known := r.resourceOf(tool)
	if !known {
		out.record(tool, wholeResult, permission.SensitivityRestricted)

		return agentquality.RestrictedPlaceholder(wholeResult)
	}

	return r.walk(resource.String(), tool, "", value, out)
}

func (r *Redactor) args(tool string, args map[string]any, out *redaction) map[string]any {
	if len(args) == 0 {
		return args
	}

	redacted, ok := r.value(tool, args, out).(map[string]any)
	if !ok {
		return map[string]any{wholeResult: agentquality.RestrictedPlaceholder(wholeResult)}
	}

	return redacted
}

func (r *Redactor) walk(resource, tool, path string, value any, out *redaction) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, nested := range typed {
			fieldPath := joinPath(path, key)
			if sensitivity, restricted := r.restricted(resource, key, fieldPath); restricted {
				out.record(tool, fieldPath, sensitivity)
				result[key] = agentquality.RestrictedPlaceholder(key)

				continue
			}
			result[key] = r.walk(resource, tool, fieldPath, nested, out)
		}

		return result
	case []any:
		result := make([]any, len(typed))
		for i, nested := range typed {
			result[i] = r.walk(resource, tool, path, nested, out)
		}

		return result
	default:
		return value
	}
}

func (r *Redactor) restricted(
	resource, field, path string,
) (permission.FieldSensitivity, bool) {
	sensitivity := r.sensitivity.GetFieldSensitivity(resource, field)
	if path != field {
		if nested := r.sensitivity.GetFieldSensitivity(resource, path); nested.Level() >
			sensitivity.Level() {
			sensitivity = nested
		}
	}

	return sensitivity, sensitivity.Level() >= permission.SensitivityRestricted.Level()
}

func (r *Redactor) toolContent(tool, content string, out *redaction) string {
	name, payload, fenced := agentruntime.UnfenceToolResult(content)
	if !fenced {
		return content
	}
	if name == "" {
		name = tool
	}

	return agentruntime.FenceToolResult(name, r.payload(name, payload, out))
}

func (r *Redactor) payload(tool, payload string, out *redaction) string {
	if agentquality.IsRuntimeTool(tool) {
		return payload
	}

	var decoded any
	if err := sonic.UnmarshalString(payload, &decoded); err != nil {
		out.record(tool, wholeResult, permission.SensitivityRestricted)

		return agentquality.RestrictedPlaceholder(wholeResult)
	}

	encoded, err := sonic.MarshalString(r.value(tool, decoded, out))
	if err != nil {
		out.record(tool, wholeResult, permission.SensitivityRestricted)

		return agentquality.RestrictedPlaceholder(wholeResult)
	}

	return encoded
}

func (r *Redactor) decodedResult(tool, content string, failed bool, out *redaction) any {
	if failed {
		return strings.TrimSpace(content)
	}

	name, payload, fenced := agentruntime.UnfenceToolResult(content)
	if !fenced {
		payload = content
		name = tool
	}
	if agentquality.IsRuntimeTool(name) {
		return nil
	}

	var decoded any
	if err := sonic.UnmarshalString(payload, &decoded); err != nil {
		out.record(tool, wholeResult, permission.SensitivityRestricted)

		return agentquality.RestrictedPlaceholder(wholeResult)
	}

	return r.value(tool, decoded, out)
}

func (r *Redactor) History(
	messages []agentquality.HistoryMessage,
	out *redaction,
) []agentquality.HistoryMessage {
	redacted := make([]agentquality.HistoryMessage, 0, len(messages))
	for _, message := range messages {
		switch message.Role {
		case conversation.RoleTool:
			if !message.ToolFailed {
				message.Content = r.toolContent(message.ToolName, message.Content, out)
			}
		case conversation.RoleAssistant:
			calls := make([]conversation.ToolCallRecord, 0, len(message.ToolCalls))
			for _, call := range message.ToolCalls {
				call.Arguments = r.args(call.Name, call.Arguments, out)
				calls = append(calls, call)
			}
			message.ToolCalls = calls
		}
		redacted = append(redacted, message)
	}

	return redacted
}

func joinPath(parent, key string) string {
	if parent == "" {
		return key
	}

	return parent + "." + key
}
