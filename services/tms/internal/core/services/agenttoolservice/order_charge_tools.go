package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/orderservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type orderChargeKeeper interface {
	ListCharges(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
	) ([]*order.OrderCharge, error)
	AddChargeWithAllocations(
		ctx context.Context,
		req *orderservice.AddChargeRequest,
		actor *serviceports.RequestActor,
	) (*order.Order, error)
	PlanAddCharge(
		ctx context.Context,
		req *orderservice.AddChargeRequest,
	) (*orderservice.ChargePlan, error)
	UpdateCharge(
		ctx context.Context,
		req *orderservice.UpdateChargeRequest,
		actor *serviceports.RequestActor,
	) (*order.Order, error)
	PlanUpdateCharge(
		ctx context.Context,
		req *orderservice.UpdateChargeRequest,
	) (*orderservice.ChargePlan, error)
	SetChargeAllocations(
		ctx context.Context,
		req *orderservice.SetChargeAllocationsRequest,
		actor *serviceports.RequestActor,
	) (*order.Order, error)
	PlanSetChargeAllocations(
		ctx context.Context,
		req *orderservice.SetChargeAllocationsRequest,
	) (*orderservice.ChargePlan, error)
	RemoveCharge(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
		chargeID pulid.ID,
		actor *serviceports.RequestActor,
	) (*order.Order, error)
	PlanRemoveCharge(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
		chargeID pulid.ID,
	) (*orderservice.ChargePlan, error)
}

