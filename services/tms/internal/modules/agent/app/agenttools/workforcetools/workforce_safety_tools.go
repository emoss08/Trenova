package workforcetools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workersafetyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramSafetyEventID = "safetyEventId"

const paramRecognitionID = "recognitionId"

const paramViolationID = "violationId"

const paramEventKind = "kind"

const paramEventSeverity = "severity"

const paramEventLocation = "location"

const paramPreventable = "preventable"

const paramPoints = "points"

const paramReferenceNumber = "referenceNumber"

const paramInspectionLevel = "inspectionLevel"

const paramInspectionRes = "inspectionResult"

const paramOutOfService = "outOfService"

const paramFineAmount = "fineAmount"

const paramCostAmount = "costAmount"

const paramResolutionText = "resolution"

const paramEventMove = "action"

const paramRecognitionKind = "kind"

const paramTitle = "title"

const paramVisibleToWorker = "visibleToWorker"

const paramCSABasic = "basic"

const paramViolationCode = "code"

const paramSeverityWeight = "severityWeight"

const kindSafetyEvent = "safety event"

const kindRecognition = "recognition"

const kindSafetyViolation = "roadside violation"

const maxSafetyPoints = 100

const maxInspectionLevel = 8

const maxSeverityWeight = 10

type safetyEventMove string

const safetyEventReview = safetyEventMove("Review")

const safetyEventClose = safetyEventMove("Close")

const safetyEventReopen = safetyEventMove("Reopen")

func (m safetyEventMove) done() string {
	switch m {
	case safetyEventClose:
		return "closed"
	case safetyEventReview:
		return "put under review"
	case safetyEventReopen:
		return "reopened"
	}
	return string(m)
}

var safetyEventKinds = agenttoolschema.Source(
	"worker.safetyEventKind",
	worker.SafetyEventKindValues(),
)

var safetySeverities = agenttoolschema.Source(
	"worker.safetySeverity",
	worker.SafetySeverityValues(),
)

var inspectionResults = agenttoolschema.Source(
	"worker.inspectionResult",
	worker.InspectionResultValues(),
)

var recognitionKinds = agenttoolschema.Source(
	"worker.recognitionKind",
	worker.RecognitionKindValues(),
)

var csaBasics = agenttoolschema.Source("worker.csaBasic", worker.AllCSABasics())

var safetyEventMoves = agenttoolschema.Source(
	"workerSafetyEvent.agentMove",
	[]safetyEventMove{safetyEventReview, safetyEventClose, safetyEventReopen},
)

var safetyEventFields = []string{
	paramEventKind, paramEventSeverity, common.FieldStatus, wfParamOccurred, paramEventLocation,
	common.FieldDescription, paramPreventable, paramPoints, "pointsExpireAt", paramReferenceNumber,
	common.FieldShipmentID, paramInspectionLevel, paramInspectionRes, paramOutOfService,
	paramFineAmount, paramCostAmount, wfFieldDocument, paramResolutionText,
}

type SafetyKeeper interface {
	GetEvent(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*worker.WorkerSafetyEvent, error)
	PlanCreateEvent(
		ctx context.Context,
		entity *worker.WorkerSafetyEvent,
		userID pulid.ID,
	) (*worker.WorkerSafetyEvent, error)
	CreateEvent(
		ctx context.Context,
		entity *worker.WorkerSafetyEvent,
		userID pulid.ID,
	) (*worker.WorkerSafetyEvent, error)
	PlanUpdateEvent(
		ctx context.Context,
		entity *worker.WorkerSafetyEvent,
	) (*workersafetyservice.EventChange, error)
	UpdateEvent(
		ctx context.Context,
		entity *worker.WorkerSafetyEvent,
		userID pulid.ID,
	) (*worker.WorkerSafetyEvent, error)
	PlanCloseEvent(
		ctx context.Context,
		req *workersafetyservice.EventStatusRequest,
	) (*workersafetyservice.EventChange, error)
	CloseEvent(
		ctx context.Context,
		req *workersafetyservice.EventStatusRequest,
	) (*worker.WorkerSafetyEvent, error)
	PlanReviewEvent(
		ctx context.Context,
		req *workersafetyservice.EventStatusRequest,
	) (*workersafetyservice.EventChange, error)
	ReviewEvent(
		ctx context.Context,
		req *workersafetyservice.EventStatusRequest,
	) (*worker.WorkerSafetyEvent, error)
	PlanReopenEvent(
		ctx context.Context,
		req *workersafetyservice.EventStatusRequest,
	) (*workersafetyservice.EventChange, error)
	ReopenEvent(
		ctx context.Context,
		req *workersafetyservice.EventStatusRequest,
	) (*worker.WorkerSafetyEvent, error)
	PlanDeleteEvent(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*worker.WorkerSafetyEvent, error)
	DeleteEvent(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		userID pulid.ID,
	) error
	PlanGiveRecognition(
		ctx context.Context,
		entity *worker.WorkerRecognition,
		userID pulid.ID,
	) (*worker.WorkerRecognition, error)
	GiveRecognition(
		ctx context.Context,
		entity *worker.WorkerRecognition,
		userID pulid.ID,
	) (*worker.WorkerRecognition, error)
	PlanDeleteRecognition(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*worker.WorkerRecognition, error)
	DeleteRecognition(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		userID pulid.ID,
	) error
	PlanRecordViolation(
		ctx context.Context,
		req *workersafetyservice.RecordViolationRequest,
	) (*worker.WorkerSafetyViolation, error)
	RecordViolation(
		ctx context.Context,
		req *workersafetyservice.RecordViolationRequest,
	) (*worker.WorkerSafetyViolation, error)
	PlanUpdateViolation(
		ctx context.Context,
		req *workersafetyservice.UpdateViolationRequest,
	) (*workersafetyservice.ViolationChange, error)
	UpdateViolation(
		ctx context.Context,
		req *workersafetyservice.UpdateViolationRequest,
	) (*worker.WorkerSafetyViolation, error)
	GetViolation(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*worker.WorkerSafetyViolation, error)
	PlanDeleteViolation(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*worker.WorkerSafetyViolation, error)
	DeleteViolation(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		userID pulid.ID,
	) error
}

