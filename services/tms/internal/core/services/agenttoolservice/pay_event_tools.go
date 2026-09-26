package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/driversettlementservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramPayEventID       = "payEventId"
	paramPayEventIDs      = "payEventIds"
	paramHoldReason       = "reason"
	maxHoldReasonChars    = 500
	maxPayEventsPerAttach = 50
	fieldOnHold           = "onHold"
	fieldHoldReason       = "holdReason"
	payEventSources       = "list_driver_pay_events"
)

type payEventHolder interface {
	PlanPayEventHold(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		eventID pulid.ID,
		reason string,
		hold bool,
	) (*driversettlementservice.PayEventPlan, error)
	HoldPayEvent(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		eventID pulid.ID,
		reason string,
		actor *serviceports.RequestActor,
	) (*driversettlement.PayEvent, error)
	ReleasePayEvent(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		eventID pulid.ID,
		actor *serviceports.RequestActor,
	) (*driversettlement.PayEvent, error)
}

type payEventHoldTool struct {
	events payEventHolder
	hold   bool
}

var (
	_ serviceports.ToolPreviewer = (*payEventHoldTool)(nil)
	_ serviceports.ToolValidator = (*payEventHoldTool)(nil)
)

func provideHoldDriverPayEventTool(s *driversettlementservice.Service) serviceports.AgentTool {
	return &payEventHoldTool{events: s, hold: true}
}

func provideReleaseDriverPayEventTool(s *driversettlementservice.Service) serviceports.AgentTool {
	return &payEventHoldTool{events: s}
}

func (t *payEventHoldTool) Name() string {
	if t.hold {
		return "hold_driver_pay_event"
	}

	return "release_driver_pay_event"
}

func (t *payEventHoldTool) Description() string {
	if t.hold {
		return "Hold a driver's accrued pay for one load so it skips settlement until " +
			"released, with the reason. The driver is told in the driver portal why pay " +
			"was deferred, so write the reason for them, such as paperwork still to come or a " +
			"load under review. Use release_driver_pay_event when it is resolved."
	}

	return "Release a held driver pay event back into the pool the next settlement draws " +
		"from. Where the settlement control attaches accruals to open drafts, it joins the " +
		"driver's open draft at once. Release only once what the hold waited on is resolved."
}

func (t *payEventHoldTool) ParamSchema() map[string]any {
	properties := map[string]any{
		paramPayEventID: map[string]any{
			toolschema.KeyType: toolschema.TypeString,
			toolschema.KeyDescription: "The pay event, from " + payEventSources +
				" or the lines get_driver_settlement lists. Never guess one.",
		},
	}
	required := []string{paramPayEventID}
	if t.hold {
		properties[paramHoldReason] = stringProperty("Why the pay is held, in words for the "+
			"driver.", maxHoldReasonChars)
		required = append(required, paramHoldReason)
	}

	return map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyRequired:             required,
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *payEventHoldTool) Policy() serviceports.ToolPolicy {
	policy := serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceDriverSettlement,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Returns held pay to the settlement pool inside Trenova; nothing is paid " +
			"until a settlement is approved, and the pay is held again the same way.",
	}
	if t.hold {
		policy.MaxTier = agent.TierActWithApproval
		policy.Egress = []agent.EgressClass{agent.EgressDriverVisible}
		policy.Rationale = "Defers a driver's pay and tells the driver why in the driver " +
			"portal; it is released the same way."
	}

	return policy
}

type payEventHoldRequest struct {
	tenant  pagination.TenantInfo
	eventID pulid.ID
	reason  string
}

func (t *payEventHoldTool) request(
	params *serviceports.ToolExecuteParams,
) (*payEventHoldRequest, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}
	eventID, err := requirePulid(params.Params, paramPayEventID)
	if err != nil {
		return nil, err
	}
	req := &payEventHoldRequest{tenant: tenantFrom(*params), eventID: eventID}
	if t.hold {
		if req.reason, err = boundedString(
			params.Params,
			paramHoldReason,
			maxHoldReasonChars,
			true,
		); err != nil {
			return nil, err
		}
	}

	return req, nil
}

func (t *payEventHoldTool) plan(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*driversettlementservice.PayEventPlan, error) {
	req, err := t.request(params)
	if err != nil {
		return nil, err
	}

	return t.events.PlanPayEventHold(ctx, req.tenant, req.eventID, req.reason, t.hold)
}

