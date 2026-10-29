package shipmentedittools

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentcoverage"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const paramWindowStart = "windowStart"

const paramWindowEnd = "windowEnd"

const paramUnits = "units"

const maxChargeUnits = 10000

var stopScheduleLabels = map[string]string{
	paramWindowStart: "Window opens",
	paramWindowEnd:   "Window closes",
}

type rescheduleStopTool struct {
	reads    shipmentpreview.ShipmentReader
	writes   shipmentpreview.ShipmentWriter
	partners shipmentpreview.ShipmentPartnerNotices
}

func NewRescheduleStopTool(
	reads shipmentpreview.ShipmentReader,
	writes shipmentpreview.ShipmentWriter,
	partners shipmentpreview.ShipmentPartnerNotices,
) agenttoolports.AgentTool {
	return &rescheduleStopTool{reads: reads, writes: writes, partners: partners}
}

func (t *rescheduleStopTool) Name() string { return "shipment_reschedule_stop" }

func (t *rescheduleStopTool) SearchTerms() []string {
	return []string{
		"move the appointment", "change the pickup time", "new appointment", "stop window",
	}
}

func (t *rescheduleStopTool) Prerequisites() []string {
	return []string{shipmenttools.ToolShipmentList, "shipment_get"}
}

func (t *rescheduleStopTool) Recipe() []string {
	return []string{shipmenttools.ToolShipmentList, "shipment_get", "shipment_reschedule_stop"}
}

func (t *rescheduleStopTool) Description() string {
	return "Move a saved shipment's pickup or delivery appointment: the stop's planned " +
		"window, local to the stop. Give the new window start; the window end moves with " +
		"it, keeping its length, unless you give one. Take the stop id from shipment_get."
}

func (t *rescheduleStopTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			toolparams.ParamShipmentID: agenttoolschema.RecordIDText(permission.ResourceShipment,
				"The shipment the stop is on, from shipment_list."),
			"stopId": agenttoolschema.RecordIDText(permission.ResourceShipmentStop,
				"The stop to reschedule, from the moves' stops in shipment_get."),
			paramWindowStart: agenttoolschema.LocalDateTime(
				"When the stop's window now opens, in the stop location's local time. This is " +
					"the plan; an actual arrival or departure goes to dispatch_record_stop_actual."),
			paramWindowEnd: agenttoolschema.LocalDateTime(
				"When it now closes, in the same local time. Leave it out to keep the " +
					"window's length."),
		},
		"required":             []string{toolparams.ParamShipmentID, "stopId", paramWindowStart},
		"additionalProperties": false,
	}
}

func (t *rescheduleStopTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceShipment,
		Operation:     permission.OpUpdate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale: "A changed appointment is sent as an EDI tender change to the trading " +
			"partners the load was tendered to.",
	}
}

type stopSchedulePlan struct {
	before *shipment.Shipment
	after  *shipment.Shipment
	stop   *shipment.Stop
	zone   *time.Location
	was    stopScheduleView
}

type stopScheduleView struct {
	WindowStart string `json:"windowStart"`
	WindowEnd   string `json:"windowEnd,omitempty"`
}

func (t *rescheduleStopTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}

	_, err = t.writes.Update(ctx, plan.after, params.Actor)

	return err
}

func (t *rescheduleStopTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return toolspec.PreviewValidates(ctx, t, &params)
}