func allocationsProperty(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeArray,
		toolschema.KeyDescription: description,
		toolschema.KeyMaxItems:    maxChargeAllocations,
		toolschema.KeyItems: map[string]any{
			toolschema.KeyType: toolschema.TypeObject,
			toolschema.KeyProperties: map[string]any{
				paramBillToCustomer: idProperty("The customer who pays this share, from " +
					"list_customers. Never guess one."),
				paramAllocMethod: agenttoolschema.Enum("Whether the share is a percent of "+
					"the charge or a fixed amount.", allocationMethods),
				paramAllocPercent: amountProperty("The share as a percent such as 60, " +
					"when method is Percent."),
				paramAmount: amountProperty("The share as an amount such as 250.00, when " +
					"method is Amount."),
			},
			toolschema.KeyRequired:             []string{paramBillToCustomer, paramAllocMethod},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}

type allocationArg struct {
	BillToCustomerID string `json:"billToCustomerId"`
	Method           string `json:"method"`
	Percent          any    `json:"percent"`
	Amount           any    `json:"amount"`
}

// readAllocations reads a charge's payer split. Absent is nil, which leaves the
// split alone; an empty list bills the whole charge to the order's customer.
// A payer already on the charge keeps its row, so the split is edited rather
// than replaced.
func readAllocations(
	params map[string]any,
	tenant pagination.TenantInfo,
	kind shipment.ChargeAllocationKind,
	current []*shipment.ChargeAllocation,
) ([]*shipment.ChargeAllocation, error) {
	if _, given := params[paramAllocations]; !given {
		return nil, nil
	}

	var args []allocationArg
	if err := decodeParam(params, paramAllocations, &args); err != nil {
		return nil, err
	}
	if len(args) > maxChargeAllocations {
		return nil, fmt.Errorf("parameter %q holds %d payers; at most %d split one charge",
			paramAllocations, len(args), maxChargeAllocations)
	}

	existing := make(map[pulid.ID]*shipment.ChargeAllocation, len(current))
	for _, row := range current {
		if row != nil {
			existing[row.BillToCustomerID] = row
		}
	}

	rows := make([]*shipment.ChargeAllocation, 0, len(args))
	for index, arg := range args {
		row, err := allocationRow(index, arg, tenant, kind)
		if err != nil {
			return nil, err
		}
		if kept, ok := existing[row.BillToCustomerID]; ok {
			row.ID = kept.ID
			row.Version = kept.Version
		}
		rows = append(rows, row)
	}

	return rows, nil
}

func allocationRow(
	index int,
	arg allocationArg,
	tenant pagination.TenantInfo,
	kind shipment.ChargeAllocationKind,
) (*shipment.ChargeAllocation, error) {
	path := fmt.Sprintf("%s[%d]", paramAllocations, index)
	payer, err := pulid.Parse(strings.TrimSpace(arg.BillToCustomerID))
	if err != nil || payer.IsNil() {
		return nil, fmt.Errorf("%s.%s must be a customer id", path, paramBillToCustomer)
	}
	method, err := requireEnum(map[string]any{paramAllocMethod: arg.Method}, paramAllocMethod,
		allocationMethods.Values)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	row := &shipment.ChargeAllocation{
		OrganizationID:   tenant.OrgID,
		BusinessUnitID:   tenant.BuID,
		ChargeKind:       kind,
		BillToCustomerID: payer,
		Method:           method,
		Sequence:         int16(index), //nolint:gosec // bounded by maxChargeAllocations
	}
	share := map[string]any{paramAllocPercent: arg.Percent, paramAmount: arg.Amount}
	percent, hasPercent, err := optionalDecimal(share, paramAllocPercent)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	amount, hasAmount, err := optionalDecimal(share, paramAmount)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	switch method {
	case shipment.ChargeAllocationMethodPercent:
		if !hasPercent || hasAmount {
			return nil, fmt.Errorf("%s gives a percent and no amount when method is Percent",
				path)
		}
		row.Percent = decimal.NewNullDecimal(percent)
	case shipment.ChargeAllocationMethodAmount:
		if !hasAmount || hasPercent {
			return nil, fmt.Errorf("%s gives an amount and no percent when method is Amount",
				path)
		}
		row.Amount = decimal.NewNullDecimal(amount)
	}

	return row, nil
}

func chargeIDProperty() map[string]any {
	return idProperty("The order charge, from the charges get_order lists. Never guess one.")
}

type chargeView struct {
	Description string              `json:"description"`
	Amount      decimal.NullDecimal `json:"amount"`
	Split       string              `json:"split"`
}

func chargeViewOf(charge *order.OrderCharge) *chargeView {
	if charge == nil {
		return nil
	}

	return &chargeView{
		Description: charge.Description,
		Amount:      decimal.NewNullDecimal(charge.Amount),
		Split:       splitWords(charge.Allocations),
	}
}

func splitWords(rows []*shipment.ChargeAllocation) string {
	if len(rows) == 0 {
		return "Billed to the order's customer"
	}

	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		switch row.Method {
		case shipment.ChargeAllocationMethodPercent:
			parts = append(parts, row.Percent.Decimal.String()+"%")
		case shipment.ChargeAllocationMethodAmount:
			parts = append(parts, row.Amount.Decimal.StringFixed(2))
		}
	}

	return fmt.Sprintf("%s: %s", countOf(len(parts), "payer"), strings.Join(parts, " / "))
}

func chargeLabel(ord *order.Order, charge *order.OrderCharge) string {
	return fmt.Sprintf("Order %s charge %q", ord.OrderNumber, charge.Description)
}

func renderChargePlan(verb string) func(any, *orderservice.ChargePlan) (*agent.ToolPreview, error) {
	return func(_ any, plan *orderservice.ChargePlan) (*agent.ToolPreview, error) {
		shown := plan.After
		if shown == nil {
			shown = plan.Before
		}
		rec := toolpreview.Record{
			Resource: permission.ResourceOrder,
			ID:       plan.Order.ID,
			Label:    chargeLabel(plan.Order, shown),
			Version:  pinnedVersion(plan.Order.Version),
		}

		var (
			change *agent.RecordChange
			err    error
		)
		switch {
		case plan.Before == nil:
			change, err = toolpreview.Create(rec, chargeViewOf(plan.After))
		case plan.After == nil:
			change, err = toolpreview.Delete(rec, chargeViewOf(plan.Before))
		default:
			change, err = toolpreview.Changed(rec, chargeViewOf(plan.Before),
				chargeViewOf(plan.After))
		}
		if err != nil {
			return nil, err
		}
		toolpreview.AttachMoney(change, toolpreview.MoneyBlock(plan.Order.CurrencyCode,
			agent.MoneyLine{
				Label:  shown.Description,
				Before: chargeAmount(plan.Before),
				After:  chargeAmount(plan.After),
			}), toolpreview.SensitiveAs("totalAmount"))

		return toolpreview.Build(fmt.Sprintf("Would %s a charge on order %s; the order's "+
			"total follows.", verb, plan.Order.OrderNumber), change), nil
	}
}

