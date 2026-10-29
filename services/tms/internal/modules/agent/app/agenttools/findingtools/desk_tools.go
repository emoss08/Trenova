package findingtools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/detentionpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionreview"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelevent"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workercredentialservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	toolDetentionGetOccurrence              = "detention_get_occurrence"
	toolCarrierResolveIntelligenceEvent     = "carrier_resolve_intelligence_event"
	toolCarrierAcknowledgeIntelligenceEvent = "carrier_acknowledge_intelligence_event"
)

// A tender block is an override with a life: carrier records are
// corrected, and a block nobody revisits outlives the problem.
const tenderBlockDays = 30

const secondsPerBlockDay = int64(86400)

// The longest a renewal can be asked for before the ask stops being a
// reminder and becomes a plan.
const maxRenewalNoteChars = 2000

var carrierFindingResolutions = agenttoolschema.Source(
	"carrierIntel.findingResolution",
	[]carrierintelevent.EventResolution{
		carrierintelevent.EventResolutionCarrierUpdated,
		carrierintelevent.EventResolutionCarrierBlocked,
		carrierintelevent.EventResolutionNoActionRequired,
		carrierintelevent.EventResolutionFalsePositive,
	},
)

type detentionEscalator interface {
	Escalate(
		ctx context.Context,
		params detentionreview.EscalateParams,
	) (*detention.DetentionOccurrence, error)
	GetOccurrenceDetail(
		ctx context.Context,
		req *detentionports.GetDetentionOccurrenceByIDRequest,
	) (*detentionreview.OccurrenceDetail, error)
}

type EscalateDetentionTool struct {
	detention detentionEscalator
}

func NewEscalateDetentionTool(escalator detentionEscalator) agenttoolports.AgentTool {
	return &EscalateDetentionTool{detention: escalator}
}

func (t *EscalateDetentionTool) Name() string { return "detention_escalate" }

func (t *EscalateDetentionTool) Recipe() []string {
	return []string{toolDetentionGetOccurrence, "detention_escalate"}
}

func (t *EscalateDetentionTool) Description() string {
	return "Hand a detention clock to a person, with the reason it cannot be worked " +
		"automatically. Use it when the notice window has closed, a gate is holding the " +
		"notice back, or the customer has nobody on file to send it to. This does not change " +
		"the charge — waiving, approving and disputing stay a person's decision — it " +
		"only puts the occurrence in front of one."
}

func (t *EscalateDetentionTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"occurrenceId": agenttoolschema.KindID(
				"The occurrence id, from detention_list_occurrences or "+
					"detention_get_occurrence.",
				permission.KindDetentionOccurrence,
			),
			toolparams.ParamReason: map[string]any{
				"type": "string",
				"description": "Why this needs a person, in a sentence they can act on: " +
					"what is blocking the notice, and what the charge stands at.",
			},
		},
		"required":             []string{"occurrenceId", toolparams.ParamReason},
		"additionalProperties": false,
	}
}

func (t *EscalateDetentionTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceDetention,
		Operation:     permission.OpUpdate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale:     "Hands a detention clock to a person inside the organization.",
	}
}

func (t *EscalateDetentionTool) arguments(
	params agenttoolports.ToolExecuteParams,
) (pulid.ID, string, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return "", "", err
	}

	occurrenceID, err := toolparams.RequirePulid(params.Params, "occurrenceId")
	if err != nil {
		return "", "", err
	}
	reason, err := toolparams.RequireString(params.Params, toolparams.ParamReason)
	if err != nil {
		return "", "", err
	}

	return occurrenceID, strings.TrimSpace(reason), nil
}

func (t *EscalateDetentionTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
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

func (t *EscalateDetentionTool) occurrence(
	ctx context.Context,
	occurrenceID pulid.ID,
	params agenttoolports.ToolExecuteParams,
) (*detention.DetentionOccurrence, error) {
	detail, err := t.detention.GetOccurrenceDetail(
		ctx,
		&detentionports.GetDetentionOccurrenceByIDRequest{
			OccurrenceID: occurrenceID,
			TenantInfo:   toolparams.TenantFrom(params),
		},
	)
	if err != nil {
		return nil, err
	}

	return detail.Occurrence, nil
}

func (t *EscalateDetentionTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	occurrenceID, reason, err := t.arguments(params)
	if err != nil {
		return err
	}

	_, err = t.detention.Escalate(ctx, detentionreview.EscalateParams{
		OccurrenceID: occurrenceID,
		TenantInfo:   toolparams.TenantFrom(params),
		Reason:       reason,
		UserID:       params.Actor.UserID,
	})

	return err
}

func (t *EscalateDetentionTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, "occurrenceId", permission.ResourceDetention)
}