func (t *rescheduleStopTool) plan(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*stopSchedulePlan, error) {
	if err := toolguard.GuardExecute(t, *params); err != nil {
		return nil, err
	}

	shipmentID, err := toolparams.RequirePulid(params.Params, toolparams.ParamShipmentID)
	if err != nil {
		return nil, err
	}
	stopID, err := toolparams.RequirePulid(params.Params, "stopId")
	if err != nil {
		return nil, err
	}

	before, err := t.reads.Get(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:              shipmentID,
		TenantInfo:      toolparams.TenantFrom(*params),
		ShipmentOptions: shipmentports.ShipmentOptions{ExpandShipmentDetails: true},
	})
	if err != nil {
		return nil, err
	}

	after := shipmentcoverage.CloneForUpdate(before)
	stop := stopOnShipment(after, stopID)
	if stop == nil {
		return nil, errortypes.NewValidationError("stopId", errortypes.ErrInvalid,
			"That stop is not on this shipment; take the stop id from shipment_get's moves")
	}
	if stop.Status == shipment.StopStatusCompleted || stop.Status == shipment.StopStatusCanceled {
		return nil, errortypes.NewBusinessError(
			"This stop is {0} and can no longer be rescheduled", string(stop.Status),
		)
	}

	stopZone := ""
	if stop.Location != nil {
		stopZone = stop.Location.Timezone
	}
	zone, _ := timeutils.ResolveZone(stopZone, params.Timezone)

	plan := &stopSchedulePlan{
		before: before,
		after:  after,
		stop:   stop,
		zone:   zone,
		was:    scheduleViewOf(stop, zone),
	}

	start, err := toolparams.RequireLocalTime(params.Params, paramWindowStart, zone)
	if err != nil {
		return nil, err
	}
	end, err := toolparams.OptionalLocalTime(params.Params, paramWindowEnd, zone)
	if err != nil {
		return nil, err
	}
	if end == nil && stop.ScheduledWindowEnd != nil && *stop.ScheduledWindowEnd > 0 {
		shifted := start + (*stop.ScheduledWindowEnd - stop.ScheduledWindowStart)
		end = &shifted
	}
	if end != nil && *end < start {
		return nil, errortypes.NewValidationError(paramWindowEnd, errortypes.ErrInvalid,
			"The window cannot close before it opens")
	}

	stop.ScheduledWindowStart = start
	stop.ScheduledWindowEnd = end

	return plan, nil
}

func stopOnShipment(entity *shipment.Shipment, stopID pulid.ID) *shipment.Stop {
	for _, move := range entity.Moves {
		if move == nil {
			continue
		}
		if stop := shipmentpreview.FindStop(move, stopID); stop != nil {
			return stop
		}
	}

	return nil
}

func scheduleViewOf(stop *shipment.Stop, zone *time.Location) stopScheduleView {
	view := stopScheduleView{
		WindowStart: timeutils.FormatLocalDateTime(stop.ScheduledWindowStart, zone),
	}
	if stop.ScheduledWindowEnd != nil && *stop.ScheduledWindowEnd > 0 {
		view.WindowEnd = timeutils.FormatLocalDateTime(*stop.ScheduledWindowEnd, zone)
	}

	return view
}

func (t *rescheduleStopTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		if changepreview.IsRefusal(err) {
			return changepreview.WarnWouldFail(
				toolpreview.Build("Would reschedule the stop."),
				err,
			), nil
		}

		return nil, err
	}

	now := scheduleViewOf(plan.stop, plan.zone)
	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceShipment,
		ID:       plan.before.ID,
		Label:    plan.before.ProNumber + " · " + shipmentpreview.StopLabel(plan.stop),
		Version:  changepreview.PinnedVersion(plan.before.Version),
	}, &plan.was, &now, toolpreview.Labels(stopScheduleLabels))
	if err != nil {
		return nil, err
	}

	changes, err := shipmentpreview.TenderChangeNotices(
		ctx,
		t.partners,
		plan.before,
		plan.after,
		params.Actor,
	)
	if err != nil {
		return nil, err
	}

	_, zoneName := timeutils.ResolveZone(plan.zone.String())
	summary := fmt.Sprintf("Would move the %s on shipment %s to open %s (%s).",
		shipmentpreview.StopLabel(plan.stop), plan.before.ProNumber, now.WindowStart, zoneName)

	return toolpreview.Build(
		summary,
		append([]*agentpreview.RecordChange{change}, changes...)...), nil
}

func (t *rescheduleStopTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, toolparams.ParamShipmentID, permission.ResourceShipment)
}

