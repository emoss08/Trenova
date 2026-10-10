package agenttoolservice

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const (
	paramWindowStart = "windowStart"
	paramWindowEnd   = "windowEnd"
	paramUnits       = "units"
	maxChargeUnits   = 10000
)

var stopScheduleLabels = map[string]string{
	paramWindowStart: "Window opens",
	paramWindowEnd:   "Window closes",
}

type rescheduleStopTool struct {
	shipments shipmentWriter
	partners  shipmentPartnerNotices
}

func newRescheduleStopTool(
	shipments shipmentWriter,
	partners shipmentPartnerNotices,
) serviceports.AgentTool {
	return &rescheduleStopTool{shipments: shipments, partners: partners}
}

func (t *rescheduleStopTool) Name() string { return "reschedule_stop" }

func (t *rescheduleStopTool) SearchTerms() []string {
	return []string{
		"move the appointment", "change the pickup time", "new appointment", "stop window",
	}
}

func (t *rescheduleStopTool) Prerequisites() []string {
	return []string{"search_shipments", "get_shipment"}
}

func (t *rescheduleStopTool) Recipe() []string {
	return []string{"search_shipments", "get_shipment", "reschedule_stop"}
}

func (t *rescheduleStopTool) Description() string {
	return "Move a saved shipment's pickup or delivery appointment: the stop's planned " +
		"window, local to the stop. Give the new window start; the window end moves with " +
		"it, keeping its length, unless you give one. Take the stop id from get_shipment."
}

func (t *rescheduleStopTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentId": agenttoolschema.RecordIDText(permission.ResourceShipment,
				"The shipment the stop is on, from search_shipments or list_shipments."),
			"stopId": agenttoolschema.RecordIDText(permission.ResourceShipmentStop,
				"The stop to reschedule, from the moves' stops in get_shipment."),
			paramWindowStart: agenttoolschema.LocalDateTime(
				"When the stop's window now opens, in the stop location's local time. This is " +
					"the plan; an actual arrival or departure goes to record_stop_actual."),
			paramWindowEnd: agenttoolschema.LocalDateTime(
				"When it now closes, in the same local time. Leave it out to keep the " +
					"window's length."),
		},
		"required":             []string{"shipmentId", "stopId", paramWindowStart},
		"additionalProperties": false,
	}
}

func (t *rescheduleStopTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceShipment,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
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

func (t *rescheduleStopTool) Execute(ctx context.Context, params serviceports.ToolExecuteParams) error {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}

	_, err = t.shipments.Update(ctx, plan.after, params.Actor)

	return err
}

func (t *rescheduleStopTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return previewValidates(ctx, t, &params)
}

func (t *rescheduleStopTool) plan(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*stopSchedulePlan, error) {
	if err := guardExecute(t, *params); err != nil {
		return nil, err
	}

	shipmentID, err := requirePulid(params.Params, "shipmentId")
	if err != nil {
		return nil, err
	}
	stopID, err := requirePulid(params.Params, "stopId")
	if err != nil {
		return nil, err
	}

	before, err := t.shipments.Get(ctx, &repositories.GetShipmentByIDRequest{
		ID:              shipmentID,
		TenantInfo:      tenantFrom(*params),
		ShipmentOptions: repositories.ShipmentOptions{ExpandShipmentDetails: true},
	})
	if err != nil {
		return nil, err
	}

	after := shipment.CloneForUpdate(before)
	stop := stopOnShipment(after, stopID)
	if stop == nil {
		return nil, errortypes.NewValidationError("stopId", errortypes.ErrInvalid,
			"That stop is not on this shipment; take the stop id from get_shipment's moves")
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

	start, err := requireLocalTime(params.Params, paramWindowStart, zone)
	if err != nil {
		return nil, err
	}
	end, err := optionalLocalTime(params.Params, paramWindowEnd, zone)
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
		if stop := findStop(move, stopID); stop != nil {
			return stop
		}
	}

	return nil
}

