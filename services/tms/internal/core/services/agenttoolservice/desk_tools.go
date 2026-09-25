package agenttoolservice

import (
	"context"
	"errors"
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

func (t *escalateDetentionTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceDetentionPolicy,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "Hands a detention clock to a person inside the organization.",
	}
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

type detentionApprover interface {
	Approve(
		ctx context.Context,
		params detentionservice.ApproveParams,
	) (*detention.DetentionOccurrence, error)
	PreviewApprove(
		ctx context.Context,
		params *detentionservice.ApproveParams,
	) (*detentionservice.OccurrenceChange, error)
	GetOccurrenceDetail(
		ctx context.Context,
		req *repositories.GetDetentionOccurrenceByIDRequest,
	) (*detentionservice.OccurrenceDetail, error)
}

// ErrApprovalNeedsAPerson is an approve_detention or waive_detention call that
// did not come from a proposal a person approved. The tool's tier ceiling already keeps the
// runtime from running it on its own; this is the same rule where the money
// moves.
var ErrApprovalNeedsAPerson = errors.New(
	"a detention charge is approved or waived only once a person approves the proposal",
)

type approveDetentionTool struct {
	detention detentionApprover
}

func newApproveDetentionTool(detention detentionApprover) serviceports.AgentTool {
	return &approveDetentionTool{detention: detention}
}

func (t *approveDetentionTool) Name() string { return "approve_detention" }

func (t *approveDetentionTool) Description() string {
	return "Propose approving a detention charge that is waiting on approval, so it can " +
		"be invoiced. list_detention_desk shows these as AwaitingApproval, and while one " +
		"waits its shipment cannot be billed. The charge is approved exactly as calculated. " +
		"Propose it only when the evidence holds up: arrival and departure on record, the " +
		"notice sent inside the policy's window or none required, and nothing on file that " +
		"contradicts the time. Say what shows that in evidence. Only a pending charge can be " +
		"approved, and a person always decides; when the evidence does not hold up, propose " +
		"waive_detention instead or leave it for a person."
}

func (t *approveDetentionTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"occurrenceId": map[string]any{
				"type": "string",
				"description": "The occurrence id, from list_detention_desk or " +
					"get_detention_occurrence.",
			},
			"evidence": map[string]any{
				"type": "string",
				"description": "What on the occurrence justifies billing it, in a sentence " +
					"the approver can check: the times on record, the notice, and anything " +
					"else on file that supports the charge.",
			},
		},
		"required":             []string{"occurrenceId", "evidence"},
		"additionalProperties": false,
	}
}

func (t *approveDetentionTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceDetentionPolicy,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierPropose,
		Egress:        []agent.EgressClass{agent.EgressMoney},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Releases a held detention charge onto the customer's invoice; only a " +
			"person approves it.",
	}
}

func (t *approveDetentionTool) arguments(
	params serviceports.ToolExecuteParams,
) (pulid.ID, string, error) {
	if err := guardExecute(t, params); err != nil {
		return "", "", err
	}

	occurrenceID, err := requirePulid(params.Params, "occurrenceId")
	if err != nil {
		return "", "", err
	}
	evidence, err := requireString(params.Params, "evidence")
	if err != nil {
		return "", "", err
	}

	return occurrenceID, strings.TrimSpace(evidence), nil
}

func (t *approveDetentionTool) pending(
	ctx context.Context,
	occurrenceID pulid.ID,
	params serviceports.ToolExecuteParams,
) (*detentionservice.OccurrenceDetail, error) {
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

	if status := detail.Occurrence.Status; status != detention.OccurrenceStatusPending {
		return nil, fmt.Errorf(
			"detention occurrence %s is %s; only a pending charge can be approved",
			occurrenceID, status,
		)
	}

	return detail, nil
}

func (t *approveDetentionTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	occurrenceID, _, err := t.arguments(params)
	if err != nil {
		return err
	}

	_, err = t.pending(ctx, occurrenceID, params)

	return err
}

// request is the approval both the preview and the write make: a pending
// charge, approved as the person who approved the proposal, with the evidence
// as its note.
func (t *approveDetentionTool) request(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (detentionservice.ApproveParams, *detentionservice.OccurrenceDetail, error) {
	occurrenceID, evidence, err := t.arguments(params)
	if err != nil {
		return detentionservice.ApproveParams{}, nil, err
	}

	detail, err := t.pending(ctx, occurrenceID, params)
	if err != nil {
		return detentionservice.ApproveParams{}, nil, err
	}

	return detentionservice.ApproveParams{
		OccurrenceID: occurrenceID,
		TenantInfo:   tenantFrom(params),
		UserID:       params.Actor.UserID,
		Note:         evidence,
	}, detail, nil
}

func (t *approveDetentionTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if _, _, err := t.arguments(params); err != nil {
		return err
	}
	if !params.ApprovedFromProposal() {
		return ErrApprovalNeedsAPerson
	}

	request, _, err := t.request(ctx, params)
	if err != nil {
		return err
	}

	_, err = t.detention.Approve(ctx, request)

	return err
}

func (t *approveDetentionTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "occurrenceId", permission.ResourceDetentionPolicy)
}

// --------------------------------------------------------------- credential

type credentialActor interface {
	RequestRenewal(ctx context.Context, req workercredentialservice.RenewalRequest) error
	PreviewRenewal(
		ctx context.Context,
		req *workercredentialservice.RenewalRequest,
	) (*workercredentialservice.RenewalPreview, error)
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

func (t *requestCredentialRenewalTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceWorkerCredential,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressDriverVisible},
		Effect:        agent.ToolEffectChange,
		Idempotent:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "Sends a driver a renewal request the model wrote.",
	}
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

func (t *requestCredentialRenewalTool) Target(
	params map[string]any,
) (serviceports.ToolTarget, bool) {
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

func (t *placeWorkerDispatchHoldTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceWorkerDispatchHold,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "Keeps a driver off new freight inside Trenova; the driver is not messaged.",
	}
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

func (t *placeWorkerDispatchHoldTool) Target(
	params map[string]any,
) (serviceports.ToolTarget, bool) {
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

func (t *carrierIntelEventTool) Policy() serviceports.ToolPolicy {
	rationale := "Acknowledges a carrier finding inside Trenova."
	if t.resolves {
		rationale = "Closes a carrier finding inside Trenova."
	}

	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceCarrierIntelligence,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     rationale,
	}
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
	if comment.Origin() != shipment.CommentOriginAgent {
		return false
	}
	tool, _ := comment.Metadata["tool"].(string)

	return tool == "email_customer"
}
