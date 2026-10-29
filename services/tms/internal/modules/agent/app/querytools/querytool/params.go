package querytool

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/toolschema"
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
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeInteger,
		toolschema.KeyDescription: description,
	}
}

func BoolParam(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeBoolean,
		toolschema.KeyDescription: description,
	}
}

func StringParam(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyDescription: description,
	}
}

func EnumParam(description string, source agenttoolschema.EnumSource[string]) map[string]any {
	return agenttoolschema.Enum(description, source)
}

func ObjectSchema(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyAdditionalProperties: false,
	}
	if len(required) > 0 {
		schema[toolschema.KeyRequired] = required
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
