package agentquerytoolservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	wfFieldDescription = "description"
	wfFieldNotes       = "notes"
	wfFieldReason      = "reason"
	wfFieldResolution  = "resolution"
	wfParamOpenOnly    = "openOnly"
	wfParamClosed      = "includeClosed"
	absentNotCompleted = "not completed"
	absentNotDue       = "no due date"
	absentNotExpiring  = "does not expire"
	absentNotReturned  = "not back yet"
	absentNotRequested = "not requested"
	absentNotFinal     = "not finalized"
	absentNotReceived  = "not received"
)

type workforceRead struct {
	name        string
	description string
	searchTerms []string
	resource    permission.Resource
	rationale   string
	reads       agent.ExternalRead
	source      agent.TaintSource
	properties  map[string]any
	required    []string
	access      fieldAccess
	run         func(
		ctx context.Context,
		params *serviceports.QueryToolParams,
		gate *fieldGate,
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

func (t *workforceRead) Policy() serviceports.ToolPolicy {
	return readPolicy(t.name, readSpec{
		resource:  t.resource,
		reads:     t.reads,
		source:    t.source,
		rationale: t.rationale,
	})
}

func (t *workforceRead) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	return t.run(ctx, params, t.access.gate(ctx, params, t.resource))
}

func wfWorkerProperty(what string) map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeString,
		toolschema.KeyDescription: what + ", by id from search_worker or list_workers. A " +
			"name is not an id.",
		toolschema.KeyRecordOf: permission.ResourceWorker.String(),
	}
}

func wfBoolProperty(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeBoolean,
		toolschema.KeyDescription: description,
	}
}

func optionalWorker(params map[string]any) (pulid.ID, error) {
	if optionalString(params, paramWorkerID) == "" {
		return pulid.Nil, nil
	}

	return requirePulid(params, paramWorkerID)
}

func gatedText(gate *fieldGate, field, value string) string {
	if value == "" || !gate.show(field, field) {
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

func workforceQueryToolProviders() []any {
	return []any{
		provideListWorkerSafetyEventsTool,
		provideListDOTTestsTool,
		provideListDOTRandomDrawsTool,
		provideGetDOTRandomDrawTool,
		provideListWorkerLeaveCasesTool,
		provideListWorkerTrainingTool,
		provideListTrainingCoursesTool,
		provideListWorkerChecklistsTool,
		provideListPerformanceReviewsTool,
		provideListEmploymentVerificationsTool,
		provideListWorkerCredentialsTool,
		provideListWorkerInjuriesTool,
		provideListShipmentPermitsTool,
		provideListDriverExpensesTool,
		provideListPayrollExportsTool,
	}
}
