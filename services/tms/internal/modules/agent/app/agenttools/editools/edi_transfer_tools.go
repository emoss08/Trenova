package editools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const paramEDITransferID = "transferId"

const maxTenderDeclineChars = 500

const ediTransferIDGuidance = "The load tender transfer, from list_edi_transfers or get_edi_transfer. Never guess one."

const ediDecisionRationaleEnd = " A person always decides."

var ErrEDIDecisionNeedsAPerson = errors.New(
	"an EDI write that reaches a trading partner runs only once a person approves the proposal",
)

type ediTransferDecider interface {
	GetTransfer(
		ctx context.Context,
		req ediports.GetEDITransferByIDRequest,
	) (*edi.EDITransfer, error)
	PlanApproveTransfer(
		ctx context.Context,
		req *ediservice.ApproveTransferRequest,
		actor *permissionports.RequestActor,
	) (*ediservice.TransferApprovalPlan, error)
	ApproveTransfer(
		ctx context.Context,
		req *ediservice.ApproveTransferRequest,
		actor *permissionports.RequestActor,
	) (*edi.EDITransfer, error)
	RejectTransfer(
		ctx context.Context,
		req *ediservice.RejectTransferRequest,
		actor *permissionports.RequestActor,
	) (*edi.EDITransfer, error)
	CancelTransfer(
		ctx context.Context,
		req *ediservice.CancelTransferRequest,
		actor *permissionports.RequestActor,
	) (*edi.EDITransfer, error)
	ExpireTransfer(
		ctx context.Context,
		req *ediservice.ExpireTransferRequest,
		actor *permissionports.RequestActor,
	) (*edi.EDITransfer, error)
}

type tenderCall struct {
	transferID pulid.ID
	reason     string
	tenant     pagination.TenantInfo
	actor      *permissionports.RequestActor
}

type tenderOutcome struct {
	before   *edi.EDITransfer
	after    *edi.EDITransfer
	approval *ediservice.TransferApprovalPlan
	refusal  error
}

type transferDecision struct {
	name        string
	description string
	rationale   string
	needsReason bool
	summary     func(*tenderOutcome, *tenderCall) string
	plan        func(context.Context, ediTransferDecider, *tenderCall) (*tenderOutcome, error)
	run         func(context.Context, ediTransferDecider, *tenderCall) error
	searchTerms []string
	recipe      []string
}

type EdiTransferDecisionTool struct {
	decision transferDecision
	edi      ediTransferDecider
}

func (t *EdiTransferDecisionTool) Name() string { return t.decision.name }

func (t *EdiTransferDecisionTool) Recipe() []string { return t.decision.recipe }

func (t *EdiTransferDecisionTool) Description() string { return t.decision.description }

func (t *EdiTransferDecisionTool) SearchTerms() []string { return t.decision.searchTerms }

func (t *EdiTransferDecisionTool) ParamSchema() map[string]any {
	properties := map[string]any{
		paramEDITransferID: agenttoolschema.KindID(ediTransferIDGuidance,
			permission.KindEDITransfer),
	}
	required := []string{paramEDITransferID}
	if t.decision.needsReason {
		properties[common.FieldReason] = map[string]any{
			toolschema.KeyType:      toolschema.TypeString,
			toolschema.KeyMaxLength: maxTenderDeclineChars,
			toolschema.KeyDescription: "Why the tender is declined, in a sentence the " +
				"trading partner reads on the response it is sent.",
		}
		required = append(required, common.FieldReason)
	}

	return jsonschemautils.Object(properties, required...)
}

func (t *EdiTransferDecisionTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceEDI,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierPropose,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     t.decision.rationale + ediDecisionRationaleEnd,
	}
}

func (t *EdiTransferDecisionTool) Target(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, paramEDITransferID, permission.ResourceEDI)
}

func (t *EdiTransferDecisionTool) call(
	params *agentports.ToolExecuteParams,
) (*tenderCall, error) {
	if err := common.GuardPreview(t, params); err != nil {
		return nil, err
	}

	transferID, err := common.RequirePulid(params.Params, paramEDITransferID)
	if err != nil {
		return nil, err
	}

	call := &tenderCall{
		transferID: transferID,
		tenant:     common.TenantFrom(*params),
		actor:      params.Actor,
	}
	if !t.decision.needsReason {
		return call, nil
	}

	call.reason, err = ediservice.TransferRejectionReason(
		common.OptionalString(params.Params, common.FieldReason),
	)
	if err != nil {
		return nil, err
	}
	if len(call.reason) > maxTenderDeclineChars {
		return nil, fmt.Errorf(
			"parameter %q is %d characters; a tender response carries at most %d",
			common.FieldReason, len(call.reason), maxTenderDeclineChars,
		)
	}

	return call, nil
}

