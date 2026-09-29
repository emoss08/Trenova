package agenttoolservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/recurringshipmentservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramRecurringShipmentID = "recurringShipmentId"
	paramSourceShipmentID    = "sourceShipmentId"
	paramSeriesName          = "name"
	paramCronExpression      = "cronExpression"
	paramTimezone            = "timezone"
	paramMaxOccurrences      = "maxOccurrences"
	paramLeadTimeDays        = "leadTimeDays"
	paramSkipWeekends        = "skipWeekends"
	paramExceptionPolicy     = "exceptionPolicy"
	paramBlackoutDates       = "blackoutDates"
	paramAutoGenerate        = "autoGenerate"
	paramOccurrenceAt        = "occurrenceAt"
	kindRecurringShipment    = "recurring shipment"
	maxSeriesName            = 100
	maxCronExpression        = 100
	maxTimezoneName          = 64
	maxSeriesOccurrences     = 10000
)

var (
	seriesStatuses = agenttoolschema.Source(
		"recurringShipment.status",
		recurringshipment.StatusValues(),
	)
	seriesExceptionPolicies = agenttoolschema.Source(
		"recurringShipment.exceptionPolicy",
		recurringshipment.ExceptionPolicyValues(),
	)
	seriesFields = []string{
		paramSeriesName, fieldDescription, paramSourceShipmentID, fieldCustomerID,
		paramCronExpression, paramTimezone, fieldRecurringStart, fieldRecurringEnd,
		paramMaxOccurrences, paramLeadTimeDays, paramSkipWeekends, paramExceptionPolicy,
		paramBlackoutDates, paramAutoGenerate, fieldStatus, "nextOccurrenceAt",
	}
	seriesEditable = []string{
		paramSeriesName, fieldDescription, paramCronExpression, paramTimezone,
		fieldRecurringStart, fieldRecurringEnd, paramMaxOccurrences, paramLeadTimeDays,
		paramSkipWeekends, paramExceptionPolicy, paramBlackoutDates, paramAutoGenerate,
	}
	seriesRefs = map[string]permission.Resource{
		paramSourceShipmentID: permission.ResourceShipment,
		fieldCustomerID:       permission.ResourceCustomer,
	}
	seriesTypes = map[string]assistantartifact.DisplayType{
		fieldRecurringStart: assistantartifact.DisplayDate,
		fieldRecurringEnd:   assistantartifact.DisplayDate,
		"nextOccurrenceAt":  assistantartifact.DisplayDateTime,
	}
)

type recurringShipmentKeeper interface {
	Get(
		ctx context.Context,
		req *repositories.GetRecurringShipmentByIDRequest,
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
		req *repositories.UpdateRecurringShipmentStatusRequest,
		userID pulid.ID,
	) (*recurringshipment.RecurringShipment, error)
	PreviewUpdateStatus(
		ctx context.Context,
		req *repositories.UpdateRecurringShipmentStatusRequest,
	) (*recurringshipmentservice.SeriesChange, error)
	Generate(
		ctx context.Context,
		req *repositories.GenerateRecurringShipmentRequest,
	) (*repositories.GenerateRecurringShipmentResult, error)
	PreviewGenerate(
		ctx context.Context,
		req *repositories.GenerateRecurringShipmentRequest,
	) (*repositories.RecurringShipmentGenerationPlan, error)
}

func recurringShipmentIDProperty(what string) map[string]any {
	return idProperty(what + ", from list_recurring_shipments. Never guess one.")
}

func targetRecurringShipment(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramRecurringShipmentID, permission.ResourceRecurringShipment)
}

func seriesRecord(series *recurringshipment.RecurringShipment) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceRecurringShipment,
		ID:       series.ID,
		Label:    "Recurring shipment " + series.Name,
		Version:  pinnedVersion(series.Version),
	}
}

func seriesResult(
	action string,
	series *recurringshipment.RecurringShipment,
) *agent.ToolExecutionResult {
	result := &agent.ToolExecutionResult{Action: action, Kind: kindRecurringShipment}
	if series != nil {
		result.Name = series.Name
		result.IDs = map[string]string{paramRecurringShipmentID: series.ID.String()}
	}

	return result
}