func chargeAmount(charge *order.OrderCharge) decimal.NullDecimal {
	if charge == nil {
		return decimal.NullDecimal{}
	}

	return decimal.NewNullDecimal(charge.Amount)
}

func chargeSpec(name, description, rationale string, properties map[string]any,
	required ...string,
) *receivableSpec {
	return &receivableSpec{
		name:        name,
		description: description,
		artifact:    orderRecordEntity,
		resource:    permission.ResourceOrder,
		operation:   permission.OpUpdate,
		egress:      agent.EgressMoney,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale:   rationale,
		properties:  properties,
		required:    required,
		target:      targetOrder,
	}
}

func newAddOrderChargeTool(charges orderChargeKeeper) serviceports.AgentTool {
	return newReportingReceivableTool(chargeSpec(
		"add_order_charge",
		"Add an order-level charge, one billed once for the whole order rather than on a "+
			"shipment: a consolidation fee, a quoted flat rate. Give allocations only to "+
			"split it among several payers; left out, the order's customer pays it all.",
		"Adds to what the customer is billed; nothing is invoiced or sent, and "+
			"remove_order_charge takes it off until it is invoiced.",
		map[string]any{
			paramOrderID: orderIDProperty("The order"),
			fieldDescription: stringProperty("What the charge is for, as the invoice "+
				"line should read.", maxOrderChargeText),
			paramAmount:      amountProperty("The amount, as a decimal such as 150.00."),
			paramAllocations: allocationsProperty("How the charge is split among payers."),
		},
		paramOrderID, fieldDescription, paramAmount,
	), receivablePlan[*orderservice.AddChargeRequest, *orderservice.ChargePlan]{
		request: func(params *serviceports.ToolExecuteParams) (*orderservice.AddChargeRequest, error) {
			orderID, err := requirePulid(params.Params, paramOrderID)
			if err != nil {
				return nil, err
			}
			description, err := requireBoundedText(params.Params, fieldDescription,
				maxOrderChargeText)
			if err != nil {
				return nil, err
			}
			amount, err := decimalArg(params.Params, paramAmount, true)
			if err != nil {
				return nil, err
			}
			tenant := tenantFrom(*params)
			allocations, err := readAllocations(
				params.Params,
				tenant,
				shipment.ChargeAllocationKindOrderCharge,
				nil,
			)
			if err != nil {
				return nil, err
			}

			return &orderservice.AddChargeRequest{
				TenantInfo:  tenant,
				OrderID:     orderID,
				Description: description,
				Amount:      amount,
				Allocations: allocations,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *orderservice.AddChargeRequest,
			_ *serviceports.ToolExecuteParams,
		) (*orderservice.ChargePlan, error) {
			return charges.PlanAddCharge(ctx, req)
		},
		refused: func(*orderservice.AddChargeRequest) string {
			return "Would add a charge to the order."
		},
		render: func(req *orderservice.AddChargeRequest, plan *orderservice.ChargePlan) (*agent.ToolPreview, error) {
			return renderChargePlan("add")(req, plan)
		},
		run: func(
			ctx context.Context,
			req *orderservice.AddChargeRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			updated, err := charges.AddChargeWithAllocations(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}

			return orderResult("charge added", updated), nil
		},
	})
}

type chargeRef struct {
	tenant   pagination.TenantInfo
	orderID  pulid.ID
	chargeID pulid.ID
	params   map[string]any
}

func readChargeRef(params *serviceports.ToolExecuteParams) (*chargeRef, error) {
	orderID, err := requirePulid(params.Params, paramOrderID)
	if err != nil {
		return nil, err
	}
	chargeID, err := requirePulid(params.Params, paramChargeID)
	if err != nil {
		return nil, err
	}

	return &chargeRef{
		tenant:   tenantFrom(*params),
		orderID:  orderID,
		chargeID: chargeID,
		params:   params.Params,
	}, nil
}

func (r *chargeRef) current(
	ctx context.Context,
	charges orderChargeKeeper,
) (*order.OrderCharge, error) {
	listed, err := charges.ListCharges(ctx, r.tenant, r.orderID)
	if err != nil {
		return nil, err
	}
	for _, charge := range listed {
		if charge != nil && charge.ID == r.chargeID {
			return charge, nil
		}
	}

	return nil, fmt.Errorf("order charge %s is not on that order", r.chargeID)
}

func (r *chargeRef) updateRequest(
	ctx context.Context,
	charges orderChargeKeeper,
) (*orderservice.UpdateChargeRequest, error) {
	current, err := r.current(ctx, charges)
	if err != nil {
		return nil, err
	}
	req := &orderservice.UpdateChargeRequest{
		TenantInfo:  r.tenant,
		OrderID:     r.orderID,
		ChargeID:    r.chargeID,
		Description: current.Description,
		Amount:      current.Amount,
		Version:     current.Version,
	}
	if text, textErr := optionalBoundedText(r.params, fieldDescription,
		maxOrderChargeText); textErr != nil {
		return nil, textErr
	} else if text != nil {
		req.Description = *text
	}
	if _, given := r.params[paramAmount]; given {
		if req.Amount, err = decimalArg(r.params, paramAmount, true); err != nil {
			return nil, err
		}
	}
	if req.Allocations, err = readAllocations(
		r.params,
		r.tenant,
		shipment.ChargeAllocationKindOrderCharge,
		current.Allocations,
	); err != nil {
		return nil, err
	}

	return req, nil
}

func (r *chargeRef) allocationRequest(
	ctx context.Context,
	charges orderChargeKeeper,
) (*orderservice.SetChargeAllocationsRequest, error) {
	current, err := r.current(ctx, charges)
	if err != nil {
		return nil, err
	}
	allocations, err := readAllocations(
		r.params,
		r.tenant,
		shipment.ChargeAllocationKindOrderCharge,
		current.Allocations,
	)
	if err != nil {
		return nil, err
	}
	if allocations == nil {
		allocations = []*shipment.ChargeAllocation{}
	}

	return &orderservice.SetChargeAllocationsRequest{
		TenantInfo:  r.tenant,
		OrderID:     r.orderID,
		ChargeID:    r.chargeID,
		Allocations: allocations,
	}, nil
}

func newUpdateOrderChargeTool(charges orderChargeKeeper) serviceports.AgentTool {
	return newReportingReceivableTool(chargeSpec(
		"update_order_charge",
		"Change an order charge's description, amount or payer split. Send only what "+
			"changes. An invoiced charge cannot be changed; adjust its invoice instead.",
		"Changes what the customer is billed; nothing is invoiced or sent, and the old "+
			"amount is set back the same way until it is invoiced.",
		map[string]any{
			paramOrderID:  orderIDProperty("The order"),
			paramChargeID: chargeIDProperty(),
			fieldDescription: stringProperty("The new description, in full.",
				maxOrderChargeText),
			paramAmount: amountProperty("The new amount, as a decimal such as 150.00."),
			paramAllocations: allocationsProperty("The whole new payer split; an empty " +
				"list bills it all to the order's customer."),
		},
		paramOrderID, paramChargeID,
	), receivablePlan[*chargeRef, *orderservice.ChargePlan]{
		request: func(params *serviceports.ToolExecuteParams) (*chargeRef, error) {
			ref, err := readChargeRef(params)
			if err != nil {
				return nil, err
			}
			if !sendsAny(params.Params, fieldDescription, paramAmount, paramAllocations) {
				return nil, errNothingToChange
			}

			return ref, nil
		},
		plan: func(
			ctx context.Context,
			ref *chargeRef,
			_ *serviceports.ToolExecuteParams,
		) (*orderservice.ChargePlan, error) {
			req, err := ref.updateRequest(ctx, charges)
			if err != nil {
				return nil, err
			}

			return charges.PlanUpdateCharge(ctx, req)
		},
		refused: func(*chargeRef) string { return "Would change a charge on the order." },
		render: func(ref *chargeRef, plan *orderservice.ChargePlan) (*agent.ToolPreview, error) {
			return renderChargePlan("change")(ref, plan)
		},
		run: func(
			ctx context.Context,
			ref *chargeRef,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			req, err := ref.updateRequest(ctx, charges)
			if err != nil {
				return nil, err
			}
			updated, err := charges.UpdateCharge(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}

			return orderResult("charge updated", updated), nil
		},
	})
}

func newSetOrderChargeAllocationsTool(charges orderChargeKeeper) serviceports.AgentTool {
	return newReportingReceivableTool(chargeSpec(
		"set_order_charge_allocations",
		"Replace how one order charge is split among payers without touching the charge "+
			"itself. Percents must add up to 100 and amounts to the charge; an empty list "+
			"bills it all to the order's customer.",
		"Moves who is billed for a charge; nothing is invoiced or sent, and the split is "+
			"set back the same way until it is invoiced.",
		map[string]any{
			paramOrderID:     orderIDProperty("The order"),
			paramChargeID:    chargeIDProperty(),
			paramAllocations: allocationsProperty("The whole new payer split."),
		},
		paramOrderID, paramChargeID, paramAllocations,
	), receivablePlan[*chargeRef, *orderservice.ChargePlan]{
		request: func(params *serviceports.ToolExecuteParams) (*chargeRef, error) {
			ref, err := readChargeRef(params)
			if err != nil {
				return nil, err
			}
			if _, err = readAllocations(params.Params, ref.tenant,
				shipment.ChargeAllocationKindOrderCharge, nil); err != nil {
				return nil, err
			}
			if !sendsAny(params.Params, paramAllocations) {
				return nil, fmt.Errorf("missing required parameter %q", paramAllocations)
			}

			return ref, nil
		},
		plan: func(
			ctx context.Context,
			ref *chargeRef,
			_ *serviceports.ToolExecuteParams,
		) (*orderservice.ChargePlan, error) {
			req, err := ref.allocationRequest(ctx, charges)
			if err != nil {
				return nil, err
			}

			return charges.PlanSetChargeAllocations(ctx, req)
		},
		refused: func(*chargeRef) string { return "Would change who pays a charge on the order." },
		render: func(ref *chargeRef, plan *orderservice.ChargePlan) (*agent.ToolPreview, error) {
			return renderChargePlan("re-split")(ref, plan)
		},
		run: func(
			ctx context.Context,
			ref *chargeRef,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			req, err := ref.allocationRequest(ctx, charges)
			if err != nil {
				return nil, err
			}
			updated, err := charges.SetChargeAllocations(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}

			return orderResult("charge split updated", updated), nil
		},
	})
}

func newRemoveOrderChargeTool(charges orderChargeKeeper) serviceports.AgentTool {
	return newReportingReceivableTool(chargeSpec(
		"remove_order_charge",
		"Take a charge off an order before it is invoiced. An invoiced charge stays; "+
			"credit its invoice instead.",
		"Lowers what the customer is billed; nothing is invoiced or sent, and "+
			"add_order_charge puts it back.",
		map[string]any{
			paramOrderID:  orderIDProperty("The order"),
			paramChargeID: chargeIDProperty(),
		},
		paramOrderID, paramChargeID,
	), receivablePlan[*chargeRef, *orderservice.ChargePlan]{
		request: readChargeRef,
		plan: func(
			ctx context.Context,
			ref *chargeRef,
			_ *serviceports.ToolExecuteParams,
		) (*orderservice.ChargePlan, error) {
			return charges.PlanRemoveCharge(ctx, ref.tenant, ref.orderID, ref.chargeID)
		},
		refused: func(*chargeRef) string { return "Would remove a charge from the order." },
		render: func(ref *chargeRef, plan *orderservice.ChargePlan) (*agent.ToolPreview, error) {
			return renderChargePlan("remove")(ref, plan)
		},
		run: func(
			ctx context.Context,
			ref *chargeRef,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			updated, err := charges.RemoveCharge(ctx, ref.tenant, ref.orderID, ref.chargeID,
				params.Actor)
			if err != nil {
				return nil, err
			}

			return orderResult("charge removed", updated), nil
		},
	})
}
