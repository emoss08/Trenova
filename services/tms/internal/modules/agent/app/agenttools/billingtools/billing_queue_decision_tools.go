package billingtools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/billingqueuepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingqueueservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const paramExceptionReasonCode = "exceptionReasonCode"

const paramBillerID = "billerId"

const maxCancelReasonChars = 100

const fieldReviewCompletedAt = "reviewCompletedAt"

const fieldReviewStartedAt = "reviewStartedAt"

// queueDecision is one decision a biller makes on a queue item. The tools
// differ in the status they move the item to, what they ask for and who may
// make them; the loading, planning, checking and writing are shared.
type queueDecision struct {
	name        string
	description string
	status      billingqueue.Status
	// personOnly is a decision only a person makes: the tool proposes it,
	// runs only from a person's approval, and is planned for that person.
	personOnly bool
	egress     agent.EgressClass
	defaultTo  agent.AutonomyTier
	maxTier    agent.AutonomyTier
	// holdsWhenTainted is a decision whose notes a person reads, which a run
	// that read outside text proposes rather than writes.
	holdsWhenTainted bool
	rationale        string
	properties       map[string]any
	required         []string
	fill             func(params map[string]any, req *billingqueueports.UpdateBillingQueueStatusRequest) error
	searchTerms      []string
}

type BillingQueueDecisionTool struct {
	decision queueDecision
	billing  billingqueuepreview.BillingQueueDecider
}

func (t *BillingQueueDecisionTool) Name() string { return t.decision.name }

func (t *BillingQueueDecisionTool) Description() string { return t.decision.description }

func (t *BillingQueueDecisionTool) SearchTerms() []string { return t.decision.searchTerms }

func (t *BillingQueueDecisionTool) ParamSchema() map[string]any {
	properties := map[string]any{
		billingqueuepreview.ParamBillingQueueItemID: agenttoolschema.OfKinds(map[string]any{
			toolschema.KeyType: toolschema.TypeString,
			toolschema.KeyDescription: "The billing queue item, from billing_list_queue_items, " +
				"billing_get_queue_item or this run's subject. Never guess one.",
		}, permission.KindBillingQueueItem),
	}
	for name, property := range t.decision.properties {
		properties[name] = property
	}

	return map[string]any{
		toolschema.KeyType:       toolschema.TypeObject,
		toolschema.KeyProperties: properties,
		toolschema.KeyRequired: append(
			[]string{billingqueuepreview.ParamBillingQueueItemID},
			t.decision.required...),
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *BillingQueueDecisionTool) Policy() agenttoolports.ToolPolicy {
	policy := agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceBillingQueue,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   t.decision.defaultTo,
		MaxTier:       t.decision.maxTier,
		Egress:        []agent.EgressClass{t.decision.egress},
		Effect:        agent.ToolEffectChange,
		Reversible:    !t.decision.personOnly,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     t.decision.rationale,
	}
	if t.decision.holdsWhenTainted {
		policy.TaintHold = &agenttoolports.TaintHold{
			Description: "Its notes are what a biller or operations reads next, so a run " +
				"that has read outside text proposes it rather than writing it.",
			Applies: func(agenttoolports.ToolExecuteParams) bool { return true },
		}
	}

	return policy
}

func (t *BillingQueueDecisionTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(
		params,
		billingqueuepreview.ParamBillingQueueItemID,
		permission.ResourceBillingQueue,
	)
}

func (t *BillingQueueDecisionTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*billingqueueports.UpdateBillingQueueStatusRequest, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}

	itemID, err := toolparams.RequirePulid(
		params.Params,
		billingqueuepreview.ParamBillingQueueItemID,
	)
	if err != nil {
		return nil, err
	}

	req := &billingqueueports.UpdateBillingQueueStatusRequest{
		ItemID:     itemID,
		NewStatus:  t.decision.status,
		TenantInfo: toolparams.TenantFrom(*params),
	}
	if t.decision.fill != nil {
		if err = t.decision.fill(params.Params, req); err != nil {
			return nil, err
		}
	}

	return req, nil
}

