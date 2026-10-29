package recurringshipmenttools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/recurringshipmentservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/recurringshipmentports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramRecurringShipmentID = "recurringShipmentId"

const paramSourceShipmentID = "sourceShipmentId"

const paramSeriesName = "name"

const ParamCronExpression = "cronExpression"

const ParamTimezone = "timezone"

const paramMaxOccurrences = "maxOccurrences"

const paramLeadTimeDays = "leadTimeDays"

const paramSkipWeekends = "skipWeekends"

const paramExceptionPolicy = "exceptionPolicy"

const paramBlackoutDates = "blackoutDates"

const paramAutoGenerate = "autoGenerate"

const paramOccurrenceAt = "occurrenceAt"

const kindRecurringShipment = "recurring shipment"

const maxSeriesName = 100

const maxCronExpression = 100

const maxTimezoneName = 64

const maxSeriesOccurrences = 10000

var seriesStatuses = agenttoolschema.Source(
	"recurringShipment.status",
	recurringshipment.StatusValues(),
)

var seriesExceptionPolicies = agenttoolschema.Source(
	"recurringShipment.exceptionPolicy",
	recurringshipment.ExceptionPolicyValues(),
)

var seriesFields = []string{
	paramSeriesName,
	changepreview.FieldDescription,
	paramSourceShipmentID,
	changepreview.FieldCustomerID,
	ParamCronExpression,
	ParamTimezone,
	changepreview.FieldRecurringStart,
	changepreview.FieldRecurringEnd,
	paramMaxOccurrences,
	paramLeadTimeDays,
	paramSkipWeekends,
	paramExceptionPolicy,
	paramBlackoutDates,
	paramAutoGenerate,
	changepreview.FieldStatus,
	"nextOccurrenceAt",
}

var seriesEditable = []string{
	paramSeriesName,
	changepreview.FieldDescription,
	ParamCronExpression,
	ParamTimezone,
	changepreview.FieldRecurringStart,
	changepreview.FieldRecurringEnd,
	paramMaxOccurrences,
	paramLeadTimeDays,
	paramSkipWeekends,
	paramExceptionPolicy,
	paramBlackoutDates,
	paramAutoGenerate,
}

var seriesRefs = map[string]permission.Resource{
	paramSourceShipmentID:         permission.ResourceShipment,
	changepreview.FieldCustomerID: permission.ResourceCustomer,
}

var seriesTypes = map[string]assistantartifact.DisplayType{
	changepreview.FieldRecurringStart: assistantartifact.DisplayDate,
	changepreview.FieldRecurringEnd:   assistantartifact.DisplayDate,
	"nextOccurrenceAt":                assistantartifact.DisplayDateTime,
}

type recurringShipmentKeeper interface {
	Get(
		ctx context.Context,
		req *recurringshipmentports.GetRecurringShipmentByIDRequest,
	) (*recurringshipment.RecurringShipment, error)
	Create(
		ctx context.Context,
		entity *recurringshipment.RecurringShipment,
		userID pulid.ID,
	) (*recurringshipment.RecurringShipment, error)
	PreviewCreate(
		ctx context.Context,
		entity *recurringshipment.RecurringShipment,
		userID pulid.ID,
	) (*recurringshipmentservice.SeriesChange, error)
	Update(
		ctx context.Context,
		entity *recurringshipment.RecurringShipment,
		userID pulid.ID,
	) (*recurringshipment.RecurringShipment, error)
	PreviewUpdate(
		ctx context.Context,
		entity *recurringshipment.RecurringShipment,
	) (*recurringshipmentservice.SeriesChange, error)
	UpdateStatus(
		ctx context.Context,
		req *recurringshipmentports.UpdateRecurringShipmentStatusRequest,
		userID pulid.ID,
	) (*recurringshipment.RecurringShipment, error)
	PreviewUpdateStatus(
		ctx context.Context,
		req *recurringshipmentports.UpdateRecurringShipmentStatusRequest,
	) (*recurringshipmentservice.SeriesChange, error)
	Generate(
		ctx context.Context,
		req *recurringshipmentports.GenerateRecurringShipmentRequest,
	) (*recurringshipmentports.GenerateRecurringShipmentResult, error)
	PreviewGenerate(
		ctx context.Context,
		req *recurringshipmentports.GenerateRecurringShipmentRequest,
	) (*recurringshipmentports.RecurringShipmentGenerationPlan, error)
}