func (t *EdiTransferDecisionTool) outcome(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
) (*tenderCall, *tenderOutcome, error) {
	call, err := t.call(params)
	if err != nil {
		return nil, nil, err
	}

	outcome, err := t.decision.plan(ctx, t.edi, call)
	if err != nil {
		return nil, nil, err
	}

	return call, outcome, nil
}

func (t *EdiTransferDecisionTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	_, outcome, err := t.outcome(ctx, &params)
	if err != nil {
		return err
	}

	return outcome.refusal
}

func (t *EdiTransferDecisionTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}
	if !params.ApprovedFromProposal() {
		return ErrEDIDecisionNeedsAPerson
	}

	call, err := t.call(&params)
	if err != nil {
		return err
	}

	return t.decision.run(ctx, t.edi, call)
}

type tenderSettlement struct {
	direction string
	verb      string
	mark      func(*edi.EDITransfer)
}

func (s tenderSettlement) settle(
	ctx context.Context,
	decider ediTransferDecider,
	call *tenderCall,
) (*tenderOutcome, error) {
	transfer, err := decider.GetTransfer(ctx, ediports.GetEDITransferByIDRequest{
		ID:         call.transferID,
		TenantInfo: call.tenant,
		Direction:  s.direction,
	})
	if err != nil {
		return nil, err
	}

	outcome := &tenderOutcome{
		before:  transfer,
		refusal: ediservice.RequireActionableTransfer(transfer, s.verb),
	}
	if outcome.refusal == nil {
		after := *transfer
		s.mark(&after)
		outcome.after = &after
	}

	return outcome, nil
}

func actorUserID(actor *permissionports.RequestActor) pulid.ID {
	if actor == nil {
		return pulid.Nil
	}

	return actor.UserID
}

func planAcceptTender(
	ctx context.Context,
	decider ediTransferDecider,
	call *tenderCall,
) (*tenderOutcome, error) {
	plan, err := decider.PlanApproveTransfer(ctx, &ediservice.ApproveTransferRequest{
		TransferID: call.transferID,
		TenantInfo: call.tenant,
	}, call.actor)
	if err != nil {
		return nil, err
	}

	return &tenderOutcome{
		before:   plan.Transfer,
		after:    plan.After,
		approval: plan,
		refusal:  plan.Refusal,
	}, nil
}

func planDeclineTender(
	ctx context.Context,
	decider ediTransferDecider,
	call *tenderCall,
) (*tenderOutcome, error) {
	return tenderSettlement{
		direction: ediservice.TransferDirectionInbound,
		verb:      ediservice.TransferVerbRejected,
		mark: func(transfer *edi.EDITransfer) {
			ediservice.MarkTransferRejected(
				transfer, call.reason, actorUserID(call.actor), timeutils.NowUnix(),
			)
		},
	}.settle(ctx, decider, call)
}

func planCancelTender(
	ctx context.Context,
	decider ediTransferDecider,
	call *tenderCall,
) (*tenderOutcome, error) {
	return tenderSettlement{
		direction: ediservice.TransferDirectionOutbound,
		verb:      ediservice.TransferVerbCanceled,
		mark: func(transfer *edi.EDITransfer) {
			ediservice.MarkTransferCanceled(transfer, actorUserID(call.actor), timeutils.NowUnix())
		},
	}.settle(ctx, decider, call)
}

func planExpireTender(
	ctx context.Context,
	decider ediTransferDecider,
	call *tenderCall,
) (*tenderOutcome, error) {
	return tenderSettlement{
		direction: ediservice.TransferDirectionAny,
		verb:      ediservice.TransferVerbExpired,
		mark: func(transfer *edi.EDITransfer) {
			ediservice.MarkTransferExpired(transfer, timeutils.NowUnix())
		},
	}.settle(ctx, decider, call)
}

func NewAcceptEDILoadTenderTool(decider ediTransferDecider) agentports.AgentTool {
	return &EdiTransferDecisionTool{edi: decider, decision: transferDecision{
		name: "accept_edi_tender",
		recipe: []string{
			"list_edi_transfers",
			"get_edi_transfer",
			"get_customer",
			"search_shipments",
			"quote_shipment",
			"accept_edi_tender",
		},
		description: "Propose accepting a load tender a trading partner sent over EDI, which " +
			"creates the shipment from it and tells the partner. Read the tender with " +
			"get_edi_transfer first and check its customer, stops, dates, equipment and rate " +
			"against what this organization hauls for that customer. A tender whose mappings " +
			"are not complete is refused until an EDI administrator maps them. A person always " +
			"decides; an outside partner is sent a 990 acceptance, and another Trenova " +
			"organization sees its tender accepted.",
		rationale: "Commits the organization to haul the load, creates the shipment and " +
			"sends the partner an acceptance they act on; it cannot be taken back.",
		summary: summarizeAcceptTender,
		plan:    planAcceptTender,
		run: func(ctx context.Context, decider ediTransferDecider, call *tenderCall) error {
			_, err := decider.ApproveTransfer(ctx, &ediservice.ApproveTransferRequest{
				TransferID: call.transferID,
				TenantInfo: call.tenant,
			}, call.actor)

			return err
		},
	}}
}