func SafetyToolProviders() []any {
	return []any{
		provideOpenWorkerSafetyEventTool,
		provideUpdateWorkerSafetyEventTool,
		provideChangeWorkerSafetyEventStatusTool,
		provideDeleteWorkerSafetyEventTool,
		provideGiveWorkerRecognitionTool,
		provideDeleteWorkerRecognitionTool,
		provideRecordSafetyViolationTool,
		provideUpdateSafetyViolationTool,
		provideDeleteSafetyViolationTool,
	}
}

func safetyEventIDProperty() map[string]any {
	return agenttoolschema.RecordID(permission.ResourceWorkerSafetyEvent, "The safety event",
		"worker_list_safety_events")
}

func safetyEventRecord(event *worker.WorkerSafetyEvent) toolpreview.Record {
	return wfRecord(permission.ResourceWorkerSafetyEvent, event.ID,
		fmt.Sprintf("%s %s", event.Severity, strings.ToLower(event.Kind.String())), event.Version)
}

func safetyEventProperties() map[string]any {
	return map[string]any{
		paramEventKind:     agenttoolschema.Enum("What happened.", safetyEventKinds),
		paramEventSeverity: agenttoolschema.Enum("How serious it was.", safetySeverities),
		wfParamOccurred:    agenttoolschema.DateTime("When it happened."),
		common.FieldDescription: common.StringProperty(
			"What happened, in the words of the report or "+
				"the person who told you.",
			wfNoteChars,
		),
		paramEventLocation: common.StringProperty("Where it happened.", wfShortChars),
		paramPreventable: common.BooleanProperty("For an accident: whether it was preventable. " +
			"Leave it out unless the report says so."),
		paramPoints: common.IntegerProperty("The points it carries on the driver's record. Leave "+
			"it out on a new event to take the house scale for the kind and severity.", 0,
			maxSafetyPoints),
		paramReferenceNumber: common.StringProperty("A citation, report or inspection number.",
			wfShortChars),
		common.FieldShipmentID: agenttoolschema.RecordIDText(
			permission.ResourceShipment,
			"The load it happened on, from shipment_list or "+
				"shipment_get. Never guess one.",
		),
		paramInspectionLevel: common.IntegerProperty("For an inspection: its CVSA level.", 1,
			maxInspectionLevel),
		paramInspectionRes: agenttoolschema.Enum("For an inspection: how it ended.",
			inspectionResults),
		paramOutOfService: common.BooleanProperty("Whether the driver or the vehicle was put out " +
			"of service."),
		paramFineAmount: common.AmountProperty("A fine, as a decimal such as 250.00."),
		paramCostAmount: common.AmountProperty("What it cost the company, as a decimal such as " +
			"1800.00."),
		wfParamDocument: wfDocumentProperty(),
	}
}

type safetyEventRequired struct {
	kind, severity, occurred, description bool
}

func applySafetyEventParams(
	entity *worker.WorkerSafetyEvent,
	params map[string]any,
	required safetyEventRequired,
) error {
	if err := applySafetyEventCore(entity, params, required); err != nil {
		return err
	}
	return applySafetyEventDetail(entity, params)
}

func applySafetyEventCore(
	entity *worker.WorkerSafetyEvent,
	params map[string]any,
	required safetyEventRequired,
) error {
	kind, kindGiven, kindErr := common.OptionalEnum(params, paramEventKind, safetyEventKinds.Values)
	switch {
	case kindErr != nil:
		return kindErr
	case kindGiven:
		entity.Kind = kind
	case required.kind:
		return fmt.Errorf("missing required parameter %q", paramEventKind)
	}
	if severity, given, err := common.OptionalEnum(
		params, paramEventSeverity, safetySeverities.Values,
	); err != nil {
		return err
	} else if given {
		entity.Severity = severity
	} else if required.severity {
		return fmt.Errorf("missing required parameter %q", paramEventSeverity)
	}
	occurred, err := common.OptionalDateTime(params, wfParamOccurred)
	switch {
	case err != nil:
		return err
	case occurred != nil:
		entity.OccurredAt = *occurred
	case required.occurred:
		return fmt.Errorf("missing required parameter %q", wfParamOccurred)
	}
	description, err := common.OptionalBoundedText(params, common.FieldDescription, wfNoteChars)
	switch {
	case err != nil:
		return err
	case description != nil:
		entity.Description = *description
	case required.description:
		return fmt.Errorf("missing required parameter %q", common.FieldDescription)
	}
	return nil
}

func applySafetyEventDetail(entity *worker.WorkerSafetyEvent, params map[string]any) error {
	var err error
	location, locErr := common.OptionalBoundedText(params, paramEventLocation, wfShortChars)
	if locErr != nil {
		return locErr
	}
	if location != nil {
		entity.Location = *location
	}
	if reference, refErr := common.OptionalBoundedText(
		params, paramReferenceNumber, wfShortChars,
	); refErr != nil {
		return refErr
	} else if reference != nil {
		entity.ReferenceNumber = *reference
	}
	if entity.Preventable, err = optionalBoolParam(params, paramPreventable,
		entity.Preventable); err != nil {
		return err
	}
	if entity.OutOfService, err = optionalBoolParam(params, paramOutOfService,
		entity.OutOfService); err != nil {
		return err
	}
	if result, given, resErr := common.OptionalEnum(
		params, paramInspectionRes, inspectionResults.Values,
	); resErr != nil {
		return resErr
	} else if given {
		entity.InspectionResult = result
	}
	level, err := common.OptionalIntInRange(params, paramInspectionLevel, 1, maxInspectionLevel)
	if err != nil {
		return err
	}
	if level != nil {
		value := int16(*level) //nolint:gosec // bounded by the schema's maximum
		entity.InspectionLevel = &value
	}
	return applySafetyEventRefs(entity, params)
}