func recurringShipmentIDProperty(what string) map[string]any {
	return agenttoolschema.RecordID(permission.ResourceRecurringShipment, what,
		"shipment_list_recurring")
}

func targetRecurringShipment(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(
		params,
		paramRecurringShipmentID,
		permission.ResourceRecurringShipment,
	)
}

func seriesRecord(series *recurringshipment.RecurringShipment) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceRecurringShipment,
		ID:       series.ID,
		Label:    "Recurring shipment " + series.Name,
		Version:  changepreview.PinnedVersion(series.Version),
	}
}

func seriesResult(
	action string,
	series *recurringshipment.RecurringShipment,
) *agentrecord.ToolExecutionResult {
	result := &agentrecord.ToolExecutionResult{Action: action, Kind: kindRecurringShipment}
	if series != nil {
		result.Name = series.Name
		result.IDs = map[string]string{paramRecurringShipmentID: series.ID.String()}
	}

	return result
}

func seriesProperties() map[string]any {
	return map[string]any{
		paramSeriesName: toolparams.StringProperty("What dispatch calls the series, such as "+
			"\"Acme weekday Dallas run\".", maxSeriesName),
		changepreview.FieldDescription: toolparams.StringProperty(
			"Anything a planner should know about it.",
			toolparams.MaxOperationNoteChars,
		),
		ParamCronExpression: toolparams.StringProperty(
			"When shipments repeat, as a five-field cron "+
				"expression read in the series' timezone: \"0 8 * * 1-5\" is 08:00 every "+
				"weekday. Take it from what the person or customer asked for.",
			maxCronExpression,
		),
		ParamTimezone: toolparams.StringProperty(
			"The IANA timezone the schedule is read in, such as "+
				"America/Chicago.",
			maxTimezoneName,
		),
		changepreview.FieldRecurringStart: agenttoolschema.Date(
			"The first day shipments may be generated, in the " +
				"series' timezone.",
		),
		changepreview.FieldRecurringEnd: agenttoolschema.Date(
			"The last day shipments may be generated, in the " +
				"series' timezone. Leave out for no end.",
		),
		paramMaxOccurrences: toolparams.IntegerProperty(
			"Stop after this many shipments. Leave out for "+
				"no limit.",
			1,
			maxSeriesOccurrences,
		),
		paramLeadTimeDays: toolparams.IntegerProperty(
			"How many days before each pickup its shipment "+
				"is created.",
			0,
			recurringshipment.MaxLeadTimeDays,
		),
		paramSkipWeekends: toolparams.BooleanProperty(
			"Treat Saturday and Sunday like blackout days.",
		),
		paramExceptionPolicy: agenttoolschema.Enum("What a shipment that falls on a blackout "+
			"day or weekend does: Skip it, or move it to the business day before or after.",
			seriesExceptionPolicies),
		paramBlackoutDates: toolschema.KeepEmpty(map[string]any{
			toolschema.KeyType:        toolschema.TypeArray,
			toolschema.KeyDescription: "Days no shipment is generated, such as holidays.",
			toolschema.KeyMaxItems:    recurringshipment.MaxBlackoutDates,
			toolschema.KeyItems:       agenttoolschema.Date("A blackout day."),
		}),
		paramAutoGenerate: toolparams.BooleanProperty(
			"Create each shipment on schedule without anyone " +
				"asking. Off, a person generates each one.",
		),
	}
}

