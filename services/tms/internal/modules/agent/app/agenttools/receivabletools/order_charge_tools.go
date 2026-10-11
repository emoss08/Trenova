package receivabletools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/orderservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
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
		actor *permissionports.RequestActor,
	) (*order.Order, error)
	PlanAddCharge(
		ctx context.Context,
		req *orderservice.AddChargeRequest,
	) (*orderservice.ChargePlan, error)
	UpdateCharge(
		ctx context.Context,
		req *orderservice.UpdateChargeRequest,
		actor *permissionports.RequestActor,
	) (*order.Order, error)
	PlanUpdateCharge(
		ctx context.Context,
		req *orderservice.UpdateChargeRequest,
	) (*orderservice.ChargePlan, error)
	SetChargeAllocations(
		ctx context.Context,
		req *orderservice.SetChargeAllocationsRequest,
		actor *permissionports.RequestActor,
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
		actor *permissionports.RequestActor,
	) (*order.Order, error)
	PlanRemoveCharge(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
		chargeID pulid.ID,
	) (*orderservice.ChargePlan, error)
}

func chargeIDProperty() map[string]any {
	return agenttoolschema.KindID(
		agenttoolschema.IDDescription("The order charge", "the charges get_order lists"),
		permission.KindOrderCharge,
	)
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

	return fmt.Sprintf("%s: %s", common.CountOf(len(parts), "payer"), strings.Join(parts, " / "))
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
			Version:  common.PinnedVersion(plan.Order.Version),
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
) *common.ReceivableSpec {
	return &common.ReceivableSpec{
		Name:        name,
		Description: description,
		Artifact:    common.OrderRecordEntity,
		Resource:    permission.ResourceOrder,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressMoney,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale:   rationale,
		Properties:  properties,
		Required:    required,
		Target:      targetOrder,
	}
}

