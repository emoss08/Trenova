package safetytools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcekit"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workersafetyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/fleetsafety"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdiscipline"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

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
	workersafety.SafetyEventKindValues(),
)

var safetySeverities = agenttoolschema.Source(
	"worker.safetySeverity",
	workersafety.SafetySeverityValues(),
)

var inspectionResults = agenttoolschema.Source(
	"worker.inspectionResult",
	workersafety.InspectionResultValues(),
)

var recognitionKinds = agenttoolschema.Source(
	"worker.recognitionKind",
	workerdiscipline.RecognitionKindValues(),
)

var csaBasics = agenttoolschema.Source("worker.csaBasic", fleetsafety.AllCSABasics())

var safetyEventMoves = agenttoolschema.Source(
	"workerSafetyEvent.agentMove",
	[]safetyEventMove{safetyEventReview, safetyEventClose, safetyEventReopen},
)

var safetyEventFields = []string{
	paramEventKind,
	paramEventSeverity,
	changepreview.FieldStatus,
	workforcekit.ParamOccurred,
	paramEventLocation,
	changepreview.FieldDescription,
	paramPreventable,
	paramPoints,
	"pointsExpireAt",
	paramReferenceNumber,
	changepreview.FieldShipmentID,
	paramInspectionLevel,
	paramInspectionRes,
	paramOutOfService,
	paramFineAmount,
	paramCostAmount,
	workforcekit.FieldDocument,
	paramResolutionText,
}

