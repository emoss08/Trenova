package workforcequerytools

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	wfFieldDescription = "description"
	wfFieldNotes       = "notes"
	wfFieldReason      = "reason"
	wfFieldResolution  = "resolution"
	absentNotDue       = "no due date"
	absentNotExpiring  = "does not expire"
	absentNotReturned  = "not back yet"
	absentNotRequested = "not requested"
	absentNotReceived  = "not received"
)

type workforceRead struct {
	name        string
	description string
	searchTerms []string
	resource    permission.Resource
	rationale   string
	reads       agenttooltrust.ExternalRead
	source      agent.TaintSource
	properties  map[string]any
	required    []string
	access      fieldgate.Access
	run         func(
		ctx context.Context,
		params *agenttoolports.QueryToolParams,
		gate *fieldgate.Gate,
	) (any, error)
}

func (t *workforceRead) Name() string { return t.name }

func (t *workforceRead) Description() string { return t.description }

func (t *workforceRead) SearchTerms() []string { return t.searchTerms }

func (t *workforceRead) ParamSchema() map[string]any {
	schema := map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           t.properties,
		toolschema.KeyAdditionalProperties: false,
	}
	if len(t.required) > 0 {
		schema[toolschema.KeyRequired] = t.required
	}

	return schema
}

func (t *workforceRead) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.name, querytool.ReadSpec{
		Resource:  t.resource,
		Reads:     t.reads,
		Source:    t.source,
		Rationale: t.rationale,
	})
}

func (t *workforceRead) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	return t.run(ctx, params, t.access.Gate(ctx, params, t.resource))
}

func wfWorkerProperty(what string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyDescription: what + ", by id from worker_list. A name is not an id.",
		toolschema.KeyRecordOf:    permission.ResourceWorker.String(),
	}
}

func wfBoolProperty(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeBoolean,
		toolschema.KeyDescription: description,
	}
}

func optionalWorker(params map[string]any) (pulid.ID, error) {
	if querytool.OptionalString(params, querytool.ParamWorkerID) == "" {
		return pulid.Nil, nil
	}

	return querytool.RequirePulid(params, querytool.ParamWorkerID)
}

func gatedText(gate *fieldgate.Gate, field, value string) string {
	if value == "" || !gate.Show(field, field) {
		return ""
	}

	return value
}

func decimalText(value decimal.NullDecimal) string {
	if !value.Valid {
		return ""
	}

	return value.Decimal.String()
}

func boolRef(value *bool) string {
	switch {
	case value == nil:
		return ""
	case *value:
		return "yes"
	default:
		return "no"
	}
}

func joinStrings[T ~string](values []T) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, string(value))
	}

	return strings.Join(parts, ", ")
}

func Providers() []any {
	return []any{
		provideListShipmentPermitsTool,
		provideListPayrollExportsTool,
	}
}
