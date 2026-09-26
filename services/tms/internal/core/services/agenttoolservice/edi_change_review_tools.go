package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	paramEDIChangeID           = "changeId"
	paramEDIReviewDecision     = "decision"
	reviewApply                = "Apply"
	reviewReject               = "Reject"
	maxChangeReviewReasonChars = 500
)

type tenderChangeReviewer interface {
	GetTenderChange(
		ctx context.Context,
		req repositories.GetEDITenderChangeByIDRequest,
	) (*edi.TenderChange, error)
	ApplyTenderChange(
		ctx context.Context,
		req *ediservice.TenderChangeActionRequest,
		actor *serviceports.RequestActor,
	) (*edi.TenderChange, error)
	RejectTenderChange(
		ctx context.Context,
		req *ediservice.TenderChangeActionRequest,
		actor *serviceports.RequestActor,
	) (*edi.TenderChange, error)
}

type transferChangeReviewer interface {
	GetTransferChange(
		ctx context.Context,
		req repositories.GetEDITransferChangeByIDRequest,
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
		actor *serviceports.RequestActor,
	) (*edi.TransferChange, error)
	RejectTransferChange(
		ctx context.Context,
		req *ediservice.TransferChangeActionRequest,
		actor *serviceports.RequestActor,
	) (*edi.TransferChange, error)
}

type changeReviewCall struct {
	changeID pulid.ID
	apply    bool
	reason   string
	tenant   pagination.TenantInfo
	actor    *serviceports.RequestActor
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
	plan        func(context.Context, *changeReviewCall) (*changeReviewPlan, error)
	run         func(context.Context, *changeReviewCall) error
}

type ediChangeReviewTool struct {
	kind changeReviewKind
}

var (
	_ serviceports.ToolPreviewer = (*ediChangeReviewTool)(nil)
	_ serviceports.ToolValidator = (*ediChangeReviewTool)(nil)
	_ serviceports.TargetedTool  = (*ediChangeReviewTool)(nil)
)

func (t *ediChangeReviewTool) Name() string { return t.kind.name }

func (t *ediChangeReviewTool) Description() string { return t.kind.description }

func (t *ediChangeReviewTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramEDIChangeID: jsonschemautils.Text(t.kind.idGuidance),
		paramEDIReviewDecision: jsonschemautils.Enum(
			"Apply takes the change onto this organization's load; Reject leaves the load "+
				"as it is.",
			reviewApply, reviewReject,
		),
		fieldReason: map[string]any{
			toolschema.KeyType:      toolschema.TypeString,
			toolschema.KeyMaxLength: maxChangeReviewReasonChars,
			toolschema.KeyDescription: "Why, in a sentence the other side reads with the " +
				"change's outcome. Give one for every rejection.",
		},
	}, paramEDIChangeID, paramEDIReviewDecision)
}

func (t *ediChangeReviewTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
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

func (t *ediChangeReviewTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramEDIChangeID, permission.ResourceEDI)
}

func (t *ediChangeReviewTool) call(
	params *serviceports.ToolExecuteParams,
) (*changeReviewCall, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}

	changeID, err := requirePulid(params.Params, paramEDIChangeID)
	if err != nil {
		return nil, err
	}

	decision := optionalString(params.Params, paramEDIReviewDecision)
	if decision != reviewApply && decision != reviewReject {
		return nil, fmt.Errorf(
			"parameter %q must be Apply or Reject, not %q", paramEDIReviewDecision, decision,
		)
	}

	reason := strings.TrimSpace(optionalString(params.Params, fieldReason))
	if len(reason) > maxChangeReviewReasonChars {
		return nil, fmt.Errorf(
			"parameter %q is %d characters; a review keeps at most %d",
			fieldReason, len(reason), maxChangeReviewReasonChars,
		)
	}

	return &changeReviewCall{
		changeID: changeID,
		apply:    decision == reviewApply,
		reason:   reason,
		tenant:   tenantFrom(*params),
		actor:    params.Actor,
	}, nil
}

func (t *ediChangeReviewTool) plan(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*changeReviewPlan, error) {
	call, err := t.call(params)
	if err != nil {
		return nil, err
	}

	return t.kind.plan(ctx, call)
}

func (t *ediChangeReviewTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}

	return plan.refusal
}

func (t *ediChangeReviewTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
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

	return verbReject
}

func newReviewEDITenderChangeTool(reviewer tenderChangeReviewer) serviceports.AgentTool {
	return &ediChangeReviewTool{kind: changeReviewKind{
		name: "review_edi_tender_change",
		description: "Propose applying or rejecting a change another Trenova organization " +
			"made to a load it tendered to this one over EDI, after this organization " +
			"received it. Applying takes the new stops, dates, weights or charges onto the " +
			"pending tender or the shipment made from it; rejecting keeps the load as it is. " +
			"Compare the change with the load in list_edi_tender_changes first. A person " +
			"always decides.",
		rationale: "Changes a load this organization committed to on another " +
			"organization's word, and that organization sees the outcome.",
		idGuidance: "The tender change, from list_edi_tender_changes. Never guess one.",
		plan: func(ctx context.Context, call *changeReviewCall) (*changeReviewPlan, error) {
			change, err := reviewer.GetTenderChange(ctx, repositories.GetEDITenderChangeByIDRequest{
				ID:         call.changeID,
				TenantInfo: call.tenant,
			})
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

func newReviewEDITransferChangeTool(reviewer transferChangeReviewer) serviceports.AgentTool {
	return &ediChangeReviewTool{kind: changeReviewKind{
		name: "review_edi_transfer_change",
		description: "Propose applying or rejecting a status or cancellation the other " +
			"organization on an EDI-linked load reported. It waits for review before it " +
			"reaches this organization's shipment. Applying moves the shipment to the status reported; " +
			"rejecting leaves it where it is. Check the shipment's own tracking first, and " +
			"find the change with list_edi_transfer_changes. A person always decides.",
		rationale: "Changes a shipment on another organization's report, and that " +
			"organization sees the outcome on its own shipment.",
		idGuidance: "The transfer change, from list_edi_transfer_changes. Never guess one.",
		plan: func(ctx context.Context, call *changeReviewCall) (*changeReviewPlan, error) {
			change, err := reviewer.GetTransferChange(
				ctx,
				repositories.GetEDITransferChangeByIDRequest{
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