type SafetyKeeper interface {
	GetEvent(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*workersafety.WorkerSafetyEvent, error)
	PlanCreateEvent(
		ctx context.Context,
		entity *workersafety.WorkerSafetyEvent,
		userID pulid.ID,
	) (*workersafety.WorkerSafetyEvent, error)
	CreateEvent(
		ctx context.Context,
		entity *workersafety.WorkerSafetyEvent,
		userID pulid.ID,
	) (*workersafety.WorkerSafetyEvent, error)
	PlanUpdateEvent(
		ctx context.Context,
		entity *workersafety.WorkerSafetyEvent,
	) (*workersafetyservice.EventChange, error)
	UpdateEvent(
		ctx context.Context,
		entity *workersafety.WorkerSafetyEvent,
		userID pulid.ID,
	) (*workersafety.WorkerSafetyEvent, error)
	PlanCloseEvent(
		ctx context.Context,
		req *workersafetyservice.EventStatusRequest,
	) (*workersafetyservice.EventChange, error)
	CloseEvent(
		ctx context.Context,
		req *workersafetyservice.EventStatusRequest,
	) (*workersafety.WorkerSafetyEvent, error)
	PlanReviewEvent(
		ctx context.Context,
		req *workersafetyservice.EventStatusRequest,
	) (*workersafetyservice.EventChange, error)
	ReviewEvent(
		ctx context.Context,
		req *workersafetyservice.EventStatusRequest,
	) (*workersafety.WorkerSafetyEvent, error)
	PlanReopenEvent(
		ctx context.Context,
		req *workersafetyservice.EventStatusRequest,
	) (*workersafetyservice.EventChange, error)
	ReopenEvent(
		ctx context.Context,
		req *workersafetyservice.EventStatusRequest,
	) (*workersafety.WorkerSafetyEvent, error)
	PlanDeleteEvent(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*workersafety.WorkerSafetyEvent, error)
	DeleteEvent(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		userID pulid.ID,
	) error
	PlanGiveRecognition(
		ctx context.Context,
		entity *workerdiscipline.WorkerRecognition,
		userID pulid.ID,
	) (*workerdiscipline.WorkerRecognition, error)
	GiveRecognition(
		ctx context.Context,
		entity *workerdiscipline.WorkerRecognition,
		userID pulid.ID,
	) (*workerdiscipline.WorkerRecognition, error)
	PlanDeleteRecognition(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*workerdiscipline.WorkerRecognition, error)
	DeleteRecognition(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		userID pulid.ID,
	) error
	PlanRecordViolation(
		ctx context.Context,
		req *workersafetyservice.RecordViolationRequest,
	) (*fleetsafety.WorkerSafetyViolation, error)
	RecordViolation(
		ctx context.Context,
		req *workersafetyservice.RecordViolationRequest,
	) (*fleetsafety.WorkerSafetyViolation, error)
	PlanUpdateViolation(
		ctx context.Context,
		req *workersafetyservice.UpdateViolationRequest,
	) (*workersafetyservice.ViolationChange, error)
	UpdateViolation(
		ctx context.Context,
		req *workersafetyservice.UpdateViolationRequest,
	) (*fleetsafety.WorkerSafetyViolation, error)
	GetViolation(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*fleetsafety.WorkerSafetyViolation, error)
	PlanDeleteViolation(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*fleetsafety.WorkerSafetyViolation, error)
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

func safetyEventRecord(event *workersafety.WorkerSafetyEvent) toolpreview.Record {
	return workforcekit.Record(permission.ResourceWorkerSafetyEvent, event.ID,
		fmt.Sprintf("%s %s", event.Severity, strings.ToLower(event.Kind.String())), event.Version)
}

func safetyEventProperties() map[string]any {
	return map[string]any{
		paramEventKind:             agenttoolschema.Enum("What happened.", safetyEventKinds),
		paramEventSeverity:         agenttoolschema.Enum("How serious it was.", safetySeverities),
		workforcekit.ParamOccurred: agenttoolschema.DateTime("When it happened."),
		changepreview.FieldDescription: toolparams.StringProperty(
			"What happened, in the words of the report or "+
				"the person who told you.",
			workforcekit.NoteChars,
		),
		paramEventLocation: toolparams.StringProperty(
			"Where it happened.",
			workforcekit.ShortChars,
		),
		paramPreventable: toolparams.BooleanProperty(
			"For an accident: whether it was preventable. " +
				"Leave it out unless the report says so.",
		),
		paramPoints: toolparams.IntegerProperty(
			"The points it carries on the driver's record. Leave "+
				"it out on a new event to take the house scale for the kind and severity.",
			0,
			maxSafetyPoints,
		),
		paramReferenceNumber: toolparams.StringProperty("A citation, report or inspection number.",
			workforcekit.ShortChars),
		changepreview.FieldShipmentID: agenttoolschema.RecordIDText(
			permission.ResourceShipment,
			"The load it happened on, from shipment_list or "+
				"shipment_get. Never guess one.",
		),
		paramInspectionLevel: toolparams.IntegerProperty("For an inspection: its CVSA level.", 1,
			maxInspectionLevel),
		paramInspectionRes: agenttoolschema.Enum("For an inspection: how it ended.",
			inspectionResults),
		paramOutOfService: toolparams.BooleanProperty(
			"Whether the driver or the vehicle was put out " +
				"of service.",
		),
		paramFineAmount: toolparams.AmountProperty("A fine, as a decimal such as 250.00."),
		paramCostAmount: toolparams.AmountProperty(
			"What it cost the company, as a decimal such as " +
				"1800.00.",
		),
		workforcekit.ParamDocument: workforcekit.DocumentProperty(),
	}
}

type safetyEventRequired struct {
	kind, severity, occurred, description bool
}

func applySafetyEventParams(
	entity *workersafety.WorkerSafetyEvent,
	params map[string]any,
	required safetyEventRequired,
) error {
	if err := applySafetyEventCore(entity, params, required); err != nil {
		return err
	}
	return applySafetyEventDetail(entity, params)
}

func applySafetyEventCore(
	entity *workersafety.WorkerSafetyEvent,
	params map[string]any,
	required safetyEventRequired,
) error {
	kind, kindGiven, kindErr := toolparams.OptionalEnum(
		params,
		paramEventKind,
		safetyEventKinds.Values,
	)
	switch {
	case kindErr != nil:
		return kindErr
	case kindGiven:
		entity.Kind = kind
	case required.kind:
		return fmt.Errorf("missing required parameter %q", paramEventKind)
	}
	if severity, given, err := toolparams.OptionalEnum(
		params, paramEventSeverity, safetySeverities.Values,
	); err != nil {
		return err
	} else if given {
		entity.Severity = severity
	} else if required.severity {
		return fmt.Errorf("missing required parameter %q", paramEventSeverity)
	}
	occurred, err := toolparams.OptionalDateTime(params, workforcekit.ParamOccurred)
	switch {
	case err != nil:
		return err
	case occurred != nil:
		entity.OccurredAt = *occurred
	case required.occurred:
		return fmt.Errorf("missing required parameter %q", workforcekit.ParamOccurred)
	}
	description, err := toolparams.OptionalBoundedText(
		params,
		changepreview.FieldDescription,
		workforcekit.NoteChars,
	)
	switch {
	case err != nil:
		return err
	case description != nil:
		entity.Description = *description
	case required.description:
		return fmt.Errorf("missing required parameter %q", changepreview.FieldDescription)
	}
	return nil
}

func applySafetyEventDetail(entity *workersafety.WorkerSafetyEvent, params map[string]any) error {
	var err error
	location, locErr := toolparams.OptionalBoundedText(
		params,
		paramEventLocation,
		workforcekit.ShortChars,
	)
	if locErr != nil {
		return locErr
	}
	if location != nil {
		entity.Location = *location
	}
	if reference, refErr := toolparams.OptionalBoundedText(
		params, paramReferenceNumber, workforcekit.ShortChars,
	); refErr != nil {
		return refErr
	} else if reference != nil {
		entity.ReferenceNumber = *reference
	}
	if entity.Preventable, err = workforcekit.OptionalBoolParam(params, paramPreventable,
		entity.Preventable); err != nil {
		return err
	}
	if entity.OutOfService, err = workforcekit.OptionalBoolParam(params, paramOutOfService,
		entity.OutOfService); err != nil {
		return err
	}
	if result, given, resErr := toolparams.OptionalEnum(
		params, paramInspectionRes, inspectionResults.Values,
	); resErr != nil {
		return resErr
	} else if given {
		entity.InspectionResult = result
	}
	level, err := toolparams.OptionalIntInRange(params, paramInspectionLevel, 1, maxInspectionLevel)
	if err != nil {
		return err
	}
	if level != nil {
		value := int16(*level) //nolint:gosec // bounded by the schema's maximum
		entity.InspectionLevel = &value
	}
	return applySafetyEventRefs(entity, params)
}

func applySafetyEventRefs(entity *workersafety.WorkerSafetyEvent, params map[string]any) error {
	shipmentID, err := toolparams.OptionalID(params, changepreview.FieldShipmentID)
	if err != nil {
		return err
	}
	if !shipmentID.IsNil() {
		entity.ShipmentID = shipmentID
	}
	documentID, err := toolparams.OptionalID(params, workforcekit.ParamDocument)
	if err != nil {
		return err
	}
	if !documentID.IsNil() {
		entity.DocumentID = documentID
	}
	if fine, fineErr := workforcekit.OptionalNullDecimal(params, paramFineAmount); fineErr != nil {
		return fineErr
	} else if fine.Valid {
		entity.FineAmount = fine
	}
	if cost, costErr := workforcekit.OptionalNullDecimal(params, paramCostAmount); costErr != nil {
		return costErr
	} else if cost.Valid {
		entity.CostAmount = cost
	}
	return nil
}

func applySafetyPoints(
	entity *workersafety.WorkerSafetyEvent,
	params map[string]any,
	defaulted bool,
) error {
	points, err := toolparams.OptionalIntInRange(params, paramPoints, 0, maxSafetyPoints)
	if err != nil {
		return err
	}
	switch {
	case points != nil:
		entity.Points = int32(*points) //nolint:gosec // bounded by the schema's maximum
	case defaulted:
		entity.Points = workersafety.DefaultSafetyPoints(
			entity.Kind, entity.Severity, entity.Preventable, entity.InspectionResult,
		)
	}
	return nil
}

func newOpenWorkerSafetyEventTool(events SafetyKeeper) agenttoolports.AgentTool {
	properties := safetyEventProperties()
	properties[toolparams.ParamWorkerID] = toolparams.WorkerProperty()
	spec := workforcekit.WithSchema(workforcekit.Spec(
		"worker_open_safety_event",
		"Record a safety event on a driver's record: an accident, incident, near miss, "+
			"citation or roadside inspection, with when and what happened. It opens Open and "+
			"counts its points on the driver's scorecard; roadside violations cited on an "+
			"inspection are added after with worker_record_safety_violation.",
		"Adds an event to a driver's safety record inside Trenova; nothing is sent to the "+
			"driver, and it is corrected or deleted while it is open.",
		permission.ResourceWorkerSafetyEvent,
		permission.OpCreate,
	), properties, toolparams.ParamWorkerID, paramEventKind, paramEventSeverity, workforcekit.ParamOccurred,
		changepreview.FieldDescription)
	spec.SearchTerms = []string{"accident", "incident", "inspection", "citation", "near miss"}

	return toolspec.NewReportingReceivableTool(spec,
		toolspec.ReceivablePlan[*workersafety.WorkerSafetyEvent, *workersafety.WorkerSafetyEvent]{
			Request: func(params *agenttoolports.ToolExecuteParams) (*workersafety.WorkerSafetyEvent, error) {
				workerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamWorkerID)
				if err != nil {
					return nil, err
				}
				entity := &workersafety.WorkerSafetyEvent{
					OrganizationID: params.OrganizationID,
					BusinessUnitID: params.BusinessUnitID,
					WorkerID:       workerID,
					Status:         workersafety.SafetyEventStatusOpen,
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
				entity *workersafety.WorkerSafetyEvent,
				params *agenttoolports.ToolExecuteParams,
			) (*workersafety.WorkerSafetyEvent, error) {
				return events.PlanCreateEvent(ctx, entity, params.Actor.UserID)
			},
			Refused: func(*workersafety.WorkerSafetyEvent) string {
				return "Would record a safety event on the driver's record."
			},
			Render: func(
				_ *workersafety.WorkerSafetyEvent,
				planned *workersafety.WorkerSafetyEvent,
			) (*agentpreview.ToolPreview, error) {
				change, err := toolpreview.Create(
					workforcekit.Record(permission.ResourceWorkerSafetyEvent, pulid.Nil,
						"New safety event", 0),
					planned, workforcekit.Options(append([]string{workforcekit.FieldWorkerID},
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
				entity *workersafety.WorkerSafetyEvent,
				params *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				created, err := events.CreateEvent(ctx, entity, params.Actor.UserID)
				if err != nil {
					return nil, err
				}
				return workforcekit.Result(
					"recorded",
					kindSafetyEvent,
					workforcekit.ParamSafetyEventID,
					created.ID,
					created.WorkerID,
				), nil
			},
		})
}

type safetyEventEdit struct {
	id     pulid.ID
	params *agenttoolports.ToolExecuteParams
}

func (e *safetyEventEdit) entity(
	ctx context.Context,
	events SafetyKeeper,
) (*workersafety.WorkerSafetyEvent, error) {
	current, err := events.GetEvent(ctx, toolparams.TenantFrom(e.params), e.id)
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
	resolution, err := toolparams.OptionalBoundedText(
		e.params.Params,
		paramResolutionText,
		workforcekit.NoteChars,
	)
	if err != nil {
		return nil, err
	}
	if resolution != nil {
		entity.Resolution = *resolution
	}
	return &entity, nil
}

func newUpdateWorkerSafetyEventTool(events SafetyKeeper) agenttoolports.AgentTool {
	properties := safetyEventProperties()
	properties[workforcekit.ParamSafetyEventID] = safetyEventIDProperty()
	properties[paramResolutionText] = workforcekit.NoteProperty(
		"How it was resolved, kept on the event.",
	)
	spec := workforcekit.Targeting(workforcekit.WithSchema(workforcekit.Spec(
		"worker_update_safety_event",
		"Correct a safety event on a driver's record: what happened, when, the points it "+
			"carries, the inspection result, the fine or cost. Give only what changes. It "+
			"never moves the event's status; worker_set_safety_event_status does.",
		"Corrects an event on a driver's safety record inside Trenova; nothing is sent, and "+
			"the event is corrected again the same way.",
		permission.ResourceWorkerSafetyEvent,
		permission.OpUpdate,
	), properties, workforcekit.ParamSafetyEventID), workforcekit.ParamSafetyEventID, permission.ResourceWorkerSafetyEvent)

	return toolspec.NewReportingReceivableTool(spec,
		toolspec.ReceivablePlan[*safetyEventEdit, *workersafetyservice.EventChange]{
			Request: func(params *agenttoolports.ToolExecuteParams) (*safetyEventEdit, error) {
				id, err := toolparams.RequirePulid(params.Params, workforcekit.ParamSafetyEventID)
				if err != nil {
					return nil, err
				}
				return &safetyEventEdit{id: id, params: params}, nil
			},
			Plan: func(
				ctx context.Context,
				edit *safetyEventEdit,
				_ *agenttoolports.ToolExecuteParams,
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
			) (*agentpreview.ToolPreview, error) {
				recorded, err := toolpreview.Changed(safetyEventRecord(change.Before),
					change.Before, change.After, workforcekit.Options(safetyEventFields...)...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build("Would correct the safety event.", recorded), nil
			},
			Run: func(
				ctx context.Context,
				edit *safetyEventEdit,
				params *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				entity, err := edit.entity(ctx, events)
				if err != nil {
					return nil, err
				}
				updated, err := events.UpdateEvent(ctx, entity, params.Actor.UserID)
				if err != nil {
					return nil, err
				}
				return workforcekit.Result(
					"updated",
					kindSafetyEvent,
					workforcekit.ParamSafetyEventID,
					updated.ID,
					updated.WorkerID,
				), nil
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
	return nil, toolparams.ErrUnknownValue(paramEventMove, string(m.move), safetyEventMoves.Names())
}

func (m *safetyEventStatusMove) run(
	ctx context.Context,
	events SafetyKeeper,
) (*workersafety.WorkerSafetyEvent, error) {
	switch m.move {
	case safetyEventClose:
		return events.CloseEvent(ctx, m.req)
	case safetyEventReview:
		return events.ReviewEvent(ctx, m.req)
	case safetyEventReopen:
		return events.ReopenEvent(ctx, m.req)
	}
	return nil, toolparams.ErrUnknownValue(paramEventMove, string(m.move), safetyEventMoves.Names())
}

func newChangeWorkerSafetyEventStatusTool(events SafetyKeeper) agenttoolports.AgentTool {
	spec := workforcekit.Targeting(workforcekit.WithSchema(workforcekit.Spec(
		"worker_set_safety_event_status",
		"Move a safety event to under review, closed or back to open. Review marks it "+
			"under review while it is looked into, Close resolves it with the resolution kept "+
			"on the record, and Reopen puts a closed or reviewed event back on the open list.",
		"Moves a safety event's status inside Trenova; nothing is sent, and the event is "+
			"reopened or closed again the same way.",
		permission.ResourceWorkerSafetyEvent,
		permission.OpClose,
	), map[string]any{
		workforcekit.ParamSafetyEventID: safetyEventIDProperty(),
		paramEventMove: agenttoolschema.Enum("Review, Close or Reopen.",
			safetyEventMoves),
		paramResolutionText: workforcekit.NoteProperty("For Close: how it was resolved. Required " +
			"to close; say what was actually done, not what you composed."),
	}, workforcekit.ParamSafetyEventID, paramEventMove), workforcekit.ParamSafetyEventID,
		permission.ResourceWorkerSafetyEvent)

	return toolspec.NewReportingReceivableTool(spec,
		toolspec.ReceivablePlan[*safetyEventStatusMove, *workersafetyservice.EventChange]{
			Request: func(params *agenttoolports.ToolExecuteParams) (*safetyEventStatusMove, error) {
				id, err := toolparams.RequirePulid(params.Params, workforcekit.ParamSafetyEventID)
				if err != nil {
					return nil, err
				}
				move, err := toolparams.RequireEnum(
					params.Params,
					paramEventMove,
					safetyEventMoves.Values,
				)
				if err != nil {
					return nil, err
				}
				resolution, err := toolparams.BoundedText(
					params.Params,
					paramResolutionText,
					workforcekit.NoteChars,
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
						TenantInfo: toolparams.TenantFrom(params),
						Resolution: resolution,
						UserID:     params.Actor.UserID,
					},
				}, nil
			},
			Plan: func(
				ctx context.Context,
				move *safetyEventStatusMove,
				_ *agenttoolports.ToolExecuteParams,
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
				_ *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				saved, err := move.run(ctx, events)
				if err != nil {
					return nil, err
				}
				return workforcekit.Result(move.move.done(), kindSafetyEvent,
					workforcekit.ParamSafetyEventID, saved.ID, saved.WorkerID), nil
			},
		})
}

func renderSafetyEventStatusMove(
	move *safetyEventStatusMove,
	change *workersafetyservice.EventChange,
) (*agentpreview.ToolPreview, error) {
	recorded, err := toolpreview.Changed(
		safetyEventRecord(change.Before),
		change.Before,
		change.After,
		workforcekit.Options(
			changepreview.FieldStatus,
			workforcekit.FieldClosedAt,
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

func newDeleteWorkerSafetyEventTool(events SafetyKeeper) agenttoolports.AgentTool {
	spec := workforcekit.Targeting(workforcekit.WithSchema(workforcekit.Spec(
		"worker_delete_safety_event",
		"Delete a safety event recorded in error, taking its points off the driver's "+
			"record. A closed event is history and is refused: reopen it first if it truly "+
			"never happened.",
		"Removes an event recorded in error from a driver's safety record; the audit trail "+
			"keeps what was deleted, and it cannot be put back.",
		permission.ResourceWorkerSafetyEvent,
		permission.OpDelete,
	), map[string]any{workforcekit.ParamSafetyEventID: safetyEventIDProperty()}, workforcekit.ParamSafetyEventID),
		workforcekit.ParamSafetyEventID, permission.ResourceWorkerSafetyEvent)
	spec.MaxTier = agent.TierPropose
	spec.Reversible = false

	return toolspec.NewReceivableTool(
		spec,
		toolspec.ReceivablePlan[*workforcekit.RecordDelete, *workersafety.WorkerSafetyEvent]{
			Request: workforcekit.RecordDeleteFrom(workforcekit.ParamSafetyEventID),
			Plan: func(
				ctx context.Context,
				del *workforcekit.RecordDelete,
				_ *agenttoolports.ToolExecuteParams,
			) (*workersafety.WorkerSafetyEvent, error) {
				return events.PlanDeleteEvent(ctx, del.Tenant, del.ID)
			},
			Refused: func(*workforcekit.RecordDelete) string {
				return "Would delete a safety event from the driver's record."
			},
			Render: func(_ *workforcekit.RecordDelete, event *workersafety.WorkerSafetyEvent) (*agentpreview.ToolPreview, error) {
				change, err := toolpreview.Delete(safetyEventRecord(event), event,
					workforcekit.Options(safetyEventFields...)...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build(fmt.Sprintf(
					"Would delete the %s and its %d points from the driver's record.",
					strings.ToLower(event.Kind.String()), event.Points), change), nil
			},
			Run: func(
				ctx context.Context,
				del *workforcekit.RecordDelete,
				_ *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				return nil, events.DeleteEvent(ctx, del.Tenant, del.ID, del.UserID)
			},
		},
	)
}

func recognitionFrom(
	params *agenttoolports.ToolExecuteParams,
) (*workerdiscipline.WorkerRecognition, error) {
	workerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamWorkerID)
	if err != nil {
		return nil, err
	}
	kind, err := toolparams.RequireEnum(
		params.Params,
		paramRecognitionKind,
		recognitionKinds.Values,
	)
	if err != nil {
		return nil, err
	}
	title, err := toolparams.RequireBoundedText(
		params.Params,
		workforcekit.ParamTitle,
		workforcekit.ShortChars,
	)
	if err != nil {
		return nil, err
	}
	message, err := toolparams.BoundedText(
		params.Params,
		changepreview.FieldMessage,
		workforcekit.NoteChars,
	)
	if err != nil {
		return nil, err
	}
	visible, err := workforcekit.OptionalBoolParam(params.Params, paramVisibleToWorker, true)
	if err != nil {
		return nil, err
	}
	entity := &workerdiscipline.WorkerRecognition{
		OrganizationID:  params.OrganizationID,
		BusinessUnitID:  params.BusinessUnitID,
		WorkerID:        workerID,
		Kind:            kind,
		Title:           title,
		Message:         message,
		VisibleToWorker: visible,
	}
	occurred, err := toolparams.OptionalDateTime(params.Params, workforcekit.ParamOccurred)
	if err != nil {
		return nil, err
	}
	if occurred != nil {
		entity.OccurredAt = *occurred
	}
	return entity, nil
}

var recognitionFields = []string{
	workforcekit.FieldWorkerID,
	paramRecognitionKind,
	workforcekit.ParamTitle,
	changepreview.FieldMessage,
	workforcekit.ParamOccurred,
	paramVisibleToWorker,
}

func newGiveWorkerRecognitionTool(events SafetyKeeper) agenttoolports.AgentTool {
	spec := workforcekit.WithSchema(workforcekit.Spec(
		"worker_record_recognition",
		"Recognise a driver for a safety milestone, customer praise, performance or tenure. "+
			"When it is visible to the driver, which is the default, they are told in Dash "+
			"with the title and message you write, so write it as you would say it to them.",
		"The driver is shown the recognition and told of it in Dash when it is visible to them.",
		permission.ResourceWorkerRecognition,
		permission.OpCreate,
	), map[string]any{
		toolparams.ParamWorkerID: toolparams.WorkerProperty(),
		paramRecognitionKind:     agenttoolschema.Enum("What it is for.", recognitionKinds),
		workforcekit.ParamTitle: toolparams.StringProperty(
			"A short title the driver sees.",
			workforcekit.ShortChars,
		),
		changepreview.FieldMessage: workforcekit.NoteProperty("What you would say to them."),
		workforcekit.ParamOccurred: agenttoolschema.DateTime(
			"When it was earned. Defaults to now.",
		),
		paramVisibleToWorker: toolparams.BooleanProperty(
			"Whether the driver sees it and is told. " +
				"Defaults to true.",
		),
	}, toolparams.ParamWorkerID, paramRecognitionKind, workforcekit.ParamTitle)
	spec.Egress = agent.EgressDriverVisible

	spec.SearchTerms = []string{"safe miles award", "recognize driver", "safety award", "praise"}

	return toolspec.NewReportingReceivableTool(
		spec,
		toolspec.ReceivablePlan[*workerdiscipline.WorkerRecognition, *workerdiscipline.WorkerRecognition]{
			Request: recognitionFrom,
			Plan: func(
				ctx context.Context,
				entity *workerdiscipline.WorkerRecognition,
				params *agenttoolports.ToolExecuteParams,
			) (*workerdiscipline.WorkerRecognition, error) {
				return events.PlanGiveRecognition(ctx, entity, params.Actor.UserID)
			},
			Refused: func(*workerdiscipline.WorkerRecognition) string {
				return "Would recognise the driver."
			},
			Render: func(
				_ *workerdiscipline.WorkerRecognition,
				planned *workerdiscipline.WorkerRecognition,
			) (*agentpreview.ToolPreview, error) {
				change, err := toolpreview.Create(
					workforcekit.Record(
						permission.ResourceWorkerRecognition,
						pulid.Nil,
						planned.Title,
						0,
					),
					planned,
					workforcekit.Options(recognitionFields...)...)
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
				entity *workerdiscipline.WorkerRecognition,
				params *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				created, err := events.GiveRecognition(ctx, entity, params.Actor.UserID)
				if err != nil {
					return nil, err
				}
				return workforcekit.Result("given", kindRecognition, paramRecognitionID, created.ID,
					created.WorkerID), nil
			},
		},
	)
}

func newDeleteWorkerRecognitionTool(events SafetyKeeper) agenttoolports.AgentTool {
	spec := workforcekit.Targeting(workforcekit.WithSchema(workforcekit.Spec(
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

	return toolspec.NewReceivableTool(
		spec,
		toolspec.ReceivablePlan[*workforcekit.RecordDelete, *workerdiscipline.WorkerRecognition]{
			Request: workforcekit.RecordDeleteFrom(paramRecognitionID),
			Plan: func(
				ctx context.Context,
				del *workforcekit.RecordDelete,
				_ *agenttoolports.ToolExecuteParams,
			) (*workerdiscipline.WorkerRecognition, error) {
				return events.PlanDeleteRecognition(ctx, del.Tenant, del.ID)
			},
			Refused: func(*workforcekit.RecordDelete) string { return "Would remove a recognition." },
			Render: func(_ *workforcekit.RecordDelete, rec *workerdiscipline.WorkerRecognition) (*agentpreview.ToolPreview, error) {
				change, err := toolpreview.Delete(
					workforcekit.Record(
						permission.ResourceWorkerRecognition,
						rec.ID,
						rec.Title,
						rec.Version,
					),
					rec,
					workforcekit.Options(recognitionFields...)...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build(fmt.Sprintf("Would remove the recognition %q.",
					rec.Title), change), nil
			},
			Run: func(
				ctx context.Context,
				del *workforcekit.RecordDelete,
				_ *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				return nil, events.DeleteRecognition(ctx, del.Tenant, del.ID, del.UserID)
			},
		},
	)
}

var violationFields = []string{
	workforcekit.ParamSafetyEventID,
	workforcekit.FieldWorkerID,
	paramCSABasic,
	paramViolationCode,
	changepreview.FieldDescription,
	paramSeverityWeight,
	paramOutOfService,
}

func violationRecord(violation *fleetsafety.WorkerSafetyViolation) toolpreview.Record {
	label := violation.Description
	if violation.Code != "" {
		label = violation.Code + " " + label
	}
	return workforcekit.Record(permission.ResourceWorkerSafetyEvent, violation.ID, label,
		violation.Version)
}

func violationProperties() map[string]any {
	return map[string]any{
		paramCSABasic: agenttoolschema.Enum("The BASIC it falls under. Leave it out to "+
			"take what the event implies.", csaBasics),
		paramViolationCode: toolparams.StringProperty("The regulation cited, such as 393.47(e).",
			workforcekit.ShortChars),
		changepreview.FieldDescription: toolparams.StringProperty(
			"What was cited, as the inspection report words it.",
			workforcekit.NoteChars,
		),
		paramSeverityWeight: toolparams.IntegerProperty("The FMCSA severity weight, 1 to 10.", 1,
			maxSeverityWeight),
		paramOutOfService: toolparams.BooleanProperty("Whether this violation put the driver or " +
			"vehicle out of service."),
	}
}

func violationWeight(params map[string]any) (int16, error) {
	weight, err := toolparams.OptionalIntInRange(params, paramSeverityWeight, 1, maxSeverityWeight)
	if err != nil || weight == nil {
		return 0, err
	}
	return int16(*weight), nil //nolint:gosec // bounded by the schema's maximum
}

func newRecordSafetyViolationTool(events SafetyKeeper) agenttoolports.AgentTool {
	properties := violationProperties()
	properties[workforcekit.ParamSafetyEventID] = safetyEventIDProperty()
	spec := workforcekit.WithSchema(workforcekit.Spec(
		"worker_record_safety_violation",
		"Cite one roadside violation on a safety event, usually an inspection, with its "+
			"BASIC, the regulation, the severity weight and whether it was out of service. "+
			"An inspection that cited several gets one call each. It feeds the fleet's CSA "+
			"BASIC scores.",
		"Adds a cited violation to a safety event inside Trenova; nothing is sent, and it is "+
			"corrected or removed the same way.",
		permission.ResourceWorkerSafetyEvent,
		permission.OpCreate,
	), properties, workforcekit.ParamSafetyEventID, changepreview.FieldDescription)

	spec.SearchTerms = []string{"roadside violation", "inspection violation", "add violation"}

	return toolspec.NewReportingReceivableTool(spec, toolspec.ReceivablePlan[
		*workersafetyservice.RecordViolationRequest, *fleetsafety.WorkerSafetyViolation,
	]{
		Request: recordViolationRequest,
		Plan: func(
			ctx context.Context,
			req *workersafetyservice.RecordViolationRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*fleetsafety.WorkerSafetyViolation, error) {
			return events.PlanRecordViolation(ctx, req)
		},
		Refused: func(*workersafetyservice.RecordViolationRequest) string {
			return "Would cite a roadside violation on the safety event."
		},
		Render: func(
			_ *workersafetyservice.RecordViolationRequest,
			planned *fleetsafety.WorkerSafetyViolation,
		) (*agentpreview.ToolPreview, error) {
			change, err := toolpreview.Create(
				workforcekit.Record(
					permission.ResourceWorkerSafetyEvent,
					pulid.Nil,
					"New violation",
					0,
				),
				planned,
				workforcekit.Options(violationFields...)...)
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
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			created, err := events.RecordViolation(ctx, req)
			if err != nil {
				return nil, err
			}
			return workforcekit.Result(
				"recorded",
				kindSafetyViolation,
				paramViolationID,
				created.ID,
				created.WorkerID,
			), nil
		},
	})
}

func recordViolationRequest(
	params *agenttoolports.ToolExecuteParams,
) (*workersafetyservice.RecordViolationRequest, error) {
	eventID, err := toolparams.RequirePulid(params.Params, workforcekit.ParamSafetyEventID)
	if err != nil {
		return nil, err
	}
	description, err := toolparams.RequireBoundedText(
		params.Params,
		changepreview.FieldDescription,
		workforcekit.NoteChars,
	)
	if err != nil {
		return nil, err
	}
	basic, _, err := toolparams.OptionalEnum(params.Params, paramCSABasic, csaBasics.Values)
	if err != nil {
		return nil, err
	}
	code, err := toolparams.BoundedText(
		params.Params,
		paramViolationCode,
		workforcekit.ShortChars,
	)
	if err != nil {
		return nil, err
	}
	weight, err := violationWeight(params.Params)
	if err != nil {
		return nil, err
	}
	outOfService, err := workforcekit.OptionalBoolParam(
		params.Params,
		paramOutOfService,
		false,
	)
	if err != nil {
		return nil, err
	}
	return &workersafetyservice.RecordViolationRequest{
		TenantInfo:     toolparams.TenantFrom(params),
		SafetyEventID:  eventID,
		Basic:          basic,
		Code:           code,
		Description:    description,
		SeverityWeight: weight,
		OutOfService:   outOfService,
		UserID:         params.Actor.UserID,
	}, nil
}

type violationEdit struct {
	params *agenttoolports.ToolExecuteParams
	id     pulid.ID
}

func (e *violationEdit) request(
	ctx context.Context,
	events SafetyKeeper,
) (*workersafetyservice.UpdateViolationRequest, error) {
	tenant := toolparams.TenantFrom(e.params)
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
	if req.Basic, _, err = toolparams.OptionalEnum(e.params.Params, paramCSABasic,
		csaBasics.Values); err != nil {
		return nil, err
	}
	if code, codeErr := toolparams.OptionalBoundedText(e.params.Params, paramViolationCode,
		workforcekit.ShortChars); codeErr != nil {
		return nil, codeErr
	} else if code != nil {
		req.Code = *code
	}
	if text, textErr := toolparams.OptionalBoundedText(
		e.params.Params,
		changepreview.FieldDescription,
		workforcekit.NoteChars,
	); textErr != nil {
		return nil, textErr
	} else if text != nil {
		req.Description = *text
	}
	if weight, weightErr := violationWeight(e.params.Params); weightErr != nil {
		return nil, weightErr
	} else if weight > 0 {
		req.SeverityWeight = weight
	}
	if req.OutOfService, err = workforcekit.OptionalBoolParam(e.params.Params, paramOutOfService,
		req.OutOfService); err != nil {
		return nil, err
	}
	return req, nil
}

func violationIDProperty() map[string]any {
	return agenttoolschema.KindID(
		agenttoolschema.IDDescription("The violation", "worker_list_safety_events"),
		recordkind.SafetyViolation,
	)
}

func newUpdateSafetyViolationTool(events SafetyKeeper) agenttoolports.AgentTool {
	properties := violationProperties()
	properties[paramViolationID] = violationIDProperty()
	spec := workforcekit.Targeting(workforcekit.WithSchema(workforcekit.Spec(
		"worker_update_safety_violation",
		"Correct a roadside violation cited on a safety event: its BASIC, regulation, "+
			"wording, severity weight or out-of-service flag. Give only what changes.",
		"Corrects a cited violation inside Trenova; nothing is sent, and it is corrected "+
			"again the same way.",
		permission.ResourceWorkerSafetyEvent,
		permission.OpUpdate,
	), properties, paramViolationID), paramViolationID, permission.ResourceWorkerSafetyEvent)
	spec.SearchTerms = []string{"fix violation", "basic", "severity weight"}

	return toolspec.NewReportingReceivableTool(spec,
		toolspec.ReceivablePlan[*violationEdit, *workersafetyservice.ViolationChange]{
			Request: func(params *agenttoolports.ToolExecuteParams) (*violationEdit, error) {
				id, err := toolparams.RequirePulid(params.Params, paramViolationID)
				if err != nil {
					return nil, err
				}
				return &violationEdit{params: params, id: id}, nil
			},
			Plan: func(
				ctx context.Context,
				edit *violationEdit,
				_ *agenttoolports.ToolExecuteParams,
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
			) (*agentpreview.ToolPreview, error) {
				recorded, err := toolpreview.Changed(violationRecord(change.Before),
					change.Before, change.After, workforcekit.Options(violationFields...)...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build("Would correct the cited violation.", recorded), nil
			},
			Run: func(
				ctx context.Context,
				edit *violationEdit,
				_ *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				req, err := edit.request(ctx, events)
				if err != nil {
					return nil, err
				}
				updated, err := events.UpdateViolation(ctx, req)
				if err != nil {
					return nil, err
				}
				return workforcekit.Result(
					"updated",
					kindSafetyViolation,
					paramViolationID,
					updated.ID,
					updated.WorkerID,
				), nil
			},
		})
}

func newDeleteSafetyViolationTool(events SafetyKeeper) agenttoolports.AgentTool {
	spec := workforcekit.Targeting(workforcekit.WithSchema(workforcekit.Spec(
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

	return toolspec.NewReceivableTool(
		spec,
		toolspec.ReceivablePlan[*workforcekit.RecordDelete, *fleetsafety.WorkerSafetyViolation]{
			Request: workforcekit.RecordDeleteFrom(paramViolationID),
			Plan: func(
				ctx context.Context,
				del *workforcekit.RecordDelete,
				_ *agenttoolports.ToolExecuteParams,
			) (*fleetsafety.WorkerSafetyViolation, error) {
				return events.PlanDeleteViolation(ctx, del.Tenant, del.ID)
			},
			Refused: func(*workforcekit.RecordDelete) string { return "Would remove a cited violation." },
			Render: func(
				_ *workforcekit.RecordDelete,
				violation *fleetsafety.WorkerSafetyViolation,
			) (*agentpreview.ToolPreview, error) {
				change, err := toolpreview.Delete(violationRecord(violation), violation,
					workforcekit.Options(violationFields...)...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build(fmt.Sprintf("Would remove the %s violation.",
					violation.Basic), change), nil
			},
			Run: func(
				ctx context.Context,
				del *workforcekit.RecordDelete,
				_ *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				return nil, events.DeleteViolation(ctx, del.Tenant, del.ID, del.UserID)
			},
		},
	)
}

func provideOpenWorkerSafetyEventTool(s *workersafetyservice.Service) agenttoolports.AgentTool {
	return newOpenWorkerSafetyEventTool(s)
}

func provideUpdateWorkerSafetyEventTool(s *workersafetyservice.Service) agenttoolports.AgentTool {
	return newUpdateWorkerSafetyEventTool(s)
}

func provideChangeWorkerSafetyEventStatusTool(
	s *workersafetyservice.Service,
) agenttoolports.AgentTool {
	return newChangeWorkerSafetyEventStatusTool(s)
}

func provideDeleteWorkerSafetyEventTool(s *workersafetyservice.Service) agenttoolports.AgentTool {
	return newDeleteWorkerSafetyEventTool(s)
}

func provideGiveWorkerRecognitionTool(s *workersafetyservice.Service) agenttoolports.AgentTool {
	return newGiveWorkerRecognitionTool(s)
}

func provideDeleteWorkerRecognitionTool(s *workersafetyservice.Service) agenttoolports.AgentTool {
	return newDeleteWorkerRecognitionTool(s)
}

func provideRecordSafetyViolationTool(s *workersafetyservice.Service) agenttoolports.AgentTool {
	return newRecordSafetyViolationTool(s)
}

func provideUpdateSafetyViolationTool(s *workersafetyservice.Service) agenttoolports.AgentTool {
	return newUpdateSafetyViolationTool(s)
}

func provideDeleteSafetyViolationTool(s *workersafetyservice.Service) agenttoolports.AgentTool {
	return newDeleteSafetyViolationTool(s)
}
