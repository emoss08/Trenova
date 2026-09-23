package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/carrierintelservice"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/internal/core/services/workercredentialservice"
	"github.com/emoss08/trenova/internal/core/services/workerservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

/*
The writes the autonomous desks run on.

Each one is reversible or additive on purpose. A detention clock is escalated
rather than waived, a driver is held rather than deactivated, a carrier is
blocked with an override that expires — because a desk acts on a schedule and
a mistake it makes has to be something a person can undo on Monday.
*/

const (
	// A tender block is an override with a life: carrier records are
	// corrected, and a block nobody revisits outlives the problem.
	tenderBlockDays    = 30
	secondsPerBlockDay = int64(86400)
	// The longest a renewal can be asked for before the ask stops being a
	// reminder and becomes a plan.
	maxRenewalNoteChars = 2000
)

// ---------------------------------------------------------------- detention

type detentionEscalator interface {
	Escalate(
		ctx context.Context,
		params detentionservice.EscalateParams,
	) (*detention.DetentionOccurrence, error)
	GetOccurrenceDetail(
		ctx context.Context,
		req *repositories.GetDetentionOccurrenceByIDRequest,
	) (*detentionservice.OccurrenceDetail, error)
}

type escalateDetentionTool struct {
	detention detentionEscalator
}

func newEscalateDetentionTool(detention detentionEscalator) serviceports.AgentTool {
	return &escalateDetentionTool{detention: detention}
}

func (t *escalateDetentionTool) Name() string { return "escalate_detention" }

func (t *escalateDetentionTool) Description() string {
	return "Hand a detention clock to a person, with the reason it cannot be worked " +
		"automatically. Use it when the notice window has closed, a gate is holding the " +
		"notice back, or the customer has nobody on file to send it to. This does not change " +
		"the charge — waiving, approving and disputing stay a person's decision — it " +
		"only puts the occurrence in front of one."
}

func (t *escalateDetentionTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"occurrenceId": map[string]any{
				"type": "string",
				"description": "The occurrence id, from list_detention_desk or " +
					"get_detention_occurrence.",
			},
			"reason": map[string]any{
				"type": "string",
				"description": "Why this needs a person, in a sentence they can act on: " +
					"what is blocking the notice, and what the charge stands at.",
			},
		},
		"required":             []string{"occurrenceId", "reason"},
		"additionalProperties": false,
	}
}

func (t *escalateDetentionTool) Reversible() bool { return true }

func (t *escalateDetentionTool) PermissionResource() permission.Resource {
	return permission.ResourceDetentionPolicy
}

func (t *escalateDetentionTool) PermissionOperation() permission.Operation {
	return permission.OpUpdate
}

func (t *escalateDetentionTool) RequiresIdempotencyKey() bool { return false }

func (t *escalateDetentionTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierPropose
}

func (t *escalateDetentionTool) arguments(
	params serviceports.ToolExecuteParams,
) (pulid.ID, string, error) {
	if err := guardExecute(t, params); err != nil {
		return "", "", err
	}

	occurrenceID, err := requirePulid(params.Params, "occurrenceId")
	if err != nil {
		return "", "", err
	}
	reason, err := requireString(params.Params, "reason")
	if err != nil {
		return "", "", err
	}

	return occurrenceID, strings.TrimSpace(reason), nil
}

func (t *escalateDetentionTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	occurrenceID, _, err := t.arguments(params)
	if err != nil {
		return err
	}

	occurrence, err := t.occurrence(ctx, occurrenceID, params)
	if err != nil {
		return err
	}

	return occurrence.Escalate(timeutils.NowUnix())
}

