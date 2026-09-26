package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/billingqueueservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	paramBillingQueueItemID  = "billingQueueItemId"
	paramExceptionReasonCode = "exceptionReasonCode"
	paramDecisionNotes       = "notes"
	paramReviewNotes         = "reviewNotes"
	paramCancelReason        = "cancelReason"
	paramBillerID            = "billerId"
	maxCancelReasonChars     = 100
	maxDecisionNoteChars     = 2000
)

// ErrDecisionNeedsAPerson is an approval or a cancellation that did not come
// from a proposal a person approved. The tool's tier ceiling already keeps the
// runtime from running it on its own, and the billing queue refuses an agent
// either decision; this is the same rule where the tool runs.
var ErrDecisionNeedsAPerson = errors.New(
	"a billing queue item is approved or canceled only once a person approves the proposal",
)

type billingQueueDecider interface {
	GetByID(
		ctx context.Context,
		req *repositories.GetBillingQueueItemByIDRequest,
	) (*billingqueue.BillingQueueItem, error)
	UpdateStatus(
		ctx context.Context,
		req *serviceports.UpdateBillingQueueStatusRequest,
		actor *serviceports.RequestActor,
	) (*billingqueue.BillingQueueItem, error)
	AssignBiller(
		ctx context.Context,
		req *serviceports.AssignBillerRequest,
		actor *serviceports.RequestActor,
	) (*billingqueue.BillingQueueItem, error)
}

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
	fill             func(params map[string]any, req *serviceports.UpdateBillingQueueStatusRequest) error
}

type billingQueueDecisionTool struct {
	decision queueDecision
	billing  billingQueueDecider
}

var (
	_ serviceports.ToolPreviewer = (*billingQueueDecisionTool)(nil)
	_ serviceports.ToolValidator = (*billingQueueDecisionTool)(nil)
	_ serviceports.TargetedTool  = (*billingQueueDecisionTool)(nil)
)

func (t *billingQueueDecisionTool) Name() string { return t.decision.name }

func (t *billingQueueDecisionTool) Description() string { return t.decision.description }

func (t *billingQueueDecisionTool) ParamSchema() map[string]any {
	properties := map[string]any{
		paramBillingQueueItemID: map[string]any{
			toolschema.KeyType: toolschema.TypeString,
			toolschema.KeyDescription: "The billing queue item, from list_billing_queue_items, " +
				"get_billing_queue_item or this run's subject. Never guess one.",
		},
	}
	for name, property := range t.decision.properties {
		properties[name] = property
	}

	return map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyRequired:             append([]string{paramBillingQueueItemID}, t.decision.required...),
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *billingQueueDecisionTool) Policy() serviceports.ToolPolicy {
	policy := serviceports.ToolPolicy{
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
		policy.TaintHold = &serviceports.TaintHold{
			Description: "Its notes are what a biller or operations reads next, so a run " +
				"that has read outside text proposes it rather than writing it.",
			Applies: func(serviceports.ToolExecuteParams) bool { return true },
		}
	}

	return policy
}

func (t *billingQueueDecisionTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramBillingQueueItemID, permission.ResourceBillingQueue)
}

func (t *billingQueueDecisionTool) request(
	params *serviceports.ToolExecuteParams,
) (*serviceports.UpdateBillingQueueStatusRequest, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}

	itemID, err := requirePulid(params.Params, paramBillingQueueItemID)
	if err != nil {
		return nil, err
	}

	req := &serviceports.UpdateBillingQueueStatusRequest{
		ItemID:     itemID,
		NewStatus:  t.decision.status,
		TenantInfo: tenantFrom(*params),
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
func (t *billingQueueDecisionTool) plan(
	item *billingqueue.BillingQueueItem,
	req *serviceports.UpdateBillingQueueStatusRequest,
	actor *serviceports.RequestActor,
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

func (t *billingQueueDecisionTool) requiredFields(item *billingqueue.BillingQueueItem) error {
	multiErr := errortypes.NewMultiError()
	billingqueueservice.CheckStatusFields(item, multiErr)
	if !multiErr.HasErrors() {
		return nil
	}

	kept := errortypes.NewMultiError()
	for _, entry := range multiErr.Errors {
		// The approver who cancels is who the item records; a proposal is
		// checked before anyone has.
		if t.decision.personOnly && entry.Field == "canceledById" {
			continue
		}
		kept.Add(entry.Field, entry.Code, entry.Message, entry.Args...)
	}
	if kept.HasErrors() {
		return kept
	}

	return nil
}

func (t *billingQueueDecisionTool) load(
	ctx context.Context,
	req *serviceports.UpdateBillingQueueStatusRequest,
) (*billingqueue.BillingQueueItem, error) {
	return t.billing.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:                req.ItemID,
		TenantInfo:            req.TenantInfo,
		ExpandShipmentDetails: t.decision.status == billingqueue.StatusApproved,
	})
}

