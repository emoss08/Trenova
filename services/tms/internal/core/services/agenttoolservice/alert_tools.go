package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/shared/pulid"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/domain/tablechangealert"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/internal/core/services/tablechangealertservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/cronutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

/*
The two ways of being told rather than having to look.

A report on a schedule answers a question every week without anybody asking
it. A change alert answers one the moment the answer changes. Between them
they are most of what "let me know when..." means, and both were surfaces
somebody had to find, open and fill in a form on — which is why almost nobody
has one.

Neither invents its trigger. A cron expression is parsed before it is stored,
because a schedule that never fires is indistinguishable from one that has not
fired yet, and a person finds out weeks later that the report they asked for
has been silent the whole time.
*/

type scheduleWriter interface {
	CreateSchedule(
		ctx context.Context,
		req *reporting.SaveScheduleRequest,
	) (*report.ReportSchedule, error)
	GetDefinition(
		ctx context.Context,
		req *reporting.GetDefinitionRequest,
	) (*report.ReportDefinition, error)
}

type scheduleReportTool struct {
	schedules scheduleWriter
}

func newScheduleReportTool(schedules *reporting.Service) serviceports.AgentTool {
	return &scheduleReportTool{schedules: schedules}
}

func (t *scheduleReportTool) Name() string { return "schedule_report" }

func (t *scheduleReportTool) Description() string {
	return "Put a saved report on a schedule and email it to people. Use it when somebody " +
		"wants a number regularly rather than now — \"send me the unbilled aging every " +
		"Monday morning\". The report has to exist already; list_reports finds its id. " +
		"Times are read in the timezone given, so \"Monday morning\" means theirs."
}

func (t *scheduleReportTool) Prerequisites() []string {
	return []string{"list_reports"}
}

func (t *scheduleReportTool) ParamSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"definitionId", "cronExpression", "emailRecipients"},
		"properties": map[string]any{
			"definitionId": map[string]any{
				"type":        "string",
				"description": "The saved report to run, from list_reports.",
			},
			"cronExpression": map[string]any{
				"type": "string",
				"description": "When to run it, as five cron fields. " +
					"\"0 7 * * 1\" is 07:00 every Monday.",
			},
			"timezone": map[string]any{
				"type": "string",
				"description": "The zone the schedule is read in, as an IANA name " +
					"(America/New_York). Defaults to the organization's.",
			},
			"emailRecipients": map[string]any{
				"type":        "array",
				"maxItems":    20,
				"items":       map[string]any{"type": "string"},
				"description": "The email addresses it goes to.",
			},
			"formats": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Which formats to attach. Defaults to the report's own.",
			},
			"attach": map[string]any{
				"type": "boolean",
				"description": "Attach the file rather than only linking to it. " +
					"Defaults to true.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *scheduleReportTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceReport,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		Idempotent:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Emails a report on a schedule to whatever addresses the call names, " +
			"which may be outside the organization.",
	}
}

func (t *scheduleReportTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := t.validateArgs(params.Params); err != nil {
		return err
	}

	// The report is looked up now, not when the schedule first fires: a
	// schedule on a report that does not exist is refused at save anyway,
	// and after approval is too late for the model to find the right one.
	raw := optionalString(params.Params, "definitionId")
	definitionID, err := pulid.Parse(raw)
	if err == nil {
		_, err = t.schedules.GetDefinition(ctx, &reporting.GetDefinitionRequest{
			Request:      reporting.Request{TenantInfo: tenantFrom(params)},
			DefinitionID: definitionID,
		})
	}
	if err != nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("definitionId", errortypes.ErrInvalid, fmt.Sprintf(
			"%q is not a saved report you can open; find the one you mean with list_reports",
			raw,
		))
		return multiErr
	}

	return nil
}