type detentionApprover interface {
	Approve(
		ctx context.Context,
		params detentionreview.ApproveParams,
	) (*detention.DetentionOccurrence, error)
	PreviewApprove(
		ctx context.Context,
		params *detentionreview.ApproveParams,
	) (*detentionreview.OccurrenceChange, error)
	GetOccurrenceDetail(
		ctx context.Context,
		req *detentionports.GetDetentionOccurrenceByIDRequest,
	) (*detentionreview.OccurrenceDetail, error)
}

type ApproveDetentionTool struct {
	detention detentionApprover
}

func NewApproveDetentionTool(approver detentionApprover) agenttoolports.AgentTool {
	return &ApproveDetentionTool{detention: approver}
}

func (t *ApproveDetentionTool) Name() string { return "detention_approve" }

func (t *ApproveDetentionTool) Prerequisites() []string {
	return []string{"detention_list_occurrences", toolDetentionGetOccurrence}
}

func (t *ApproveDetentionTool) Recipe() []string {
	return []string{"detention_list_occurrences", toolDetentionGetOccurrence, "detention_approve"}
}

func (t *ApproveDetentionTool) Description() string {
	return "Propose approving a detention charge that is waiting on approval, so it can " +
		"be invoiced. detention_list_occurrences shows these as AwaitingApproval, and while one " +
		"waits its shipment cannot be billed. The charge is approved exactly as calculated. " +
		"Propose it only when the evidence holds up: arrival and departure on record, the " +
		"notice sent inside the policy's window or none required, and nothing on file that " +
		"contradicts the time. Say what shows that in evidence. Only a pending charge can be " +
		"approved, and a person always decides; when the evidence does not hold up, propose " +
		"detention_waive instead or leave it for a person."
}

func (t *ApproveDetentionTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"occurrenceId": agenttoolschema.KindID(
				"The occurrence id, from detention_list_occurrences or "+
					"detention_get_occurrence.",
				permission.KindDetentionOccurrence,
			),
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

func (t *ApproveDetentionTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceDetention,
		Operation:     permission.OpUpdate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierPropose,
		Egress:        []agent.EgressClass{agent.EgressMoney},
		Effect:        agenttooltrust.ToolEffectChange,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale: "Releases a held detention charge onto the customer's invoice; only a " +
			"person approves it.",
	}
}

func (t *ApproveDetentionTool) arguments(
	params agenttoolports.ToolExecuteParams,
) (pulid.ID, string, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return "", "", err
	}

	occurrenceID, err := toolparams.RequirePulid(params.Params, "occurrenceId")
	if err != nil {
		return "", "", err
	}
	evidence, err := toolparams.RequireString(params.Params, "evidence")
	if err != nil {
		return "", "", err
	}

	return occurrenceID, strings.TrimSpace(evidence), nil
}