func applySafetyEventRefs(entity *worker.WorkerSafetyEvent, params map[string]any) error {
	shipmentID, err := common.OptionalID(params, common.FieldShipmentID)
	if err != nil {
		return err
	}
	if !shipmentID.IsNil() {
		entity.ShipmentID = shipmentID
	}
	documentID, err := common.OptionalID(params, wfParamDocument)
	if err != nil {
		return err
	}
	if !documentID.IsNil() {
		entity.DocumentID = documentID
	}
	if fine, fineErr := optionalNullDecimal(params, paramFineAmount); fineErr != nil {
		return fineErr
	} else if fine.Valid {
		entity.FineAmount = fine
	}
	if cost, costErr := optionalNullDecimal(params, paramCostAmount); costErr != nil {
		return costErr
	} else if cost.Valid {
		entity.CostAmount = cost
	}
	return nil
}

func applySafetyPoints(
	entity *worker.WorkerSafetyEvent,
	params map[string]any,
	defaulted bool,
) error {
	points, err := common.OptionalIntInRange(params, paramPoints, 0, maxSafetyPoints)
	if err != nil {
		return err
	}
	switch {
	case points != nil:
		entity.Points = int32(*points) //nolint:gosec // bounded by the schema's maximum
	case defaulted:
		entity.Points = worker.DefaultSafetyPoints(
			entity.Kind, entity.Severity, entity.Preventable, entity.InspectionResult,
		)
	}
	return nil
}

func newOpenWorkerSafetyEventTool(events SafetyKeeper) agentports.AgentTool {
	properties := safetyEventProperties()
	properties[common.ParamWorkerID] = common.WorkerProperty()
	spec := withSchema(wfSpec(
		"worker_open_safety_event",
		"Record a safety event on a driver's record: an accident, incident, near miss, "+
			"citation or roadside inspection, with when and what happened. It opens Open and "+
			"counts its points on the driver's scorecard; roadside violations cited on an "+
			"inspection are added after with worker_record_safety_violation.",
		"Adds an event to a driver's safety record inside Trenova; nothing is sent to the "+
			"driver, and it is corrected or deleted while it is open.",
		permission.ResourceWorkerSafetyEvent,
		permission.OpCreate,
	), properties, common.ParamWorkerID, paramEventKind, paramEventSeverity, wfParamOccurred,
		common.FieldDescription)
	spec.SearchTerms = []string{"accident", "incident", "inspection", "citation", "near miss"}

	return common.NewReportingReceivableTool(spec,
		common.ReceivablePlan[*worker.WorkerSafetyEvent, *worker.WorkerSafetyEvent]{
			Request: func(params *agentports.ToolExecuteParams) (*worker.WorkerSafetyEvent, error) {
				workerID, err := common.RequirePulid(params.Params, common.ParamWorkerID)
				if err != nil {
					return nil, err
				}
				entity := &worker.WorkerSafetyEvent{
					OrganizationID: params.OrganizationID,
					BusinessUnitID: params.BusinessUnitID,
					WorkerID:       workerID,
					Status:         worker.SafetyEventStatusOpen,
				}
				if err = applySafetyEventParams(entity, params.Params, safetyEventRequired{
					kind: true, severity: true, occurred: true, description: true,
				}); err != nil {
					return nil, err
				}
				return entity, applySafetyPoints(entity, params.Params, true)
			},
			Plan: func(
				ctx context.Context,
				entity *worker.WorkerSafetyEvent,
				params *agentports.ToolExecuteParams,
			) (*worker.WorkerSafetyEvent, error) {
				return events.PlanCreateEvent(ctx, entity, params.Actor.UserID)
			},
			Refused: func(*worker.WorkerSafetyEvent) string {
				return "Would record a safety event on the driver's record."
			},
			Render: func(
				_ *worker.WorkerSafetyEvent,
				planned *worker.WorkerSafetyEvent,
			) (*agent.ToolPreview, error) {
				change, err := toolpreview.Create(
					wfRecord(permission.ResourceWorkerSafetyEvent, pulid.Nil,
						"New safety event", 0),
					planned, wfOptions(append([]string{wfFieldWorkerID},
						safetyEventFields...)...)...,
				)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build(fmt.Sprintf(
					"Would record a %s %s on the driver's record carrying %d points.",
					strings.ToLower(planned.Severity.String()),
					strings.ToLower(planned.Kind.String()), planned.Points,
				), change), nil
			},
			Run: func(
				ctx context.Context,
				entity *worker.WorkerSafetyEvent,
				params *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				created, err := events.CreateEvent(ctx, entity, params.Actor.UserID)
				if err != nil {
					return nil, err
				}
				return wfResult("recorded", kindSafetyEvent, paramSafetyEventID, created.ID,
					created.WorkerID), nil
			},
		})
}

type safetyEventEdit struct {
	id     pulid.ID
	params *agentports.ToolExecuteParams
}

func (e *safetyEventEdit) entity(
	ctx context.Context,
	events SafetyKeeper,
) (*worker.WorkerSafetyEvent, error) {
	current, err := events.GetEvent(ctx, common.TenantFrom(*e.params), e.id)
	if err != nil {
		return nil, err
	}
	entity := *current
	entity.Worker = nil
	entity.Document = nil
	if err = applySafetyEventParams(&entity, e.params.Params, safetyEventRequired{}); err != nil {
		return nil, err
	}
	if err = applySafetyPoints(&entity, e.params.Params, false); err != nil {
		return nil, err
	}
	resolution, err := common.OptionalBoundedText(e.params.Params, paramResolutionText, wfNoteChars)
	if err != nil {
		return nil, err
	}
	if resolution != nil {
		entity.Resolution = *resolution
	}
	return &entity, nil
}

