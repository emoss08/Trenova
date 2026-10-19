package masterdatatools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workercredentialservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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
	[]carrierintel.EventResolution{
		carrierintel.EventResolutionCarrierUpdated,
		carrierintel.EventResolutionCarrierBlocked,
		carrierintel.EventResolutionNoActionRequired,
		carrierintel.EventResolutionFalsePositive,
	},
)

type detentionEscalator interface {
	Escalate(
		ctx context.Context,
		params detentionservice.EscalateParams,
	) (*detention.DetentionOccurrence, error)
	GetOccurrenceDetail(
		ctx context.Context,
		req *detentionports.GetDetentionOccurrenceByIDRequest,
	) (*detentionservice.OccurrenceDetail, error)
}

type EscalateDetentionTool struct {
	detention detentionEscalator
}

func NewEscalateDetentionTool(escalator detentionEscalator) agentports.AgentTool {
	return &EscalateDetentionTool{detention: escalator}
}

func (t *EscalateDetentionTool) Name() string { return "detention_escalate" }

func (t *EscalateDetentionTool) Recipe() []string {
	return []string{"detention_get_occurrence", "detention_escalate"}
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
			common.ParamReason: map[string]any{
				"type": "string",
				"description": "Why this needs a person, in a sentence they can act on: " +
					"what is blocking the notice, and what the charge stands at.",
			},
		},
		"required":             []string{"occurrenceId", common.ParamReason},
		"additionalProperties": false,
	}
}

func (t *EscalateDetentionTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceDetention,
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

func (t *EscalateDetentionTool) arguments(
	params agentports.ToolExecuteParams,
) (pulid.ID, string, error) {
	if err := common.GuardExecute(t, params); err != nil {
		return "", "", err
	}

	occurrenceID, err := common.RequirePulid(params.Params, "occurrenceId")
	if err != nil {
		return "", "", err
	}
	reason, err := common.RequireString(params.Params, common.ParamReason)
	if err != nil {
		return "", "", err
	}

	return occurrenceID, strings.TrimSpace(reason), nil
}

func (t *EscalateDetentionTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams,
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
	params agentports.ToolExecuteParams,
) (*detention.DetentionOccurrence, error) {
	detail, err := t.detention.GetOccurrenceDetail(
		ctx,
		&detentionports.GetDetentionOccurrenceByIDRequest{
			OccurrenceID: occurrenceID,
			TenantInfo:   common.TenantFrom(params),
		},
	)
	if err != nil {
		return nil, err
	}

	return detail.Occurrence, nil
}

func (t *EscalateDetentionTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams,
) error {
	occurrenceID, reason, err := t.arguments(params)
	if err != nil {
		return err
	}

	_, err = t.detention.Escalate(ctx, detentionservice.EscalateParams{
		OccurrenceID: occurrenceID,
		TenantInfo:   common.TenantFrom(params),
		Reason:       reason,
		UserID:       params.Actor.UserID,
	})

	return err
}

func (t *EscalateDetentionTool) Target(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, "occurrenceId", permission.ResourceDetention)
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
		req *detentionports.GetDetentionOccurrenceByIDRequest,
	) (*detentionservice.OccurrenceDetail, error)
}

type ApproveDetentionTool struct {
	detention detentionApprover
}

func NewApproveDetentionTool(approver detentionApprover) agentports.AgentTool {
	return &ApproveDetentionTool{detention: approver}
}

func (t *ApproveDetentionTool) Name() string { return "detention_approve" }

func (t *ApproveDetentionTool) Prerequisites() []string {
	return []string{"detention_list_occurrences", "detention_get_occurrence"}
}

