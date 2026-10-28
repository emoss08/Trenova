package editools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const paramEDIChangeID = "changeId"

const paramEDIReviewDecision = "decision"

const reviewApply = "Apply"

const reviewReject = "Reject"

const maxChangeReviewReasonChars = 500

var ediReviewDecisions = agenttoolschema.Source(
	"edi.changeReviewDecision",
	[]string{reviewApply, reviewReject},
)

type tenderChangeReviewer interface {
	GetTenderChange(
		ctx context.Context,
		req editenderports.GetEDITenderChangeByIDRequest,
	) (*edi.TenderChange, error)
	ApplyTenderChange(
		ctx context.Context,
		req *ediservice.TenderChangeActionRequest,
		actor *permissionports.RequestActor,
	) (*edi.TenderChange, error)
	RejectTenderChange(
		ctx context.Context,
		req *ediservice.TenderChangeActionRequest,
		actor *permissionports.RequestActor,
	) (*edi.TenderChange, error)
}

type transferChangeReviewer interface {
	GetTransferChange(
		ctx context.Context,
		req editenderports.GetEDITransferChangeByIDRequest,
	) (*edi.TransferChange, error)
	CheckTransferChangeReview(
		ctx context.Context,
		change *edi.TransferChange,
		tenantInfo pagination.TenantInfo,
	) error
	PlanTransferChangeEffect(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		change *edi.TransferChange,
	) (*ediservice.TransferChangeEffect, error)
	ApplyTransferChange(
		ctx context.Context,
		req *ediservice.TransferChangeActionRequest,
		actor *permissionports.RequestActor,
	) (*edi.TransferChange, error)
	RejectTransferChange(
		ctx context.Context,
		req *ediservice.TransferChangeActionRequest,
		actor *permissionports.RequestActor,
	) (*edi.TransferChange, error)
}

type changeReviewCall struct {
	changeID pulid.ID
	apply    bool
	reason   string
	tenant   pagination.TenantInfo
	actor    *permissionports.RequestActor
}

type changeReviewPlan struct {
	summary string
	changes []*agent.RecordChange
	partial bool
	refusal error
}

type changeReviewKind struct {
	name        string
	description string
	rationale   string
	idGuidance  string
	changeKind  permission.RecordKind
	plan        func(context.Context, *changeReviewCall) (*changeReviewPlan, error)
	run         func(context.Context, *changeReviewCall) error
	recipe      []string
}

type EdiChangeReviewTool struct {
	kind changeReviewKind
}

func (t *EdiChangeReviewTool) Name() string { return t.kind.name }

func (t *EdiChangeReviewTool) Recipe() []string { return t.kind.recipe }

func (t *EdiChangeReviewTool) Description() string { return t.kind.description }

func (t *EdiChangeReviewTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramEDIChangeID: agenttoolschema.KindID(t.kind.idGuidance, t.kind.changeKind),
		paramEDIReviewDecision: agenttoolschema.Enum(
			"Apply takes the change onto this organization's load; Reject leaves the load "+
				"as it is.",
			ediReviewDecisions,
		),
		changepreview.FieldReason: map[string]any{
			toolschema.KeyType:      toolschema.TypeString,
			toolschema.KeyMaxLength: maxChangeReviewReasonChars,
			toolschema.KeyDescription: "Why, in a sentence the other side reads with the " +
				"change's outcome. Give one for every rejection.",
		},
	}, paramEDIChangeID, paramEDIReviewDecision)
}

func (t *EdiChangeReviewTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
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
		Rationale:     t.kind.rationale + ediDecisionRationaleEnd,
	}
}

func (t *EdiChangeReviewTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, paramEDIChangeID, permission.ResourceEDI)
}

func (t *EdiChangeReviewTool) call(
	params *agenttoolports.ToolExecuteParams,
) (*changeReviewCall, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}

	changeID, err := toolparams.RequirePulid(params.Params, paramEDIChangeID)
	if err != nil {
		return nil, err
	}

	decision := toolparams.OptionalString(params.Params, paramEDIReviewDecision)
	if decision != reviewApply && decision != reviewReject {
		return nil, fmt.Errorf(
			"parameter %q must be Apply or Reject, not %q", paramEDIReviewDecision, decision,
		)
	}

	reason := strings.TrimSpace(toolparams.OptionalString(params.Params, changepreview.FieldReason))
	if len(reason) > maxChangeReviewReasonChars {
		return nil, fmt.Errorf(
			"parameter %q is %d characters; a review keeps at most %d",
			changepreview.FieldReason, len(reason), maxChangeReviewReasonChars,
		)
	}

	return &changeReviewCall{
		changeID: changeID,
		apply:    decision == reviewApply,
		reason:   reason,
		tenant:   toolparams.TenantFrom(*params),
		actor:    params.Actor,
	}, nil
}