// applySeriesArgs lays what the call sends over a series. Days are read in the
// series' own timezone, the one its schedule runs in: a start is the first
// moment of that day and an end the last.
func applySeriesArgs(series *recurringshipment.RecurringShipment, params map[string]any) error {
	for key, target := range map[string]*string{
		paramSeriesName:     &series.Name,
		ParamCronExpression: &series.CronExpression,
		ParamTimezone:       &series.Timezone,
	} {
		limit := maxSeriesName
		switch key {
		case ParamCronExpression:
			limit = maxCronExpression
		case ParamTimezone:
			limit = maxTimezoneName
		}
		value, err := toolparams.OptionalBoundedText(params, key, limit)
		if err != nil {
			return err
		}
		if value != nil {
			*target = *value
		}
	}
	description, err := toolparams.OptionalBoundedText(
		params,
		changepreview.FieldDescription,
		toolparams.MaxOperationNoteChars,
	)
	if err != nil {
		return err
	}
	if description != nil {
		series.Description = *description
	}
	if err = applySeriesFlags(series, params); err != nil {
		return err
	}

	return applySeriesDays(series, params)
}

func applySeriesFlags(series *recurringshipment.RecurringShipment, params map[string]any) error {
	if _, given := params[paramMaxOccurrences]; given {
		limit, err := toolparams.RequireIntInRange(
			params,
			paramMaxOccurrences,
			1,
			maxSeriesOccurrences,
		)
		if err != nil {
			return err
		}
		occurrences := int32(limit) //nolint:gosec // bounded by maxSeriesOccurrences
		series.MaxOccurrences = &occurrences
	}
	if _, given := params[paramLeadTimeDays]; given {
		days, err := toolparams.RequireIntInRange(params, paramLeadTimeDays, 0,
			recurringshipment.MaxLeadTimeDays)
		if err != nil {
			return err
		}
		series.LeadTimeDays = int16(days) //nolint:gosec // bounded by MaxLeadTimeDays
	}
	for key, target := range map[string]*bool{
		paramSkipWeekends: &series.SkipWeekends,
		paramAutoGenerate: &series.AutoGenerate,
	} {
		value, err := toolparams.OptionalBoolPointer(params, key)
		if err != nil {
			return err
		}
		if value != nil {
			*target = *value
		}
	}
	policy, given, err := toolparams.OptionalEnum(
		params,
		paramExceptionPolicy,
		seriesExceptionPolicies.Values,
	)
	if err != nil {
		return err
	}
	if given {
		series.ExceptionPolicy = policy
	}

	return nil
}

func applySeriesDays(series *recurringshipment.RecurringShipment, params map[string]any) error {
	if !toolparams.SendsAny(
		params,
		changepreview.FieldRecurringStart,
		changepreview.FieldRecurringEnd,
		paramBlackoutDates,
	) {
		return nil
	}
	location, err := time.LoadLocation(series.Timezone)
	if err != nil || strings.TrimSpace(series.Timezone) == "" {
		return fmt.Errorf("parameter %q must be an IANA timezone such as America/Chicago "+
			"before days can be read", ParamTimezone)
	}

	start, given, err := optionalLocalDay(params, changepreview.FieldRecurringStart, location)
	if err != nil {
		return err
	}
	if given && start != nil {
		series.StartDate = start.Unix()
	}
	end, given, err := optionalLocalDay(params, changepreview.FieldRecurringEnd, location)
	if err != nil {
		return err
	}
	if given {
		series.EndDate = nil
		if end != nil {
			last := end.AddDate(0, 0, 1).Unix() - 1
			series.EndDate = &last
		}
	}
	if _, listed := params[paramBlackoutDates]; listed {
		var days []string
		if err = toolparams.DecodeParam(params, paramBlackoutDates, &days); err != nil {
			return err
		}
		if len(days) > recurringshipment.MaxBlackoutDates {
			return fmt.Errorf("parameter %q holds %d days; at most %d are kept",
				paramBlackoutDates, len(days), recurringshipment.MaxBlackoutDates)
		}
		for index, day := range days {
			if _, dayErr := time.Parse(
				toolparams.DayLayout,
				strings.TrimSpace(day),
			); dayErr != nil {
				return fmt.Errorf("%s[%d] must be YYYY-MM-DD, got %q", paramBlackoutDates, index,
					day)
			}
			days[index] = strings.TrimSpace(day)
		}
		series.BlackoutDates = days
	}

	return nil
}