func (t *escalateDetentionTool) Simulate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolSimulation, error) {
	occurrenceID, reason, err := t.arguments(params)
	if err != nil {
		return nil, err
	}

	occurrence, err := t.occurrence(ctx, occurrenceID, params)
	if err != nil {
		return nil, err
	}

	before := *occurrence
	if eErr := occurrence.Escalate(timeutils.NowUnix()); eErr != nil {
		return nil, eErr
	}

	changes := []agent.FieldChange{
		{Field: "requiresApproval", From: "false", To: "true"},
	}
	if occurrence.NotificationStatus != before.NotificationStatus {
		changes = append(changes, agent.FieldChange{
			Field: "notificationStatus",
			From:  string(before.NotificationStatus),
			To:    string(occurrence.NotificationStatus),
		})
	}

	return &agent.ToolSimulation{
		Summary: fmt.Sprintf(
			"Would put detention occurrence %s (%d billable minutes) in front of a person: %s",
			occurrenceID, before.BillableMinutes, reason,
		),
		Changes:   changes,
		Previewed: true,
	}, nil
}

func (t *escalateDetentionTool) occurrence(
	ctx context.Context,
	occurrenceID pulid.ID,
	params serviceports.ToolExecuteParams,
) (*detention.DetentionOccurrence, error) {
	detail, err := t.detention.GetOccurrenceDetail(
		ctx,
		&repositories.GetDetentionOccurrenceByIDRequest{
			OccurrenceID: occurrenceID,
			TenantInfo:   tenantFrom(params),
		},
	)
	if err != nil {
		return nil, err
	}

	return detail.Occurrence, nil
}

func (t *escalateDetentionTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	occurrenceID, reason, err := t.arguments(params)
	if err != nil {
		return err
	}

	_, err = t.detention.Escalate(ctx, detentionservice.EscalateParams{
		OccurrenceID: occurrenceID,
		TenantInfo:   tenantFrom(params),
		Reason:       reason,
		UserID:       params.Actor.UserID,
	})

	return err
}

func (t *escalateDetentionTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "occurrenceId", permission.ResourceDetentionPolicy)
}

// --------------------------------------------------------------- credential

type credentialActor interface {
	RequestRenewal(ctx context.Context, req workercredentialservice.RenewalRequest) error
}

type requestCredentialRenewalTool struct {
	credentials credentialActor
}

func newRequestCredentialRenewalTool(credentials credentialActor) serviceports.AgentTool {
	return &requestCredentialRenewalTool{credentials: credentials}
}

func (t *requestCredentialRenewalTool) Name() string { return "request_credential_renewal" }

func (t *requestCredentialRenewalTool) Description() string {
	return "Ask a driver to renew the papers that are coming due, in one message covering " +
		"every credential rather than one per certificate. Name each credential and its " +
		"expiry date. This only asks; it does not record a renewal or change what is on file."
}

func (t *requestCredentialRenewalTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"workerId": map[string]any{
				"type":        "string",
				"description": "The driver whose papers are due, from get_worker.",
			},
			"credentialIds": map[string]any{
				"type": "array",
				"description": "Every credential this ask covers, by the id " +
					"list_expiring_credentials or get_worker_credential returns, or the one " +
					"the event that started this run names. Never invent one.",
				"items":    map[string]any{"type": "string"},
				"minItems": 1,
			},
			"note": map[string]any{
				"type": "string",
				"description": "What the driver is being asked for, in their words: " +
					"which papers, by when, and where to send them.",
			},
		},
		"required":             []string{"workerId", "credentialIds", "note"},
		"additionalProperties": false,
	}
}

func (t *requestCredentialRenewalTool) Reversible() bool { return false }

func (t *requestCredentialRenewalTool) PermissionResource() permission.Resource {
	return permission.ResourceWorkerCredential
}

func (t *requestCredentialRenewalTool) PermissionOperation() permission.Operation {
	return permission.OpUpdate
}

// Asking twice for the same renewal is a driver told twice, so the ask is
// keyed and replayed rather than repeated.
func (t *requestCredentialRenewalTool) RequiresIdempotencyKey() bool { return true }

// Asking a driver to renew a paper takes nothing away and is the same thing
// the compliance sweep already does by notification, so it is the one write
// on these desks that can earn its way to running unattended.
func (t *requestCredentialRenewalTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierPropose
}