func newUpdateWorkerSafetyEventTool(events SafetyKeeper) agentports.AgentTool {
	properties := safetyEventProperties()
	properties[paramSafetyEventID] = safetyEventIDProperty()
	properties[paramResolutionText] = wfNoteProperty("How it was resolved, kept on the event.")
	spec := targeting(withSchema(wfSpec(
		"worker_update_safety_event",
		"Correct a safety event on a driver's record: what happened, when, the points it "+
			"carries, the inspection result, the fine or cost. Give only what changes. It "+
			"never moves the event's status; worker_set_safety_event_status does.",
		"Corrects an event on a driver's safety record inside Trenova; nothing is sent, and "+
			"the event is corrected again the same way.",
		permission.ResourceWorkerSafetyEvent,
		permission.OpUpdate,
	), properties, paramSafetyEventID), paramSafetyEventID, permission.ResourceWorkerSafetyEvent)

	return common.NewReportingReceivableTool(spec,
		common.ReceivablePlan[*safetyEventEdit, *workersafetyservice.EventChange]{
			Request: func(params *agentports.ToolExecuteParams) (*safetyEventEdit, error) {
				id, err := common.RequirePulid(params.Params, paramSafetyEventID)
				if err != nil {
					return nil, err
				}
				return &safetyEventEdit{id: id, params: params}, nil
			},
			Plan: func(
				ctx context.Context,
				edit *safetyEventEdit,
				_ *agentports.ToolExecuteParams,
			) (*workersafetyservice.EventChange, error) {
				entity, err := edit.entity(ctx, events)
				if err != nil {
					return nil, err
				}
				return events.PlanUpdateEvent(ctx, entity)
			},
			Refused: func(*safetyEventEdit) string {
				return "Would correct a safety event on the driver's record."
			},
			Render: func(
				_ *safetyEventEdit,
				change *workersafetyservice.EventChange,
			) (*agent.ToolPreview, error) {
				recorded, err := toolpreview.Changed(safetyEventRecord(change.Before),
					change.Before, change.After, wfOptions(safetyEventFields...)...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build("Would correct the safety event.", recorded), nil
			},
			Run: func(
				ctx context.Context,
				edit *safetyEventEdit,
				params *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				entity, err := edit.entity(ctx, events)
				if err != nil {
					return nil, err
				}
				updated, err := events.UpdateEvent(ctx, entity, params.Actor.UserID)
				if err != nil {
					return nil, err
				}
				return wfResult("updated", kindSafetyEvent, paramSafetyEventID, updated.ID,
					updated.WorkerID), nil
			},
		})
}

type safetyEventStatusMove struct {
	move safetyEventMove
	req  *workersafetyservice.EventStatusRequest
}

func (m *safetyEventStatusMove) plan(
	ctx context.Context,
	events SafetyKeeper,
) (*workersafetyservice.EventChange, error) {
	switch m.move {
	case safetyEventClose:
		return events.PlanCloseEvent(ctx, m.req)
	case safetyEventReview:
		return events.PlanReviewEvent(ctx, m.req)
	case safetyEventReopen:
		return events.PlanReopenEvent(ctx, m.req)
	}
	return nil, common.ErrUnknownValue(paramEventMove, string(m.move), safetyEventMoves.Names())
}

func (m *safetyEventStatusMove) run(
	ctx context.Context,
	events SafetyKeeper,
) (*worker.WorkerSafetyEvent, error) {
	switch m.move {
	case safetyEventClose:
		return events.CloseEvent(ctx, m.req)
	case safetyEventReview:
		return events.ReviewEvent(ctx, m.req)
	case safetyEventReopen:
		return events.ReopenEvent(ctx, m.req)
	}
	return nil, common.ErrUnknownValue(paramEventMove, string(m.move), safetyEventMoves.Names())
}

func newChangeWorkerSafetyEventStatusTool(events SafetyKeeper) agentports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"worker_set_safety_event_status",
		"Move a safety event to under review, closed or back to open. Review marks it "+
			"under review while it is looked into, Close resolves it with the resolution kept "+
			"on the record, and Reopen puts a closed or reviewed event back on the open list.",
		"Moves a safety event's status inside Trenova; nothing is sent, and the event is "+
			"reopened or closed again the same way.",
		permission.ResourceWorkerSafetyEvent,
		permission.OpClose,
	), map[string]any{
		paramSafetyEventID: safetyEventIDProperty(),
		paramEventMove: agenttoolschema.Enum("Review, Close or Reopen.",
			safetyEventMoves),
		paramResolutionText: wfNoteProperty("For Close: how it was resolved. Required " +
			"to close; say what was actually done, not what you composed."),
	}, paramSafetyEventID, paramEventMove), paramSafetyEventID,
		permission.ResourceWorkerSafetyEvent)

	return common.NewReportingReceivableTool(spec,
		common.ReceivablePlan[*safetyEventStatusMove, *workersafetyservice.EventChange]{
			Request: func(params *agentports.ToolExecuteParams) (*safetyEventStatusMove, error) {
				id, err := common.RequirePulid(params.Params, paramSafetyEventID)
				if err != nil {
					return nil, err
				}
				move, err := common.RequireEnum(
					params.Params,
					paramEventMove,
					safetyEventMoves.Values,
				)
				if err != nil {
					return nil, err
				}
				resolution, err := common.BoundedText(
					params.Params,
					paramResolutionText,
					wfNoteChars,
				)
				if err != nil {
					return nil, err
				}
				if move == safetyEventClose && resolution == "" {
					return nil, fmt.Errorf("parameter %q is required to close an event",
						paramResolutionText)
				}
				return &safetyEventStatusMove{
					move: move,
					req: &workersafetyservice.EventStatusRequest{
						ID:         id,
						TenantInfo: common.TenantFrom(*params),
						Resolution: resolution,
						UserID:     params.Actor.UserID,
					},
				}, nil
			},
			Plan: func(
				ctx context.Context,
				move *safetyEventStatusMove,
				_ *agentports.ToolExecuteParams,
			) (*workersafetyservice.EventChange, error) {
				return move.plan(ctx, events)
			},
			Refused: func(move *safetyEventStatusMove) string {
				return fmt.Sprintf("Would %s the safety event.",
					strings.ToLower(string(move.move)))
			},
			Render: renderSafetyEventStatusMove,
			Run: func(
				ctx context.Context,
				move *safetyEventStatusMove,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				saved, err := move.run(ctx, events)
				if err != nil {
					return nil, err
				}
				return wfResult(move.move.done(), kindSafetyEvent,
					paramSafetyEventID, saved.ID, saved.WorkerID), nil
			},
		})
}