func (t *billingQueueDecisionTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
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

	return detentionHold(item, req.NewStatus)
}

// detentionHold refuses an approval while a detention charge on the shipment
// waits on its own approval, as the billing queue does.
func detentionHold(item *billingqueue.BillingQueueItem, status billingqueue.Status) error {
	if status != billingqueue.StatusApproved || len(item.DetentionHolds) == 0 {
		return nil
	}

	return errortypes.NewValidationError(
		"detentionHolds",
		errortypes.ErrInvalidOperation,
		"{0} on this item's shipment still waits on approval; approve or waive it first",
		countOf(len(item.DetentionHolds), "detention charge"),
	)
}

func (t *billingQueueDecisionTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}
	if t.decision.personOnly && !params.ApprovedFromProposal() {
		return ErrDecisionNeedsAPerson
	}

	req, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.billing.UpdateStatus(ctx, req, params.Actor)

	return err
}

func reasonCodeNames() []string {
	codes := []billingqueue.ExceptionReasonCode{
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
	}
	names := make([]string, 0, len(codes))
	for _, code := range codes {
		names = append(names, string(code))
	}

	return names
}

func reasonCodeProperty(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyEnum:        reasonCodeNames(),
		toolschema.KeyDescription: description,
	}
}

func notesProperty(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		"maxLength":               maxDecisionNoteChars,
		toolschema.KeyDescription: description,
	}
}

// fillException reads the reason and notes an exception or a send-back
// records. The billing queue's own rule decides which need notes.
func fillException(params map[string]any, req *serviceports.UpdateBillingQueueStatusRequest) error {
	raw, err := requireString(params, paramExceptionReasonCode)
	if err != nil {
		return err
	}

	code := billingqueue.ExceptionReasonCode(strings.TrimSpace(raw))
	if !code.IsValid() {
		return fmt.Errorf(
			"exceptionReasonCode %q is not one of %s",
			raw,
			strings.Join(reasonCodeNames(), ", "),
		)
	}
	req.ExceptionReasonCode = &code
	req.ExceptionNotes = strings.TrimSpace(optionalString(params, paramDecisionNotes))

	return nil
}

func fillHold(params map[string]any, req *serviceports.UpdateBillingQueueStatusRequest) error {
	notes, err := requireString(params, paramDecisionNotes)
	if err != nil {
		return err
	}
	req.ReviewNotes = strings.TrimSpace(notes)

	return nil
}

func fillApproval(params map[string]any, req *serviceports.UpdateBillingQueueStatusRequest) error {
	req.ReviewNotes = strings.TrimSpace(optionalString(params, paramReviewNotes))

	return nil
}

func fillCancel(params map[string]any, req *serviceports.UpdateBillingQueueStatusRequest) error {
	reason, err := requireString(params, paramCancelReason)
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

func newSendBackToOpsTool(billing billingQueueDecider) serviceports.AgentTool {
	return &billingQueueDecisionTool{billing: billing, decision: queueDecision{
		name: "send_billing_item_back_to_ops",
		description: "Send a billing queue item in review back to operations to fix, with the " +
			"reason, when the fix belongs to the shipment rather than the bill: a missing " +
			"document operations must collect, a wrong weight, a missing reference number. " +
			"Operations sees a high-priority note on the shipment saying why. Read the item " +
			"with get_billing_queue_item first. Notes are required when the reason is Other.",
		status:           billingqueue.StatusSentBackToOps,
		egress:           agent.EgressInternal,
		defaultTo:        agent.TierPropose,
		maxTier:          agent.TierAutoExecute,
		holdsWhenTainted: true,
		rationale: "Returns an item to operations with a note on its shipment inside Trenova; " +
			"it creates no money and the item comes back when operations fixes it.",
		properties: map[string]any{
			paramExceptionReasonCode: reasonCodeProperty("Why it goes back."),
			paramDecisionNotes: notesProperty("What operations has to fix, in a sentence " +
				"they can act on. Required when the reason is Other."),
		},
		required: []string{paramExceptionReasonCode},
		fill:     fillException,
	}}
}

func newMoveToExceptionTool(billing billingQueueDecider) serviceports.AgentTool {
	return &billingQueueDecisionTool{billing: billing, decision: queueDecision{
		name: "move_billing_item_to_exception",
		description: "Move a billing queue item in review into exception when the bill itself " +
			"is wrong and a biller has to resolve it: a rate that disagrees with the agreement, " +
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
			paramDecisionNotes: notesProperty("What is wrong and what would clear it, with " +
				"the figures you checked."),
		},
		required: []string{paramExceptionReasonCode, paramDecisionNotes},
		fill:     fillException,
	}}
}