func scheduleViewOf(stop *shipment.Stop, zone *time.Location) stopScheduleView {
	view := stopScheduleView{WindowStart: timeutils.FormatLocalDateTime(stop.ScheduledWindowStart, zone)}
	if stop.ScheduledWindowEnd != nil && *stop.ScheduledWindowEnd > 0 {
		view.WindowEnd = timeutils.FormatLocalDateTime(*stop.ScheduledWindowEnd, zone)
	}

	return view
}

func (t *rescheduleStopTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		if isRefusal(err) {
			return warnWouldFail(toolpreview.Build("Would reschedule the stop."), err), nil
		}

		return nil, err
	}

	now := scheduleViewOf(plan.stop, plan.zone)
	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceShipment,
		ID:       plan.before.ID,
		Label:    plan.before.ProNumber + " · " + stopLabel(plan.stop),
		Version:  pinnedVersion(plan.before.Version),
	}, &plan.was, &now, toolpreview.Labels(stopScheduleLabels))
	if err != nil {
		return nil, err
	}

	changes, err := tenderChangeNotices(ctx, t.partners, plan.before, plan.after, params.Actor)
	if err != nil {
		return nil, err
	}

	_, zoneName := timeutils.ResolveZone(plan.zone.String())
	summary := fmt.Sprintf("Would move the %s on shipment %s to open %s (%s).",
		stopLabel(plan.stop), plan.before.ProNumber, now.WindowStart, zoneName)

	return toolpreview.Build(summary, append([]*agent.RecordChange{change}, changes...)...), nil
}

func (t *rescheduleStopTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "shipmentId", permission.ResourceShipment)
}

type addShipmentChargeTool struct {
	shipments    shipmentWriter
	accessorials repositories.AccessorialChargeRepository
}

func newAddShipmentChargeTool(
	shipments shipmentWriter,
	accessorials repositories.AccessorialChargeRepository,
) serviceports.AgentTool {
	return &addShipmentChargeTool{shipments: shipments, accessorials: accessorials}
}

func (t *addShipmentChargeTool) Name() string { return "add_shipment_charge" }

func (t *addShipmentChargeTool) SearchTerms() []string {
	return []string{
		"lumper", "add a charge", "accessorial", "layover", "tarp fee",
		"bill an extra charge",
	}
}

func (t *addShipmentChargeTool) Prerequisites() []string {
	return []string{"search_shipments", "list_accessorial_charges"}
}

func (t *addShipmentChargeTool) Recipe() []string {
	return []string{"search_shipments", "list_accessorial_charges", "add_shipment_charge"}
}

func (t *addShipmentChargeTool) Description() string {
	return "Add an accessorial charge to a saved shipment not yet invoiced, at the charge's " +
		"own rate unless you give an amount. It takes a lumper, layover, tarp, extra stop or " +
		"any charge from list_accessorial_charges. An invoiced shipment is corrected with a " +
		"credit memo."
}

func (t *addShipmentChargeTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentId": agenttoolschema.RecordIDText(permission.ResourceShipment,
				"The shipment to charge, from search_shipments or list_shipments."),
			"accessorialChargeId": agenttoolschema.RecordIDText(
				permission.ResourceAccessorialCharge,
				"The charge, from list_accessorial_charges. Not detention: a detention "+
					"charge is approved with approve_detention."),
			paramUnits: map[string]any{
				"type": "integer",
				"description": "How many units: hours, stops, days or pieces for a per-unit " +
					"charge. 1 for a flat charge. Defaults to 1.",
				"minimum": 1,
				"maximum": maxChargeUnits,
			},
			paramAmount: stringProperty(
				"The rate per unit, or the flat amount, as a decimal such as 150.00, when the "+
					"person named one; leave it out to use the charge's configured rate.", 0),
		},
		"required":             []string{"shipmentId", "accessorialChargeId"},
		"additionalProperties": false,
	}
}

func (t *addShipmentChargeTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceShipment,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressMoney},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "Adds to what the customer is billed for the shipment.",
	}
}