// plan is the item as the decision leaves it, decided by the billing queue's
// own plan and required-field rules. A person-only decision is planned for
// the person who will approve it, since that person is who makes it.
func (t *BillingQueueDecisionTool) plan(
	item *billingqueue.BillingQueueItem,
	req *billingqueueports.UpdateBillingQueueStatusRequest,
	actor *permissionports.RequestActor,
	now int64,
) error {
	var err error
	if t.decision.personOnly {
		err = billingqueueservice.PlanTransition(item, req, actor, now)
	} else {
		err = billingqueueservice.PlanStatusChange(item, req, actor, now)
	}
	if err != nil {
		return err
	}

	return t.requiredFields(item)
}

func (t *BillingQueueDecisionTool) requiredFields(item *billingqueue.BillingQueueItem) error {
	multiErr := errortypes.NewMultiError()
	billingqueueservice.CheckStatusFields(item, multiErr)
	if !multiErr.HasErrors() {
		return nil
	}

	kept := errortypes.NewMultiError()
	for _, entry := range multiErr.Errors {
		// The approver who cancels is who the item records; a proposal is
		// checked before anyone has.
		if t.decision.personOnly && entry.Field == changepreview.FieldCanceledByID {
			continue
		}
		kept.Add(entry.Field, entry.Code, entry.Message, entry.Args...)
	}
	if kept.HasErrors() {
		return kept
	}

	return nil
}

func (t *BillingQueueDecisionTool) load(
	ctx context.Context,
	req *billingqueueports.UpdateBillingQueueStatusRequest,
) (*billingqueue.BillingQueueItem, error) {
	return t.billing.GetByID(ctx, &billingqueueports.GetBillingQueueItemByIDRequest{
		ItemID:                req.ItemID,
		TenantInfo:            req.TenantInfo,
		ExpandShipmentDetails: t.decision.status == billingqueue.StatusApproved,
	})
}

func (t *BillingQueueDecisionTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}

	item, err := t.load(ctx, req)
	if err != nil {
		return err
	}
	if err = t.plan(item, req, params.Actor, timeutils.NowUnix()); err != nil {
		return err
	}

	return approvalBlock(item, req.NewStatus)
}

// approvalBlock refuses an approval the billing queue would refuse when it
// ran: a detention charge on the shipment waiting on its own approval, a
// rating the tariff held for review, or a flagged check nobody has settled.
func approvalBlock(item *billingqueue.BillingQueueItem, status billingqueue.Status) error {
	if status != billingqueue.StatusApproved {
		return nil
	}
	if len(item.DetentionHolds) > 0 {
		return errortypes.NewValidationError(
			"detentionHolds",
			errortypes.ErrInvalidOperation,
			"{0} on this item's shipment still waits on approval; approve or waive it first",
			changepreview.CountOf(len(item.DetentionHolds), "detention charge"),
		)
	}
	if hold := item.RatingReviewHold; hold != nil {
		return errortypes.NewValidationError(
			"ratingReviewHold",
			errortypes.ErrInvalidOperation,
			"The tariff held this item's rating for review ({0}); a person approves the rating first",
			hold.Summary(),
		)
	}
	if open := item.OpenIssues(); len(open) > 0 {
		return errortypes.NewValidationError(
			"issues",
			errortypes.ErrInvalidOperation,
			"{0} on this item still need settling before it can be approved: {1}. "+
				"A biller settles each in the item's review",
			changepreview.CountOf(len(open), "flagged check"),
			issueSummaries(open),
		)
	}

	return nil
}

func issueSummaries(issues []*billingqueue.Issue) string {
	summaries := make([]string, 0, len(issues))
	for _, issue := range issues {
		summaries = append(summaries, issue.Summary)
	}

	return strings.Join(summaries, "; ")
}

func (t *BillingQueueDecisionTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}
	if t.decision.personOnly && !params.ApprovedFromProposal() {
		return billingqueuepreview.ErrDecisionNeedsAPerson
	}

	req, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.billing.UpdateStatus(ctx, req, params.Actor)

	return err
}

var billingReasonCodes = agenttoolschema.Source(
	"billingQueue.exceptionReasonCode",
	[]billingqueue.ExceptionReasonCode{
		billingqueue.ExceptionMissingDocumentation,
		billingqueue.ExceptionIncorrectRates,
		billingqueue.ExceptionWeightDiscrepancy,
		billingqueue.ExceptionAccessorialDispute,
		billingqueue.ExceptionDuplicateCharge,
		billingqueue.ExceptionMissingReferenceNumber,
		billingqueue.ExceptionCustomerInformationError,
		billingqueue.ExceptionServiceFailure,
		billingqueue.ExceptionRateNotOnFile,
		billingqueue.ExceptionOther,
	},
)