func (t *EdiChangeReviewTool) plan(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*changeReviewPlan, error) {
	call, err := t.call(params)
	if err != nil {
		return nil, err
	}

	return t.kind.plan(ctx, call)
}

func (t *EdiChangeReviewTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}

	return plan.refusal
}

func (t *EdiChangeReviewTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}
	if !params.ApprovedFromProposal() {
		return ErrEDIDecisionNeedsAPerson
	}

	call, err := t.call(&params)
	if err != nil {
		return err
	}

	return t.kind.run(ctx, call)
}

func (c *changeReviewCall) mark() *ediservice.ChangeReviewMark {
	return &ediservice.ChangeReviewMark{
		Applied:    c.apply,
		ReviewerID: actorUserID(c.actor),
		Reason:     c.reason,
		At:         timeutils.NowUnix(),
	}
}

func (c *changeReviewCall) verb() string {
	if c.apply {
		return "apply"
	}

	return changepreview.VerbReject
}

func NewReviewEDITenderChangeTool(reviewer tenderChangeReviewer) agenttoolports.AgentTool {
	return &EdiChangeReviewTool{kind: changeReviewKind{
		name:   "edi_review_tender_change",
		recipe: []string{"edi_list_tender_changes", "edi_review_tender_change"},
		description: "Propose applying or rejecting a change another Trenova organization " +
			"made to a load it tendered to this one over EDI, after this organization " +
			"received it. Applying takes the new stops, dates, weights or charges onto the " +
			"pending tender or the shipment made from it; rejecting keeps the load as it is. " +
			"Compare the change with the load in edi_list_tender_changes first. A person " +
			"always decides.",
		rationale: "Changes a load this organization committed to on another " +
			"organization's word, and that organization sees the outcome.",
		idGuidance: "The tender change, from edi_list_tender_changes. Never guess one.",
		changeKind: permission.KindEDITenderChange,
		plan: func(ctx context.Context, call *changeReviewCall) (*changeReviewPlan, error) {
			change, err := reviewer.GetTenderChange(
				ctx,
				editenderports.GetEDITenderChangeByIDRequest{
					ID:         call.changeID,
					TenantInfo: call.tenant,
				},
			)
			if err != nil {
				return nil, err
			}

			return planTenderChangeReview(change, call)
		},
		run: func(ctx context.Context, call *changeReviewCall) error {
			req := &ediservice.TenderChangeActionRequest{
				ChangeID:   call.changeID,
				TenantInfo: call.tenant,
				Reason:     call.reason,
			}
			var err error
			if call.apply {
				_, err = reviewer.ApplyTenderChange(ctx, req, call.actor)
			} else {
				_, err = reviewer.RejectTenderChange(ctx, req, call.actor)
			}

			return err
		},
	}}
}

func NewReviewEDITransferChangeTool(reviewer transferChangeReviewer) agenttoolports.AgentTool {
	return &EdiChangeReviewTool{kind: changeReviewKind{
		name: "edi_review_transfer_change",
		recipe: []string{
			"edi_list_transfer_changes",
			"shipment_get_tracking",
			"edi_review_transfer_change",
		},
		description: "Propose applying or rejecting a status or cancellation the other " +
			"organization on an EDI-linked load reported. It waits for review before it " +
			"reaches this organization's shipment. Applying moves the shipment to the status reported; " +
			"rejecting leaves it where it is. Check the shipment's own tracking first, and " +
			"find the change with edi_list_transfer_changes. A person always decides.",
		rationale: "Changes a shipment on another organization's report, and that " +
			"organization sees the outcome on its own shipment.",
		idGuidance: "The transfer change, from edi_list_transfer_changes. Never guess one.",
		changeKind: permission.KindEDITransferChange,
		plan: func(ctx context.Context, call *changeReviewCall) (*changeReviewPlan, error) {
			change, err := reviewer.GetTransferChange(
				ctx,
				editenderports.GetEDITransferChangeByIDRequest{
					ID:         call.changeID,
					TenantInfo: call.tenant,
				},
			)
			if err != nil {
				return nil, err
			}

			return planTransferChangeReview(ctx, reviewer, change, call)
		},
		run: func(ctx context.Context, call *changeReviewCall) error {
			req := &ediservice.TransferChangeActionRequest{
				ChangeID:   call.changeID,
				TenantInfo: call.tenant,
				Reason:     call.reason,
			}
			var err error
			if call.apply {
				_, err = reviewer.ApplyTransferChange(ctx, req, call.actor)
			} else {
				_, err = reviewer.RejectTransferChange(ctx, req, call.actor)
			}

			return err
		},
	}}
}