func (t *requestCredentialRenewalTool) arguments(
	params serviceports.ToolExecuteParams,
) (workercredentialservice.RenewalRequest, error) {
	var out workercredentialservice.RenewalRequest
	if err := guardExecute(t, params); err != nil {
		return out, err
	}

	workerID, err := requirePulid(params.Params, "workerId")
	if err != nil {
		return out, err
	}

	var rawIDs []string
	if err = decodeParam(params.Params, "credentialIds", &rawIDs); err != nil {
		return out, err
	}
	if len(rawIDs) == 0 {
		return out, fmt.Errorf("parameter %q needs at least one credential", "credentialIds")
	}
	credentialIDs := make([]pulid.ID, 0, len(rawIDs))
	for _, raw := range rawIDs {
		id, parseErr := pulid.Parse(raw)
		if parseErr != nil {
			return out, fmt.Errorf("credential id %q is not an id: %w", raw, parseErr)
		}
		credentialIDs = append(credentialIDs, id)
	}

	note, err := requireString(params.Params, "note")
	if err != nil {
		return out, err
	}
	if len(note) > maxRenewalNoteChars {
		return out, fmt.Errorf(
			"parameter %q is %d characters, and the ask is capped at %d",
			"note", len(note), maxRenewalNoteChars,
		)
	}

	return workercredentialservice.RenewalRequest{
		WorkerID:      workerID,
		CredentialIDs: credentialIDs,
		Note:          strings.TrimSpace(note),
		TenantInfo:    tenantFrom(params),
		RequestedByID: params.Actor.UserID,
		CorrelationID: params.IdempotencyKey,
	}, nil
}

func (t *requestCredentialRenewalTool) Validate(
	_ context.Context,
	params serviceports.ToolExecuteParams,
) error {
	_, err := t.arguments(params)

	return err
}

func (t *requestCredentialRenewalTool) Simulate(
	_ context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolSimulation, error) {
	request, err := t.arguments(params)
	if err != nil {
		return nil, err
	}

	return &agent.ToolSimulation{
		Summary: fmt.Sprintf(
			"Would ask driver %s to renew %d credential(s).",
			request.WorkerID, len(request.CredentialIDs),
		),
		Changes: []agent.FieldChange{
			{Field: "renewalRequestedAt", From: "", To: "now"},
		},
		Previewed: true,
	}, nil
}

func (t *requestCredentialRenewalTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	request, err := t.arguments(params)
	if err != nil {
		return err
	}

	return t.credentials.RequestRenewal(ctx, request)
}

func (t *requestCredentialRenewalTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "workerId", permission.ResourceWorker)
}

type workerHolder interface {
	SetDispatchHold(
		ctx context.Context,
		req workerservice.DispatchHoldRequest,
	) (*worker.Worker, error)
	Get(ctx context.Context, req repositories.GetWorkerByIDRequest) (*worker.Worker, error)
}

type placeWorkerDispatchHoldTool struct {
	workers workerHolder
}

func newPlaceWorkerDispatchHoldTool(workers workerHolder) serviceports.AgentTool {
	return &placeWorkerDispatchHoldTool{workers: workers}
}

func (t *placeWorkerDispatchHoldTool) Name() string { return "place_worker_dispatch_hold" }

func (t *placeWorkerDispatchHoldTool) Description() string {
	return "Stop a driver being given new freight, because a credential that governs " +
		"driving has expired or will expire before a renewal could land. Say which " +
		"paper and when it lapsed. This is reversible and a person can lift it; it does " +
		"not touch the driver's employment or their current load."
}

func (t *placeWorkerDispatchHoldTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"workerId": map[string]any{
				"type":        "string",
				"description": "The driver to hold, from get_worker.",
			},
			"reason": map[string]any{
				"type": "string",
				"description": "Which paper lapsed and when, in a sentence a dispatcher " +
					"can read on the board.",
			},
		},
		"required":             []string{"workerId", "reason"},
		"additionalProperties": false,
	}
}

func (t *placeWorkerDispatchHoldTool) Reversible() bool { return true }

func (t *placeWorkerDispatchHoldTool) PermissionResource() permission.Resource {
	return permission.ResourceWorkerDispatchHold
}