func reasonCodeProperty(description string) map[string]any {
	return agenttoolschema.Enum(description, billingReasonCodes)
}

func notesProperty(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyMaxLength:   billingqueuepreview.MaxDecisionNoteChars,
		toolschema.KeyDescription: description,
	}
}

// fillException reads the reason and notes an exception or a send-back
// records. The billing queue's own rule decides which need notes.
func fillException(
	params map[string]any,
	req *billingqueueports.UpdateBillingQueueStatusRequest,
) error {
	raw, err := toolparams.RequireString(params, paramExceptionReasonCode)
	if err != nil {
		return err
	}

	code := billingqueue.ExceptionReasonCode(strings.TrimSpace(raw))
	if !code.IsValid() {
		return fmt.Errorf(
			"exceptionReasonCode %q is not one of %s",
			raw,
			strings.Join(billingReasonCodes.Names(), ", "),
		)
	}
	req.ExceptionReasonCode = &code
	req.ExceptionNotes = strings.TrimSpace(toolparams.OptionalString(params, toolparams.ParamNote))

	return nil
}

func fillHold(params map[string]any, req *billingqueueports.UpdateBillingQueueStatusRequest) error {
	notes, err := toolparams.RequireString(params, toolparams.ParamNote)
	if err != nil {
		return err
	}
	req.ReviewNotes = strings.TrimSpace(notes)

	return nil
}

func fillApproval(
	params map[string]any,
	req *billingqueueports.UpdateBillingQueueStatusRequest,
) error {
	req.ReviewNotes = strings.TrimSpace(
		toolparams.OptionalString(params, billingqueuepreview.ParamReviewNotes),
	)

	return nil
}

func fillCancel(
	params map[string]any,
	req *billingqueueports.UpdateBillingQueueStatusRequest,
) error {
	reason, err := toolparams.RequireString(params, toolparams.ParamReason)
	if err != nil {
		return err
	}
	reason = strings.TrimSpace(reason)
	if len(reason) > maxCancelReasonChars {
		return fmt.Errorf(
			"cancelReason is %d characters; the billing queue keeps at most %d",
			len(reason),
			maxCancelReasonChars,
		)
	}
	req.CancelReason = reason

	return nil
}

func NewSendBackToOpsTool(
	billing billingqueuepreview.BillingQueueDecider,
) agenttoolports.AgentTool {
	return &BillingQueueDecisionTool{billing: billing, decision: queueDecision{
		name: "billing_return_to_operations",
		description: "Send a billing queue item in review back to operations to fix, with a " +
			"reason. Use it when the fix belongs to the shipment rather than the bill: a missing " +
			"document operations must collect, a wrong weight, a missing reference number. " +
			"Operations sees a high-priority note on the shipment saying why. Read the item " +
			"with billing_get_queue_item first. Notes are required when the reason is Other.",
		status:           billingqueue.StatusSentBackToOps,
		egress:           agent.EgressInternal,
		defaultTo:        agent.TierPropose,
		maxTier:          agent.TierAutoExecute,
		holdsWhenTainted: true,
		rationale: "Returns an item to operations with a note on its shipment inside Trenova; " +
			"it creates no money and the item comes back when operations fixes it.",
		properties: map[string]any{
			paramExceptionReasonCode: reasonCodeProperty("Why it goes back."),
			toolparams.ParamNote: notesProperty("What operations has to fix, in a sentence " +
				"they can act on. Required when the reason is Other."),
		},
		required: []string{paramExceptionReasonCode},
		fill:     fillException,
	}}
}