func seriesProperties() map[string]any {
	return map[string]any{
		paramSeriesName: stringProperty("What dispatch calls the series, such as "+
			"\"Acme weekday Dallas run\".", maxSeriesName),
		fieldDescription: stringProperty("Anything a planner should know about it.",
			maxOperationNoteChars),
		paramCronExpression: stringProperty("When shipments repeat, as a five-field cron "+
			"expression read in the series' timezone: \"0 8 * * 1-5\" is 08:00 every "+
			"weekday. Take it from what the person or customer asked for.",
			maxCronExpression),
		paramTimezone: stringProperty("The IANA timezone the schedule is read in, such as "+
			"America/Chicago.", maxTimezoneName),
		fieldRecurringStart: dateProperty("The first day shipments may be generated, in the " +
			"series' timezone."),
		fieldRecurringEnd: dateProperty("The last day shipments may be generated, in the " +
			"series' timezone. Leave out for no end."),
		paramMaxOccurrences: integerProperty("Stop after this many shipments. Leave out for "+
			"no limit.", 1, maxSeriesOccurrences),
		paramLeadTimeDays: integerProperty("How many days before each pickup its shipment "+
			"is created.", 0, recurringshipment.MaxLeadTimeDays),
		paramSkipWeekends: booleanProperty("Treat Saturday and Sunday like blackout days."),
		paramExceptionPolicy: agenttoolschema.Enum("What a shipment that falls on a blackout "+
			"day or weekend does: Skip it, or move it to the business day before or after.",
			seriesExceptionPolicies),
		paramBlackoutDates: map[string]any{
			toolschema.KeyType:        toolschema.TypeArray,
			toolschema.KeyDescription: "Days no shipment is generated, such as holidays.",
			toolschema.KeyMaxItems:    recurringshipment.MaxBlackoutDates,
			toolschema.KeyItems:       dateProperty("A blackout day."),
		},
		paramAutoGenerate: booleanProperty("Create each shipment on schedule without anyone " +
			"asking. Off, a person generates each one."),
	}
}

// applySeriesArgs lays what the call sends over a series. Days are read in the
// series' own timezone, the one its schedule runs in: a start is the first
// moment of that day and an end the last.
func applySeriesArgs(series *recurringshipment.RecurringShipment, params map[string]any) error {
	for key, target := range map[string]*string{
		paramSeriesName:     &series.Name,
		paramCronExpression: &series.CronExpression,
		paramTimezone:       &series.Timezone,
	} {
		limit := maxSeriesName
		switch key {
		case paramCronExpression:
			limit = maxCronExpression
		case paramTimezone:
			limit = maxTimezoneName
		}
		value, err := optionalBoundedText(params, key, limit)
		if err != nil {
			return err
		}
		if value != nil {
			*target = *value
		}
	}
	description, err := optionalBoundedText(params, fieldDescription, maxOperationNoteChars)
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
		limit, err := requireIntInRange(params, paramMaxOccurrences, 1, maxSeriesOccurrences)
		if err != nil {
			return err
		}
		occurrences := int32(limit) //nolint:gosec // bounded by maxSeriesOccurrences
		series.MaxOccurrences = &occurrences
	}
	if _, given := params[paramLeadTimeDays]; given {
		days, err := requireIntInRange(params, paramLeadTimeDays, 0,
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
		value, err := optionalBoolPointer(params, key)
		if err != nil {
			return err
		}
		if value != nil {
			*target = *value
		}
	}
	policy, given, err := optionalEnum(params, paramExceptionPolicy, seriesExceptionPolicies.Values)
	if err != nil {
		return err
	}
	if given {
		series.ExceptionPolicy = policy
	}

	return nil
}