func (t *ApproveDetentionTool) pending(
	ctx context.Context,
	occurrenceID pulid.ID,
	params agenttoolports.ToolExecuteParams,
) (*detentionreview.OccurrenceDetail, error) {
	detail, err := t.detention.GetOccurrenceDetail(
		ctx,
		&detentionports.GetDetentionOccurrenceByIDRequest{
			OccurrenceID: occurrenceID,
			TenantInfo:   toolparams.TenantFrom(params),
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

func (t *ApproveDetentionTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
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
func (t *ApproveDetentionTool) request(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) (detentionreview.ApproveParams, *detentionreview.OccurrenceDetail, error) {
	occurrenceID, evidence, err := t.arguments(params)
	if err != nil {
		return detentionreview.ApproveParams{}, nil, err
	}

	detail, err := t.pending(ctx, occurrenceID, params)
	if err != nil {
		return detentionreview.ApproveParams{}, nil, err
	}

	return detentionreview.ApproveParams{
		OccurrenceID: occurrenceID,
		TenantInfo:   toolparams.TenantFrom(params),
		UserID:       params.Actor.UserID,
		Note:         evidence,
	}, detail, nil
}

func (t *ApproveDetentionTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	if _, _, err := t.arguments(params); err != nil {
		return err
	}
	if !params.ApprovedFromProposal() {
		return detentionpreview.ErrApprovalNeedsAPerson
	}

	request, _, err := t.request(ctx, params)
	if err != nil {
		return err
	}

	_, err = t.detention.Approve(ctx, request)

	return err
}

func (t *ApproveDetentionTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, "occurrenceId", permission.ResourceDetention)
}

type credentialActor interface {
	RequestRenewal(ctx context.Context, req workercredentialservice.RenewalRequest) error
	PreviewRenewal(
		ctx context.Context,
		req *workercredentialservice.RenewalRequest,
	) (*workercredentialservice.RenewalPreview, error)
}

type RequestCredentialRenewalTool struct {
	credentials credentialActor
}

func NewRequestCredentialRenewalTool(credentials credentialActor) agenttoolports.AgentTool {
	return &RequestCredentialRenewalTool{credentials: credentials}
}

func (t *RequestCredentialRenewalTool) Name() string { return "worker_request_credential_renewal" }

func (t *RequestCredentialRenewalTool) Recipe() []string {
	return []string{
		"worker_list_credentials",
		"worker_get",
		"worker_request_credential_renewal",
	}
}

func (t *RequestCredentialRenewalTool) Description() string {
	return "Ask a driver to renew the papers that are coming due, in one message covering " +
		"every credential rather than one per certificate. Name each credential and its " +
		"expiry date. This only asks; it does not record a renewal or change what is on file."
}

func (t *RequestCredentialRenewalTool) SearchTerms() []string {
	return []string{"ask to renew", "renewal reminder", "medical card renewal", "license renewal"}
}

func (t *RequestCredentialRenewalTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			toolparams.ParamWorkerID: agenttoolschema.RecordIDText(
				permission.ResourceWorker,
				"The driver whose papers are due, from worker_get.",
			),
			"credentialIds": agenttoolschema.OfResource(map[string]any{
				"type": "array",
				"description": "Every credential this ask covers, by the id " +
					"worker_list_credentials or worker_get_credential returns, or the one " +
					"the event that started this run names. Never invent one.",
				toolparams.SchemaItems: map[string]any{"type": "string"},
				"minItems":             1,
			}, permission.ResourceWorkerCredential),
			toolparams.ParamNote: map[string]any{
				"type": "string",
				"description": "What the driver is being asked for, in their words: " +
					"which papers, by when, and where to send them.",
			},
		},
		"required": []string{
			toolparams.ParamWorkerID,
			"credentialIds",
			toolparams.ParamNote,
		},
		"additionalProperties": false,
	}
}

func (t *RequestCredentialRenewalTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceWorkerCredential,
		Operation:     permission.OpUpdate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressDriverVisible},
		Effect:        agenttooltrust.ToolEffectChange,
		Idempotent:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale:     "Sends a driver a renewal request the model wrote.",
	}
}

func (t *RequestCredentialRenewalTool) arguments(
	params agenttoolports.ToolExecuteParams,
) (workercredentialservice.RenewalRequest, error) {
	var out workercredentialservice.RenewalRequest
	if err := toolguard.GuardExecute(t, params); err != nil {
		return out, err
	}

	workerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamWorkerID)
	if err != nil {
		return out, err
	}

	var rawIDs []string
	if err = toolparams.DecodeParam(params.Params, "credentialIds", &rawIDs); err != nil {
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

	note, err := toolparams.RequireString(params.Params, toolparams.ParamNote)
	if err != nil {
		return out, err
	}
	if len(note) > maxRenewalNoteChars {
		return out, fmt.Errorf(
			"parameter %q is %d characters, and the ask is capped at %d",
			toolparams.ParamNote, len(note), maxRenewalNoteChars,
		)
	}

	return workercredentialservice.RenewalRequest{
		WorkerID:      workerID,
		CredentialIDs: credentialIDs,
		Note:          strings.TrimSpace(note),
		TenantInfo:    toolparams.TenantFrom(params),
		RequestedByID: params.Actor.UserID,
		CorrelationID: params.IdempotencyKey,
	}, nil
}

func (t *RequestCredentialRenewalTool) Validate(
	_ context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	_, err := t.arguments(params)

	return err
}

func (t *RequestCredentialRenewalTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	request, err := t.arguments(params)
	if err != nil {
		return err
	}

	return t.credentials.RequestRenewal(ctx, request)
}