func newHoldBillingQueueItemTool(billing billingQueueDecider) serviceports.AgentTool {
	return &billingQueueDecisionTool{billing: billing, decision: queueDecision{
		name: "hold_billing_queue_item",
		description: "Put a billing queue item that is waiting for or in review on hold, with " +
			"a note saying what it waits on, when it cannot be billed yet but nothing is wrong " +
			"with it: a document on its way, a customer's confirmation. A held item is taken " +
			"off hold only when the note's condition is met.",
		status:           billingqueue.StatusOnHold,
		egress:           agent.EgressInternal,
		defaultTo:        agent.TierPropose,
		maxTier:          agent.TierAutoExecute,
		holdsWhenTainted: true,
		rationale: "Parks an item inside Trenova with a note; it creates no money and is " +
			"undone by moving the item on.",
		properties: map[string]any{
			paramDecisionNotes: notesProperty("What the item waits on, so whoever takes it " +
				"off hold knows when."),
		},
		required: []string{paramDecisionNotes},
		fill:     fillHold,
	}}
}

func newCancelBillingQueueItemTool(billing billingQueueDecider) serviceports.AgentTool {
	return &billingQueueDecisionTool{billing: billing, decision: queueDecision{
		name: "cancel_billing_queue_item",
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
			paramCancelReason: map[string]any{
				toolschema.KeyType:        toolschema.TypeString,
				"maxLength":               maxCancelReasonChars,
				toolschema.KeyDescription: "Why it must not bill, in a short phrase.",
			},
		},
		required: []string{paramCancelReason},
		fill:     fillCancel,
	}}
}

// assignBillerTool names the biller who reviews an item. An item waiting for
// review moves into review with them, as the queue's own assignment does.
type assignBillerTool struct {
	billing billingQueueDecider
}

var (
	_ serviceports.ToolPreviewer = (*assignBillerTool)(nil)
	_ serviceports.ToolValidator = (*assignBillerTool)(nil)
	_ serviceports.TargetedTool  = (*assignBillerTool)(nil)
)

func newAssignBillerTool(billing billingQueueDecider) serviceports.AgentTool {
	return &assignBillerTool{billing: billing}
}

func (t *assignBillerTool) Name() string { return "assign_billing_queue_biller" }

func (t *assignBillerTool) Description() string {
	return "Assign the biller who reviews a billing queue item. An item still waiting for " +
		"review moves into review with them. Take the biller from the item's customer's " +
		"default biller or the person who asked; never guess a user id."
}

func (t *assignBillerTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			paramBillingQueueItemID: map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "The billing queue item, from " +
					"list_billing_queue_items or get_billing_queue_item.",
			},
			paramBillerID: map[string]any{
				toolschema.KeyType:        toolschema.TypeString,
				toolschema.KeyDescription: "The biller's user id.",
			},
		},
		toolschema.KeyRequired:             []string{paramBillingQueueItemID, paramBillerID},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *assignBillerTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
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

func (t *assignBillerTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramBillingQueueItemID, permission.ResourceBillingQueue)
}

func (t *assignBillerTool) request(
	params *serviceports.ToolExecuteParams,
) (*serviceports.AssignBillerRequest, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}

	itemID, err := requirePulid(params.Params, paramBillingQueueItemID)
	if err != nil {
		return nil, err
	}
	billerID, err := requirePulid(params.Params, paramBillerID)
	if err != nil {
		return nil, err
	}

	return &serviceports.AssignBillerRequest{
		ItemID:     itemID,
		BillerID:   billerID,
		TenantInfo: tenantFrom(*params),
	}, nil
}

func (t *assignBillerTool) load(
	ctx context.Context,
	req *serviceports.AssignBillerRequest,
) (*billingqueue.BillingQueueItem, error) {
	return t.billing.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		ItemID:     req.ItemID,
		TenantInfo: req.TenantInfo,
	})
}

func (t *assignBillerTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}

	item, err := t.load(ctx, req)
	if err != nil {
		return err
	}

	return billingqueueservice.PlanAssignBiller(item, req.BillerID, timeutils.NowUnix())
}

func (t *assignBillerTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	req, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.billing.AssignBiller(ctx, req, params.Actor)

	return err
}
