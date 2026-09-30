package agentquerytoolservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/orgstructureservice"
	"github.com/emoss08/trenova/internal/core/services/schedulingservice"
	"github.com/emoss08/trenova/internal/core/services/teamscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	fieldSeriesName        = "name"
	maxScheduleAssignments = 12
	maxScheduleSwaps       = 20
	maxShiftTemplates      = 100
)

var recurringShipmentStatuses = []string{
	string(recurringshipment.StatusActive),
	string(recurringshipment.StatusPaused),
	string(recurringshipment.StatusExpired),
}

func operationsQueryToolProviders() []any {
	return []any{
		newListRecurringShipmentsTool,
		provideGetWorkerScheduleTool,
		provideListShiftTemplatesTool,
	}
}

type recurringShipmentRow struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	Status           string       `json:"status"`
	CustomerID       string       `json:"customerId,omitempty"`
	Customer         string       `json:"customer,omitempty"`
	Lane             string       `json:"lane,omitempty"`
	SourceShipmentID string       `json:"sourceShipmentId"`
	Schedule         string       `json:"schedule"`
	Timezone         string       `json:"timezone"`
	AutoGenerate     bool         `json:"autoGenerate"`
	NextOccurrenceAt optionalDate `json:"nextOccurrenceAt"`
	GenerationCount  int64        `json:"generationCount"`
	LastShipmentID   string       `json:"lastGeneratedShipmentId,omitempty"`
}

func recurringShipmentRowFrom(entity *recurringshipment.RecurringShipment) recurringShipmentRow {
	row := recurringShipmentRow{
		ID:               entity.ID.String(),
		Name:             entity.Name,
		Status:           string(entity.Status),
		CustomerID:       pulidString(entity.CustomerID),
		SourceShipmentID: entity.SourceShipmentID.String(),
		Schedule:         entity.CronExpression,
		Timezone:         entity.Timezone,
		AutoGenerate:     entity.AutoGenerate,
		NextOccurrenceAt: expectedDate(typeutils.ValueOrZero(entity.NextOccurrenceAt), "none scheduled"),
		GenerationCount:  entity.GenerationCount,
		LastShipmentID:   pulidString(entity.LastGeneratedShipmentID),
	}
	if entity.Customer != nil {
		row.Customer = entity.Customer.Name
	}
	if entity.OriginLocation != nil && entity.DestinationLocation != nil {
		row.Lane = entity.OriginLocation.Name + " to " + entity.DestinationLocation.Name
	}

	return row
}

func newListRecurringShipmentsTool(
	repo repositories.RecurringShipmentRepository,
) serviceports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "list_recurring_shipments",
		entityPlural: "recurring shipments",
		summary: "List recurring shipments: the series that copy one saved shipment on a " +
			"schedule for a customer's standing lane, with when each next generates. Use " +
			"the id with update_recurring_shipment, set_recurring_shipment_status or " +
			"generate_recurring_shipment.",
		resource: permission.ResourceRecurringShipment,
		config:   querybuilder.GetFieldConfiguration((*recurringshipment.RecurringShipment)(nil)),
		fields: []listField{
			{
				Name:   paramStatus,
				Kind:   filterEnum,
				Values: recurringShipmentStatuses,
				Note:   "Paused generates nothing until resumed; Expired has ended",
			},
			{Name: fieldSeriesName, Kind: filterText, Sortable: true},
			{Name: "autoGenerate", Kind: filterBool},
			{Name: "nextOccurrenceAt", Kind: filterDate, Sortable: true},
			{Name: fieldCreatedAt, Kind: filterDate, Sortable: true},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.List(ctx, &repositories.ListRecurringShipmentsRequest{Filter: opts})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *recurringshipment.RecurringShipment) any {
				return recurringShipmentRowFrom(item)
			}), nil
		},
	})
}

type scheduleReader interface {
	ListAssignments(
		ctx context.Context,
		req *repositories.ListShiftAssignmentsRequest,
	) ([]*worker.WorkerShiftAssignment, error)
	ListPreferences(
		ctx context.Context,
		req *repositories.ListAvailabilityPreferencesRequest,
	) ([]*worker.WorkerAvailabilityPreference, error)
	ListSwaps(
		ctx context.Context,
		req *repositories.ListShiftSwapsRequest,
	) ([]*worker.ShiftSwapRequest, error)
	ListTemplates(
		ctx context.Context,
		req *repositories.ListShiftTemplatesRequest,
	) ([]*worker.ShiftTemplate, error)
}

type getWorkerScheduleTool struct {
	schedules scheduleReader
	scope     teamscope.Guard
	access    fieldAccess
}

func provideGetWorkerScheduleTool(
	schedules *schedulingservice.Service,
	permissions serviceports.PermissionEngine,
	teams *orgstructureservice.Service,
) serviceports.AgentQueryTool {
	return newGetWorkerScheduleTool(schedules, permissions, teams)
}