func (t *placeWorkerDispatchHoldTool) PermissionOperation() permission.Operation {
	return permission.OpCreate
}

func (t *placeWorkerDispatchHoldTool) RequiresIdempotencyKey() bool { return false }

func (t *placeWorkerDispatchHoldTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierPropose
}

func (t *placeWorkerDispatchHoldTool) arguments(
	params serviceports.ToolExecuteParams,
) (workerservice.DispatchHoldRequest, error) {
	var out workerservice.DispatchHoldRequest
	if err := guardExecute(t, params); err != nil {
		return out, err
	}

	workerID, err := requirePulid(params.Params, "workerId")
	if err != nil {
		return out, err
	}
	reason, err := requireString(params.Params, "reason")
	if err != nil {
		return out, err
	}

	return workerservice.DispatchHoldRequest{
		WorkerID:   workerID,
		Held:       true,
		Reason:     strings.TrimSpace(reason),
		TenantInfo: tenantFrom(params),
		Actor:      params.Actor,
	}, nil
}

func (t *placeWorkerDispatchHoldTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	request, err := t.arguments(params)
	if err != nil {
		return err
	}

	held, err := t.workers.Get(ctx, repositories.GetWorkerByIDRequest{
		ID:         request.WorkerID,
		TenantInfo: request.TenantInfo,
	})
	if err != nil {
		return err
	}
	if !held.CanBeAssigned {
		return fmt.Errorf("driver %s is already off dispatch", request.WorkerID)
	}

	return nil
}

func (t *placeWorkerDispatchHoldTool) Simulate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolSimulation, error) {
	request, err := t.arguments(params)
	if err != nil {
		return nil, err
	}
	if vErr := t.Validate(ctx, params); vErr != nil {
		return nil, vErr
	}

	return &agent.ToolSimulation{
		Summary: fmt.Sprintf(
			"Would stop driver %s being given new freight: %s",
			request.WorkerID, request.Reason,
		),
		Changes: []agent.FieldChange{
			{Field: "canBeAssigned", From: "true", To: "false"},
		},
		Previewed: true,
	}, nil
}

func (t *placeWorkerDispatchHoldTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	request, err := t.arguments(params)
	if err != nil {
		return err
	}

	_, err = t.workers.SetDispatchHold(ctx, request)

	return err
}

func (t *placeWorkerDispatchHoldTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "workerId", permission.ResourceWorker)
}

// ------------------------------------------------------------ carrier risk

type carrierIntelActor interface {
	AcknowledgeEvents(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		eventIDs []pulid.ID,
	) (int, error)
	ResolveEvent(
		ctx context.Context,
		req *carrierintelservice.ResolveEventRequest,
	) (*carrierintel.CarrierIntelEvent, error)
	GetEvent(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		eventID pulid.ID,
	) (*carrierintel.CarrierIntelEvent, error)
}

// The two tools share everything but their name, their outcome and whether
// they ask for one. Each is its own type so its name is a constant a reader
// — and the permission coverage test — can find without running it.
type carrierIntelEventTool struct {
	intel    carrierIntelActor
	resolves bool
}

type acknowledgeCarrierIntelEventTool struct{ carrierIntelEventTool }

type resolveCarrierIntelEventTool struct{ carrierIntelEventTool }

func newAcknowledgeCarrierIntelEventTool(intel carrierIntelActor) serviceports.AgentTool {
	return &acknowledgeCarrierIntelEventTool{
		carrierIntelEventTool{intel: intel},
	}
}

func newResolveCarrierIntelEventTool(intel carrierIntelActor) serviceports.AgentTool {
	return &resolveCarrierIntelEventTool{
		carrierIntelEventTool{intel: intel, resolves: true},
	}
}

func (t *carrierIntelEventTool) Name() string {
	if t.resolves {
		return "resolve_carrier_intel_event"
	}

	return "acknowledge_carrier_intel_event"
}

func (t *acknowledgeCarrierIntelEventTool) Name() string {
	return "acknowledge_carrier_intel_event"
}

func (t *resolveCarrierIntelEventTool) Name() string { return "resolve_carrier_intel_event" }

