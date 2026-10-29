package querytool

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
)

// ViewRunner reads a composed view through a table's own list: the
// repository list tools and the ones that read through GraphQL alike.
type ViewRunner interface {
	RunView(
		ctx context.Context,
		params *agenttoolports.QueryToolParams,
		query string,
		filters []domaintypes.FieldFilter,
		sorting []domaintypes.SortField,
		limit int,
	) (*ViewRun, error)
}

var (
	ErrMissingActor   = errors.New("query tool requires an actor")
	ErrTenantMismatch = errors.New("query tool parameters do not match the actor tenant")
)

// GuardQuery repeats the tenant check the write tools make. The tenant comes
// from the actor rather than from the model either way, but asserting it here
// means a future caller that builds params by hand cannot skip it silently.
func GuardQuery(params *agenttoolports.QueryToolParams) error {
	if params == nil || params.Actor == nil {
		return ErrMissingActor
	}

	if params.Actor.OrganizationID != params.OrganizationID ||
		params.Actor.BusinessUnitID != params.BusinessUnitID {
		return ErrTenantMismatch
	}

	return nil
}

func RequireString(params map[string]any, key string) (string, error) {
	raw, ok := params[key]
	if !ok {
		return "", fmt.Errorf("missing required parameter %q", key)
	}

	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("parameter %q must be a non-empty string", key)
	}

	return strings.TrimSpace(value), nil
}

// OptionalString reads a string the caller may leave out.
//
// Absent, null and the empty string all mean "not given", which is the same
// thing to every filter that takes one: a model that omits an argument and one
// that sends "" are asking for the same unfiltered result.
func OptionalString(params map[string]any, key string) string {
	raw, ok := params[key]
	if !ok || raw == nil {
		return ""
	}

	value, ok := raw.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(value)
}

// OptionalStrings reads a list of strings the caller may leave out, accepting
// every shape a model reaches for: the array, a lone string, a comma-separated
// string, and a single-key object wrapping one of those. asList already knows
// them, so the coercion lives in one place rather than per tool.
func OptionalStrings(params map[string]any, key string) []string {
	raw, ok := params[key]
	if !ok || raw == nil {
		return nil
	}

	list, ok := AsList(raw).([]any)
	if !ok {
		return nil
	}

	values := make([]string, 0, len(list))
	for _, entry := range list {
		text, isText := entry.(string)
		if text = strings.TrimSpace(text); isText && text != "" {
			values = append(values, text)
		}
	}
	if len(values) == 0 {
		return nil
	}

	return values
}

func RequirePulid(params map[string]any, key string) (pulid.ID, error) {
	value, err := RequireString(params, key)
	if err != nil {
		return pulid.Nil, err
	}

	id, err := pulid.Parse(value)
	if err != nil {
		return pulid.Nil, fmt.Errorf("parameter %q is not a valid id: %w", key, err)
	}

	return id, nil
}

// OptionalBool reads a flag the caller may leave out. Anything that is not a
// literal true is false, including a model that helpfully sends the string
// "true" — a flag that silently widens a compliance query on a type confusion
// is worse than one that stays narrow.
func OptionalBool(params map[string]any, key string) bool {
	value, _ := params[key].(bool)

	return value
}

func OptionalInt(params map[string]any, key string, fallback int) int {
	raw, ok := params[key]
	if !ok {
		return fallback
	}

	switch value := raw.(type) {
	case float64:
		return int(value)
	case int:
		return value
	case int64:
		return int(value)
	case string:
		// "30" is 30. Models send numbers as strings often enough that
		// refusing the string form turned a valid request into "needs a
		// whole number of days", which the model then retried identically.
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return fallback
		}

		return parsed
	default:
		return fallback
	}
}

// maxUnixSeconds is the last instant a seconds parameter is read as: a value
// past it is almost always milliseconds, and reading it as seconds would put
// the window centuries away and match nothing without saying why.
const maxUnixSeconds = int64(9_999_999_999)

// optionalUnixSeconds reads an instant given as whole Unix seconds. Unlike
// optionalInt it refuses what it cannot read, because a time bound dropped
// silently widens the answer to everything.
func optionalUnixSeconds(params map[string]any, key string) (int64, error) {
	raw, ok := params[key]
	if !ok || raw == nil {
		return 0, nil
	}

	var value int64
	switch typed := raw.(type) {
	case float64:
		if typed != math.Trunc(typed) {
			return 0, fmt.Errorf("parameter %q must be whole Unix seconds, not %v", key, typed)
		}
		value = int64(typed)
	case int:
		value = int64(typed)
	case int64:
		value = typed
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parameter %q must be whole Unix seconds, not %q", key, typed)
		}
		value = parsed
	default:
		return 0, fmt.Errorf("parameter %q must be whole Unix seconds", key)
	}

	switch {
	case value < 0:
		return 0, fmt.Errorf("parameter %q must not be negative", key)
	case value > maxUnixSeconds:
		return 0, fmt.Errorf(
			"parameter %q is %d, which reads as milliseconds; give Unix seconds", key, value,
		)
	}

	return value, nil
}