func newGetWorkerScheduleTool(
	schedules scheduleReader,
	permissions serviceports.PermissionEngine,
	teams teamscope.Teams,
) serviceports.AgentQueryTool {
	return &getWorkerScheduleTool{
		schedules: schedules,
		scope:     teamscope.Guard{Permissions: permissions, Teams: teams},
		access:    newFieldAccess(permissions),
	}
}

func (t *getWorkerScheduleTool) Name() string { return "get_worker_schedule" }

func (t *getWorkerScheduleTool) Description() string {
	return "Read one worker's schedule: the shift patterns they have been on (the current " +
		"one open-ended), what they said about each weekday, and their open shift swaps. " +
		"It gives the shiftAssignmentId and swapId the scheduling tools take."
}

func (t *getWorkerScheduleTool) ParamSchema() map[string]any {
	return idSchema("workerId", "The worker, from search_worker or list_workers.")
}

func (t *getWorkerScheduleTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceWorkerSchedule})
}

type scheduleAssignmentRow struct {
	ShiftAssignmentID string       `json:"shiftAssignmentId"`
	ShiftTemplateID   string       `json:"shiftTemplateId"`
	Shift             string       `json:"shift,omitempty"`
	EffectiveFrom     optionalDate `json:"effectiveFrom"`
	EffectiveTo       optionalDate `json:"effectiveTo"`
	CycleOffsetWeeks  int16        `json:"cycleOffsetWeeks"`
	Notes             string       `json:"notes,omitempty"`
}

type schedulePreferenceRow struct {
	DayOfWeek  string `json:"dayOfWeek"`
	Preference string `json:"preference"`
	Note       string `json:"note,omitempty"`
}

type scheduleSwapRow struct {
	SwapID                string       `json:"swapId"`
	Status                string       `json:"status"`
	RequestingWorkerID    string       `json:"requestingWorkerId"`
	CounterpartyWorkerID  string       `json:"counterpartyWorkerId,omitempty"`
	ShiftDate             optionalDate `json:"shiftDate"`
	CounterpartyShiftDate optionalDate `json:"counterpartyShiftDate"`
	Reason                string       `json:"reason,omitempty"`
}

type workerScheduleView struct {
	WorkerID    string                  `json:"workerId"`
	Assignments []scheduleAssignmentRow `json:"assignments"`
	Preferences []schedulePreferenceRow `json:"preferences"`
	OpenSwaps   []scheduleSwapRow       `json:"openSwaps"`
	Withheld    []string                `json:"withheldByAccess,omitempty"`
}

func (t *getWorkerScheduleTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	workerID, err := requirePulid(params.Params, "workerId")
	if err != nil {
		return nil, err
	}
	tenant := tenantOf(params)
	if err = t.scope.Require(ctx, &teamscope.Request{
		Actor:      params.Actor,
		TenantInfo: tenant,
		Resource:   permission.ResourceWorkerSchedule,
		Operation:  permission.OpRead,
		WorkerID:   workerID,
	}); err != nil {
		return nil, err
	}

	view := &workerScheduleView{WorkerID: workerID.String()}
	if view.Assignments, err = t.assignments(ctx, tenant, workerID); err != nil {
		return nil, err
	}
	if view.Preferences, err = t.preferences(ctx, tenant, workerID); err != nil {
		return nil, err
	}
	if !t.access.mayRead(ctx, params, permission.ResourceShiftSwap) {
		view.OpenSwaps = []scheduleSwapRow{}
		view.Withheld = []string{"openSwaps"}

		return view, nil
	}
	if view.OpenSwaps, err = t.swaps(ctx, tenant, workerID); err != nil {
		return nil, err
	}

	return view, nil
}

func (t *getWorkerScheduleTool) assignments(
	ctx context.Context,
	tenant pagination.TenantInfo,
	workerID pulid.ID,
) ([]scheduleAssignmentRow, error) {
	assignments, err := t.schedules.ListAssignments(ctx, &repositories.ListShiftAssignmentsRequest{
		TenantInfo:      tenant,
		WorkerID:        workerID,
		IncludeTemplate: true,
		Limit:           maxScheduleAssignments,
	})
	if err != nil {
		return nil, err
	}

	rows := make([]scheduleAssignmentRow, 0, len(assignments))
	for _, assignment := range assignments {
		if assignment == nil {
			continue
		}
		row := scheduleAssignmentRow{
			ShiftAssignmentID: assignment.ID.String(),
			ShiftTemplateID:   assignment.ShiftTemplateID.String(),
			EffectiveFrom:     recordedDate(assignment.EffectiveFrom),
			EffectiveTo:       expectedDate(typeutils.ValueOrZero(assignment.EffectiveTo), absentOpenEnded),
			CycleOffsetWeeks:  assignment.CycleOffsetWeeks,
			Notes:             strings.TrimSpace(assignment.Notes),
		}
		if assignment.ShiftTemplate != nil {
			row.Shift = assignment.ShiftTemplate.Name
		}
		rows = append(rows, row)
	}

	return rows, nil
}

