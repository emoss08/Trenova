package paytools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/settlementpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramPayEventID = "payEventId"

const ParamPayEventIDs = "payEventIds"

const paramHoldReason = "reason"

const maxHoldReasonChars = 500

const maxPayEventsPerAttach = 50

const fieldOnHold = "onHold"

const fieldHoldReason = "holdReason"

const payEventSources = "settlement_list_pay_events"

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
		actor *permissionports.RequestActor,
	) (*driversettlement.PayEvent, error)
	ReleasePayEvent(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		eventID pulid.ID,
		actor *permissionports.RequestActor,
	) (*driversettlement.PayEvent, error)
}

type PayEventHoldTool struct {
	events payEventHolder
	hold   bool
}

func ProvideHoldDriverPayEventTool(s *driversettlementservice.Service) agenttoolports.AgentTool {
	return &PayEventHoldTool{events: s, hold: true}
}

func ProvideReleaseDriverPayEventTool(s *driversettlementservice.Service) agenttoolports.AgentTool {
	return &PayEventHoldTool{events: s}
}

func (t *PayEventHoldTool) Name() string {
	if t.hold {
		return "settlement_hold_pay_event"
	}

	return "settlement_release_pay_event"
}

func (t *PayEventHoldTool) Description() string {
	if t.hold {
		return "Hold a driver's accrued pay for one load so it skips settlement until " +
			"released, with the reason. The driver is told in the driver portal why pay " +
			"was deferred, so write the reason for them, such as paperwork still to come or a " +
			"load under review. Use settlement_release_pay_event when it is resolved."
	}

	return "Release a held driver pay event back into the pool the next settlement draws " +
		"from. Where the settlement control attaches accruals to open drafts, it joins the " +
		"driver's open draft at once. Release only once what the hold waited on is resolved."
}