// optionalLocalDay reads a YYYY-MM-DD as the start of that day in location. An
// empty string is given with no day, which clears an optional date.
func optionalLocalDay(
	params map[string]any,
	key string,
	location *time.Location,
) (*time.Time, bool, error) {
	raw, given := params[key]
	if !given || raw == nil {
		return nil, false, nil
	}
	text, isText := raw.(string)
	if !isText {
		return nil, true, fmt.Errorf("parameter %q must be YYYY-MM-DD", key)
	}
	if strings.TrimSpace(text) == "" {
		return nil, true, nil
	}
	day, err := time.ParseInLocation(toolparams.DayLayout, strings.TrimSpace(text), location)
	if err != nil {
		return nil, true, fmt.Errorf("parameter %q must be YYYY-MM-DD, got %q", key, text)
	}

	return &day, true, nil
}

func renderSeriesChange(
	verb string,
	plan *recurringshipmentservice.SeriesChange,
) (*agentpreview.ToolPreview, error) {
	options := []toolpreview.Option{
		toolpreview.Only(seriesFields...),
		toolpreview.WithRefs(seriesRefs),
		toolpreview.Types(seriesTypes),
	}
	if plan.Before == nil {
		change, err := toolpreview.Create(seriesRecord(plan.After), plan.After, options...)
		if err != nil {
			return nil, err
		}

		return toolpreview.Build(fmt.Sprintf("Would start recurring shipment %q, copying "+
			"its source shipment on schedule.", plan.After.Name), change), nil
	}

	change, err := toolpreview.Changed(seriesRecord(plan.Before), plan.Before, plan.After,
		options...)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf("Would %s recurring shipment %q.", verb,
		plan.Before.Name), change), nil
}

func NewCreateRecurringShipmentTool(series recurringShipmentKeeper) agenttoolports.AgentTool {
	properties := seriesProperties()
	properties[paramSourceShipmentID] = shipmentpreview.ShipmentIDProperty(
		"The shipment every one copies " +
			"(customer, stops, commodities and charges)",
	)

	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name:        "shipment_create_recurring",
		SearchTerms: []string{"every weekday", "repeat every week", "standing lane"},
		Description: "Start a recurring shipment: a series that copies one saved shipment " +
			"on a schedule, for a customer's standing lane. Give the source shipment, a " +
			"name, the cron schedule and its timezone, all from what the person or the " +
			"customer asked for. It starts Active.",
		Resource:    permission.ResourceRecurringShipment,
		Operation:   permission.OpCreate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Sets up a schedule inside Trenova that creates shipments later; nothing " +
			"is sent, and pausing the series stops it.",
		Properties: properties,
		Required: []string{
			paramSourceShipmentID, paramSeriesName, ParamCronExpression, ParamTimezone,
		},
	}, toolspec.ReceivablePlan[*recurringshipment.RecurringShipment, *recurringshipmentservice.SeriesChange]{
		Request: func(
			params *agenttoolports.ToolExecuteParams,
		) (*recurringshipment.RecurringShipment, error) {
			sourceID, err := toolparams.RequirePulid(params.Params, paramSourceShipmentID)
			if err != nil {
				return nil, err
			}
			for _, key := range []string{paramSeriesName, ParamCronExpression, ParamTimezone} {
				if _, err = toolparams.RequireString(params.Params, key); err != nil {
					return nil, err
				}
			}
			entity := &recurringshipment.RecurringShipment{
				OrganizationID:   params.OrganizationID,
				BusinessUnitID:   params.BusinessUnitID,
				SourceShipmentID: sourceID,
				Status:           recurringshipment.StatusActive,
				ExceptionPolicy:  recurringshipment.ExceptionPolicySkip,
				LeadTimeDays:     1,
			}
			if err = applySeriesArgs(entity, params.Params); err != nil {
				return nil, err
			}

			return entity, nil
		},
		Plan: func(
			ctx context.Context,
			entity *recurringshipment.RecurringShipment,
			params *agenttoolports.ToolExecuteParams,
		) (*recurringshipmentservice.SeriesChange, error) {
			probe := *entity

			return series.PreviewCreate(ctx, &probe, params.Actor.UserID)
		},
		Refused: func(*recurringshipment.RecurringShipment) string {
			return "Would start a recurring shipment."
		},
		Render: func(
			_ *recurringshipment.RecurringShipment,
			plan *recurringshipmentservice.SeriesChange,
		) (*agentpreview.ToolPreview, error) {
			return renderSeriesChange("start", plan)
		},
		Run: func(
			ctx context.Context,
			entity *recurringshipment.RecurringShipment,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			created, err := series.Create(ctx, entity, params.Actor.UserID)
			if err != nil {
				return nil, err
			}

			return seriesResult("created", created), nil
		},
	})
}