// Each tool states its own permission rather than inheriting it. A tool whose
// resource is only readable by following an embedded type is one a reader has
// to run to understand, and one the coverage check cannot see at all.
func (t *acknowledgeCarrierIntelEventTool) PermissionResource() permission.Resource {
	return permission.ResourceCarrierIntelligence
}

func (t *acknowledgeCarrierIntelEventTool) PermissionOperation() permission.Operation {
	return permission.OpUpdate
}

func (t *resolveCarrierIntelEventTool) PermissionResource() permission.Resource {
	return permission.ResourceCarrierIntelligence
}

func (t *resolveCarrierIntelEventTool) PermissionOperation() permission.Operation {
	return permission.OpUpdate
}

func (t *carrierIntelEventTool) Description() string {
	if t.resolves {
		return "Close a carrier finding, saying what came of it. The gate blocks a " +
			"disqualified carrier on its own, so this is how the outcome is recorded: " +
			"CarrierUpdated when the record now shows it fixed, CarrierBlocked when the " +
			"carrier is not to be used, NoActionRequired when it never bore on " +
			"eligibility, FalsePositive when the finding itself was wrong. Closing a " +
			"finding that is still true hides it from the people who need it."
	}

	return "Mark a carrier finding as seen when it does not bear on whether the carrier " +
		"can be given freight. Use it for a new address, a changed contact, a score " +
		"that moved within its band. A finding that does bear on eligibility is not " +
		"acknowledged: it is acted on, then closed with resolve_carrier_intel_event."
}

func (t *carrierIntelEventTool) ParamSchema() map[string]any {
	verb := "acknowledging"
	if t.resolves {
		verb = "closing"
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"eventId": map[string]any{
				"type":        "string",
				"description": "The finding id from get_carrier_intel_event.",
			},
			"note": map[string]any{
				"type":        "string",
				"description": "Why you are " + verb + " it, for the carrier's record.",
			},
			"resolution": map[string]any{
				"type": "string",
				"enum": []string{
					"CarrierUpdated", "CarrierBlocked",
					"NoActionRequired", "FalsePositive",
				},
				"description": "What came of the finding. Only used when closing one.",
			},
		},
		"required":             t.required(),
		"additionalProperties": false,
	}
}

// required and resolutionProperty are what separate the two tools: closing a
// finding has to say what came of it, acknowledging one does not.
func (t *carrierIntelEventTool) required() []string {
	if t.resolves {
		return []string{"eventId", "resolution", "note"}
	}

	return []string{"eventId", "note"}
}

func (t *carrierIntelEventTool) Reversible() bool { return false }

func (t *carrierIntelEventTool) PermissionResource() permission.Resource {
	return permission.ResourceCarrierIntelligence
}

func (t *carrierIntelEventTool) PermissionOperation() permission.Operation {
	return permission.OpUpdate
}

func (t *carrierIntelEventTool) RequiresIdempotencyKey() bool { return false }

func (t *carrierIntelEventTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierPropose
}

func (t *carrierIntelEventTool) arguments(
	params serviceports.ToolExecuteParams,
) (*carrierintelservice.ResolveEventRequest, error) {
	if err := guardExecute(t, params); err != nil {
		return nil, err
	}

	eventID, err := requirePulid(params.Params, "eventId")
	if err != nil {
		return nil, err
	}
	note, err := requireString(params.Params, "note")
	if err != nil {
		return nil, err
	}

	request := &carrierintelservice.ResolveEventRequest{
		EventID:    eventID,
		TenantInfo: tenantFrom(params),
		Note:       strings.TrimSpace(note),
	}
	if !t.resolves {
		return request, nil
	}

	rawResolution, err := requireString(params.Params, "resolution")
	if err != nil {
		return nil, err
	}
	resolution := carrierintel.EventResolution(rawResolution)
	if !resolution.IsValid() {
		return nil, fmt.Errorf("%q is not an outcome a finding can be closed with", rawResolution)
	}
	request.Resolution = resolution

	return request, nil
}