func renderSafetyEventStatusMove(
	move *safetyEventStatusMove,
	change *workersafetyservice.EventChange,
) (*agent.ToolPreview, error) {
	recorded, err := toolpreview.Changed(
		safetyEventRecord(change.Before),
		change.Before,
		change.After,
		wfOptions(
			common.FieldStatus,
			wfFieldClosedAt,
			"closedById",
			paramResolutionText,
		)...)
	if err != nil {
		return nil, err
	}
	summary := fmt.Sprintf("Would move the safety event from %s to %s.",
		change.Before.Status, change.After.Status)
	if change.Before.Status == change.After.Status {
		summary = fmt.Sprintf("The safety event is already %s; %s changes nothing.",
			change.Before.Status, strings.ToLower(string(move.move)))
	}
	return toolpreview.Build(summary, recorded), nil
}

type recordDelete struct {
	id     pulid.ID
	tenant pagination.TenantInfo
	userID pulid.ID
}

func recordDeleteFrom(key string) func(*agentports.ToolExecuteParams) (*recordDelete, error) {
	return func(params *agentports.ToolExecuteParams) (*recordDelete, error) {
		id, err := common.RequirePulid(params.Params, key)
		if err != nil {
			return nil, err
		}
		return &recordDelete{
			id:     id,
			tenant: common.TenantFrom(*params),
			userID: params.Actor.UserID,
		}, nil
	}
}

func newDeleteWorkerSafetyEventTool(events SafetyKeeper) agentports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"worker_delete_safety_event",
		"Delete a safety event recorded in error, taking its points off the driver's "+
			"record. A closed event is history and is refused: reopen it first if it truly "+
			"never happened.",
		"Removes an event recorded in error from a driver's safety record; the audit trail "+
			"keeps what was deleted, and it cannot be put back.",
		permission.ResourceWorkerSafetyEvent,
		permission.OpDelete,
	), map[string]any{paramSafetyEventID: safetyEventIDProperty()}, paramSafetyEventID),
		paramSafetyEventID, permission.ResourceWorkerSafetyEvent)
	spec.MaxTier = agent.TierPropose
	spec.Reversible = false

	return common.NewReceivableTool(
		spec,
		common.ReceivablePlan[*recordDelete, *worker.WorkerSafetyEvent]{
			Request: recordDeleteFrom(paramSafetyEventID),
			Plan: func(
				ctx context.Context,
				del *recordDelete,
				_ *agentports.ToolExecuteParams,
			) (*worker.WorkerSafetyEvent, error) {
				return events.PlanDeleteEvent(ctx, del.tenant, del.id)
			},
			Refused: func(*recordDelete) string {
				return "Would delete a safety event from the driver's record."
			},
			Render: func(_ *recordDelete, event *worker.WorkerSafetyEvent) (*agent.ToolPreview, error) {
				change, err := toolpreview.Delete(safetyEventRecord(event), event,
					wfOptions(safetyEventFields...)...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build(fmt.Sprintf(
					"Would delete the %s and its %d points from the driver's record.",
					strings.ToLower(event.Kind.String()), event.Points), change), nil
			},
			Run: func(
				ctx context.Context,
				del *recordDelete,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				return nil, events.DeleteEvent(ctx, del.tenant, del.id, del.userID)
			},
		},
	)
}

func recognitionFrom(params *agentports.ToolExecuteParams) (*worker.WorkerRecognition, error) {
	workerID, err := common.RequirePulid(params.Params, common.ParamWorkerID)
	if err != nil {
		return nil, err
	}
	kind, err := common.RequireEnum(params.Params, paramRecognitionKind, recognitionKinds.Values)
	if err != nil {
		return nil, err
	}
	title, err := common.RequireBoundedText(params.Params, paramTitle, wfShortChars)
	if err != nil {
		return nil, err
	}
	message, err := common.BoundedText(params.Params, common.FieldMessage, wfNoteChars)
	if err != nil {
		return nil, err
	}
	visible, err := optionalBoolParam(params.Params, paramVisibleToWorker, true)
	if err != nil {
		return nil, err
	}
	entity := &worker.WorkerRecognition{
		OrganizationID:  params.OrganizationID,
		BusinessUnitID:  params.BusinessUnitID,
		WorkerID:        workerID,
		Kind:            kind,
		Title:           title,
		Message:         message,
		VisibleToWorker: visible,
	}
	occurred, err := common.OptionalDateTime(params.Params, wfParamOccurred)
	if err != nil {
		return nil, err
	}
	if occurred != nil {
		entity.OccurredAt = *occurred
	}
	return entity, nil
}

