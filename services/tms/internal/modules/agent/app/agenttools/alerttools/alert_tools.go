package alerttools

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatatools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/app/tablechangealertservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tablechangealert"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/cronutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const maxSchedulesCompared = 50

const schemaItems = "items"

type scheduleWriter interface {
	CreateSchedule(
		ctx context.Context,
		req *reporting.SaveScheduleRequest,
	) (*report.ReportSchedule, error)
	GetDefinition(
		ctx context.Context,
		req *reporting.GetDefinitionRequest,
	) (*report.ReportDefinition, error)
	PreviewSchedule(
		ctx context.Context,
		req *reporting.SaveScheduleRequest,
	) (*reporting.SchedulePreview, error)
	ListSchedules(
		ctx context.Context,
		req *reporting.ListSchedulesRequest,
	) ([]*report.ReportSchedule, error)
}

type ScheduleReportTool struct {
	schedules scheduleWriter
}

func NewScheduleReportTool(schedules *reporting.Service) agentports.AgentTool {
	return &ScheduleReportTool{schedules: schedules}
}

func (t *ScheduleReportTool) Name() string { return "schedule_report" }

func (t *ScheduleReportTool) Recipe() []string {
	return []string{"list_reports", "schedule_report"}
}

func (t *ScheduleReportTool) Description() string {
	return "Put a saved report on a schedule and email it to people. Use it when somebody " +
		"wants a number regularly rather than now — \"send me the unbilled aging every " +
		"Monday morning\". The report has to exist already; list_reports finds its id. " +
		"Times are read in the timezone given, so \"Monday morning\" means theirs."
}

func (t *ScheduleReportTool) Prerequisites() []string {
	return []string{"list_reports"}
}