type addShipmentChargeTool struct {
	reads        shipmentpreview.ShipmentReader
	writes       shipmentpreview.ShipmentWriter
	accessorials accessorialchargeports.AccessorialChargeRepository
}

func NewAddShipmentChargeTool(
	reads shipmentpreview.ShipmentReader,
	writes shipmentpreview.ShipmentWriter,
	accessorials accessorialchargeports.AccessorialChargeRepository,
) agenttoolports.AgentTool {
	return &addShipmentChargeTool{reads: reads, writes: writes, accessorials: accessorials}
}

func (t *addShipmentChargeTool) Name() string { return "shipment_add_charge" }

func (t *addShipmentChargeTool) SearchTerms() []string {
	return []string{
		"lumper", "add a charge", "accessorial", "layover", "tarp fee",
		"bill an extra charge",
	}
}

func (t *addShipmentChargeTool) Prerequisites() []string {
	return []string{shipmenttools.ToolShipmentList, "masterdata_list_accessorial_charges"}
}

func (t *addShipmentChargeTool) Recipe() []string {
	return []string{
		shipmenttools.ToolShipmentList,
		"masterdata_list_accessorial_charges",
		"shipment_add_charge",
	}
}

func (t *addShipmentChargeTool) Description() string {
	return "Add an accessorial charge to a saved shipment not yet invoiced, at the charge's " +
		"own rate unless you give an amount. It takes a lumper, layover, tarp, extra stop or " +
		"any charge from masterdata_list_accessorial_charges. An invoiced shipment is corrected with a " +
		"credit memo."
}

func (t *addShipmentChargeTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			toolparams.ParamShipmentID: agenttoolschema.RecordIDText(permission.ResourceShipment,
				"The shipment to charge, from shipment_list."),
			"accessorialChargeId": agenttoolschema.RecordIDText(
				permission.ResourceAccessorialCharge,
				"The charge, from masterdata_list_accessorial_charges. Not detention: a detention "+
					"charge is approved with detention_approve."),
			paramUnits: map[string]any{
				"type": "integer",
				"description": "How many units: hours, stops, days or pieces for a per-unit " +
					"charge. 1 for a flat charge. Defaults to 1.",
				"minimum": 1,
				"maximum": maxChargeUnits,
			},
			toolparams.ParamAmount: toolparams.StringProperty(
				"The rate per unit, or the flat amount, as a decimal such as 150.00, when the "+
					"person named one; leave it out to use the charge's configured rate.", 0),
		},
		"required":             []string{toolparams.ParamShipmentID, "accessorialChargeId"},
		"additionalProperties": false,
	}
}

func (t *addShipmentChargeTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceShipment,
		Operation:     permission.OpUpdate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressMoney},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale:     "Adds to what the customer is billed for the shipment.",
	}
}

type shipmentChargePlan struct {
	before      *shipment.Shipment
	after       *shipment.Shipment
	accessorial *accessorialcharge.AccessorialCharge
	charge      *shipment.AdditionalCharge
}

func (t *addShipmentChargeTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}

	_, err = t.writes.Update(ctx, plan.after, params.Actor)

	return err
}

func (t *addShipmentChargeTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return toolspec.PreviewValidates(ctx, t, &params)
}