var recognitionFields = []string{
	wfFieldWorkerID, paramRecognitionKind, paramTitle, common.FieldMessage, wfParamOccurred,
	paramVisibleToWorker,
}

func newGiveWorkerRecognitionTool(events SafetyKeeper) agentports.AgentTool {
	spec := withSchema(wfSpec(
		"worker_record_recognition",
		"Recognise a driver for a safety milestone, customer praise, performance or tenure. "+
			"When it is visible to the driver, which is the default, they are told in Dash "+
			"with the title and message you write, so write it as you would say it to them.",
		"The driver is shown the recognition and told of it in Dash when it is visible to them.",
		permission.ResourceWorkerRecognition,
		permission.OpCreate,
	), map[string]any{
		common.ParamWorkerID: common.WorkerProperty(),
		paramRecognitionKind: agenttoolschema.Enum("What it is for.", recognitionKinds),
		paramTitle:           common.StringProperty("A short title the driver sees.", wfShortChars),
		common.FieldMessage:  wfNoteProperty("What you would say to them."),
		wfParamOccurred:      agenttoolschema.DateTime("When it was earned. Defaults to now."),
		paramVisibleToWorker: common.BooleanProperty("Whether the driver sees it and is told. " +
			"Defaults to true."),
	}, common.ParamWorkerID, paramRecognitionKind, paramTitle)
	spec.Egress = agent.EgressDriverVisible

	spec.SearchTerms = []string{"safe miles award", "recognize driver", "safety award", "praise"}

	return common.NewReportingReceivableTool(spec,
		common.ReceivablePlan[*worker.WorkerRecognition, *worker.WorkerRecognition]{
			Request: recognitionFrom,
			Plan: func(
				ctx context.Context,
				entity *worker.WorkerRecognition,
				params *agentports.ToolExecuteParams,
			) (*worker.WorkerRecognition, error) {
				return events.PlanGiveRecognition(ctx, entity, params.Actor.UserID)
			},
			Refused: func(*worker.WorkerRecognition) string {
				return "Would recognise the driver."
			},
			Render: func(
				_ *worker.WorkerRecognition,
				planned *worker.WorkerRecognition,
			) (*agent.ToolPreview, error) {
				change, err := toolpreview.Create(
					wfRecord(permission.ResourceWorkerRecognition, pulid.Nil, planned.Title, 0),
					planned, wfOptions(recognitionFields...)...)
				if err != nil {
					return nil, err
				}
				summary := fmt.Sprintf("Would recognise the driver: %q.", planned.Title)
				if planned.VisibleToWorker {
					summary += " The driver is told in Dash."
				}
				return toolpreview.Build(summary, change), nil
			},
			Run: func(
				ctx context.Context,
				entity *worker.WorkerRecognition,
				params *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				created, err := events.GiveRecognition(ctx, entity, params.Actor.UserID)
				if err != nil {
					return nil, err
				}
				return wfResult("given", kindRecognition, paramRecognitionID, created.ID,
					created.WorkerID), nil
			},
		})
}

func newDeleteWorkerRecognitionTool(events SafetyKeeper) agentports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"worker_delete_recognition",
		"Remove a recognition given in error. If it was visible, it leaves the driver's "+
			"record in Dash too.",
		"Removes a recognition from a driver's record; it is given again the same way.",
		permission.ResourceWorkerRecognition,
		permission.OpDelete,
	), map[string]any{
		paramRecognitionID: agenttoolschema.RecordIDText(
			permission.ResourceWorkerRecognition,
			"The recognition, from worker_list_safety_events. "+
				"Never guess one.",
		),
	}, paramRecognitionID), paramRecognitionID, permission.ResourceWorkerRecognition)
	spec.SearchTerms = []string{"kudos", "remove recognition"}
	spec.MaxTier = agent.TierPropose
	spec.Reversible = false

	return common.NewReceivableTool(
		spec,
		common.ReceivablePlan[*recordDelete, *worker.WorkerRecognition]{
			Request: recordDeleteFrom(paramRecognitionID),
			Plan: func(
				ctx context.Context,
				del *recordDelete,
				_ *agentports.ToolExecuteParams,
			) (*worker.WorkerRecognition, error) {
				return events.PlanDeleteRecognition(ctx, del.tenant, del.id)
			},
			Refused: func(*recordDelete) string { return "Would remove a recognition." },
			Render: func(_ *recordDelete, rec *worker.WorkerRecognition) (*agent.ToolPreview, error) {
				change, err := toolpreview.Delete(
					wfRecord(permission.ResourceWorkerRecognition, rec.ID, rec.Title, rec.Version),
					rec, wfOptions(recognitionFields...)...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build(fmt.Sprintf("Would remove the recognition %q.",
					rec.Title), change), nil
			},
			Run: func(
				ctx context.Context,
				del *recordDelete,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				return nil, events.DeleteRecognition(ctx, del.tenant, del.id, del.userID)
			},
		},
	)
}

var violationFields = []string{
	paramSafetyEventID, wfFieldWorkerID, paramCSABasic, paramViolationCode, common.FieldDescription,
	paramSeverityWeight, paramOutOfService,
}

func violationRecord(violation *worker.WorkerSafetyViolation) toolpreview.Record {
	label := violation.Description
	if violation.Code != "" {
		label = violation.Code + " " + label
	}
	return wfRecord(permission.ResourceWorkerSafetyEvent, violation.ID, label,
		violation.Version)
}