func (t *payEventHoldTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}

	return plan.Refusal
}

func (t *payEventHoldTool) Execute(
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
	if t.hold {
		_, err = t.events.HoldPayEvent(ctx, req.tenant, req.eventID, req.reason, params.Actor)

		return err
	}
	_, err = t.events.ReleasePayEvent(ctx, req.tenant, req.eventID, params.Actor)

	return err
}

func payEventRecord(event *driversettlement.PayEvent) toolpreview.Record {
	label := event.ProNumber
	if strings.TrimSpace(label) == "" {
		label = "Pay event"
	}

	return toolpreview.Record{
		Resource: permission.ResourceDriverSettlement,
		ID:       event.ID,
		Label:    label,
		Version:  pinnedVersion(event.Version),
	}
}

func payEventDriver(event *driversettlement.PayEvent) string {
	if event.Worker != nil {
		return event.Worker.FullName()
	}

	return "the driver"
}

func (t *payEventHoldTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return nil, err
	}

	verb := "release"
	if t.hold {
		verb = "hold"
	}
	summary := fmt.Sprintf(
		"Would %s %s's pay for %s.",
		verb,
		payEventDriver(plan.Before),
		payEventRecord(plan.Before).Label,
	)
	if plan.Refusal != nil {
		return wouldFail(toolpreview.Build(summary), plan.Refusal)
	}
	if !plan.Changed {
		return toolpreview.Build(fmt.Sprintf(
			"%s's pay for %s is already %s; nothing would change.",
			payEventDriver(plan.Before),
			payEventRecord(plan.Before).Label,
			map[bool]string{true: "held", false: "released"}[t.hold],
		)), nil
	}

	change, err := toolpreview.Changed(
		payEventRecord(plan.Before),
		plan.Before,
		plan.After,
		toolpreview.Only(fieldOnHold, fieldHoldReason),
	)
	if err != nil {
		return nil, err
	}
	changes := []*agent.RecordChange{change}
	if t.hold {
		changes = append(changes, toolpreview.Send(payEventRecord(plan.Before),
			&agent.MessagePreview{
				Channel: agent.MessageChannelDash,
				To:      []string{payEventDriver(plan.Before)},
				Body:    plan.After.HoldReason,
			}))
		summary += " The driver is told why in the driver portal."
	}
	if plan.AutoAttach {
		summary += " The settlement control attaches it to the driver's open draft " +
			"settlement, which is recalculated."
	}

	return toolpreview.Build(summary, changes...), nil
}

type payEventTransferrer interface {
	PlanEventTransfer(
		ctx context.Context,
		req *driversettlementservice.EventTransferRequest,
	) (*driversettlementservice.ActionPlan, error)
	AttachPayEvents(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		settlementID pulid.ID,
		eventIDs []pulid.ID,
		actor *serviceports.RequestActor,
	) (*driversettlement.Settlement, error)
	DetachPayEvent(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		settlementID, eventID pulid.ID,
		actor *serviceports.RequestActor,
	) (*driversettlement.Settlement, error)
}

type payEventTransferTool struct {
	events payEventTransferrer
	detach bool
}

var (
	_ serviceports.ToolPreviewer = (*payEventTransferTool)(nil)
	_ serviceports.ToolValidator = (*payEventTransferTool)(nil)
	_ serviceports.TargetedTool  = (*payEventTransferTool)(nil)
)

func provideAttachPayEventsTool(s *driversettlementservice.Service) serviceports.AgentTool {
	return &payEventTransferTool{events: s}
}

func provideDetachPayEventTool(s *driversettlementservice.Service) serviceports.AgentTool {
	return &payEventTransferTool{events: s, detach: true}
}

func (t *payEventTransferTool) Name() string {
	if t.detach {
		return "detach_pay_event_from_settlement"
	}

	return "attach_pay_events_to_settlement"
}

func (t *payEventTransferTool) Description() string {
	if t.detach {
		return "Take one load's pay off a draft driver settlement and return it to the " +
			"unsettled pool, such as pay that belongs to the next period or is disputed. " +
			"Its earning lines leave the settlement and the totals are recomputed."
	}

	return "Add a driver's accrued pay events to their draft driver settlement, such as a " +
		"late load that missed generation. Each event's earning lines join the settlement, " +
		"a held event is released as it is added, and the totals are recomputed."
}

