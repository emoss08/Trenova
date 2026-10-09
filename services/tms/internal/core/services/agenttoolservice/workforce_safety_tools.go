package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/internal/core/services/workersafetyservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramSafetyEventID   = "safetyEventId"
	paramRecognitionID   = "recognitionId"
	paramViolationID     = "violationId"
	paramEventKind       = "kind"
	paramEventSeverity   = "severity"
	paramEventLocation   = "location"
	paramPreventable     = "preventable"
	paramPoints          = "points"
	paramReferenceNumber = "referenceNumber"
	paramInspectionLevel = "inspectionLevel"
	paramInspectionRes   = "inspectionResult"
	paramOutOfService    = "outOfService"
	paramFineAmount      = "fineAmount"
	paramCostAmount      = "costAmount"
	paramResolutionText  = "resolution"
	paramEventMove       = "action"
	paramRecognitionKind = "kind"
	paramTitle           = "title"
	paramVisibleToWorker = "visibleToWorker"
	paramCSABasic        = "basic"
	paramViolationCode   = "code"
	paramSeverityWeight  = "severityWeight"
	kindSafetyEvent      = "safety event"
	kindRecognition      = "recognition"
	kindSafetyViolation  = "roadside violation"
	maxSafetyPoints      = 100
	maxInspectionLevel   = 8
	maxSeverityWeight    = 10
)

type safetyEventMove string

const (
	safetyEventReview = safetyEventMove("Review")
	safetyEventClose  = safetyEventMove("Close")
	safetyEventReopen = safetyEventMove("Reopen")
)

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

var (
	safetyEventKinds = agenttoolschema.Source(
		"worker.safetyEventKind",
		worker.SafetyEventKindValues(),
	)
	safetySeverities = agenttoolschema.Source(
		"worker.safetySeverity",
		worker.SafetySeverityValues(),
	)
	inspectionResults = agenttoolschema.Source(
		"worker.inspectionResult",
		worker.InspectionResultValues(),
	)
	recognitionKinds = agenttoolschema.Source(
		"worker.recognitionKind",
		worker.RecognitionKindValues(),
	)
	csaBasics        = agenttoolschema.Source("worker.csaBasic", worker.AllCSABasics())
	safetyEventMoves = agenttoolschema.Source(
		"workerSafetyEvent.agentMove",
		[]safetyEventMove{safetyEventReview, safetyEventClose, safetyEventReopen},
	)
	safetyEventFields = []string{
		paramEventKind, paramEventSeverity, fieldStatus, wfParamOccurred, paramEventLocation,
		fieldDescription, paramPreventable, paramPoints, "pointsExpireAt", paramReferenceNumber,
		fieldShipmentID, paramInspectionLevel, paramInspectionRes, paramOutOfService,
		paramFineAmount, paramCostAmount, wfFieldDocument, paramResolutionText,
	}
)