func violationProperties() map[string]any {
	return map[string]any{
		paramCSABasic: agenttoolschema.Enum("The BASIC it falls under. Leave it out to "+
			"take what the event implies.", csaBasics),
		paramViolationCode: common.StringProperty("The regulation cited, such as 393.47(e).",
			wfShortChars),
		common.FieldDescription: common.StringProperty(
			"What was cited, as the inspection report words it.",
			wfNoteChars,
		),
		paramSeverityWeight: common.IntegerProperty("The FMCSA severity weight, 1 to 10.", 1,
			maxSeverityWeight),
		paramOutOfService: common.BooleanProperty("Whether this violation put the driver or " +
			"vehicle out of service."),
	}
}

func violationWeight(params map[string]any) (int16, error) {
	weight, err := common.OptionalIntInRange(params, paramSeverityWeight, 1, maxSeverityWeight)
	if err != nil || weight == nil {
		return 0, err
	}
	return int16(*weight), nil //nolint:gosec // bounded by the schema's maximum
}

func newRecordSafetyViolationTool(events SafetyKeeper) agentports.AgentTool {
	properties := violationProperties()
	properties[paramSafetyEventID] = safetyEventIDProperty()
	spec := withSchema(wfSpec(
		"worker_record_safety_violation",
		"Cite one roadside violation on a safety event, usually an inspection, with its "+
			"BASIC, the regulation, the severity weight and whether it was out of service. "+
			"An inspection that cited several gets one call each. It feeds the fleet's CSA "+
			"BASIC scores.",
		"Adds a cited violation to a safety event inside Trenova; nothing is sent, and it is "+
			"corrected or removed the same way.",
		permission.ResourceWorkerSafetyEvent,
		permission.OpCreate,
	), properties, paramSafetyEventID, common.FieldDescription)

	spec.SearchTerms = []string{"roadside violation", "inspection violation", "add violation"}

	return common.NewReportingReceivableTool(spec, common.ReceivablePlan[
		*workersafetyservice.RecordViolationRequest, *worker.WorkerSafetyViolation,
	]{
		Request: func(
			params *agentports.ToolExecuteParams,
		) (*workersafetyservice.RecordViolationRequest, error) {
			eventID, err := common.RequirePulid(params.Params, paramSafetyEventID)
			if err != nil {
				return nil, err
			}
			description, err := common.RequireBoundedText(
				params.Params,
				common.FieldDescription,
				wfNoteChars,
			)
			if err != nil {
				return nil, err
			}
			basic, _, err := common.OptionalEnum(params.Params, paramCSABasic, csaBasics.Values)
			if err != nil {
				return nil, err
			}
			code, err := common.BoundedText(params.Params, paramViolationCode, wfShortChars)
			if err != nil {
				return nil, err
			}
			weight, err := violationWeight(params.Params)
			if err != nil {
				return nil, err
			}
			outOfService, err := optionalBoolParam(params.Params, paramOutOfService, false)
			if err != nil {
				return nil, err
			}
			return &workersafetyservice.RecordViolationRequest{
				TenantInfo:     common.TenantFrom(*params),
				SafetyEventID:  eventID,
				Basic:          basic,
				Code:           code,
				Description:    description,
				SeverityWeight: weight,
				OutOfService:   outOfService,
				UserID:         params.Actor.UserID,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *workersafetyservice.RecordViolationRequest,
			_ *agentports.ToolExecuteParams,
		) (*worker.WorkerSafetyViolation, error) {
			return events.PlanRecordViolation(ctx, req)
		},
		Refused: func(*workersafetyservice.RecordViolationRequest) string {
			return "Would cite a roadside violation on the safety event."
		},
		Render: func(
			_ *workersafetyservice.RecordViolationRequest,
			planned *worker.WorkerSafetyViolation,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Create(
				wfRecord(permission.ResourceWorkerSafetyEvent, pulid.Nil, "New violation", 0),
				planned, wfOptions(violationFields...)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf(
				"Would cite a %s violation, severity weight %d, on the safety event.",
				planned.Basic, planned.SeverityWeight), change), nil
		},
		Run: func(
			ctx context.Context,
			req *workersafetyservice.RecordViolationRequest,
			_ *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			created, err := events.RecordViolation(ctx, req)
			if err != nil {
				return nil, err
			}
			return wfResult("recorded", kindSafetyViolation, paramViolationID, created.ID,
				created.WorkerID), nil
		},
	})
}

type violationEdit struct {
	params *agentports.ToolExecuteParams
	id     pulid.ID
}

func (e *violationEdit) request(
	ctx context.Context,
	events SafetyKeeper,
) (*workersafetyservice.UpdateViolationRequest, error) {
	tenant := common.TenantFrom(*e.params)
	current, err := events.GetViolation(ctx, tenant, e.id)
	if err != nil {
		return nil, err
	}
	req := &workersafetyservice.UpdateViolationRequest{
		TenantInfo:     tenant,
		ID:             e.id,
		Code:           current.Code,
		Description:    current.Description,
		SeverityWeight: current.SeverityWeight,
		OutOfService:   current.OutOfService,
		UserID:         e.params.Actor.UserID,
	}
	if req.Basic, _, err = common.OptionalEnum(e.params.Params, paramCSABasic,
		csaBasics.Values); err != nil {
		return nil, err
	}
	if code, codeErr := common.OptionalBoundedText(e.params.Params, paramViolationCode,
		wfShortChars); codeErr != nil {
		return nil, codeErr
	} else if code != nil {
		req.Code = *code
	}
	if text, textErr := common.OptionalBoundedText(e.params.Params, common.FieldDescription,
		wfNoteChars); textErr != nil {
		return nil, textErr
	} else if text != nil {
		req.Description = *text
	}
	if weight, weightErr := violationWeight(e.params.Params); weightErr != nil {
		return nil, weightErr
	} else if weight > 0 {
		req.SeverityWeight = weight
	}
	if req.OutOfService, err = optionalBoolParam(e.params.Params, paramOutOfService,
		req.OutOfService); err != nil {
		return nil, err
	}
	return req, nil
}

func violationIDProperty() map[string]any {
	return agenttoolschema.KindID(
		agenttoolschema.IDDescription("The violation", "worker_list_safety_events"),
		permission.KindSafetyViolation,
	)
}

func newUpdateSafetyViolationTool(events SafetyKeeper) agentports.AgentTool {
	properties := violationProperties()
	properties[paramViolationID] = violationIDProperty()
	spec := targeting(withSchema(wfSpec(
		"worker_update_safety_violation",
		"Correct a roadside violation cited on a safety event: its BASIC, regulation, "+
			"wording, severity weight or out-of-service flag. Give only what changes.",
		"Corrects a cited violation inside Trenova; nothing is sent, and it is corrected "+
			"again the same way.",
		permission.ResourceWorkerSafetyEvent,
		permission.OpUpdate,
	), properties, paramViolationID), paramViolationID, permission.ResourceWorkerSafetyEvent)
	spec.SearchTerms = []string{"fix violation", "basic", "severity weight"}

	return common.NewReportingReceivableTool(spec,
		common.ReceivablePlan[*violationEdit, *workersafetyservice.ViolationChange]{
			Request: func(params *agentports.ToolExecuteParams) (*violationEdit, error) {
				id, err := common.RequirePulid(params.Params, paramViolationID)
				if err != nil {
					return nil, err
				}
				return &violationEdit{params: params, id: id}, nil
			},
			Plan: func(
				ctx context.Context,
				edit *violationEdit,
				_ *agentports.ToolExecuteParams,
			) (*workersafetyservice.ViolationChange, error) {
				req, err := edit.request(ctx, events)
				if err != nil {
					return nil, err
				}
				return events.PlanUpdateViolation(ctx, req)
			},
			Refused: func(*violationEdit) string { return "Would correct a cited violation." },
			Render: func(
				_ *violationEdit,
				change *workersafetyservice.ViolationChange,
			) (*agent.ToolPreview, error) {
				recorded, err := toolpreview.Changed(violationRecord(change.Before),
					change.Before, change.After, wfOptions(violationFields...)...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build("Would correct the cited violation.", recorded), nil
			},
			Run: func(
				ctx context.Context,
				edit *violationEdit,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				req, err := edit.request(ctx, events)
				if err != nil {
					return nil, err
				}
				updated, err := events.UpdateViolation(ctx, req)
				if err != nil {
					return nil, err
				}
				return wfResult("updated", kindSafetyViolation, paramViolationID, updated.ID,
					updated.WorkerID), nil
			},
		})
}

func newDeleteSafetyViolationTool(events SafetyKeeper) agentports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"worker_delete_safety_violation",
		"Remove a roadside violation cited on a safety event in error, taking it out of "+
			"the fleet's CSA BASIC scores.",
		"Removes a cited violation inside Trenova; the audit trail keeps what was removed, "+
			"and it is cited again with worker_record_safety_violation.",
		permission.ResourceWorkerSafetyEvent,
		permission.OpDelete,
	), map[string]any{paramViolationID: violationIDProperty()}, paramViolationID),
		paramViolationID, permission.ResourceWorkerSafetyEvent)
	spec.SearchTerms = []string{"remove violation", "csa"}
	spec.MaxTier = agent.TierPropose
	spec.Reversible = false

	return common.NewReceivableTool(
		spec,
		common.ReceivablePlan[*recordDelete, *worker.WorkerSafetyViolation]{
			Request: recordDeleteFrom(paramViolationID),
			Plan: func(
				ctx context.Context,
				del *recordDelete,
				_ *agentports.ToolExecuteParams,
			) (*worker.WorkerSafetyViolation, error) {
				return events.PlanDeleteViolation(ctx, del.tenant, del.id)
			},
			Refused: func(*recordDelete) string { return "Would remove a cited violation." },
			Render: func(
				_ *recordDelete,
				violation *worker.WorkerSafetyViolation,
			) (*agent.ToolPreview, error) {
				change, err := toolpreview.Delete(violationRecord(violation), violation,
					wfOptions(violationFields...)...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build(fmt.Sprintf("Would remove the %s violation.",
					violation.Basic), change), nil
			},
			Run: func(
				ctx context.Context,
				del *recordDelete,
				_ *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				return nil, events.DeleteViolation(ctx, del.tenant, del.id, del.userID)
			},
		},
	)
}