func (t *getWorkerScheduleTool) preferences(
	ctx context.Context,
	tenant pagination.TenantInfo,
	workerID pulid.ID,
) ([]schedulePreferenceRow, error) {
	preferences, err := t.schedules.ListPreferences(ctx,
		&repositories.ListAvailabilityPreferencesRequest{TenantInfo: tenant, WorkerID: workerID})
	if err != nil {
		return nil, err
	}

	rows := make([]schedulePreferenceRow, 0, len(preferences))
	for _, preference := range preferences {
		if preference == nil {
			continue
		}
		rows = append(rows, schedulePreferenceRow{
			DayOfWeek:  time.Weekday(preference.DayOfWeek).String(),
			Preference: string(preference.Preference),
			Note:       strings.TrimSpace(preference.Note),
		})
	}

	return rows, nil
}

func (t *getWorkerScheduleTool) swaps(
	ctx context.Context,
	tenant pagination.TenantInfo,
	workerID pulid.ID,
) ([]scheduleSwapRow, error) {
	swaps, err := t.schedules.ListSwaps(ctx, &repositories.ListShiftSwapsRequest{
		TenantInfo: tenant,
		WorkerID:   workerID,
		OpenOnly:   true,
		Limit:      maxScheduleSwaps,
	})
	if err != nil {
		return nil, err
	}

	rows := make([]scheduleSwapRow, 0, len(swaps))
	for _, swap := range swaps {
		if swap == nil {
			continue
		}
		rows = append(rows, scheduleSwapRow{
			SwapID:               swap.ID.String(),
			Status:               string(swap.Status),
			RequestingWorkerID:   swap.RequestingWorkerID.String(),
			CounterpartyWorkerID: pulidString(swap.CounterpartyWorkerID),
			ShiftDate:            recordedDate(swap.ShiftDate),
			CounterpartyShiftDate: expectedDate(
				typeutils.ValueOrZero(swap.CounterpartyShiftDate),
				"a hand-off",
			),
			Reason: strings.TrimSpace(swap.Reason),
		})
	}

	return rows, nil
}

type listShiftTemplatesTool struct {
	schedules scheduleReader
}

func provideListShiftTemplatesTool(
	schedules *schedulingservice.Service,
) serviceports.AgentQueryTool {
	return newListShiftTemplatesTool(schedules)
}

func newListShiftTemplatesTool(schedules scheduleReader) serviceports.AgentQueryTool {
	return &listShiftTemplatesTool{schedules: schedules}
}

func (t *listShiftTemplatesTool) Name() string { return "list_shift_templates" }

func (t *listShiftTemplatesTool) Description() string {
	return "List the organization's active shift patterns: the days, start time and length " +
		"of each, and how many weeks it rotates over. assign_worker_shift takes the id."
}

func (t *listShiftTemplatesTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           map[string]any{},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *listShiftTemplatesTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceShiftTemplate})
}

type shiftTemplateRow struct {
	ShiftTemplateID string `json:"shiftTemplateId"`
	Code            string `json:"code"`
	Name            string `json:"name"`
	Days            string `json:"days"`
	Starts          string `json:"starts"`
	Hours           string `json:"hours"`
	CycleWeeks      int16  `json:"cycleWeeks"`
}

func (t *listShiftTemplatesTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	templates, err := t.schedules.ListTemplates(ctx, &repositories.ListShiftTemplatesRequest{
		TenantInfo: tenantOf(params),
		ActiveOnly: true,
		Limit:      maxShiftTemplates,
	})
	if err != nil {
		return nil, err
	}

	rows := make([]shiftTemplateRow, 0, len(templates))
	for _, template := range templates {
		if template == nil {
			continue
		}
		rows = append(rows, shiftTemplateRow{
			ShiftTemplateID: template.ID.String(),
			Code:            template.Code,
			Name:            template.Name,
			Days:            shiftDays(template.DaysOfWeek),
			Starts:          minuteOfDay(template.StartMinute),
			Hours:           shiftLength(template.DurationMinutes),
			CycleWeeks:      template.CycleWeeks,
		})
	}

	return map[string]any{"shiftTemplates": rows}, nil
}

// shiftDays reads the pattern's seven flags, Sunday first.
func shiftDays(flags string) string {
	days := make([]string, 0, len(flags))
	for index, flag := range flags {
		if flag == '1' && index < 7 {
			days = append(days, time.Weekday(index).String()[:3])
		}
	}
	if len(days) == 0 {
		return "none"
	}

	return strings.Join(days, " ")
}

func shiftLength(minutes int16) string {
	return fmt.Sprintf("%d:%02d", minutes/60, minutes%60)
}

func minuteOfDay(minutes int16) string {
	return time.Date(0, 1, 1, 0, int(minutes), 0, 0, time.UTC).Format("15:04")
}