type seriesEdit struct {
	get    recurringshipmentports.GetRecurringShipmentByIDRequest
	params map[string]any
}

func (e *seriesEdit) entity(
	ctx context.Context,
	series recurringShipmentKeeper,
) (*recurringshipment.RecurringShipment, error) {
	current, err := series.Get(ctx, &e.get)
	if err != nil {
		return nil, err
	}
	if err = applySeriesArgs(current, e.params); err != nil {
		return nil, err
	}

	return current, nil
}

func NewUpdateRecurringShipmentTool(series recurringShipmentKeeper) agenttoolports.AgentTool {
	properties := seriesProperties()
	properties[paramRecurringShipmentID] = recurringShipmentIDProperty("The series to change")

	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name: "shipment_update_recurring",
		Description: "Change a recurring shipment's name, schedule, timezone, dates, " +
			"limits, blackout days or whether it generates on its own. Send only what " +
			"changes; an empty endDate removes the end. Pause or resume it with " +
			"shipment_set_recurring_status.",
		Resource:    permission.ResourceRecurringShipment,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Changes when a series creates shipments inside Trenova; nothing is " +
			"sent, and the old schedule is set back the same way.",
		Properties: properties,
		Required:   []string{paramRecurringShipmentID},
		Target:     targetRecurringShipment,
	}, toolspec.ReceivablePlan[*seriesEdit, *recurringshipmentservice.SeriesChange]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*seriesEdit, error) {
			id, err := toolparams.RequirePulid(params.Params, paramRecurringShipmentID)
			if err != nil {
				return nil, err
			}
			if !toolparams.SendsAny(params.Params, seriesEditable...) {
				return nil, toolparams.ErrNothingToChange
			}
			probe := &recurringshipment.RecurringShipment{Timezone: "UTC"}
			if err = applySeriesArgs(probe, params.Params); err != nil {
				return nil, err
			}

			return &seriesEdit{
				get: recurringshipmentports.GetRecurringShipmentByIDRequest{
					ID:         id,
					TenantInfo: toolparams.TenantFrom(params),
				},
				params: params.Params,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			edit *seriesEdit,
			_ *agenttoolports.ToolExecuteParams,
		) (*recurringshipmentservice.SeriesChange, error) {
			entity, err := edit.entity(ctx, series)
			if err != nil {
				return nil, err
			}

			return series.PreviewUpdate(ctx, entity)
		},
		Refused: func(*seriesEdit) string { return "Would change the recurring shipment." },
		Render: func(
			_ *seriesEdit,
			plan *recurringshipmentservice.SeriesChange,
		) (*agentpreview.ToolPreview, error) {
			return renderSeriesChange("change", plan)
		},
		Run: func(
			ctx context.Context,
			edit *seriesEdit,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			entity, err := edit.entity(ctx, series)
			if err != nil {
				return nil, err
			}
			updated, err := series.Update(ctx, entity, params.Actor.UserID)
			if err != nil {
				return nil, err
			}

			return seriesResult("updated", updated), nil
		},
	})
}