func provideOpenWorkerSafetyEventTool(s *workersafetyservice.Service) agentports.AgentTool {
	return newOpenWorkerSafetyEventTool(s)
}

func provideUpdateWorkerSafetyEventTool(s *workersafetyservice.Service) agentports.AgentTool {
	return newUpdateWorkerSafetyEventTool(s)
}

func provideChangeWorkerSafetyEventStatusTool(
	s *workersafetyservice.Service,
) agentports.AgentTool {
	return newChangeWorkerSafetyEventStatusTool(s)
}

func provideDeleteWorkerSafetyEventTool(s *workersafetyservice.Service) agentports.AgentTool {
	return newDeleteWorkerSafetyEventTool(s)
}

func provideGiveWorkerRecognitionTool(s *workersafetyservice.Service) agentports.AgentTool {
	return newGiveWorkerRecognitionTool(s)
}

func provideDeleteWorkerRecognitionTool(s *workersafetyservice.Service) agentports.AgentTool {
	return newDeleteWorkerRecognitionTool(s)
}

func provideRecordSafetyViolationTool(s *workersafetyservice.Service) agentports.AgentTool {
	return newRecordSafetyViolationTool(s)
}

func provideUpdateSafetyViolationTool(s *workersafetyservice.Service) agentports.AgentTool {
	return newUpdateSafetyViolationTool(s)
}

func provideDeleteSafetyViolationTool(s *workersafetyservice.Service) agentports.AgentTool {
	return newDeleteSafetyViolationTool(s)
}