func (t *scheduleReportTool) validateArgs(params map[string]any) error {
	multiErr := errortypes.NewMultiError()

	if optionalString(params, "definitionId") == "" {
		multiErr.Add("definitionId", errortypes.ErrRequired, "Name the report to run")
	}

	// A cron nobody parsed is a schedule that silently never fires.
	expression := optionalString(params, "cronExpression")
	switch {
	case expression == "":
		multiErr.Add("cronExpression", errortypes.ErrRequired, "Say when it should run")
	default:
		// The same check the service runs, run early so a bad expression is
		// refused on the card rather than at the moment it should have fired.
		timezone := optionalString(params, "timezone")
		if _, err := cronutils.NextRun(expression, timezone, timeutils.NowUnix()); err != nil {
			multiErr.Add("cronExpression", errortypes.ErrInvalid,
				fmt.Sprintf("%q is not a schedule: %s", expression, err.Error()))
		}
	}

	if len(stringSliceParam(params, "emailRecipients")) == 0 {
		multiErr.Add("emailRecipients", errortypes.ErrRequired, "Say who it goes to")
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (t *scheduleReportTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	definitionID, err := requirePulid(params.Params, "definitionId")
	if err != nil {
		return err
	}

	attach := true
	if raw, ok := params.Params["attach"].(bool); ok {
		attach = raw
	}

	_, err = t.schedules.CreateSchedule(ctx, &reporting.SaveScheduleRequest{
		Request:         reporting.Request{TenantInfo: tenantFrom(params)},
		DefinitionID:    definitionID,
		CronExpression:  optionalString(params.Params, "cronExpression"),
		Timezone:        optionalString(params.Params, "timezone"),
		Formats:         stringSliceParam(params.Params, "formats"),
		EmailRecipients: stringSliceParam(params.Params, "emailRecipients"),
		EmailAttach:     attach,
		Enabled:         true,
	})

	return err
}

func (t *scheduleReportTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "definitionId", permission.ResourceReport)
}

type alertWriter interface {
	CreateSubscription(
		ctx context.Context,
		entity *tablechangealert.TCASubscription,
	) (*tablechangealert.TCASubscription, error)
}

type createTableChangeAlertTool struct {
	alerts alertWriter
}

func newCreateTableChangeAlertTool(
	alerts *tablechangealertservice.Service,
) serviceports.AgentTool {
	return &createTableChangeAlertTool{alerts: alerts}
}

func (t *createTableChangeAlertTool) Name() string { return "create_table_change_alert" }

func (t *createTableChangeAlertTool) Description() string {
	return "Be told when a record changes in a particular way: a shipment reaching " +
		"Delayed, a customer's credit hold coming on, a rate agreement's expiry being set. " +
		"Use it for \"let me know when...\" and \"flag it if...\". It watches the table " +
		"itself, so it fires however the change was made — by a person, an import, an " +
		"integration or another agent."
}

func (t *createTableChangeAlertTool) ParamSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"name", "tableName", "eventTypes"},
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "What the alert is called, in the words of the thing watched.",
			},
			"tableName": map[string]any{
				"type":        "string",
				"description": "The table to watch, e.g. shipments, customers, workers.",
			},
			"eventTypes": map[string]any{
				"type":     "array",
				"maxItems": 3,
				"description": "Which changes to watch: INSERT for a new record, UPDATE for " +
					"an edit, DELETE for a removal. A status change is an UPDATE.",
				"items": map[string]any{
					"type": "string",
					"enum": []string{"INSERT", "UPDATE", "DELETE"},
				},
			},
			"watchedColumns": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
				"description": "Only fire when one of these columns changed. " +
					"Leave empty to fire on any change.",
			},
			"conditions": map[string]any{
				"type":     "array",
				"maxItems": 6,
				"description": "Narrow when it fires: each compares a column (field) to a " +
					"value, e.g. status changed_to Delayed. is_null, is_not_null and changed " +
					"take no value. Leave empty to fire on every matching event.",
				"items": map[string]any{
					"type":     "object",
					"required": []string{"field", "operator"},
					"properties": map[string]any{
						"field":    map[string]any{"type": "string"},
						"operator": map[string]any{"type": "string", "enum": alertOperators()},
						"value":    map[string]any{"type": "string"},
					},
					"additionalProperties": false,
				},
			},
			"conditionMatch": map[string]any{
				"type":        "string",
				"enum":        []string{"all", "any"},
				"description": "Whether every condition must hold, or any one. Defaults to all.",
			},
			"customMessage": map[string]any{
				"type":        "string",
				"description": "What the notification should say.",
			},
		},
		"additionalProperties": false,
	}
}