func (t *ScheduleReportTool) ParamSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"definitionId", "cronExpression", "emailRecipients"},
		"properties": map[string]any{
			"definitionId": agenttoolschema.RecordIDText(
				permission.ResourceReport,
				"The saved report to run, from list_reports.",
			),
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
				schemaItems:   map[string]any{"type": "string"},
				"description": "The email addresses it goes to.",
			},
			"formats": map[string]any{
				"type":        "array",
				schemaItems:   agenttoolschema.Enum("", common.ReportFormats),
				"description": "Which formats to attach. Defaults to xlsx, as the schedule form does.",
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

func (t *ScheduleReportTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
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

func (t *ScheduleReportTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams,
) error {
	if err := t.validateArgs(params.Params); err != nil {
		return err
	}

	// The report is looked up now, not when the schedule first fires: a
	// schedule on a report that does not exist is refused at save anyway,
	// and after approval is too late for the model to find the right one.
	raw := common.OptionalString(params.Params, "definitionId")
	definitionID, err := pulid.Parse(raw)
	if err == nil {
		_, err = t.schedules.GetDefinition(ctx, &reporting.GetDefinitionRequest{
			Request:      reporting.Request{TenantInfo: common.TenantFrom(params)},
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

	return t.refuseRepeat(ctx, &params, definitionID)
}

// refuseRepeat refuses a schedule the report already has: the same time, zone
// and recipients. Approving one twice sent every email twice; three identical
// Monday schedules piled up on one report before this.
func (t *ScheduleReportTool) refuseRepeat(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
	definitionID pulid.ID,
) error {
	existing, err := t.schedules.ListSchedules(ctx, &reporting.ListSchedulesRequest{
		Request:      reporting.Request{TenantInfo: common.TenantFrom(*params)},
		DefinitionID: definitionID,
		EnabledOnly:  true,
		Limit:        maxSchedulesCompared,
	})
	if err != nil {
		return err
	}

	expression := strings.TrimSpace(common.OptionalString(params.Params, "cronExpression"))
	timezone := strings.TrimSpace(common.OptionalString(params.Params, "timezone"))
	recipients := common.StringSliceParam(params.Params, "emailRecipients")
	for _, schedule := range existing {
		if schedule == nil || schedule.CronExpression != expression ||
			!strings.EqualFold(schedule.Timezone, timezone) ||
			!sendsTo(schedule.Delivery, recipients) {
			continue
		}

		return errortypes.NewValidationError("cronExpression", errortypes.ErrDuplicate, fmt.Sprintf(
			"This report already runs on that schedule to those recipients (%s); tell the "+
				"person it is set up rather than adding a second one", schedule.ID))
	}

	return nil
}

func sendsTo(delivery *report.ScheduleDelivery, recipients []string) bool {
	if delivery == nil {
		return len(recipients) == 0
	}
	for _, recipient := range recipients {
		if !slices.ContainsFunc(delivery.EmailRecipients, func(have string) bool {
			return strings.EqualFold(strings.TrimSpace(have), strings.TrimSpace(recipient))
		}) {
			return false
		}
	}

	return true
}

func (t *ScheduleReportTool) validateArgs(params map[string]any) error {
	multiErr := errortypes.NewMultiError()

	if common.OptionalString(params, "definitionId") == "" {
		multiErr.Add("definitionId", errortypes.ErrRequired, "Name the report to run")
	}

	// A cron nobody parsed is a schedule that silently never fires.
	expression := common.OptionalString(params, "cronExpression")
	switch {
	case expression == "":
		multiErr.Add("cronExpression", errortypes.ErrRequired, "Say when it should run")
	default:
		// The same check the service runs, run early so a bad expression is
		// refused on the card rather than at the moment it should have fired.
		timezone := common.OptionalString(params, "timezone")
		if _, err := cronutils.NextRun(expression, timezone, timeutils.NowUnix()); err != nil {
			multiErr.Add("cronExpression", errortypes.ErrInvalid,
				fmt.Sprintf("%q is not a schedule: %s", expression, err.Error()))
		}
	}

	if len(common.StringSliceParam(params, "emailRecipients")) == 0 {
		multiErr.Add("emailRecipients", errortypes.ErrRequired, "Say who it goes to")
	}

	for i, format := range common.ScheduleFormats(params) {
		if !report.Format(format).IsValid() {
			multiErr.Add(fmt.Sprintf("formats[%d]", i), errortypes.ErrInvalid,
				fmt.Sprintf("%q is not a report format; use xlsx, csv, pdf or json", format))
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (t *ScheduleReportTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams,
) error {
	request, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.schedules.CreateSchedule(ctx, request)

	return err
}

// request is the schedule the preview shows and the write creates.
func (t *ScheduleReportTool) request(
	params *agentports.ToolExecuteParams,
) (*reporting.SaveScheduleRequest, error) {
	if err := common.GuardExecute(t, *params); err != nil {
		return nil, err
	}

	definitionID, err := common.RequirePulid(params.Params, "definitionId")
	if err != nil {
		return nil, err
	}

	attach := true
	if raw, ok := params.Params["attach"].(bool); ok {
		attach = raw
	}

	return &reporting.SaveScheduleRequest{
		Request:         reporting.Request{TenantInfo: common.TenantFrom(*params)},
		DefinitionID:    definitionID,
		CronExpression:  common.OptionalString(params.Params, "cronExpression"),
		Timezone:        common.OptionalString(params.Params, "timezone"),
		Formats:         common.ScheduleFormats(params.Params),
		EmailRecipients: common.StringSliceParam(params.Params, "emailRecipients"),
		EmailAttach:     attach,
		Enabled:         true,
	}, nil
}

func (t *ScheduleReportTool) Target(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, "definitionId", permission.ResourceReport)
}

type alertWriter interface {
	CheckSubscription(ctx context.Context, entity *tablechangealert.TCASubscription) error
	CreateSubscription(
		ctx context.Context,
		entity *tablechangealert.TCASubscription,
	) (*tablechangealert.TCASubscription, error)
}

type CreateTableChangeAlertTool struct {
	alerts alertWriter
}

func NewCreateTableChangeAlertTool(
	alerts *tablechangealertservice.Service,
) agentports.AgentTool {
	return &CreateTableChangeAlertTool{alerts: alerts}
}

func (t *CreateTableChangeAlertTool) Name() string { return "create_table_change_alert" }

func (t *CreateTableChangeAlertTool) Description() string {
	return "Be told when a record changes in a particular way: a shipment reaching " +
		"Delayed, a customer's credit hold coming on, a rate agreement's expiry being set. " +
		"Use it for \"let me know when...\" and \"flag it if...\". It watches the table " +
		"itself, so it fires however the change was made — by a person, an import, an " +
		"integration or another agent."
}

func (t *CreateTableChangeAlertTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"required": []string{
			masterdatatools.MdName,
			masterdatatools.AlertTableName,
			"eventTypes",
		},
		"properties":           alertProperties(""),
		"additionalProperties": false,
	}
}

func alertProperties(keep string) map[string]any {
	return map[string]any{
		masterdatatools.MdName: map[string]any{
			"type":        "string",
			"description": "What the alert is called, in the words of the thing watched." + keep,
		},
		masterdatatools.AlertTableName: map[string]any{
			"type":        "string",
			"description": "The table to watch, e.g. shipments, customers, workers." + keep,
		},
		"eventTypes": map[string]any{
			"type":     "array",
			"maxItems": 3,
			"description": "Which changes to watch: INSERT for a new record, UPDATE for " +
				"an edit, DELETE for a removal. A status change is an UPDATE." + keep,
			schemaItems: agenttoolschema.Enum("", alertEventTypes),
		},
		"watchedColumns": map[string]any{
			"type":      "array",
			schemaItems: map[string]any{"type": "string"},
			"description": "Only fire when one of these columns changed. " +
				"Leave empty to fire on any change." + keep,
		},
		"conditions": map[string]any{
			"type":     "array",
			"maxItems": 6,
			"description": "Narrow when it fires: each compares a column (field) to a " +
				"value, e.g. status changed_to Delayed. is_null, is_not_null and changed " +
				"take no value. Leave empty to fire on every matching event." + keep,
			schemaItems: map[string]any{
				"type":     "object",
				"required": []string{"field", "operator"},
				"properties": map[string]any{
					"field":    map[string]any{"type": "string"},
					"operator": agenttoolschema.Enum("", alertOperators),
					"value":    map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		"conditionMatch": agenttoolschema.Enum(
			"Whether every condition must hold, or any one. Defaults to all."+keep,
			alertConditionMatches,
		),
		masterdatatools.AlertCustomMessage: map[string]any{
			"type":        "string",
			"description": "What the notification should say." + keep,
		},
	}
}

var alertOperators = agenttoolschema.Source(
	"tableChangeAlert.conditionOperator",
	[]tablechangealert.ConditionOperator{
		tablechangealert.OpEq, tablechangealert.OpNeq,
		tablechangealert.OpGt, tablechangealert.OpGte,
		tablechangealert.OpLt, tablechangealert.OpLte,
		tablechangealert.OpIsNull, tablechangealert.OpIsNotNull,
		tablechangealert.OpContains, tablechangealert.OpNotContains,
		tablechangealert.OpChangedTo, tablechangealert.OpChangedFrom,
		tablechangealert.OpChanged,
	},
)

var alertEventTypes = agenttoolschema.Source(
	"tableChangeAlert.eventType",
	[]tablechangealert.EventType{
		tablechangealert.EventTypeInsert,
		tablechangealert.EventTypeUpdate,
		tablechangealert.EventTypeDelete,
	},
)

var alertConditionMatches = agenttoolschema.Source(
	"tableChangeAlert.conditionMatch",
	[]string{"all", "any"},
)

func (t *CreateTableChangeAlertTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
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

func (t *CreateTableChangeAlertTool) Validate(
	_ context.Context,
	params agentports.ToolExecuteParams,
) error {
	return t.validateArgs(params.Params)
}

func (t *CreateTableChangeAlertTool) validateArgs(params map[string]any) error {
	multiErr := errortypes.NewMultiError()

	if common.OptionalString(params, "name") == "" {
		multiErr.Add("name", errortypes.ErrRequired, "Give the alert a name")
	}
	if common.OptionalString(params, "tableName") == "" {
		multiErr.Add("tableName", errortypes.ErrRequired, "Say which table to watch")
	}

	events := common.StringSliceParam(params, "eventTypes")
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
			Field:    common.OptionalString(object, "field"),
			Operator: tablechangealert.ConditionOperator(common.OptionalString(object, "operator")),
		}
		if value, present := object["value"]; present {
			condition.Value = value
		}
		conditions = append(conditions, condition)
	}

	return conditions
}

func (t *CreateTableChangeAlertTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams,
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}

	_, err := t.alerts.CreateSubscription(ctx, alertSubscription(&params))

	return err
}

func alertSubscription(params *agentports.ToolExecuteParams) *tablechangealert.TCASubscription {
	match := common.OptionalString(params.Params, "conditionMatch")
	if match == "" {
		match = "all"
	}

	events := common.StringSliceParam(params.Params, "eventTypes")
	for i, event := range events {
		events[i] = strings.ToUpper(event)
	}

	return &tablechangealert.TCASubscription{
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
		UserID:         params.Actor.UserID,
		Name:           common.OptionalString(params.Params, "name"),
		TableName:      common.OptionalString(params.Params, "tableName"),
		EventTypes:     events,
		WatchedColumns: common.StringSliceParam(params.Params, "watchedColumns"),
		Conditions:     alertConditions(params.Params),
		ConditionMatch: match,
		CustomMessage:  common.OptionalString(params.Params, "customMessage"),
		Status:         tablechangealert.SubscriptionStatusActive,
	}
}