type safetyKeeper interface {
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

var _ safetyKeeper = (*workersafetyservice.Service)(nil)

func safetyToolProviders() []any {
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
		"list_worker_safety_events")
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
		fieldDescription: stringProperty("What happened, in the words of the report or "+
			"the person who told you.", wfNoteChars),
		paramEventLocation: stringProperty("Where it happened.", wfShortChars),
		paramPreventable: booleanProperty("For an accident: whether it was preventable. " +
			"Leave it out unless the report says so."),
		paramPoints: integerProperty("The points it carries on the driver's record. Leave "+
			"it out on a new event to take the house scale for the kind and severity.", 0,
			maxSafetyPoints),
		paramReferenceNumber: stringProperty("A citation, report or inspection number.",
			wfShortChars),
		fieldShipmentID: agenttoolschema.RecordIDText(
			permission.ResourceShipment,
			"The load it happened on, from search_shipments or "+
				"get_shipment. Never guess one.",
		),
		paramInspectionLevel: integerProperty("For an inspection: its CVSA level.", 1,
			maxInspectionLevel),
		paramInspectionRes: agenttoolschema.Enum("For an inspection: how it ended.",
			inspectionResults),
		paramOutOfService: booleanProperty("Whether the driver or the vehicle was put out " +
			"of service."),
		paramFineAmount: amountProperty("A fine, as a decimal such as 250.00."),
		paramCostAmount: amountProperty("What it cost the company, as a decimal such as " +
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
	kind, kindGiven, kindErr := optionalEnum(params, paramEventKind, safetyEventKinds.Values)
	switch {
	case kindErr != nil:
		return kindErr
	case kindGiven:
		entity.Kind = kind
	case required.kind:
		return fmt.Errorf("missing required parameter %q", paramEventKind)
	}
	if severity, given, err := optionalEnum(
		params, paramEventSeverity, safetySeverities.Values,
	); err != nil {
		return err
	} else if given {
		entity.Severity = severity
	} else if required.severity {
		return fmt.Errorf("missing required parameter %q", paramEventSeverity)
	}
	occurred, err := optionalDateTime(params, wfParamOccurred)
	switch {
	case err != nil:
		return err
	case occurred != nil:
		entity.OccurredAt = *occurred
	case required.occurred:
		return fmt.Errorf("missing required parameter %q", wfParamOccurred)
	}
	description, err := optionalBoundedText(params, fieldDescription, wfNoteChars)
	switch {
	case err != nil:
		return err
	case description != nil:
		entity.Description = *description
	case required.description:
		return fmt.Errorf("missing required parameter %q", fieldDescription)
	}
	return nil
}

func applySafetyEventDetail(entity *worker.WorkerSafetyEvent, params map[string]any) error {
	var err error
	location, locErr := optionalBoundedText(params, paramEventLocation, wfShortChars)
	if locErr != nil {
		return locErr
	}
	if location != nil {
		entity.Location = *location
	}
	if reference, refErr := optionalBoundedText(
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
	if result, given, resErr := optionalEnum(
		params, paramInspectionRes, inspectionResults.Values,
	); resErr != nil {
		return resErr
	} else if given {
		entity.InspectionResult = result
	}
	level, err := optionalIntInRange(params, paramInspectionLevel, 1, maxInspectionLevel)
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
	shipmentID, err := optionalID(params, fieldShipmentID)
	if err != nil {
		return err
	}
	if !shipmentID.IsNil() {
		entity.ShipmentID = shipmentID
	}
	documentID, err := optionalID(params, wfParamDocument)
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
	points, err := optionalIntInRange(params, paramPoints, 0, maxSafetyPoints)
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

func newOpenWorkerSafetyEventTool(events safetyKeeper) serviceports.AgentTool {
	properties := safetyEventProperties()
	properties[paramWorkerID] = workerProperty()
	spec := withSchema(wfSpec(
		"open_worker_safety_event",
		"Record a safety event on a driver's record: an accident, incident, near miss, "+
			"citation or roadside inspection, with when and what happened. It opens Open and "+
			"counts its points on the driver's scorecard; roadside violations cited on an "+
			"inspection are added after with record_safety_violation.",
		"Adds an event to a driver's safety record inside Trenova; nothing is sent to the "+
			"driver, and it is corrected or deleted while it is open.",
		permission.ResourceWorkerSafetyEvent,
		permission.OpCreate,
	), properties, paramWorkerID, paramEventKind, paramEventSeverity, wfParamOccurred,
		fieldDescription)
	spec.searchTerms = []string{"accident", "incident", "inspection", "citation", "near miss"}

	return newReportingReceivableTool(spec,
		receivablePlan[*worker.WorkerSafetyEvent, *worker.WorkerSafetyEvent]{
			request: func(params *serviceports.ToolExecuteParams) (*worker.WorkerSafetyEvent, error) {
				workerID, err := requirePulid(params.Params, paramWorkerID)
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
			plan: func(
				ctx context.Context,
				entity *worker.WorkerSafetyEvent,
				params *serviceports.ToolExecuteParams,
			) (*worker.WorkerSafetyEvent, error) {
				return events.PlanCreateEvent(ctx, entity, params.Actor.UserID)
			},
			refused: func(*worker.WorkerSafetyEvent) string {
				return "Would record a safety event on the driver's record."
			},
			render: func(
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
			run: func(
				ctx context.Context,
				entity *worker.WorkerSafetyEvent,
				params *serviceports.ToolExecuteParams,
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
	params *serviceports.ToolExecuteParams
}

func (e *safetyEventEdit) entity(
	ctx context.Context,
	events safetyKeeper,
) (*worker.WorkerSafetyEvent, error) {
	current, err := events.GetEvent(ctx, tenantFrom(*e.params), e.id)
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
	resolution, err := optionalBoundedText(e.params.Params, paramResolutionText, wfNoteChars)
	if err != nil {
		return nil, err
	}
	if resolution != nil {
		entity.Resolution = *resolution
	}
	return &entity, nil
}

func newUpdateWorkerSafetyEventTool(events safetyKeeper) serviceports.AgentTool {
	properties := safetyEventProperties()
	properties[paramSafetyEventID] = safetyEventIDProperty()
	properties[paramResolutionText] = wfNoteProperty("How it was resolved, kept on the event.")
	spec := targeting(withSchema(wfSpec(
		"update_worker_safety_event",
		"Correct a safety event on a driver's record: what happened, when, the points it "+
			"carries, the inspection result, the fine or cost. Give only what changes. It "+
			"never moves the event's status; change_worker_safety_event_status does.",
		"Corrects an event on a driver's safety record inside Trenova; nothing is sent, and "+
			"the event is corrected again the same way.",
		permission.ResourceWorkerSafetyEvent,
		permission.OpUpdate,
	), properties, paramSafetyEventID), paramSafetyEventID, permission.ResourceWorkerSafetyEvent)

	return newReportingReceivableTool(spec,
		receivablePlan[*safetyEventEdit, *workersafetyservice.EventChange]{
			request: func(params *serviceports.ToolExecuteParams) (*safetyEventEdit, error) {
				id, err := requirePulid(params.Params, paramSafetyEventID)
				if err != nil {
					return nil, err
				}
				return &safetyEventEdit{id: id, params: params}, nil
			},
			plan: func(
				ctx context.Context,
				edit *safetyEventEdit,
				_ *serviceports.ToolExecuteParams,
			) (*workersafetyservice.EventChange, error) {
				entity, err := edit.entity(ctx, events)
				if err != nil {
					return nil, err
				}
				return events.PlanUpdateEvent(ctx, entity)
			},
			refused: func(*safetyEventEdit) string {
				return "Would correct a safety event on the driver's record."
			},
			render: func(
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
			run: func(
				ctx context.Context,
				edit *safetyEventEdit,
				params *serviceports.ToolExecuteParams,
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
	events safetyKeeper,
) (*workersafetyservice.EventChange, error) {
	switch m.move {
	case safetyEventClose:
		return events.PlanCloseEvent(ctx, m.req)
	case safetyEventReview:
		return events.PlanReviewEvent(ctx, m.req)
	case safetyEventReopen:
		return events.PlanReopenEvent(ctx, m.req)
	}
	return nil, errUnknownValue(paramEventMove, string(m.move), safetyEventMoves.Names())
}

func (m *safetyEventStatusMove) run(
	ctx context.Context,
	events safetyKeeper,
) (*worker.WorkerSafetyEvent, error) {
	switch m.move {
	case safetyEventClose:
		return events.CloseEvent(ctx, m.req)
	case safetyEventReview:
		return events.ReviewEvent(ctx, m.req)
	case safetyEventReopen:
		return events.ReopenEvent(ctx, m.req)
	}
	return nil, errUnknownValue(paramEventMove, string(m.move), safetyEventMoves.Names())
}

func newChangeWorkerSafetyEventStatusTool(events safetyKeeper) serviceports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"change_worker_safety_event_status",
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

	return newReportingReceivableTool(spec,
		receivablePlan[*safetyEventStatusMove, *workersafetyservice.EventChange]{
			request: func(params *serviceports.ToolExecuteParams) (*safetyEventStatusMove, error) {
				id, err := requirePulid(params.Params, paramSafetyEventID)
				if err != nil {
					return nil, err
				}
				move, err := requireEnum(params.Params, paramEventMove, safetyEventMoves.Values)
				if err != nil {
					return nil, err
				}
				resolution, err := boundedText(params.Params, paramResolutionText, wfNoteChars)
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
						TenantInfo: tenantFrom(*params),
						Resolution: resolution,
						UserID:     params.Actor.UserID,
					},
				}, nil
			},
			plan: func(
				ctx context.Context,
				move *safetyEventStatusMove,
				_ *serviceports.ToolExecuteParams,
			) (*workersafetyservice.EventChange, error) {
				return move.plan(ctx, events)
			},
			refused: func(move *safetyEventStatusMove) string {
				return fmt.Sprintf("Would %s the safety event.",
					strings.ToLower(string(move.move)))
			},
			render: func(
				move *safetyEventStatusMove,
				change *workersafetyservice.EventChange,
			) (*agent.ToolPreview, error) {
				recorded, err := toolpreview.Changed(safetyEventRecord(change.Before),
					change.Before, change.After,
					wfOptions(fieldStatus, wfFieldClosedAt, "closedById", paramResolutionText)...)
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
			},
			run: func(
				ctx context.Context,
				move *safetyEventStatusMove,
				_ *serviceports.ToolExecuteParams,
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

type recordDelete struct {
	id     pulid.ID
	tenant pagination.TenantInfo
	userID pulid.ID
}

func recordDeleteFrom(key string) func(*serviceports.ToolExecuteParams) (*recordDelete, error) {
	return func(params *serviceports.ToolExecuteParams) (*recordDelete, error) {
		id, err := requirePulid(params.Params, key)
		if err != nil {
			return nil, err
		}
		return &recordDelete{id: id, tenant: tenantFrom(*params), userID: params.Actor.UserID}, nil
	}
}

func newDeleteWorkerSafetyEventTool(events safetyKeeper) serviceports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"delete_worker_safety_event",
		"Delete a safety event recorded in error, taking its points off the driver's "+
			"record. A closed event is history and is refused: reopen it first if it truly "+
			"never happened.",
		"Removes an event recorded in error from a driver's safety record; the audit trail "+
			"keeps what was deleted, and it cannot be put back.",
		permission.ResourceWorkerSafetyEvent,
		permission.OpDelete,
	), map[string]any{paramSafetyEventID: safetyEventIDProperty()}, paramSafetyEventID),
		paramSafetyEventID, permission.ResourceWorkerSafetyEvent)
	spec.maxTier = agent.TierPropose
	spec.reversible = false

	return newReceivableTool(spec, receivablePlan[*recordDelete, *worker.WorkerSafetyEvent]{
		request: recordDeleteFrom(paramSafetyEventID),
		plan: func(
			ctx context.Context,
			del *recordDelete,
			_ *serviceports.ToolExecuteParams,
		) (*worker.WorkerSafetyEvent, error) {
			return events.PlanDeleteEvent(ctx, del.tenant, del.id)
		},
		refused: func(*recordDelete) string {
			return "Would delete a safety event from the driver's record."
		},
		render: func(_ *recordDelete, event *worker.WorkerSafetyEvent) (*agent.ToolPreview, error) {
			change, err := toolpreview.Delete(safetyEventRecord(event), event,
				wfOptions(safetyEventFields...)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf(
				"Would delete the %s and its %d points from the driver's record.",
				strings.ToLower(event.Kind.String()), event.Points), change), nil
		},
		run: func(
			ctx context.Context,
			del *recordDelete,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			return nil, events.DeleteEvent(ctx, del.tenant, del.id, del.userID)
		},
	})
}

func recognitionFrom(params *serviceports.ToolExecuteParams) (*worker.WorkerRecognition, error) {
	workerID, err := requirePulid(params.Params, paramWorkerID)
	if err != nil {
		return nil, err
	}
	kind, err := requireEnum(params.Params, paramRecognitionKind, recognitionKinds.Values)
	if err != nil {
		return nil, err
	}
	title, err := requireBoundedText(params.Params, paramTitle, wfShortChars)
	if err != nil {
		return nil, err
	}
	message, err := boundedText(params.Params, fieldMessage, wfNoteChars)
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
	occurred, err := optionalDateTime(params.Params, wfParamOccurred)
	if err != nil {
		return nil, err
	}
	if occurred != nil {
		entity.OccurredAt = *occurred
	}
	return entity, nil
}

var recognitionFields = []string{
	wfFieldWorkerID, paramRecognitionKind, paramTitle, fieldMessage, wfParamOccurred,
	paramVisibleToWorker,
}

func newGiveWorkerRecognitionTool(events safetyKeeper) serviceports.AgentTool {
	spec := withSchema(wfSpec(
		"give_worker_recognition",
		"Recognise a driver for a safety milestone, customer praise, performance or tenure. "+
			"When it is visible to the driver, which is the default, they are told in Dash "+
			"with the title and message you write, so write it as you would say it to them.",
		"The driver is shown the recognition and told of it in Dash when it is visible to them.",
		permission.ResourceWorkerRecognition,
		permission.OpCreate,
	), map[string]any{
		paramWorkerID:        workerProperty(),
		paramRecognitionKind: agenttoolschema.Enum("What it is for.", recognitionKinds),
		paramTitle:           stringProperty("A short title the driver sees.", wfShortChars),
		fieldMessage:         wfNoteProperty("What you would say to them."),
		wfParamOccurred:      agenttoolschema.DateTime("When it was earned. Defaults to now."),
		paramVisibleToWorker: booleanProperty("Whether the driver sees it and is told. " +
			"Defaults to true."),
	}, paramWorkerID, paramRecognitionKind, paramTitle)
	spec.egress = agent.EgressDriverVisible

	spec.searchTerms = []string{"safe miles award", "recognize driver", "safety award", "praise"}

	return newReportingReceivableTool(spec,
		receivablePlan[*worker.WorkerRecognition, *worker.WorkerRecognition]{
			request: recognitionFrom,
			plan: func(
				ctx context.Context,
				entity *worker.WorkerRecognition,
				params *serviceports.ToolExecuteParams,
			) (*worker.WorkerRecognition, error) {
				return events.PlanGiveRecognition(ctx, entity, params.Actor.UserID)
			},
			refused: func(*worker.WorkerRecognition) string {
				return "Would recognise the driver."
			},
			render: func(
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
			run: func(
				ctx context.Context,
				entity *worker.WorkerRecognition,
				params *serviceports.ToolExecuteParams,
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

func newDeleteWorkerRecognitionTool(events safetyKeeper) serviceports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"delete_worker_recognition",
		"Remove a recognition given in error. If it was visible, it leaves the driver's "+
			"record in Dash too.",
		"Removes a recognition from a driver's record; it is given again the same way.",
		permission.ResourceWorkerRecognition,
		permission.OpDelete,
	), map[string]any{
		paramRecognitionID: agenttoolschema.RecordIDText(
			permission.ResourceWorkerRecognition,
			"The recognition, from list_worker_safety_events. "+
				"Never guess one.",
		),
	}, paramRecognitionID), paramRecognitionID, permission.ResourceWorkerRecognition)
	spec.searchTerms = []string{"kudos", "remove recognition"}
	spec.maxTier = agent.TierPropose
	spec.reversible = false

	return newReceivableTool(spec, receivablePlan[*recordDelete, *worker.WorkerRecognition]{
		request: recordDeleteFrom(paramRecognitionID),
		plan: func(
			ctx context.Context,
			del *recordDelete,
			_ *serviceports.ToolExecuteParams,
		) (*worker.WorkerRecognition, error) {
			return events.PlanDeleteRecognition(ctx, del.tenant, del.id)
		},
		refused: func(*recordDelete) string { return "Would remove a recognition." },
		render: func(_ *recordDelete, rec *worker.WorkerRecognition) (*agent.ToolPreview, error) {
			change, err := toolpreview.Delete(
				wfRecord(permission.ResourceWorkerRecognition, rec.ID, rec.Title, rec.Version),
				rec, wfOptions(recognitionFields...)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf("Would remove the recognition %q.",
				rec.Title), change), nil
		},
		run: func(
			ctx context.Context,
			del *recordDelete,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			return nil, events.DeleteRecognition(ctx, del.tenant, del.id, del.userID)
		},
	})
}

var violationFields = []string{
	paramSafetyEventID, wfFieldWorkerID, paramCSABasic, paramViolationCode, fieldDescription,
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
		paramViolationCode: stringProperty("The regulation cited, such as 393.47(e).",
			wfShortChars),
		fieldDescription: stringProperty("What was cited, as the inspection report words it.",
			wfNoteChars),
		paramSeverityWeight: integerProperty("The FMCSA severity weight, 1 to 10.", 1,
			maxSeverityWeight),
		paramOutOfService: booleanProperty("Whether this violation put the driver or " +
			"vehicle out of service."),
	}
}

func violationWeight(params map[string]any) (int16, error) {
	weight, err := optionalIntInRange(params, paramSeverityWeight, 1, maxSeverityWeight)
	if err != nil || weight == nil {
		return 0, err
	}
	return int16(*weight), nil //nolint:gosec // bounded by the schema's maximum
}

func newRecordSafetyViolationTool(events safetyKeeper) serviceports.AgentTool {
	properties := violationProperties()
	properties[paramSafetyEventID] = safetyEventIDProperty()
	spec := withSchema(wfSpec(
		"record_safety_violation",
		"Cite one roadside violation on a safety event, usually an inspection, with its "+
			"BASIC, the regulation, the severity weight and whether it was out of service. "+
			"An inspection that cited several gets one call each. It feeds the fleet's CSA "+
			"BASIC scores.",
		"Adds a cited violation to a safety event inside Trenova; nothing is sent, and it is "+
			"corrected or removed the same way.",
		permission.ResourceWorkerSafetyEvent,
		permission.OpCreate,
	), properties, paramSafetyEventID, fieldDescription)

	spec.searchTerms = []string{"roadside violation", "inspection violation", "add violation"}

	return newReportingReceivableTool(spec, receivablePlan[
		*workersafetyservice.RecordViolationRequest, *worker.WorkerSafetyViolation,
	]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*workersafetyservice.RecordViolationRequest, error) {
			eventID, err := requirePulid(params.Params, paramSafetyEventID)
			if err != nil {
				return nil, err
			}
			description, err := requireBoundedText(params.Params, fieldDescription, wfNoteChars)
			if err != nil {
				return nil, err
			}
			basic, _, err := optionalEnum(params.Params, paramCSABasic, csaBasics.Values)
			if err != nil {
				return nil, err
			}
			code, err := boundedText(params.Params, paramViolationCode, wfShortChars)
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
				TenantInfo:     tenantFrom(*params),
				SafetyEventID:  eventID,
				Basic:          basic,
				Code:           code,
				Description:    description,
				SeverityWeight: weight,
				OutOfService:   outOfService,
				UserID:         params.Actor.UserID,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *workersafetyservice.RecordViolationRequest,
			_ *serviceports.ToolExecuteParams,
		) (*worker.WorkerSafetyViolation, error) {
			return events.PlanRecordViolation(ctx, req)
		},
		refused: func(*workersafetyservice.RecordViolationRequest) string {
			return "Would cite a roadside violation on the safety event."
		},
		render: func(
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
		run: func(
			ctx context.Context,
			req *workersafetyservice.RecordViolationRequest,
			_ *serviceports.ToolExecuteParams,
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
	params *serviceports.ToolExecuteParams
	id     pulid.ID
}

func (e *violationEdit) request(
	ctx context.Context,
	events safetyKeeper,
) (*workersafetyservice.UpdateViolationRequest, error) {
	tenant := tenantFrom(*e.params)
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
	if req.Basic, _, err = optionalEnum(e.params.Params, paramCSABasic,
		csaBasics.Values); err != nil {
		return nil, err
	}
	if code, codeErr := optionalBoundedText(e.params.Params, paramViolationCode,
		wfShortChars); codeErr != nil {
		return nil, codeErr
	} else if code != nil {
		req.Code = *code
	}
	if text, textErr := optionalBoundedText(e.params.Params, fieldDescription,
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
		agenttoolschema.IDDescription("The violation", "list_worker_safety_events"),
		permission.KindSafetyViolation,
	)
}

func newUpdateSafetyViolationTool(events safetyKeeper) serviceports.AgentTool {
	properties := violationProperties()
	properties[paramViolationID] = violationIDProperty()
	spec := targeting(withSchema(wfSpec(
		"update_safety_violation",
		"Correct a roadside violation cited on a safety event: its BASIC, regulation, "+
			"wording, severity weight or out-of-service flag. Give only what changes.",
		"Corrects a cited violation inside Trenova; nothing is sent, and it is corrected "+
			"again the same way.",
		permission.ResourceWorkerSafetyEvent,
		permission.OpUpdate,
	), properties, paramViolationID), paramViolationID, permission.ResourceWorkerSafetyEvent)
	spec.searchTerms = []string{"fix violation", "basic", "severity weight"}

	return newReportingReceivableTool(spec,
		receivablePlan[*violationEdit, *workersafetyservice.ViolationChange]{
			request: func(params *serviceports.ToolExecuteParams) (*violationEdit, error) {
				id, err := requirePulid(params.Params, paramViolationID)
				if err != nil {
					return nil, err
				}
				return &violationEdit{params: params, id: id}, nil
			},
			plan: func(
				ctx context.Context,
				edit *violationEdit,
				_ *serviceports.ToolExecuteParams,
			) (*workersafetyservice.ViolationChange, error) {
				req, err := edit.request(ctx, events)
				if err != nil {
					return nil, err
				}
				return events.PlanUpdateViolation(ctx, req)
			},
			refused: func(*violationEdit) string { return "Would correct a cited violation." },
			render: func(
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
			run: func(
				ctx context.Context,
				edit *violationEdit,
				_ *serviceports.ToolExecuteParams,
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

func newDeleteSafetyViolationTool(events safetyKeeper) serviceports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"delete_safety_violation",
		"Remove a roadside violation cited on a safety event in error, taking it out of "+
			"the fleet's CSA BASIC scores.",
		"Removes a cited violation inside Trenova; the audit trail keeps what was removed, "+
			"and it is cited again with record_safety_violation.",
		permission.ResourceWorkerSafetyEvent,
		permission.OpDelete,
	), map[string]any{paramViolationID: violationIDProperty()}, paramViolationID),
		paramViolationID, permission.ResourceWorkerSafetyEvent)
	spec.searchTerms = []string{"remove violation", "csa"}
	spec.maxTier = agent.TierPropose
	spec.reversible = false

	return newReceivableTool(spec, receivablePlan[*recordDelete, *worker.WorkerSafetyViolation]{
		request: recordDeleteFrom(paramViolationID),
		plan: func(
			ctx context.Context,
			del *recordDelete,
			_ *serviceports.ToolExecuteParams,
		) (*worker.WorkerSafetyViolation, error) {
			return events.PlanDeleteViolation(ctx, del.tenant, del.id)
		},
		refused: func(*recordDelete) string { return "Would remove a cited violation." },
		render: func(
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
		run: func(
			ctx context.Context,
			del *recordDelete,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			return nil, events.DeleteViolation(ctx, del.tenant, del.id, del.userID)
		},
	})
}

func provideOpenWorkerSafetyEventTool(s *workersafetyservice.Service) serviceports.AgentTool {
	return newOpenWorkerSafetyEventTool(s)
}

func provideUpdateWorkerSafetyEventTool(s *workersafetyservice.Service) serviceports.AgentTool {
	return newUpdateWorkerSafetyEventTool(s)
}

func provideChangeWorkerSafetyEventStatusTool(
	s *workersafetyservice.Service,
) serviceports.AgentTool {
	return newChangeWorkerSafetyEventStatusTool(s)
}

func provideDeleteWorkerSafetyEventTool(s *workersafetyservice.Service) serviceports.AgentTool {
	return newDeleteWorkerSafetyEventTool(s)
}

func provideGiveWorkerRecognitionTool(s *workersafetyservice.Service) serviceports.AgentTool {
	return newGiveWorkerRecognitionTool(s)
}

func provideDeleteWorkerRecognitionTool(s *workersafetyservice.Service) serviceports.AgentTool {
	return newDeleteWorkerRecognitionTool(s)
}

func provideRecordSafetyViolationTool(s *workersafetyservice.Service) serviceports.AgentTool {
	return newRecordSafetyViolationTool(s)
}

func provideUpdateSafetyViolationTool(s *workersafetyservice.Service) serviceports.AgentTool {
	return newUpdateSafetyViolationTool(s)
}

func provideDeleteSafetyViolationTool(s *workersafetyservice.Service) serviceports.AgentTool {
	return newDeleteSafetyViolationTool(s)
}