func NewMoveToExceptionTool(
	billing billingqueuepreview.BillingQueueDecider,
) agenttoolports.AgentTool {
	return &BillingQueueDecisionTool{billing: billing, decision: queueDecision{
		name: "billing_raise_exception",
		description: "Move a billing queue item in review into exception for a biller to " +
			"resolve. Use it when the bill itself is wrong: a rate that disagrees with the agreement, " +
			"a disputed accessorial, a duplicate charge. Give the reason and notes that say " +
			"what is wrong and what would clear it.",
		status:           billingqueue.StatusException,
		egress:           agent.EgressInternal,
		defaultTo:        agent.TierPropose,
		maxTier:          agent.TierAutoExecute,
		holdsWhenTainted: true,
		rationale: "Marks an item for a biller to resolve inside Trenova; it creates no money " +
			"and a biller moves it back when it is resolved.",
		properties: map[string]any{
			paramExceptionReasonCode: reasonCodeProperty("What is wrong with the bill."),
			toolparams.ParamNote: notesProperty("What is wrong and what would clear it, with " +
				"the figures you checked."),
		},
		required: []string{paramExceptionReasonCode, toolparams.ParamNote},
		fill:     fillException,
	}}
}

func NewHoldBillingQueueItemTool(
	billing billingqueuepreview.BillingQueueDecider,
) agenttoolports.AgentTool {
	return &BillingQueueDecisionTool{billing: billing, decision: queueDecision{
		name:        "billing_hold_queue_item",
		searchTerms: []string{"park item", "park until"},
		description: "Put a billing queue item on hold, with a note saying what it waits on. " +
			"Use it when an item waiting for or in review cannot be billed yet but nothing is " +
			"wrong with it: a document on its way, a customer's confirmation. A held item is taken " +
			"off hold only when the note's condition is met.",
		status:           billingqueue.StatusOnHold,
		egress:           agent.EgressInternal,
		defaultTo:        agent.TierPropose,
		maxTier:          agent.TierAutoExecute,
		holdsWhenTainted: true,
		rationale: "Parks an item inside Trenova with a note; it creates no money and is " +
			"undone by moving the item on.",
		properties: map[string]any{
			toolparams.ParamNote: notesProperty("What the item waits on, so whoever takes it " +
				"off hold knows when."),
		},
		required: []string{toolparams.ParamNote},
		fill:     fillHold,
	}}
}

func NewCancelBillingQueueItemTool(
	billing billingqueuepreview.BillingQueueDecider,
) agenttoolports.AgentTool {
	return &BillingQueueDecisionTool{billing: billing, decision: queueDecision{
		name: "billing_cancel_queue_item",
		description: "Propose canceling a billing queue item that should never be billed, " +
			"such as a duplicate of another item. Canceling drops the charge from billing for " +
			"good, so a person always decides; say which item it duplicates or why it must not " +
			"bill.",
		status:     billingqueue.StatusCanceled,
		personOnly: true,
		egress:     agent.EgressMoney,
		defaultTo:  agent.TierPropose,
		maxTier:    agent.TierPropose,
		rationale: "Drops a charge from billing for good, so only a person decides; the " +
			"agent proposes it.",
		properties: map[string]any{
			toolparams.ParamReason: map[string]any{
				toolschema.KeyType:        toolschema.TypeString,
				toolschema.KeyMaxLength:   maxCancelReasonChars,
				toolschema.KeyDescription: "Why it must not bill, in a short phrase.",
			},
		},
		required: []string{toolparams.ParamReason},
		fill:     fillCancel,
	}}
}

// assignBillerTool names the biller who reviews an item. An item waiting for
// review moves into review with them, as the queue's own assignment does.
type AssignBillerTool struct {
	billing billingqueuepreview.BillingQueueDecider
}

func NewAssignBillerTool(billing billingqueuepreview.BillingQueueDecider) agenttoolports.AgentTool {
	return &AssignBillerTool{billing: billing}
}

func (t *AssignBillerTool) Name() string { return toolAssignBiller }

func (t *AssignBillerTool) SearchTerms() []string {
	return []string{"biller", "reviewer"}
}

func (t *AssignBillerTool) Description() string {
	return "Assign the biller who reviews a billing queue item. An item still waiting for " +
		"review moves into review with them. Leave billerId out to assign the person who " +
		"asked, which is what \"assign me\", \"start reviewing\" or \"get these ready to " +
		"post\" means; name a biller only when they named one or the customer's default " +
		"biller is on another of its items. Never guess a user id. For more than one item, " +
		"call billing_assign_biller once with all of them instead."
}

func (t *AssignBillerTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			billingqueuepreview.ParamBillingQueueItemID: agenttoolschema.OfKinds(map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "The billing queue item, from " +
					"billing_list_queue_items or billing_get_queue_item.",
			}, permission.KindBillingQueueItem),
			paramBillerID: billerProperty(),
		},
		toolschema.KeyRequired:             []string{billingqueuepreview.ParamBillingQueueItemID},
		toolschema.KeyAdditionalProperties: false,
	}
}