func applySeriesDays(series *recurringshipment.RecurringShipment, params map[string]any) error {
	if !sendsAny(params, fieldRecurringStart, fieldRecurringEnd, paramBlackoutDates) {
		return nil
	}
	location, err := time.LoadLocation(series.Timezone)
	if err != nil || strings.TrimSpace(series.Timezone) == "" {
		return fmt.Errorf("parameter %q must be an IANA timezone such as America/Chicago "+
			"before days can be read", paramTimezone)
	}

	start, given, err := optionalLocalDay(params, fieldRecurringStart, location)
	if err != nil {
		return err
	}
	if given && start != nil {
		series.StartDate = start.Unix()
	}
	end, given, err := optionalLocalDay(params, fieldRecurringEnd, location)
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
		if err = decodeParam(params, paramBlackoutDates, &days); err != nil {
			return err
		}
		if len(days) > recurringshipment.MaxBlackoutDates {
			return fmt.Errorf("parameter %q holds %d days; at most %d are kept",
				paramBlackoutDates, len(days), recurringshipment.MaxBlackoutDates)
		}
		for index, day := range days {
			if _, dayErr := time.Parse(dayLayout, strings.TrimSpace(day)); dayErr != nil {
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
	day, err := time.ParseInLocation(dayLayout, strings.TrimSpace(text), location)
	if err != nil {
		return nil, true, fmt.Errorf("parameter %q must be YYYY-MM-DD, got %q", key, text)
	}

	return &day, true, nil
}

func renderSeriesChange(
	verb string,
	plan *recurringshipmentservice.SeriesChange,
) (*agent.ToolPreview, error) {
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

func newCreateRecurringShipmentTool(series recurringShipmentKeeper) serviceports.AgentTool {
	properties := seriesProperties()
	properties[paramSourceShipmentID] = shipmentIDProperty("The shipment every one copies " +
		"(customer, stops, commodities and charges)")

	return newReceivableTool(&receivableSpec{
		name: "create_recurring_shipment",
		description: "Start a recurring shipment: a series that copies one saved shipment " +
			"on a schedule, for a customer's standing lane. Give the source shipment, a " +
			"name, the cron schedule and its timezone, all from what the person or the " +
			"customer asked for. It starts Active.",
		resource:    permission.ResourceRecurringShipment,
		operation:   permission.OpCreate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Sets up a schedule inside Trenova that creates shipments later; nothing " +
			"is sent, and pausing the series stops it.",
		properties: properties,
		required: []string{
			paramSourceShipmentID, paramSeriesName, paramCronExpression, paramTimezone,
		},
	}, receivablePlan[*recurringshipment.RecurringShipment, *recurringshipmentservice.SeriesChange]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*recurringshipment.RecurringShipment, error) {
			sourceID, err := requirePulid(params.Params, paramSourceShipmentID)
			if err != nil {
				return nil, err
			}
			for _, key := range []string{paramSeriesName, paramCronExpression, paramTimezone} {
				if _, err = requireString(params.Params, key); err != nil {
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
		plan: func(
			ctx context.Context,
			entity *recurringshipment.RecurringShipment,
			params *serviceports.ToolExecuteParams,
		) (*recurringshipmentservice.SeriesChange, error) {
			probe := *entity

			return series.PreviewCreate(ctx, &probe, params.Actor.UserID)
		},
		refused: func(*recurringshipment.RecurringShipment) string {
			return "Would start a recurring shipment."
		},
		render: func(
			_ *recurringshipment.RecurringShipment,
			plan *recurringshipmentservice.SeriesChange,
		) (*agent.ToolPreview, error) {
			return renderSeriesChange("start", plan)
		},
		run: func(
			ctx context.Context,
			entity *recurringshipment.RecurringShipment,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			created, err := series.Create(ctx, entity, params.Actor.UserID)
			if err != nil {
				return nil, err
			}

			return seriesResult("created", created), nil
		},
	})
}

type seriesEdit struct {
	get    repositories.GetRecurringShipmentByIDRequest
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

func newUpdateRecurringShipmentTool(series recurringShipmentKeeper) serviceports.AgentTool {
	properties := seriesProperties()
	properties[paramRecurringShipmentID] = recurringShipmentIDProperty("The series to change")

	return newReceivableTool(&receivableSpec{
		name: "update_recurring_shipment",
		description: "Change a recurring shipment's name, schedule, timezone, dates, " +
			"limits, blackout days or whether it generates on its own. Send only what " +
			"changes; an empty endDate removes the end. Pause or resume it with " +
			"set_recurring_shipment_status.",
		resource:    permission.ResourceRecurringShipment,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Changes when a series creates shipments inside Trenova; nothing is " +
			"sent, and the old schedule is set back the same way.",
		properties: properties,
		required:   []string{paramRecurringShipmentID},
		target:     targetRecurringShipment,
	}, receivablePlan[*seriesEdit, *recurringshipmentservice.SeriesChange]{
		request: func(params *serviceports.ToolExecuteParams) (*seriesEdit, error) {
			id, err := requirePulid(params.Params, paramRecurringShipmentID)
			if err != nil {
				return nil, err
			}
			if !sendsAny(params.Params, seriesEditable...) {
				return nil, errNothingToChange
			}
			probe := &recurringshipment.RecurringShipment{Timezone: "UTC"}
			if err = applySeriesArgs(probe, params.Params); err != nil {
				return nil, err
			}

			return &seriesEdit{
				get: repositories.GetRecurringShipmentByIDRequest{
					ID:         id,
					TenantInfo: tenantFrom(*params),
				},
				params: params.Params,
			}, nil
		},
		plan: func(
			ctx context.Context,
			edit *seriesEdit,
			_ *serviceports.ToolExecuteParams,
		) (*recurringshipmentservice.SeriesChange, error) {
			entity, err := edit.entity(ctx, series)
			if err != nil {
				return nil, err
			}

			return series.PreviewUpdate(ctx, entity)
		},
		refused: func(*seriesEdit) string { return "Would change the recurring shipment." },
		render: func(
			_ *seriesEdit,
			plan *recurringshipmentservice.SeriesChange,
		) (*agent.ToolPreview, error) {
			return renderSeriesChange("change", plan)
		},
		run: func(
			ctx context.Context,
			edit *seriesEdit,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
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
	get    repositories.GetRecurringShipmentByIDRequest
	status recurringshipment.Status
}

func (c *seriesStatusChange) request(
	ctx context.Context,
	series recurringShipmentKeeper,
) (*repositories.UpdateRecurringShipmentStatusRequest, error) {
	current, err := series.Get(ctx, &c.get)
	if err != nil {
		return nil, err
	}

	return &repositories.UpdateRecurringShipmentStatusRequest{
		TenantInfo:          c.get.TenantInfo,
		RecurringShipmentID: c.get.ID,
		Status:              c.status,
		Version:             current.Version,
	}, nil
}

func newSetRecurringShipmentStatusTool(series recurringShipmentKeeper) serviceports.AgentTool {
	return newReceivableTool(&receivableSpec{
		name: "set_recurring_shipment_status",
		description: "Pause a recurring shipment so it creates nothing, resume it (Active) " +
			"from its next future slot without back-filling what it missed, or end it " +
			"(Expired). A resumed series with no slot left ends on its own.",
		resource:    permission.ResourceRecurringShipment,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Starts or stops a schedule inside Trenova; nothing is sent, and the " +
			"status is set back the same way.",
		properties: map[string]any{
			paramRecurringShipmentID: recurringShipmentIDProperty("The series"),
			fieldStatus:              agenttoolschema.Enum("The status to set.", seriesStatuses),
		},
		required: []string{paramRecurringShipmentID, fieldStatus},
		target:   targetRecurringShipment,
	}, receivablePlan[*seriesStatusChange, *recurringshipmentservice.SeriesChange]{
		request: func(params *serviceports.ToolExecuteParams) (*seriesStatusChange, error) {
			id, err := requirePulid(params.Params, paramRecurringShipmentID)
			if err != nil {
				return nil, err
			}
			status, err := requireEnum(params.Params, fieldStatus, seriesStatuses.Values)
			if err != nil {
				return nil, err
			}

			return &seriesStatusChange{
				get: repositories.GetRecurringShipmentByIDRequest{
					ID:         id,
					TenantInfo: tenantFrom(*params),
				},
				status: status,
			}, nil
		},
		plan: func(
			ctx context.Context,
			change *seriesStatusChange,
			_ *serviceports.ToolExecuteParams,
		) (*recurringshipmentservice.SeriesChange, error) {
			req, err := change.request(ctx, series)
			if err != nil {
				return nil, err
			}

			return series.PreviewUpdateStatus(ctx, req)
		},
		refused: func(change *seriesStatusChange) string {
			return "Would set the recurring shipment " + string(change.status) + "."
		},
		render: func(
			change *seriesStatusChange,
			plan *recurringshipmentservice.SeriesChange,
		) (*agent.ToolPreview, error) {
			return renderSeriesChange("set "+string(change.status), plan)
		},
		run: func(
			ctx context.Context,
			change *seriesStatusChange,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
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

func newGenerateRecurringShipmentTool(series recurringShipmentKeeper) serviceports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "generate_recurring_shipment",
		artifact: shipmentRecordEntity,
		description: "Create the shipment for a recurring shipment's next slot now, or for " +
			"the slot at occurrenceAt, instead of waiting for the schedule. A slot that " +
			"already has its shipment is not made twice.",
		resource:    permission.ResourceRecurringShipment,
		operation:   permission.OpDuplicate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Creates one shipment inside Trenova from the series' source; nothing is " +
			"sent, and a shipment made in error is canceled.",
		properties: map[string]any{
			paramRecurringShipmentID: recurringShipmentIDProperty("The series"),
			paramOccurrenceAt: dateTimeProperty("The slot to generate, one the schedule " +
				"produces. Leave out for the next one."),
		},
		required: []string{paramRecurringShipmentID},
		target:   targetRecurringShipment,
	}, receivablePlan[*repositories.GenerateRecurringShipmentRequest, *repositories.RecurringShipmentGenerationPlan]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*repositories.GenerateRecurringShipmentRequest, error) {
			id, err := requirePulid(params.Params, paramRecurringShipmentID)
			if err != nil {
				return nil, err
			}
			occurrence, err := optionalDateTime(params.Params, paramOccurrenceAt)
			if err != nil {
				return nil, err
			}

			return &repositories.GenerateRecurringShipmentRequest{
				TenantInfo:          tenantFrom(*params),
				RecurringShipmentID: id,
				OccurrenceAt:        occurrence,
				Trigger:             recurringshipment.RunTriggerManual,
				RequestedBy:         params.Actor.UserID,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *repositories.GenerateRecurringShipmentRequest,
			_ *serviceports.ToolExecuteParams,
		) (*repositories.RecurringShipmentGenerationPlan, error) {
			probe := *req

			return series.PreviewGenerate(ctx, &probe)
		},
		refused: func(*repositories.GenerateRecurringShipmentRequest) string {
			return "Would create a shipment from the recurring shipment."
		},
		render: renderGenerate,
		run: func(
			ctx context.Context,
			req *repositories.GenerateRecurringShipmentRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			generated, err := series.Generate(ctx, req)
			if err != nil {
				return nil, err
			}
			result := seriesResult("generated", generated.Series)
			if generated.Shipment != nil {
				if result.IDs == nil {
					result.IDs = map[string]string{}
				}
				result.IDs[paramShipmentID] = generated.Shipment.ID.String()
				result.Name = generated.Shipment.ProNumber
				result.Record = recordOf(shipmentRecordEntity, generated.Shipment.ID)
			}

			return result, nil
		},
	})
}

func renderGenerate(
	_ *repositories.GenerateRecurringShipmentRequest,
	plan *repositories.RecurringShipmentGenerationPlan,
) (*agent.ToolPreview, error) {
	slot := time.Unix(plan.Occurrence.At, 0).UTC().Format(time.RFC3339)
	if plan.AlreadyGenerated {
		return toolpreview.Build(fmt.Sprintf(
			"Would make nothing: the %s slot of recurring shipment %q already has its shipment.",
			slot, plan.Series.Name,
		)), nil
	}

	change, err := shipmentCopyChange(
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
