package agenttoolservice

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
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

// optionalBool reads a flag the caller may leave out. Only a literal true
// counts; a model that sends the string "true" gets the safe default.
func optionalBool(params map[string]any, key string) bool {
	value, _ := params[key].(bool)

	return value
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

	if err := jsonutils.Convert(raw, out); err != nil {
		return fmt.Errorf("parameter %q: %w", key, err)
	}

	return nil
}

// tenantFrom builds the tenant envelope every write travels in.
//
// The organization, the business unit and the acting user all come from the
// authenticated actor, never from the model's arguments. guardExecute has
// already asserted that the actor and the params agree, so there is one place
// a tool could get this wrong and it is not here.
func tenantFrom(params serviceports.ToolExecuteParams) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  params.OrganizationID,
		BuID:   params.BusinessUnitID,
		UserID: params.Actor.UserID,
	}
}

// requirePulidSlice reads a bounded list of ids.
//
// The cap is enforced here rather than left to the schema, because a schema's
// maxItems is a hint to the model and nothing more: the tool is what decides
// how much of a fleet, or a ledger, one call may touch.
func requirePulidSlice(params map[string]any, key string, limit int) ([]pulid.ID, error) {
	raw, ok := params[key]
	if !ok {
		return nil, fmt.Errorf("missing required parameter %q", key)
	}

	values, ok := raw.([]any)
	if !ok || len(values) == 0 {
		return nil, fmt.Errorf("parameter %q must be a non-empty array of ids", key)
	}

	if len(values) > limit {
		return nil, fmt.Errorf(
			"parameter %q holds %d ids, which is more than the %d this tool will change "+
				"at once; split it into smaller calls", key, len(values), limit,
		)
	}

	ids := make([]pulid.ID, 0, len(values))
	for index, value := range values {
		text, ok := value.(string)
		if !ok || strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf("parameter %q[%d] must be a non-empty id", key, index)
		}

		id, err := pulid.Parse(strings.TrimSpace(text))
		if err != nil {
			return nil, fmt.Errorf("parameter %q[%d] is not a valid id: %w", key, index, err)
		}

		ids = append(ids, id)
	}

	return ids, nil
}

// targetOf names the record a call acts on, from the argument that carries its
// id. A missing or malformed id yields no target, and the call is still made:
// the tool's own Execute reports the bad argument in its own words, and a
// proposal without a target simply skips the staleness check.
func targetOf(
	params map[string]any,
	key string,
	resource permission.Resource,
) (serviceports.ToolTarget, bool) {
	raw, _ := params[key].(string)
	id, err := pulid.Parse(raw)
	if err != nil || id.IsNil() {
		return serviceports.ToolTarget{}, false
	}

	return serviceports.ToolTarget{Resource: resource, ID: id}, true
}

// requireMoney reads an amount the model wrote as a decimal string or a
// number, in major units, and returns it in minor units. It must be
// positive: a zero or negative amount is never what a payment means.
func requireMoney(params map[string]any, key string) (int64, error) {
	minor, present, err := optionalMoney(params, key)
	if err != nil {
		return 0, err
	}
	if !present {
		return 0, fmt.Errorf("parameter %q is required", key)
	}

	return minor, nil
}

func optionalMoney(params map[string]any, key string) (int64, bool, error) {
	raw, ok := params[key]
	if !ok || raw == nil {
		return 0, false, nil
	}

	var value decimal.Decimal
	switch typed := raw.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0, false, nil
		}
		parsed, err := decimal.NewFromString(trimmed)
		if err != nil {
			return 0, false, fmt.Errorf("parameter %q must be a decimal amount such as 1250.00", key)
		}
		value = parsed
	case float64:
		value = decimal.NewFromFloat(typed)
	case int:
		value = decimal.NewFromInt(int64(typed))
	case int64:
		value = decimal.NewFromInt(typed)
	default:
		return 0, false, fmt.Errorf("parameter %q must be a decimal amount such as 1250.00", key)
	}

	if !value.IsPositive() {
		return 0, false, fmt.Errorf("parameter %q must be greater than zero", key)
	}

	return money.MinorUnits(value), true, nil
}

// requireDay reads a YYYY-MM-DD as the start of that day in UTC, which is
// how a date-only value is stored on a payment.
func requireDay(params map[string]any, key string) (int64, error) {
	raw, err := requireString(params, key)
	if err != nil {
		return 0, err
	}

	day, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("parameter %q must be YYYY-MM-DD, got %q", key, raw)
	}

	return day.Unix(), nil
}