type seriesStatusChange struct {
	get    recurringshipmentports.GetRecurringShipmentByIDRequest
	status recurringshipment.Status
}

func (c *seriesStatusChange) request(
	ctx context.Context,
	series recurringShipmentKeeper,
) (*recurringshipmentports.UpdateRecurringShipmentStatusRequest, error) {
	current, err := series.Get(ctx, &c.get)
	if err != nil {
		return nil, err
	}

	return &recurringshipmentports.UpdateRecurringShipmentStatusRequest{
		TenantInfo:          c.get.TenantInfo,
		RecurringShipmentID: c.get.ID,
		Status:              c.status,
		Version:             current.Version,
	}, nil
}

func NewSetRecurringShipmentStatusTool(series recurringShipmentKeeper) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name: "shipment_set_recurring_status",
		Description: "Pause a recurring shipment so it creates nothing, resume it (Active) " +
			"from its next future slot without back-filling what it missed, or end it " +
			"(Expired). A resumed series with no slot left ends on its own.",
		Resource:    permission.ResourceRecurringShipment,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Starts or stops a schedule inside Trenova; nothing is sent, and the " +
			"status is set back the same way.",
		Properties: map[string]any{
			paramRecurringShipmentID:  recurringShipmentIDProperty("The series"),
			changepreview.FieldStatus: agenttoolschema.Enum("The status to set.", seriesStatuses),
		},
		Required: []string{paramRecurringShipmentID, changepreview.FieldStatus},
		Target:   targetRecurringShipment,
	}, toolspec.ReceivablePlan[*seriesStatusChange, *recurringshipmentservice.SeriesChange]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*seriesStatusChange, error) {
			id, err := toolparams.RequirePulid(params.Params, paramRecurringShipmentID)
			if err != nil {
				return nil, err
			}
			status, err := toolparams.RequireEnum(
				params.Params,
				changepreview.FieldStatus,
				seriesStatuses.Values,
			)
			if err != nil {
				return nil, err
			}

			return &seriesStatusChange{
				get: recurringshipmentports.GetRecurringShipmentByIDRequest{
					ID:         id,
					TenantInfo: toolparams.TenantFrom(params),
				},
				status: status,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			change *seriesStatusChange,
			_ *agenttoolports.ToolExecuteParams,
		) (*recurringshipmentservice.SeriesChange, error) {
			req, err := change.request(ctx, series)
			if err != nil {
				return nil, err
			}

			return series.PreviewUpdateStatus(ctx, req)
		},
		Refused: func(change *seriesStatusChange) string {
			return "Would set the recurring shipment " + string(change.status) + "."
		},
		Render: func(
			change *seriesStatusChange,
			plan *recurringshipmentservice.SeriesChange,
		) (*agentpreview.ToolPreview, error) {
			return renderSeriesChange("set "+string(change.status), plan)
		},
		Run: func(
			ctx context.Context,
			change *seriesStatusChange,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			req, err := change.request(ctx, series)
			if err != nil {
				return nil, err
			}
			updated, err := series.UpdateStatus(ctx, req, params.Actor.UserID)
			if err != nil {
				return nil, err
			}

			return seriesResult("set "+string(change.status), updated), nil
		},
	})
}