// OptionalObject reads a nested object the caller may leave out. A model that
// sends null rather than omitting the key means the same thing, and a tool that
// treats the two differently fails on a distinction the model cannot see.
func OptionalObject(params map[string]any, key string) map[string]any {
	value, _ := params[key].(map[string]any)

	return value
}

// DecodeParam reads a structured argument into a typed value. A model that
// sends a list of objects gets each one checked against the struct rather
// than picked apart by hand at every tool.
func DecodeParam(params map[string]any, key string, out any) error {
	raw, ok := params[key]
	if !ok {
		return fmt.Errorf("missing required parameter %q", key)
	}

	if err := jsonutils.Convert(raw, out); err != nil {
		return fmt.Errorf("parameter %q: %w", key, err)
	}

	return nil
}

func ReadDay(params map[string]any, key string, clk filtercatalog.Clock) (int64, error) {
	raw := OptionalString(params, key)
	if raw == "" {
		return 0, nil
	}

	day, ok := clk.Day(raw)
	if !ok {
		return 0, fmt.Errorf("parameter %q must be a date as YYYY-MM-DD", key)
	}

	return day, nil
}

func EndOfDay(clk filtercatalog.Clock, day int64) int64 {
	return clk.DayEnd(day)
}

func IntParam(description string) map[string]any {
	return map[string]any{"type": "integer", "description": description}
}

func BoolParam(description string) map[string]any {
	return map[string]any{"type": "boolean", "description": description}
}

func StringParam(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func EnumParam(description string, source agenttoolschema.EnumSource[string]) map[string]any {
	return agenttoolschema.Enum(description, source)
}

func ObjectSchema(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

func OptionalID(params map[string]any, key string) (pulid.ID, error) {
	id, _, err := OptionalPulid(params, key)

	return id, err
}

func ValidEnum(
	params map[string]any,
	key string,
	source agenttoolschema.EnumSource[string],
) (string, error) {
	value := OptionalString(params, key)
	if value == "" || slices.Contains(source.Values, value) {
		return value, nil
	}

	return "", fmt.Errorf(
		"%s %q is not one of %s", key, value, strings.Join(source.Values, ", "),
	)
}

// optionalEnums reads a list of enum values the caller may leave out and
// refuses any value the domain does not know, naming it, so a misspelt status
// is a correctable error rather than a filter that silently matches nothing.
func optionalEnums[T ~string](
	params map[string]any,
	key string,
	valid func(T) bool,
) ([]T, error) {
	raw := OptionalStrings(params, key)
	if len(raw) == 0 {
		return nil, nil
	}

	values := make([]T, 0, len(raw))
	for _, value := range raw {
		typed := T(value)
		if !valid(typed) {
			return nil, fmt.Errorf("%s %q is not a known value", key, value)
		}
		values = append(values, typed)
	}

	return values, nil
}

func OptionalBoolPointer(params map[string]any, key string) *bool {
	value, ok := params[key].(bool)
	if !ok {
		return nil
	}

	return &value
}

func PointerIDString(id *pulid.ID) string {
	if id == nil {
		return ""
	}

	return PulidString(*id)
}

const (
	ParamStatus         = "status"
	ParamCustomerID     = "customerId"
	ParamCarrierID      = "carrierId"
	ParamWorkerID       = "workerId"
	ParamAsOf           = "asOf"
	paramFromDate       = "fromDate"
	paramToDate         = "toDate"
	ParamStartDate      = "startDate"
	ParamFiscalPeriodID = "fiscalPeriodId"
	ParamSettlementID   = "settlementId"
	ParamPartnerID      = "partnerId"
	ParamInboundFileID  = "inboundFileId"
	ParamRateMatrixID   = "rateMatrixId"
	ParamDirection      = "direction"
	ParamSeverity       = "severity"

	LabelCustomer = "customer"
	LabelCarrier  = "carrier"

	FieldTotalAmount   = "totalAmount"
	fieldTotalMiles    = "totalMiles"
	FieldAmount        = "amount"
	fieldInputs        = "inputs"
	FieldAmountMinor   = "amountMinor"
	WithheldAmounts    = "amounts"
	withheldLineAmount = "lines.amount"

	OnThePage = "the page you are on."
)