func (t *carrierIntelEventTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	request, err := t.arguments(params)
	if err != nil {
		return err
	}

	event, err := t.intel.GetEvent(ctx, request.TenantInfo, request.EventID)
	if err != nil {
		return err
	}
	if event.Status.IsClosed() {
		return fmt.Errorf("carrier finding %s is already %s", request.EventID, event.Status)
	}

	return nil
}

func (t *carrierIntelEventTool) Simulate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolSimulation, error) {
	request, err := t.arguments(params)
	if err != nil {
		return nil, err
	}

	event, err := t.intel.GetEvent(ctx, request.TenantInfo, request.EventID)
	if err != nil {
		return nil, err
	}
	if event.Status.IsClosed() {
		return nil, fmt.Errorf("carrier finding %s is already %s", request.EventID, event.Status)
	}

	to := carrierintel.EventStatusAcknowledged
	outcome := ""
	if t.resolves {
		to = carrierintel.EventStatusResolved
		outcome = " as " + string(request.Resolution)
	}

	return &agent.ToolSimulation{
		Summary: fmt.Sprintf(
			"Would close carrier finding %s (%s, %s) to %s%s: %s",
			request.EventID, event.Category, event.Severity, to, outcome, request.Note,
		),
		Changes: []agent.FieldChange{
			{Field: "status", From: string(event.Status), To: string(to)},
		},
		Previewed: true,
	}, nil
}

func (t *carrierIntelEventTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	request, err := t.arguments(params)
	if err != nil {
		return err
	}

	if t.resolves {
		_, err = t.intel.ResolveEvent(ctx, request)

		return err
	}

	_, err = t.intel.AcknowledgeEvents(ctx, request.TenantInfo, []pulid.ID{request.EventID})

	return err
}

func (t *carrierIntelEventTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "eventId", permission.ResourceCarrierIntelligence)
}

// ------------------------------------------------------- the repeat guard

// recentUpdateWindow is how long an update to a customer about one shipment
// stands for. The customer update desk is woken by every arrival and every
// departure, so a shipment moving through a yard can raise several runs
// within a few minutes; without this the customer hears about each.
const recentUpdateWindow = int64(3600)

// recentAgentCommentPageSize bounds the read. The guard only needs to know
// whether anything was said, and the newest few comments answer that.
const recentAgentCommentPageSize = 25

/*
alreadyToldCustomer reports whether an agent already emailed this customer
about this shipment inside the window.

The instruction to not repeat an action "the shipment's comments show was
taken in the last hour" was prompt text, which means it held exactly as well
as the model's attention. The comment the send writes is the record; reading
it back before sending is what makes the rule a rule.
*/
func alreadyToldCustomer(
	ctx context.Context,
	comments serviceports.ShipmentCommentService,
	tenant pagination.TenantInfo,
	shipmentID pulid.ID,
	now int64,
) (bool, error) {
	if comments == nil {
		return false, nil
	}

	page, err := comments.ListByShipmentID(ctx, &repositories.ListShipmentCommentsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: recentAgentCommentPageSize},
		},
		Cursor:     pagination.CursorInfo{Limit: recentAgentCommentPageSize},
		ShipmentID: shipmentID,
		Filters: repositories.ShipmentCommentListFilters{
			Types: []shipment.CommentType{shipment.CommentTypeCustomerUpdate},
		},
	})
	if err != nil {
		return false, err
	}

	for _, comment := range page.Items {
		if comment == nil || comment.CreatedAt < now-recentUpdateWindow {
			continue
		}
		if wroteByAgentEmail(comment) {
			return true, nil
		}
	}

	return false, nil
}

// wroteByAgentEmail reads the stamp the send leaves, rather than the comment
// body: a dispatcher who typed an update by hand has not sent the customer
// an email, and must not silence one.
func wroteByAgentEmail(comment *shipment.ShipmentComment) bool {
	if comment.Metadata == nil {
		return false
	}
	if source, _ := comment.Metadata["source"].(string); source != "agent" {
		return false
	}
	tool, _ := comment.Metadata["tool"].(string)

	return tool == "email_customer"
}