func (t *RequestCredentialRenewalTool) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, toolparams.ParamWorkerID, permission.ResourceWorker)
}

type workerHolder interface {
	SetDispatchHold(
		ctx context.Context,
		req workerservice.DispatchHoldRequest,
	) (*worker.Worker, error)
	Get(ctx context.Context, req workerports.GetWorkerByIDRequest) (*worker.Worker, error)
}

type PlaceWorkerDispatchHoldTool struct {
	workers workerHolder
}

func NewPlaceWorkerDispatchHoldTool(workers workerHolder) agenttoolports.AgentTool {
	return &PlaceWorkerDispatchHoldTool{workers: workers}
}

func (t *PlaceWorkerDispatchHoldTool) Name() string { return "worker_place_dispatch_hold" }

func (t *PlaceWorkerDispatchHoldTool) Recipe() []string {
	return []string{"worker_list_credentials", "worker_get", "worker_place_dispatch_hold"}
}

func (t *PlaceWorkerDispatchHoldTool) SearchTerms() []string {
	return []string{"stop dispatching", "dispatch block", "no loads", "hold until renewed"}
}

func (t *PlaceWorkerDispatchHoldTool) Description() string {
	return "Stop a driver being given new freight, because a credential that governs " +
		"driving has expired or will expire before a renewal could land. Say which " +
		"paper and when it lapsed. This is reversible and a person can lift it; it does " +
		"not touch the driver's employment or their current load."
}

func (t *PlaceWorkerDispatchHoldTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			toolparams.ParamWorkerID: agenttoolschema.RecordIDText(
				permission.ResourceWorker,
				"The driver to hold, from worker_get.",
			),
			toolparams.ParamReason: map[string]any{
				"type": "string",
				"description": "Which paper lapsed and when, in a sentence a dispatcher " +
					"can read on the board.",
			},
		},
		"required":             []string{toolparams.ParamWorkerID, toolparams.ParamReason},
		"additionalProperties": false,
	}
}

func (t *PlaceWorkerDispatchHoldTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceWorkerDispatchHold,
		Operation:     permission.OpCreate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale:     "Keeps a driver off new freight inside Trenova; the driver is not messaged.",
	}
}

func (t *PlaceWorkerDispatchHoldTool) arguments(
	params agenttoolports.ToolExecuteParams,
) (workerservice.DispatchHoldRequest, error) {
	var out workerservice.DispatchHoldRequest
	if err := toolguard.GuardExecute(t, params); err != nil {
		return out, err
	}

	workerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamWorkerID)
	if err != nil {
		return out, err
	}
	reason, err := toolparams.RequireString(params.Params, toolparams.ParamReason)
	if err != nil {
		return out, err
	}

	return workerservice.DispatchHoldRequest{
		WorkerID:   workerID,
		Held:       true,
		Reason:     strings.TrimSpace(reason),
		TenantInfo: toolparams.TenantFrom(params),
		Actor:      params.Actor,
	}, nil
}

func (t *PlaceWorkerDispatchHoldTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	request, err := t.arguments(params)
	if err != nil {
		return err
	}

	held, err := t.workers.Get(ctx, workerports.GetWorkerByIDRequest{
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

func (t *PlaceWorkerDispatchHoldTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	request, err := t.arguments(params)
	if err != nil {
		return err
	}

	_, err = t.workers.SetDispatchHold(ctx, request)

	return err
}

func (t *PlaceWorkerDispatchHoldTool) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, toolparams.ParamWorkerID, permission.ResourceWorker)
}

type carrierIntelActor interface {
	AcknowledgeEvents(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		eventIDs []pulid.ID,
	) (int, error)
	ResolveEvent(
		ctx context.Context,
		req *carrierintelservice.ResolveEventRequest,
	) (*carrierintelevent.CarrierIntelEvent, error)
	GetEvent(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		eventID pulid.ID,
	) (*carrierintelevent.CarrierIntelEvent, error)
}

// The two tools share everything but their name, their outcome and whether
// they ask for one. Each is its own type so its name is a constant a reader
// — and the permission coverage test — can find without running it.
type carrierIntelEventTool struct {
	intel    carrierIntelActor
	resolves bool
}

type AcknowledgeCarrierIntelEventTool struct{ carrierIntelEventTool }

type ResolveCarrierIntelEventTool struct{ carrierIntelEventTool }

func NewAcknowledgeCarrierIntelEventTool(intel carrierIntelActor) agenttoolports.AgentTool {
	return &AcknowledgeCarrierIntelEventTool{
		carrierIntelEventTool{intel: intel},
	}
}

func NewResolveCarrierIntelEventTool(intel carrierIntelActor) agenttoolports.AgentTool {
	return &ResolveCarrierIntelEventTool{
		carrierIntelEventTool{intel: intel, resolves: true},
	}
}

func (t *carrierIntelEventTool) Name() string {
	if t.resolves {
		return toolCarrierResolveIntelligenceEvent
	}

	return toolCarrierAcknowledgeIntelligenceEvent
}

func (t *AcknowledgeCarrierIntelEventTool) Name() string {
	return toolCarrierAcknowledgeIntelligenceEvent
}

func (t *AcknowledgeCarrierIntelEventTool) Recipe() []string {
	return []string{"carrier_get_intelligence_event", toolCarrierAcknowledgeIntelligenceEvent}
}

func (t *ResolveCarrierIntelEventTool) Name() string { return toolCarrierResolveIntelligenceEvent }

func (t *ResolveCarrierIntelEventTool) Recipe() []string {
	return []string{
		"carrier_get_intelligence_event",
		"carrier_get",
		toolCarrierResolveIntelligenceEvent,
	}
}

func (t *ResolveCarrierIntelEventTool) SearchTerms() []string {
	return []string{"carrier finding", "insurance lapse", "finding resolved"}
}

func (t *carrierIntelEventTool) Policy() agenttoolports.ToolPolicy {
	rationale := "Acknowledges a carrier finding inside Trenova."
	if t.resolves {
		rationale = "Closes a carrier finding inside Trenova."
	}

	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceCarrierIntelligence,
		Operation:     permission.OpUpdate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		ReadsExternal: agenttooltrust.ExternalReadNever,
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
		"acknowledged: it is acted on, then closed with carrier_resolve_intelligence_event."
}

func (t *carrierIntelEventTool) ParamSchema() map[string]any {
	verb := "acknowledging"
	if t.resolves {
		verb = "closing"
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"eventId": agenttoolschema.KindID(
				"The finding id, from carrier_list_intelligence_events, carrier_get_intelligence_event "+
					"or this run's subject. Never guess one.",
				permission.KindCarrierIntelligenceEvent,
			),
			toolparams.ParamNote: map[string]any{
				"type":        "string",
				"description": "Why you are " + verb + " it, for the carrier's record.",
			},
			changepreview.FieldResolution: agenttoolschema.Enum(
				"What came of the finding. Only used when closing one.",
				carrierFindingResolutions,
			),
		},
		"required":             t.required(),
		"additionalProperties": false,
	}
}

// required and resolutionProperty are what separate the two tools: closing a
// finding has to say what came of it, acknowledging one does not.
func (t *carrierIntelEventTool) required() []string {
	if t.resolves {
		return []string{"eventId", "resolution", toolparams.ParamNote}
	}

	return []string{"eventId", toolparams.ParamNote}
}

func (t *carrierIntelEventTool) arguments(
	params agenttoolports.ToolExecuteParams,
) (*carrierintelservice.ResolveEventRequest, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return nil, err
	}

	eventID, err := toolparams.RequirePulid(params.Params, "eventId")
	if err != nil {
		return nil, err
	}
	note, err := toolparams.RequireString(params.Params, toolparams.ParamNote)
	if err != nil {
		return nil, err
	}

	request := &carrierintelservice.ResolveEventRequest{
		EventID:    eventID,
		TenantInfo: toolparams.TenantFrom(params),
		Note:       strings.TrimSpace(note),
	}
	if !t.resolves {
		return request, nil
	}

	rawResolution, err := toolparams.RequireString(params.Params, "resolution")
	if err != nil {
		return nil, err
	}
	resolution := carrierintelevent.EventResolution(rawResolution)
	if !resolution.IsValid() {
		return nil, fmt.Errorf("%q is not an outcome a finding can be closed with", rawResolution)
	}
	request.Resolution = resolution

	return request, nil
}

func (t *carrierIntelEventTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
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
	params agenttoolports.ToolExecuteParams,
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

func (t *carrierIntelEventTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, "eventId", permission.ResourceCarrierIntelligence)
}