type shipmentChargePlan struct {
	before      *shipment.Shipment
	after       *shipment.Shipment
	accessorial *accessorialcharge.AccessorialCharge
	charge      *shipment.AdditionalCharge
}

func (t *addShipmentChargeTool) Execute(ctx context.Context, params serviceports.ToolExecuteParams) error {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}

	_, err = t.shipments.Update(ctx, plan.after, params.Actor)

	return err
}

func (t *addShipmentChargeTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return previewValidates(ctx, t, &params)
}

func (t *addShipmentChargeTool) plan(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*shipmentChargePlan, error) {
	if err := guardExecute(t, *params); err != nil {
		return nil, err
	}

	shipmentID, err := requirePulid(params.Params, "shipmentId")
	if err != nil {
		return nil, err
	}
	chargeID, err := requirePulid(params.Params, "accessorialChargeId")
	if err != nil {
		return nil, err
	}

	units := 1
	if given, unitErr := optionalIntInRange(params.Params, paramUnits, 1, maxChargeUnits); unitErr != nil {
		return nil, unitErr
	} else if given != nil {
		units = *given
	}

	tenant := tenantFrom(*params)
	accessorial, err := t.accessorials.GetByID(ctx, repositories.GetAccessorialChargeByIDRequest{
		ID:         chargeID,
		TenantInfo: &tenant,
	})
	if err != nil {
		return nil, err
	}
	if accessorial.Status != domaintypes.StatusActive {
		return nil, errortypes.NewValidationError("accessorialChargeId", errortypes.ErrInvalid,
			"That accessorial charge is inactive; choose an active one from list_accessorial_charges")
	}

	amount := accessorial.Amount
	if given, ok, amountErr := optionalDecimal(params.Params, paramAmount); amountErr != nil {
		return nil, amountErr
	} else if ok {
		amount = given
	}
	if amount.IsNegative() {
		return nil, errortypes.NewValidationError(paramAmount, errortypes.ErrInvalid,
			"A charge cannot be negative; a credit is made with a credit memo")
	}

	before, err := t.shipments.Get(ctx, &repositories.GetShipmentByIDRequest{
		ID:              shipmentID,
		TenantInfo:      tenant,
		ShipmentOptions: repositories.ShipmentOptions{ExpandShipmentDetails: true},
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

	after := shipment.CloneForUpdate(before)
	after.AdditionalCharges = append(slices.Clone(before.AdditionalCharges), charge)

	return &shipmentChargePlan{before: before, after: after, accessorial: accessorial, charge: charge}, nil
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
		view.Total = plan.charge.Amount.Mul(decimal.NewFromInt(int64(plan.charge.Unit))).StringFixed(2)
	case accessorialcharge.MethodPercentage:
		view.Rate += "%"
	}

	return view
}

func (t *addShipmentChargeTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	plan, err := t.plan(ctx, &params)
	if err != nil {
		if isRefusal(err) {
			return warnWouldFail(toolpreview.Build("Would add a charge to the shipment."), err), nil
		}

		return nil, err
	}

	view := shipmentChargeViewOf(plan)
	change, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceShipment,
		ID:       plan.before.ID,
		Label:    plan.before.ProNumber + " · " + view.Charge,
		Version:  pinnedVersion(plan.before.Version),
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

func (t *addShipmentChargeTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "shipmentId", permission.ResourceShipment)
}

func tenderChangeNotices(
	ctx context.Context,
	partners shipmentPartnerNotices,
	before, after *shipment.Shipment,
	actor *serviceports.RequestActor,
) ([]*agent.RecordChange, error) {
	if partners == nil {
		return nil, nil
	}

	notices, err := partners.PreviewTenderChanges(ctx, before, after, actor)
	if err != nil {
		return nil, err
	}

	changes := make([]*agent.RecordChange, 0, len(notices))
	for _, notice := range notices {
		changes = append(changes, partnerNoticeSend(notice, ""))
	}

	return changes, nil
}