func NewGenerateRecurringShipmentTool(series recurringShipmentKeeper) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name:        "shipment_generate_recurring",
		SearchTerms: []string{"next occurrence", "generate early", "next slot", "spin up"},
		Artifact:    changepreview.ShipmentRecordEntity,
		Description: "Create the shipment for a recurring shipment's next slot now, or for " +
			"the slot at occurrenceAt, instead of waiting for the schedule. A slot that " +
			"already has its shipment is not made twice.",
		Resource:    permission.ResourceRecurringShipment,
		Operation:   permission.OpDuplicate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Creates one shipment inside Trenova from the series' source; nothing is " +
			"sent, and a shipment made in error is canceled.",
		Properties: map[string]any{
			paramRecurringShipmentID: recurringShipmentIDProperty("The series"),
			paramOccurrenceAt: agenttoolschema.DateTime("The slot to generate, one the schedule " +
				"produces. Leave out for the next one."),
		},
		Required: []string{paramRecurringShipmentID},
		Target:   targetRecurringShipment,
	}, toolspec.ReceivablePlan[*recurringshipmentports.GenerateRecurringShipmentRequest, *recurringshipmentports.RecurringShipmentGenerationPlan]{
		Request: func(
			params *agenttoolports.ToolExecuteParams,
		) (*recurringshipmentports.GenerateRecurringShipmentRequest, error) {
			id, err := toolparams.RequirePulid(params.Params, paramRecurringShipmentID)
			if err != nil {
				return nil, err
			}
			occurrence, err := toolparams.OptionalDateTime(params.Params, paramOccurrenceAt)
			if err != nil {
				return nil, err
			}

			return &recurringshipmentports.GenerateRecurringShipmentRequest{
				TenantInfo:          toolparams.TenantFrom(params),
				RecurringShipmentID: id,
				OccurrenceAt:        occurrence,
				Trigger:             recurringshipment.RunTriggerManual,
				RequestedBy:         params.Actor.UserID,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *recurringshipmentports.GenerateRecurringShipmentRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*recurringshipmentports.RecurringShipmentGenerationPlan, error) {
			probe := *req

			return series.PreviewGenerate(ctx, &probe)
		},
		Refused: func(*recurringshipmentports.GenerateRecurringShipmentRequest) string {
			return "Would create a shipment from the recurring shipment."
		},
		Render: renderGenerate,
		Run: func(
			ctx context.Context,
			req *recurringshipmentports.GenerateRecurringShipmentRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			generated, err := series.Generate(ctx, req)
			if err != nil {
				return nil, err
			}
			result := seriesResult("generated", generated.Series)
			if generated.Shipment != nil {
				if result.IDs == nil {
					result.IDs = map[string]string{}
				}
				result.IDs[toolparams.ParamShipmentID] = generated.Shipment.ID.String()
				result.Name = generated.Shipment.ProNumber
				result.Record = changepreview.RecordOf(
					changepreview.ShipmentRecordEntity,
					generated.Shipment.ID,
				)
			}

			return result, nil
		},
	})
}

func renderGenerate(
	_ *recurringshipmentports.GenerateRecurringShipmentRequest,
	plan *recurringshipmentports.RecurringShipmentGenerationPlan,
) (*agentpreview.ToolPreview, error) {
	slot := time.Unix(plan.Occurrence.At, 0).UTC().Format(time.RFC3339)
	if plan.AlreadyGenerated {
		return toolpreview.Build(fmt.Sprintf(
			"Would make nothing: the %s slot of recurring shipment %q already has its shipment.",
			slot, plan.Series.Name,
		)), nil
	}

	change, err := shipmentpreview.ShipmentCopyChange(
		fmt.Sprintf("%s, %s slot", plan.Series.Name, slot),
		plan.Series.Name,
		plan.Shipment,
	)
	if err != nil {
		return nil, err
	}
	summary := fmt.Sprintf("Would create the %s shipment of recurring shipment %q now.",
		slot, plan.Series.Name)
	if plan.Occurrence.Shifted {
		summary += " The slot moved off a blackout day or weekend."
	}

	return toolpreview.Build(summary, change), nil
}