func NewDeclineEDILoadTenderTool(decider ediTransferDecider) agentports.AgentTool {
	return &EdiTransferDecisionTool{edi: decider, decision: transferDecision{
		name:   "decline_edi_tender",
		recipe: []string{"list_edi_transfers", "get_edi_transfer", "decline_edi_tender"},
		description: "Propose declining a load tender a trading partner sent over EDI, with the " +
			"reason they are given. Use it when the tender cannot be hauled as sent: a lane or " +
			"equipment this organization does not run, dates it cannot make, a rate below the " +
			"customer's agreement. Read it with get_edi_transfer first. A person always " +
			"decides; an outside partner is sent a 990 decline carrying the reason.",
		rationale: "Turns down freight a partner offered and sends them the decline; the " +
			"partner moves the load elsewhere and it cannot be taken back.",
		needsReason: true,
		summary:     summarizeDeclineTender,
		plan:        planDeclineTender,
		run: func(ctx context.Context, decider ediTransferDecider, call *tenderCall) error {
			_, err := decider.RejectTransfer(ctx, &ediservice.RejectTransferRequest{
				TransferID: call.transferID,
				TenantInfo: call.tenant,
				Reason:     call.reason,
			}, call.actor)

			return err
		},
	}}
}

func NewCancelEDILoadTenderTool(decider ediTransferDecider) agentports.AgentTool {
	return &EdiTransferDecisionTool{edi: decider, decision: transferDecision{
		name:        "cancel_edi_tender",
		recipe:      []string{"list_edi_transfers", "cancel_edi_tender"},
		searchTerms: []string{"recall tender", "withdraw edi tender"},
		description: "Propose withdrawing a load tender this organization sent to another " +
			"organization over EDI and that it has not yet answered. The shipment's tender is " +
			"marked canceled and the receiving organization can no longer accept it. Find it " +
			"with list_edi_transfers, direction Outbound. A person always decides.",
		rationale: "Withdraws freight offered to another organization, which sees the " +
			"tender canceled; a withdrawn tender is sent afresh, never reopened.",
		summary: summarizeCancelTender,
		plan:    planCancelTender,
		run: func(ctx context.Context, decider ediTransferDecider, call *tenderCall) error {
			_, err := decider.CancelTransfer(ctx, &ediservice.CancelTransferRequest{
				TransferID: call.transferID,
				TenantInfo: call.tenant,
			}, call.actor)

			return err
		},
	}}
}

func NewExpireEDILoadTenderTool(decider ediTransferDecider) agentports.AgentTool {
	return &EdiTransferDecisionTool{edi: decider, decision: transferDecision{
		name:   "expire_edi_tender",
		recipe: []string{"list_edi_transfers", "expire_edi_tender"},
		description: "Propose closing an EDI load tender that went unanswered past the time " +
			"it had to be answered in, sent or received. Nobody can accept it afterwards, and " +
			"the shipment it was tendered from is marked expired. Check the dates with " +
			"get_edi_transfer first. A person always decides.",
		rationale: "Closes a tender on both sides of the trading relationship, which the " +
			"other side sees; an expired tender is sent afresh, never reopened.",
		summary: summarizeExpireTender,
		plan:    planExpireTender,
		run: func(ctx context.Context, decider ediTransferDecider, call *tenderCall) error {
			_, err := decider.ExpireTransfer(ctx, &ediservice.ExpireTransferRequest{
				TransferID: call.transferID,
				TenantInfo: call.tenant,
			}, call.actor)

			return err
		},
	}}
}

func transferLabel(transfer *edi.EDITransfer, orgID pulid.ID) string {
	return "Load tender" + transferIdentity(transfer, orgID)
}

func transferIdentity(transfer *edi.EDITransfer, orgID pulid.ID) string {
	if transfer == nil {
		return ""
	}

	identity := ""
	if bol := strings.TrimSpace(transfer.TenderPayload.BOL); bol != "" {
		identity += " BOL " + bol
	}
	partner := transfer.SourcePartner
	if transfer.TargetOrganizationID == orgID {
		partner = transfer.TargetPartner
	}
	if partner != nil && partner.Name != "" {
		identity += " with " + partner.Name
	}

	return identity
}
