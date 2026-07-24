package agenttoolservice

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

var (
	ErrMissingActor          = errors.New("agent tool requires an actor")
	ErrAgentCannotApprove    = errors.New("agent principals cannot approve billing queue items")
	ErrTenantMismatch        = errors.New("tool parameters do not match the actor tenant")
	ErrMissingIdempotencyKey = errors.New("idempotency key is required for this tool")
)

func guardExecute(tool serviceports.AgentTool, params serviceports.ToolExecuteParams) error {
	if params.Actor == nil {
		return ErrMissingActor
	}

	if params.Actor.IsAgent() && tool.PermissionOperation() == permission.OpApprove {
		return ErrAgentCannotApprove
	}

	if params.Actor.OrganizationID != params.OrganizationID ||
		params.Actor.BusinessUnitID != params.BusinessUnitID {
		return ErrTenantMismatch
	}

	if tool.RequiresIdempotencyKey() && strings.TrimSpace(params.IdempotencyKey) == "" {
		return ErrMissingIdempotencyKey
	}

	return nil
}

func requireString(params map[string]any, key string) (string, error) {
	raw, ok := params[key]
	if !ok {
		return "", fmt.Errorf("missing required parameter %q", key)
	}

	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("parameter %q must be a non-empty string", key)
	}

	return value, nil
}

func optionalString(params map[string]any, key string) string {
	if raw, ok := params[key]; ok {
		if value, ok := raw.(string); ok {
			return value
		}
	}

	return ""
}

func optionalInt64(params map[string]any, key string) int64 {
	raw, ok := params[key]
	if !ok {
		return 0
	}

	switch value := raw.(type) {
	case float64:
		return int64(value)
	case int64:
		return value
	case int:
		return int64(value)
	default:
		return 0
	}
}

func requirePulid(params map[string]any, key string) (pulid.ID, error) {
	value, err := requireString(params, key)
	if err != nil {
		return pulid.Nil, err
	}

	id, err := pulid.Parse(value)
	if err != nil {
		return pulid.Nil, fmt.Errorf("parameter %q is not a valid id: %w", key, err)
	}

	return id, nil
}

func decodeParam(params map[string]any, key string, out any) error {
	raw, ok := params[key]
	if !ok {
		return fmt.Errorf("missing required parameter %q", key)
	}

	encoded, err := sonic.Marshal(raw)
	if err != nil {
		return fmt.Errorf("encode parameter %q: %w", key, err)
	}

	if err = sonic.Unmarshal(encoded, out); err != nil {
		return fmt.Errorf("decode parameter %q: %w", key, err)
	}

	return nil
}