func NewAddOrderChargeTool(charges orderChargeKeeper) agentports.AgentTool {
	return common.NewReportingReceivableTool(chargeSpec(
		"add_order_charge",
		"Add an order-level charge, one billed once for the whole order rather than on a "+
			"shipment: a consolidation fee, a quoted flat rate. Give allocations only to "+
			"split it among several payers; left out, the order's customer pays it all.",
		"Adds to what the customer is billed; nothing is invoiced or sent, and "+
			"remove_order_charge takes it off until it is invoiced.",
		map[string]any{
			common.ParamOrderID: orderIDProperty("The order"),
			common.FieldDescription: common.StringProperty(
				"What the charge is for, as the invoice "+
					"line should read.",
				maxOrderChargeText,
			),
			common.ParamAmount: common.AmountProperty(
				"The amount, as a decimal such as 150.00.",
			),
			common.ParamAllocations: common.AllocationsProperty(
				"How the charge is split among payers.",
			),
		},
		common.ParamOrderID, common.FieldDescription, common.ParamAmount,
	), common.ReceivablePlan[*orderservice.AddChargeRequest, *orderservice.ChargePlan]{
		Request: func(params *agentports.ToolExecuteParams) (*orderservice.AddChargeRequest, error) {
			orderID, err := common.RequirePulid(params.Params, common.ParamOrderID)
			if err != nil {
				return nil, err
			}
			description, err := common.RequireBoundedText(params.Params, common.FieldDescription,
				maxOrderChargeText)
			if err != nil {
				return nil, err
			}
			amount, err := common.DecimalArg(params.Params, common.ParamAmount, true)
			if err != nil {
				return nil, err
			}
			tenant := common.TenantFrom(*params)
			allocations, err := common.ReadAllocations(
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
		Plan: func(
			ctx context.Context,
			req *orderservice.AddChargeRequest,
			_ *agentports.ToolExecuteParams,
		) (*orderservice.ChargePlan, error) {
			return charges.PlanAddCharge(ctx, req)
		},
		Refused: func(*orderservice.AddChargeRequest) string {
			return "Would add a charge to the order."
		},
		Render: func(req *orderservice.AddChargeRequest, plan *orderservice.ChargePlan) (*agent.ToolPreview, error) {
			return renderChargePlan("add")(req, plan)
		},
		Run: func(
			ctx context.Context,
			req *orderservice.AddChargeRequest,
			params *agentports.ToolExecuteParams,
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

func readChargeRef(params *agentports.ToolExecuteParams) (*chargeRef, error) {
	orderID, err := common.RequirePulid(params.Params, common.ParamOrderID)
	if err != nil {
		return nil, err
	}
	chargeID, err := common.RequirePulid(params.Params, paramChargeID)
	if err != nil {
		return nil, err
	}

	return &chargeRef{
		tenant:   common.TenantFrom(*params),
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
	if text, textErr := common.OptionalBoundedText(r.params, common.FieldDescription,
		maxOrderChargeText); textErr != nil {
		return nil, textErr
	} else if text != nil {
		req.Description = *text
	}
	if _, given := r.params[common.ParamAmount]; given {
		if req.Amount, err = common.DecimalArg(r.params, common.ParamAmount, true); err != nil {
			return nil, err
		}
	}
	if req.Allocations, err = common.ReadAllocations(
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
	allocations, err := common.ReadAllocations(
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

func NewUpdateOrderChargeTool(charges orderChargeKeeper) agentports.AgentTool {
	spec := chargeSpec(
		"update_order_charge",
		"Change an order charge's description, amount or payer split. Send only what "+
			"changes. An invoiced charge cannot be changed; adjust its invoice instead.",
		"Changes what the customer is billed; nothing is invoiced or sent, and the old "+
			"amount is set back the same way until it is invoiced.",
		map[string]any{
			common.ParamOrderID: orderIDProperty("The order"),
			paramChargeID:       chargeIDProperty(),
			common.FieldDescription: common.StringProperty("The new description, in full.",
				maxOrderChargeText),
			common.ParamAmount: common.AmountProperty(
				"The new amount, as a decimal such as 150.00.",
			),
			common.ParamAllocations: common.AllocationsProperty(
				"The whole new payer split; an empty " +
					"list bills it all to the order's customer.",
			),
		},
		common.ParamOrderID, paramChargeID,
	)
	spec.SearchTerms = []string{"charge amount", "correct charge", "edit accessorial"}

	return common.NewReportingReceivableTool(
		spec,
		common.ReceivablePlan[*chargeRef, *orderservice.ChargePlan]{
			Request: func(params *agentports.ToolExecuteParams) (*chargeRef, error) {
				ref, err := readChargeRef(params)
				if err != nil {
					return nil, err
				}
				if !common.SendsAny(
					params.Params,
					common.FieldDescription,
					common.ParamAmount,
					common.ParamAllocations,
				) {
					return nil, common.ErrNothingToChange
				}

				return ref, nil
			},
			Plan: func(
				ctx context.Context,
				ref *chargeRef,
				_ *agentports.ToolExecuteParams,
			) (*orderservice.ChargePlan, error) {
				req, err := ref.updateRequest(ctx, charges)
				if err != nil {
					return nil, err
				}

				return charges.PlanUpdateCharge(ctx, req)
			},
			Refused: func(*chargeRef) string { return "Would change a charge on the order." },
			Render: func(ref *chargeRef, plan *orderservice.ChargePlan) (*agent.ToolPreview, error) {
				return renderChargePlan("change")(ref, plan)
			},
			Run: func(
				ctx context.Context,
				ref *chargeRef,
				params *agentports.ToolExecuteParams,
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
		},
	)
}

func NewSetOrderChargeAllocationsTool(charges orderChargeKeeper) agentports.AgentTool {
	spec := chargeSpec(
		"set_order_charge_allocations",
		"Replace how one order charge is split among payers without touching the charge "+
			"itself. Percents must add up to 100 and amounts to the charge; an empty list "+
			"bills it all to the order's customer.",
		"Moves who is billed for a charge; nothing is invoiced or sent, and the split is "+
			"set back the same way until it is invoiced.",
		map[string]any{
			common.ParamOrderID:     orderIDProperty("The order"),
			paramChargeID:           chargeIDProperty(),
			common.ParamAllocations: common.AllocationsProperty("The whole new payer split."),
		},
		common.ParamOrderID, paramChargeID, common.ParamAllocations,
	)
	spec.SearchTerms = []string{"split charge", "payer split", "allocate charge"}

	return common.NewReportingReceivableTool(
		spec,
		common.ReceivablePlan[*chargeRef, *orderservice.ChargePlan]{
			Request: func(params *agentports.ToolExecuteParams) (*chargeRef, error) {
				ref, err := readChargeRef(params)
				if err != nil {
					return nil, err
				}
				if _, err = common.ReadAllocations(params.Params, ref.tenant,
					shipment.ChargeAllocationKindOrderCharge, nil); err != nil {
					return nil, err
				}
				if !common.SendsAny(params.Params, common.ParamAllocations) {
					return nil, fmt.Errorf("missing required parameter %q", common.ParamAllocations)
				}

				return ref, nil
			},
			Plan: func(
				ctx context.Context,
				ref *chargeRef,
				_ *agentports.ToolExecuteParams,
			) (*orderservice.ChargePlan, error) {
				req, err := ref.allocationRequest(ctx, charges)
				if err != nil {
					return nil, err
				}

				return charges.PlanSetChargeAllocations(ctx, req)
			},
			Refused: func(*chargeRef) string { return "Would change who pays a charge on the order." },
			Render: func(ref *chargeRef, plan *orderservice.ChargePlan) (*agent.ToolPreview, error) {
				return renderChargePlan("re-split")(ref, plan)
			},
			Run: func(
				ctx context.Context,
				ref *chargeRef,
				params *agentports.ToolExecuteParams,
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
		},
	)
}

func NewRemoveOrderChargeTool(charges orderChargeKeeper) agentports.AgentTool {
	spec := chargeSpec(
		"remove_order_charge",
		"Take a charge off an order before it is invoiced. An invoiced charge stays; "+
			"credit its invoice instead.",
		"Lowers what the customer is billed; nothing is invoiced or sent, and "+
			"add_order_charge puts it back.",
		map[string]any{
			common.ParamOrderID: orderIDProperty("The order"),
			paramChargeID:       chargeIDProperty(),
		},
		common.ParamOrderID, paramChargeID,
	)
	spec.SearchTerms = []string{"drop charge", "delete charge", "remove accessorial"}

	return common.NewReportingReceivableTool(
		spec,
		common.ReceivablePlan[*chargeRef, *orderservice.ChargePlan]{
			Request: readChargeRef,
			Plan: func(
				ctx context.Context,
				ref *chargeRef,
				_ *agentports.ToolExecuteParams,
			) (*orderservice.ChargePlan, error) {
				return charges.PlanRemoveCharge(ctx, ref.tenant, ref.orderID, ref.chargeID)
			},
			Refused: func(*chargeRef) string { return "Would remove a charge from the order." },
			Render: func(ref *chargeRef, plan *orderservice.ChargePlan) (*agent.ToolPreview, error) {
				return renderChargePlan("remove")(ref, plan)
			},
			Run: func(
				ctx context.Context,
				ref *chargeRef,
				params *agentports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				updated, err := charges.RemoveCharge(ctx, ref.tenant, ref.orderID, ref.chargeID,
					params.Actor)
				if err != nil {
					return nil, err
				}

				return orderResult("charge removed", updated), nil
			},
		},
	)
}