func (t *ApproveDetentionTool) Recipe() []string {
	return []string{"detention_list_occurrences", "detention_get_occurrence", "detention_approve"}
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

func (t *ApproveDetentionTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceDetention,
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

func (t *ApproveDetentionTool) arguments(
	params agentports.ToolExecuteParams,
) (pulid.ID, string, error) {
	if err := common.GuardExecute(t, params); err != nil {
		return "", "", err
	}

	occurrenceID, err := common.RequirePulid(params.Params, "occurrenceId")
	if err != nil {
		return "", "", err
	}
	evidence, err := common.RequireString(params.Params, "evidence")
	if err != nil {
		return "", "", err
	}

	return occurrenceID, strings.TrimSpace(evidence), nil
}

func (t *ApproveDetentionTool) pending(
	ctx context.Context,
	occurrenceID pulid.ID,
	params agentports.ToolExecuteParams,
) (*detentionservice.OccurrenceDetail, error) {
	detail, err := t.detention.GetOccurrenceDetail(
		ctx,
		&detentionports.GetDetentionOccurrenceByIDRequest{
			OccurrenceID: occurrenceID,
			TenantInfo:   common.TenantFrom(params),
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
	params agentports.ToolExecuteParams,
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
	params agentports.ToolExecuteParams,
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
		TenantInfo:   common.TenantFrom(params),
		UserID:       params.Actor.UserID,
		Note:         evidence,
	}, detail, nil
}

func (t *ApproveDetentionTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams,
) error {
	if _, _, err := t.arguments(params); err != nil {
		return err
	}
	if !params.ApprovedFromProposal() {
		return common.ErrApprovalNeedsAPerson
	}

	request, _, err := t.request(ctx, params)
	if err != nil {
		return err
	}

	_, err = t.detention.Approve(ctx, request)

	return err
}

func (t *ApproveDetentionTool) Target(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, "occurrenceId", permission.ResourceDetention)
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

func NewRequestCredentialRenewalTool(credentials credentialActor) agentports.AgentTool {
	return &RequestCredentialRenewalTool{credentials: credentials}
}

func (t *RequestCredentialRenewalTool) Name() string { return "worker_request_credential_renewal" }

func (t *RequestCredentialRenewalTool) Recipe() []string {
	return []string{"worker_list_expiring_credentials", "worker_get", "worker_request_credential_renewal"}
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
			common.ParamWorkerID: agenttoolschema.RecordIDText(
				permission.ResourceWorker,
				"The driver whose papers are due, from worker_get.",
			),
			"credentialIds": agenttoolschema.OfResource(map[string]any{
				"type": "array",
				"description": "Every credential this ask covers, by the id " +
					"worker_list_expiring_credentials or worker_get_credential returns, or the one " +
					"the event that started this run names. Never invent one.",
				schemaItems: map[string]any{"type": "string"},
				"minItems":  1,
			}, permission.ResourceWorkerCredential),
			common.ParamNote: map[string]any{
				"type": "string",
				"description": "What the driver is being asked for, in their words: " +
					"which papers, by when, and where to send them.",
			},
		},
		"required":             []string{common.ParamWorkerID, "credentialIds", common.ParamNote},
		"additionalProperties": false,
	}
}

func (t *RequestCredentialRenewalTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
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

func (t *RequestCredentialRenewalTool) arguments(
	params agentports.ToolExecuteParams,
) (workercredentialservice.RenewalRequest, error) {
	var out workercredentialservice.RenewalRequest
	if err := common.GuardExecute(t, params); err != nil {
		return out, err
	}

	workerID, err := common.RequirePulid(params.Params, common.ParamWorkerID)
	if err != nil {
		return out, err
	}

	var rawIDs []string
	if err = common.DecodeParam(params.Params, "credentialIds", &rawIDs); err != nil {
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

	note, err := common.RequireString(params.Params, common.ParamNote)
	if err != nil {
		return out, err
	}
	if len(note) > maxRenewalNoteChars {
		return out, fmt.Errorf(
			"parameter %q is %d characters, and the ask is capped at %d",
			common.ParamNote, len(note), maxRenewalNoteChars,
		)
	}

	return workercredentialservice.RenewalRequest{
		WorkerID:      workerID,
		CredentialIDs: credentialIDs,
		Note:          strings.TrimSpace(note),
		TenantInfo:    common.TenantFrom(params),
		RequestedByID: params.Actor.UserID,
		CorrelationID: params.IdempotencyKey,
	}, nil
}

func (t *RequestCredentialRenewalTool) Validate(
	_ context.Context,
	params agentports.ToolExecuteParams,
) error {
	_, err := t.arguments(params)

	return err
}

func (t *RequestCredentialRenewalTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams,
) error {
	request, err := t.arguments(params)
	if err != nil {
		return err
	}

	return t.credentials.RequestRenewal(ctx, request)
}

func (t *RequestCredentialRenewalTool) Target(
	params map[string]any,
) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, common.ParamWorkerID, permission.ResourceWorker)
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

func NewPlaceWorkerDispatchHoldTool(workers workerHolder) agentports.AgentTool {
	return &PlaceWorkerDispatchHoldTool{workers: workers}
}

func (t *PlaceWorkerDispatchHoldTool) Name() string { return "worker_place_dispatch_hold" }

func (t *PlaceWorkerDispatchHoldTool) Recipe() []string {
	return []string{"worker_list_expiring_credentials", "worker_get", "worker_place_dispatch_hold"}
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
			common.ParamWorkerID: agenttoolschema.RecordIDText(
				permission.ResourceWorker,
				"The driver to hold, from worker_get.",
			),
			common.ParamReason: map[string]any{
				"type": "string",
				"description": "Which paper lapsed and when, in a sentence a dispatcher " +
					"can read on the board.",
			},
		},
		"required":             []string{common.ParamWorkerID, common.ParamReason},
		"additionalProperties": false,
	}
}

func (t *PlaceWorkerDispatchHoldTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
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

func (t *PlaceWorkerDispatchHoldTool) arguments(
	params agentports.ToolExecuteParams,
) (workerservice.DispatchHoldRequest, error) {
	var out workerservice.DispatchHoldRequest
	if err := common.GuardExecute(t, params); err != nil {
		return out, err
	}

	workerID, err := common.RequirePulid(params.Params, common.ParamWorkerID)
	if err != nil {
		return out, err
	}
	reason, err := common.RequireString(params.Params, common.ParamReason)
	if err != nil {
		return out, err
	}

	return workerservice.DispatchHoldRequest{
		WorkerID:   workerID,
		Held:       true,
		Reason:     strings.TrimSpace(reason),
		TenantInfo: common.TenantFrom(params),
		Actor:      params.Actor,
	}, nil
}

func (t *PlaceWorkerDispatchHoldTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams,
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
	params agentports.ToolExecuteParams,
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
) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, common.ParamWorkerID, permission.ResourceWorker)
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

type AcknowledgeCarrierIntelEventTool struct{ carrierIntelEventTool }

type ResolveCarrierIntelEventTool struct{ carrierIntelEventTool }

func NewAcknowledgeCarrierIntelEventTool(intel carrierIntelActor) agentports.AgentTool {
	return &AcknowledgeCarrierIntelEventTool{
		carrierIntelEventTool{intel: intel},
	}
}

func NewResolveCarrierIntelEventTool(intel carrierIntelActor) agentports.AgentTool {
	return &ResolveCarrierIntelEventTool{
		carrierIntelEventTool{intel: intel, resolves: true},
	}
}

func (t *carrierIntelEventTool) Name() string {
	if t.resolves {
		return "carrier_resolve_intelligence_event"
	}

	return "carrier_acknowledge_intelligence_event"
}

func (t *AcknowledgeCarrierIntelEventTool) Name() string {
	return "carrier_acknowledge_intelligence_event"
}

func (t *AcknowledgeCarrierIntelEventTool) Recipe() []string {
	return []string{"carrier_get_intelligence_event", "carrier_acknowledge_intelligence_event"}
}

func (t *ResolveCarrierIntelEventTool) Name() string { return "carrier_resolve_intelligence_event" }

func (t *ResolveCarrierIntelEventTool) Recipe() []string {
	return []string{"carrier_get_intelligence_event", "carrier_get", "carrier_resolve_intelligence_event"}
}

func (t *ResolveCarrierIntelEventTool) SearchTerms() []string {
	return []string{"carrier finding", "insurance lapse", "finding resolved"}
}

func (t *carrierIntelEventTool) Policy() agentports.ToolPolicy {
	rationale := "Acknowledges a carrier finding inside Trenova."
	if t.resolves {
		rationale = "Closes a carrier finding inside Trenova."
	}

	return agentports.ToolPolicy{
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
			common.ParamNote: map[string]any{
				"type":        "string",
				"description": "Why you are " + verb + " it, for the carrier's record.",
			},
			common.FieldResolution: agenttoolschema.Enum(
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
		return []string{"eventId", "resolution", common.ParamNote}
	}

	return []string{"eventId", common.ParamNote}
}

func (t *carrierIntelEventTool) arguments(
	params agentports.ToolExecuteParams,
) (*carrierintelservice.ResolveEventRequest, error) {
	if err := common.GuardExecute(t, params); err != nil {
		return nil, err
	}

	eventID, err := common.RequirePulid(params.Params, "eventId")
	if err != nil {
		return nil, err
	}
	note, err := common.RequireString(params.Params, common.ParamNote)
	if err != nil {
		return nil, err
	}

	request := &carrierintelservice.ResolveEventRequest{
		EventID:    eventID,
		TenantInfo: common.TenantFrom(params),
		Note:       strings.TrimSpace(note),
	}
	if !t.resolves {
		return request, nil
	}

	rawResolution, err := common.RequireString(params.Params, "resolution")
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
	params agentports.ToolExecuteParams,
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
	params agentports.ToolExecuteParams,
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

func (t *carrierIntelEventTool) Target(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, "eventId", permission.ResourceCarrierIntelligence)
}