func (t *PayEventHoldTool) ParamSchema() map[string]any {
	properties := map[string]any{
		paramPayEventID: agenttoolschema.KindID(
			"The pay event, from "+payEventSources+
				" or the lines settlement_get_driver lists. Never guess one.",
			permission.KindPayEvent,
		),
	}
	required := []string{paramPayEventID}
	if t.hold {
		properties[paramHoldReason] = toolparams.StringProperty(
			"Why the pay is held, in words for the "+
				"driver.",
			maxHoldReasonChars,
		)
		required = append(required, paramHoldReason)
	}

	return map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyRequired:             required,
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *PayEventHoldTool) Policy() agenttoolports.ToolPolicy {
	policy := agenttoolports.ToolPolicy{
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

func (t *PayEventHoldTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*payEventHoldRequest, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}
	eventID, err := toolparams.RequirePulid(params.Params, paramPayEventID)
	if err != nil {
		return nil, err
	}
	req := &payEventHoldRequest{tenant: toolparams.TenantFrom(*params), eventID: eventID}
	if t.hold {
		if req.reason, err = toolparams.BoundedString(
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

func (t *PayEventHoldTool) plan(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*driversettlementservice.PayEventPlan, error) {
	req, err := t.request(params)
	if err != nil {
		return nil, err
	}

	return t.events.PlanPayEventHold(ctx, req.tenant, req.eventID, req.reason, t.hold)
}

func (t *PayEventHoldTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}

	return plan.Refusal
}

func (t *PayEventHoldTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
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
		Version:  changepreview.PinnedVersion(event.Version),
	}
}

func payEventDriver(event *driversettlement.PayEvent) string {
	if event.Worker != nil {
		return event.Worker.FullName()
	}

	return "the driver"
}

func (t *PayEventHoldTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
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
		return changepreview.WouldFail(toolpreview.Build(summary), plan.Refusal)
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
		actor *permissionports.RequestActor,
	) (*driversettlement.Settlement, error)
	DetachPayEvent(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		settlementID, eventID pulid.ID,
		actor *permissionports.RequestActor,
	) (*driversettlement.Settlement, error)
}

type PayEventTransferTool struct {
	events payEventTransferrer
	detach bool
}

func ProvideAttachPayEventsTool(s *driversettlementservice.Service) agenttoolports.AgentTool {
	return &PayEventTransferTool{events: s}
}

func ProvideDetachPayEventTool(s *driversettlementservice.Service) agenttoolports.AgentTool {
	return &PayEventTransferTool{events: s, detach: true}
}

func (t *PayEventTransferTool) Name() string {
	if t.detach {
		return "settlement_detach_pay_event"
	}

	return "settlement_attach_pay_events"
}

func (t *PayEventTransferTool) Description() string {
	if t.detach {
		return "Take one load's pay off a draft driver settlement and return it to the " +
			"unsettled pool, such as pay that belongs to the next period or is disputed. " +
			"Its earning lines leave the settlement and the totals are recomputed."
	}

	return "Add a driver's accrued pay events to their draft driver settlement, such as a " +
		"late load that missed generation. Each event's earning lines join the settlement, " +
		"a held event is released as it is added, and the totals are recomputed."
}

func (t *PayEventTransferTool) ParamSchema() map[string]any {
	properties := map[string]any{
		settlementpreview.ParamSettlementID: agenttoolschema.RecordIDText(
			permission.ResourceDriverSettlement,
			"The draft driver settlement, from "+settlementpreview.DriverSettlementReads+". Never guess one.",
		),
	}
	var required []string
	if t.detach {
		properties[paramPayEventID] = agenttoolschema.KindID(
			"The pay event on it, from the lines settlement_get_driver lists.",
			permission.KindPayEvent,
		)
		required = []string{settlementpreview.ParamSettlementID, paramPayEventID}
	} else {
		properties[ParamPayEventIDs] = agenttoolschema.OfKinds(map[string]any{
			toolschema.KeyType:     toolschema.TypeArray,
			toolschema.KeyMinItems: 1,
			toolschema.KeyMaxItems: maxPayEventsPerAttach,
			toolschema.KeyItems:    map[string]any{toolschema.KeyType: toolschema.TypeString},
			toolschema.KeyDescription: "The driver's accrued pay events to add, from " +
				payEventSources + " with status Accrued.",
		}, permission.KindPayEvent)
		required = []string{settlementpreview.ParamSettlementID, ParamPayEventIDs}
	}

	return map[string]any{
		toolschema.KeyType:                 toolschema.TypeObject,
		toolschema.KeyProperties:           properties,
		toolschema.KeyRequired:             required,
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *PayEventTransferTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
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

func (t *PayEventTransferTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(
		params,
		settlementpreview.ParamSettlementID,
		permission.ResourceDriverSettlement,
	)
}

func (t *PayEventTransferTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*driversettlementservice.EventTransferRequest, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}
	settlementID, err := toolparams.RequirePulid(params.Params, settlementpreview.ParamSettlementID)
	if err != nil {
		return nil, err
	}
	req := &driversettlementservice.EventTransferRequest{
		TenantInfo:   toolparams.TenantFrom(*params),
		SettlementID: settlementID,
		Detach:       t.detach,
	}
	if t.detach {
		eventID, idErr := toolparams.RequirePulid(params.Params, paramPayEventID)
		if idErr != nil {
			return nil, idErr
		}
		req.EventIDs = []pulid.ID{eventID}

		return req, nil
	}
	if req.EventIDs, err = toolparams.RequirePulidSlice(
		params.Params,
		ParamPayEventIDs,
		maxPayEventsPerAttach,
	); err != nil {
		return nil, err
	}

	return req, nil
}

func (t *PayEventTransferTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
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

func (t *PayEventTransferTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
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

func (t *PayEventTransferTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	req, err := t.request(&params)
	if err != nil {
		return nil, err
	}
	plan, err := t.events.PlanEventTransfer(ctx, req)
	if err != nil {
		return nil, err
	}

	before := settlementpreview.DriverSettlementFacts(plan.Before)
	summary := fmt.Sprintf(
		"Would add %s to driver settlement %s%s.",
		changepreview.CountOf(len(req.EventIDs), "pay event"),
		before.Number,
		settlementpreview.PayeeClause(before.Payee),
	)
	if t.detach {
		summary = fmt.Sprintf(
			"Would take a pay event off driver settlement %s%s and return it to the "+
				"unsettled pool.",
			before.Number,
			settlementpreview.PayeeClause(before.Payee),
		)
	}
	if plan.Refusal != nil {
		return changepreview.WouldFail(toolpreview.Build(summary), plan.Refusal)
	}

	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceDriverSettlement,
		ID:       before.ID,
		Label:    before.Number,
		Version:  changepreview.PinnedVersion(before.Version),
	}, plan.Before, plan.After, toolpreview.Only("shipmentCount"))
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(
		change,
		settlementpreview.SettlementMoney(
			before,
			settlementpreview.DriverSettlementFacts(plan.After),
		),
		toolpreview.SensitiveAs(settlementpreview.DriverSettlementLedger().Sensitive...),
	)

	return toolpreview.Build(summary, change), nil
}