func billerProperty() map[string]any {
	return agenttoolschema.OfResource(map[string]any{
		toolschema.KeyType: toolschema.TypeString,
		toolschema.KeyDescription: "The biller's user id, when the person named one or " +
			"the customer's default biller is the assignedBillerId billing_get_queue_item " +
			"shows on another of its items. Leave it out to assign the person who asked. " +
			"Never guess one.",
	}, permission.ResourceUser)
}

// billerOf reads the biller a call names, or the person asking when it names
// none. An unattended agent is nobody's biller, so a call from one that
// names nobody is refused with the parameter it needs. A named biller is
// checked as the queue will check it: a made-up user id was proposed, and
// failed only once a person had approved it.
func billerOf(
	ctx context.Context,
	billing billingqueuepreview.BillingQueueDecider,
	params *agenttoolports.ToolExecuteParams,
) (biller pulid.ID, asker bool, err error) {
	named, ok, err := toolparams.OptionalPulid(params.Params, paramBillerID)
	if err != nil {
		return pulid.Nil, false, errortypes.NewValidationError(
			paramBillerID, errortypes.ErrInvalid, "billerId is not a valid user id",
		)
	}
	if ok {
		if err = billing.CheckBiller(ctx, toolparams.TenantFrom(*params), named); err != nil {
			if errortypes.IsError(err) {
				return pulid.Nil, false, errortypes.NewValidationError(paramBillerID,
					errortypes.ErrInvalid, "billerId is not a user in this organization. "+
						"Leave it out to assign the person who asked, or use an id a tool "+
						"returned; never make one up")
			}

			return pulid.Nil, false, err
		}

		return named, false, nil
	}
	if params.Actor == nil || params.Actor.IsAgent() || params.Actor.UserID.IsNil() {
		return pulid.Nil, false, errortypes.NewValidationError(
			paramBillerID, errortypes.ErrRequired,
			"billerId is required: nobody is asking in person, so name the biller",
		)
	}

	return params.Actor.UserID, true, nil
}

// billerClause says who the item goes to in a preview's summary.
func billerClause(asker bool) string {
	if asker {
		return "the person who asked"
	}

	return "a biller"
}

func (t *AssignBillerTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceBillingQueue,
		Operation:     permission.OpAssign,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Names who reviews an item inside Trenova; it creates no money and is " +
			"changed by assigning someone else.",
	}
}

func (t *AssignBillerTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(
		params,
		billingqueuepreview.ParamBillingQueueItemID,
		permission.ResourceBillingQueue,
	)
}

func (t *AssignBillerTool) request(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (req *billingqueueports.AssignBillerRequest, asker bool, err error) {
	if err = toolguard.GuardPreview(t, params); err != nil {
		return nil, false, err
	}

	itemID, err := toolparams.RequirePulid(
		params.Params,
		billingqueuepreview.ParamBillingQueueItemID,
	)
	if err != nil {
		return nil, false, err
	}
	billerID, asker, err := billerOf(ctx, t.billing, params)
	if err != nil {
		return nil, false, err
	}

	return &billingqueueports.AssignBillerRequest{
		ItemID:     itemID,
		BillerID:   billerID,
		TenantInfo: toolparams.TenantFrom(*params),
	}, asker, nil
}

func (t *AssignBillerTool) load(
	ctx context.Context,
	req *billingqueueports.AssignBillerRequest,
) (*billingqueue.BillingQueueItem, error) {
	return t.billing.GetByID(ctx, &billingqueueports.GetBillingQueueItemByIDRequest{
		ItemID:     req.ItemID,
		TenantInfo: req.TenantInfo,
	})
}

func (t *AssignBillerTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, _, err := t.request(ctx, &params)
	if err != nil {
		return err
	}

	item, err := t.load(ctx, req)
	if err != nil {
		return err
	}

	return billingqueueservice.PlanAssignBiller(item, req.BillerID, timeutils.NowUnix())
}

func (t *AssignBillerTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}

	req, _, err := t.request(ctx, &params)
	if err != nil {
		return err
	}

	_, err = t.billing.AssignBiller(ctx, req, params.Actor)

	return err
}