func (t *addShipmentChargeTool) plan(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*shipmentChargePlan, error) {
	if err := toolguard.GuardExecute(t, *params); err != nil {
		return nil, err
	}

	shipmentID, err := toolparams.RequirePulid(params.Params, toolparams.ParamShipmentID)
	if err != nil {
		return nil, err
	}
	chargeID, err := toolparams.RequirePulid(params.Params, "accessorialChargeId")
	if err != nil {
		return nil, err
	}

	units := 1
	if given, unitErr := toolparams.OptionalIntInRange(
		params.Params,
		paramUnits,
		1,
		maxChargeUnits,
	); unitErr != nil {
		return nil, unitErr
	} else if given != nil {
		units = *given
	}

	tenant := toolparams.TenantFrom(*params)
	accessorial, err := t.accessorials.GetByID(
		ctx,
		accessorialchargeports.GetAccessorialChargeByIDRequest{
			ID:         chargeID,
			TenantInfo: &tenant,
		},
	)
	if err != nil {
		return nil, err
	}
	if accessorial.Status != domaintypes.StatusActive {
		return nil, errortypes.NewValidationError(
			"accessorialChargeId",
			errortypes.ErrInvalid,
			"That accessorial charge is inactive; choose an active one from masterdata_list_accessorial_charges",
		)
	}

	amount := accessorial.Amount
	if given, ok, amountErr := toolparams.OptionalDecimal(
		params.Params,
		toolparams.ParamAmount,
	); amountErr != nil {
		return nil, amountErr
	} else if ok {
		amount = given
	}
	if amount.IsNegative() {
		return nil, errortypes.NewValidationError(toolparams.ParamAmount, errortypes.ErrInvalid,
			"A charge cannot be negative; a credit is made with a credit memo")
	}

	before, err := t.reads.Get(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:              shipmentID,
		TenantInfo:      tenant,
		ShipmentOptions: shipmentports.ShipmentOptions{ExpandShipmentDetails: true},
	})
	if err != nil {
		return nil, err
	}
	if before.Status == shipment.StatusInvoiced || before.Status == shipment.StatusCanceled {
		return nil, errortypes.NewBusinessError(
			"Shipment {0} is {1}; its charges can no longer change here",
			before.ProNumber, string(before.Status),
		)
	}

	charge := &shipment.AdditionalCharge{
		OrganizationID:      before.OrganizationID,
		BusinessUnitID:      before.BusinessUnitID,
		ShipmentID:          before.ID,
		AccessorialChargeID: accessorial.ID,
		Method:              accessorial.Method,
		Amount:              amount,
		Unit:                int16(units),
		AccessorialCharge:   accessorial,
	}

	after := shipmentcoverage.CloneForUpdate(before)
	after.AdditionalCharges = append(slices.Clone(before.AdditionalCharges), charge)

	return &shipmentChargePlan{
		before:      before,
		after:       after,
		accessorial: accessorial,
		charge:      charge,
	}, nil
}

type shipmentChargeView struct {
	Charge string `json:"charge"`
	Method string `json:"method"`
	Rate   string `json:"rate"`
	Units  int16  `json:"units"`
	Total  string `json:"total,omitempty"`
}

func shipmentChargeViewOf(plan *shipmentChargePlan) *shipmentChargeView {
	view := &shipmentChargeView{
		Charge: plan.accessorial.Code + " " + plan.accessorial.Description,
		Method: string(plan.charge.Method),
		Rate:   plan.charge.Amount.StringFixed(2),
		Units:  plan.charge.Unit,
	}
	switch plan.charge.Method {
	case accessorialcharge.MethodFlat:
		view.Total = plan.charge.Amount.StringFixed(2)
	case accessorialcharge.MethodPerUnit:
		view.Total = plan.charge.Amount.Mul(decimal.NewFromInt(int64(plan.charge.Unit))).
			StringFixed(2)
	case accessorialcharge.MethodPercentage:
		view.Rate += "%"
	}

	return view
}

func (t *addShipmentChargeTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		if changepreview.IsRefusal(err) {
			return changepreview.WarnWouldFail(
				toolpreview.Build("Would add a charge to the shipment."),
				err,
			), nil
		}

		return nil, err
	}

	view := shipmentChargeViewOf(plan)
	change, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceShipment,
		ID:       plan.before.ID,
		Label:    plan.before.ProNumber + " · " + view.Charge,
		Version:  changepreview.PinnedVersion(plan.before.Version),
	}, view)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf("Would add %s to shipment %s", view.Charge, plan.before.ProNumber)
	if view.Total != "" {
		summary += " for " + view.Total
	}

	return toolpreview.Build(summary+".", change), nil
}

func (t *addShipmentChargeTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, toolparams.ParamShipmentID, permission.ResourceShipment)
}