func (t *payEventTransferTool) ParamSchema() map[string]any {
	properties := map[string]any{
		paramSettlementID: map[string]any{
			toolschema.KeyType: toolschema.TypeString,
			toolschema.KeyDescription: "The draft driver settlement, from " +
				driverSettlementReads + ". Never guess one.",
		},
	}
	var required []string
	if t.detach {
		properties[paramPayEventID] = map[string]any{
			toolschema.KeyType: toolschema.TypeString,
			toolschema.KeyDescription: "The pay event on it, from the lines " +
				"get_driver_settlement lists.",
		}
		required = []string{paramSettlementID, paramPayEventID}
	} else {
		properties[paramPayEventIDs] = map[string]any{
			toolschema.KeyType:     toolschema.TypeArray,
			toolschema.KeyMinItems: 1,
			toolschema.KeyMaxItems: maxPayEventsPerAttach,
			toolschema.KeyItems:    map[string]any{toolschema.KeyType: toolschema.TypeString},
			toolschema.KeyDescription: "The driver's accrued pay events to add, from " +
				payEventSources + " with status Accrued.",
		}
		required = []string{paramSettlementID, paramPayEventIDs}
	}

	return map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyRequired:             required,
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *payEventTransferTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceDriverSettlement,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Moves pay a driver already earned between the pool and a draft " +
			"settlement inside Trenova; nothing is paid until a person approves it.",
	}
}

func (t *payEventTransferTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramSettlementID, permission.ResourceDriverSettlement)
}

func (t *payEventTransferTool) request(
	params *serviceports.ToolExecuteParams,
) (*driversettlementservice.EventTransferRequest, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}
	settlementID, err := requirePulid(params.Params, paramSettlementID)
	if err != nil {
		return nil, err
	}
	req := &driversettlementservice.EventTransferRequest{
		TenantInfo:   tenantFrom(*params),
		SettlementID: settlementID,
		Detach:       t.detach,
	}
	if t.detach {
		eventID, idErr := requirePulid(params.Params, paramPayEventID)
		if idErr != nil {
			return nil, idErr
		}
		req.EventIDs = []pulid.ID{eventID}

		return req, nil
	}
	if req.EventIDs, err = requirePulidSlice(
		params.Params,
		paramPayEventIDs,
		maxPayEventsPerAttach,
	); err != nil {
		return nil, err
	}

	return req, nil
}

func (t *payEventTransferTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	plan, err := t.events.PlanEventTransfer(ctx, req)
	if err != nil {
		return err
	}

	return plan.Refusal
}

func (t *payEventTransferTool) Execute(
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
	if t.detach {
		_, err = t.events.DetachPayEvent(
			ctx,
			req.TenantInfo,
			req.SettlementID,
			req.EventIDs[0],
			params.Actor,
		)

		return err
	}
	_, err = t.events.AttachPayEvents(
		ctx,
		req.TenantInfo,
		req.SettlementID,
		req.EventIDs,
		params.Actor,
	)

	return err
}

func (t *payEventTransferTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	req, err := t.request(&params)
	if err != nil {
		return nil, err
	}
	plan, err := t.events.PlanEventTransfer(ctx, req)
	if err != nil {
		return nil, err
	}

	before := driverSettlementFacts(plan.Before)
	summary := fmt.Sprintf(
		"Would add %s to driver settlement %s%s.",
		countOf(len(req.EventIDs), "pay event"),
		before.number,
		payeeClause(before.payee),
	)
	if t.detach {
		summary = fmt.Sprintf(
			"Would take a pay event off driver settlement %s%s and return it to the "+
				"unsettled pool.",
			before.number,
			payeeClause(before.payee),
		)
	}
	if plan.Refusal != nil {
		return wouldFail(toolpreview.Build(summary), plan.Refusal)
	}

	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceDriverSettlement,
		ID:       before.id,
		Label:    before.number,
		Version:  pinnedVersion(before.version),
	}, plan.Before, plan.After, toolpreview.Only("shipmentCount"))
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(
		change,
		settlementMoney(before, driverSettlementFacts(plan.After)),
		toolpreview.SensitiveAs(driverSettlementLedger().sensitive...),
	)

	return toolpreview.Build(summary, change), nil
}