func alertOperators() []string {
	operators := []tablechangealert.ConditionOperator{
		tablechangealert.OpEq, tablechangealert.OpNeq,
		tablechangealert.OpGt, tablechangealert.OpGte,
		tablechangealert.OpLt, tablechangealert.OpLte,
		tablechangealert.OpIsNull, tablechangealert.OpIsNotNull,
		tablechangealert.OpContains, tablechangealert.OpNotContains,
		tablechangealert.OpChangedTo, tablechangealert.OpChangedFrom,
		tablechangealert.OpChanged,
	}
	names := make([]string, 0, len(operators))
	for _, operator := range operators {
		names = append(names, string(operator))
	}

	return names
}

func (t *createTableChangeAlertTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceTableChangeAlert,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "Creates an alert whose notices go to people inside the organization.",
	}
}

func (t *createTableChangeAlertTool) Validate(
	_ context.Context,
	params serviceports.ToolExecuteParams,
) error {
	return t.validateArgs(params.Params)
}

func (t *createTableChangeAlertTool) validateArgs(params map[string]any) error {
	multiErr := errortypes.NewMultiError()

	if optionalString(params, "name") == "" {
		multiErr.Add("name", errortypes.ErrRequired, "Give the alert a name")
	}
	if optionalString(params, "tableName") == "" {
		multiErr.Add("tableName", errortypes.ErrRequired, "Say which table to watch")
	}

	events := stringSliceParam(params, "eventTypes")
	if len(events) == 0 {
		multiErr.Add("eventTypes", errortypes.ErrRequired,
			"Say whether to watch inserts, updates or deletes")
	}
	for i, event := range events {
		if !tablechangealert.ValidEventType(event) {
			multiErr.Add(fmt.Sprintf("eventTypes[%d]", i), errortypes.ErrInvalid,
				fmt.Sprintf("%q is not INSERT, UPDATE or DELETE", event))
		}
	}

	for i, condition := range alertConditions(params) {
		if !tablechangealert.ValidConditionOperator(string(condition.Operator)) {
			multiErr.Add(fmt.Sprintf("conditions[%d].operator", i), errortypes.ErrInvalid,
				fmt.Sprintf("%q is not a condition operator", condition.Operator))
		}
		if condition.Field == "" {
			multiErr.Add(fmt.Sprintf("conditions[%d].field", i), errortypes.ErrRequired,
				"Name the column the condition is on")
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func alertConditions(params map[string]any) []tablechangealert.Condition {
	raw, _ := params["conditions"].([]any)
	conditions := make([]tablechangealert.Condition, 0, len(raw))
	for _, entry := range raw {
		object, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		condition := tablechangealert.Condition{
			Field:    optionalString(object, "field"),
			Operator: tablechangealert.ConditionOperator(optionalString(object, "operator")),
		}
		if value, present := object["value"]; present {
			condition.Value = value
		}
		conditions = append(conditions, condition)
	}

	return conditions
}

func (t *createTableChangeAlertTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	match := optionalString(params.Params, "conditionMatch")
	if match == "" {
		match = "all"
	}

	events := stringSliceParam(params.Params, "eventTypes")
	for i, event := range events {
		events[i] = strings.ToUpper(event)
	}

	_, err := t.alerts.CreateSubscription(ctx, &tablechangealert.TCASubscription{
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
		UserID:         params.Actor.UserID,
		Name:           optionalString(params.Params, "name"),
		TableName:      optionalString(params.Params, "tableName"),
		EventTypes:     events,
		WatchedColumns: stringSliceParam(params.Params, "watchedColumns"),
		Conditions:     alertConditions(params.Params),
		ConditionMatch: match,
		CustomMessage:  optionalString(params.Params, "customMessage"),
		Status:         tablechangealert.SubscriptionStatusActive,
	})

	return err
}

// stringSliceParam reads an array of strings, dropping anything that is not
// one rather than failing: a single bad entry should cost its own entry.
func stringSliceParam(params map[string]any, key string) []string {
	raw, _ := params[key].([]any)
	values := make([]string, 0, len(raw))
	for _, entry := range raw {
		if text, ok := entry.(string); ok && strings.TrimSpace(text) != "" {
			values = append(values, text)
		}
	}

	return values
}
