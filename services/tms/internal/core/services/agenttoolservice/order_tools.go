package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/orderservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	paramChargeID        = "chargeId"
	paramAllocations     = "allocations"
	paramBillToCustomer  = "billToCustomerId"
	paramAllocMethod     = "method"
	paramAllocPercent    = "percent"
	paramPONumber        = "poNumber"
	paramCurrencyCode    = "currencyCode"
	paramQuotedAmount    = "quotedAmount"
	paramBaseAmount      = "baseAmount"
	kindOrder            = "order"
	maxOrderReference    = 100
	maxOrderChargeText   = 255
	maxOrderAttachments  = 50
	maxChargeAllocations = 20
	currencyCodeLength   = 3
)

var (
	orderFields = []string{
		fieldCustomerID, paramOwnerID, paramPONumber, fieldBol, paramCurrencyCode,
		paramQuotedAmount, paramBaseAmount, fieldStatus,
	}
	orderRefs = map[string]permission.Resource{
		fieldCustomerID: permission.ResourceCustomer,
		paramOwnerID:    permission.ResourceUser,
	}
	allocationMethods = agenttoolschema.Source(
		"shipment.chargeAllocationMethod",
		shipment.ChargeAllocationMethodValues(),
	)
)

type orderKeeper interface {
	Get(ctx context.Context, req repositories.GetOrderByIDRequest) (*order.Order, error)
	Create(
		ctx context.Context,
		entity *order.Order,
		actor *serviceports.RequestActor,
	) (*order.Order, error)
	PlanCreate(ctx context.Context, entity *order.Order) (*orderservice.OrderChange, error)
	Update(
		ctx context.Context,
		entity *order.Order,
		actor *serviceports.RequestActor,
	) (*order.Order, error)
	PlanUpdate(ctx context.Context, entity *order.Order) (*orderservice.OrderChange, error)
	AttachShipments(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
		shipmentIDs []pulid.ID,
		actor *serviceports.RequestActor,
	) (*order.Order, error)
	PlanAttachShipments(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
		shipmentIDs []pulid.ID,
	) (*orderservice.MembershipPlan, error)
	DetachShipment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
		shipmentID pulid.ID,
		actor *serviceports.RequestActor,
	) (*order.Order, error)
	PlanDetachShipment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
		shipmentID pulid.ID,
	) (*orderservice.DetachPlan, error)
	Close(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
		actor *serviceports.RequestActor,
	) (*order.Order, error)
	PlanClose(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
	) (*orderservice.OrderChange, error)
	Cancel(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
		cancelReason string,
		actor *serviceports.RequestActor,
	) (*order.Order, error)
	PlanCancel(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
	) (*orderservice.CancelPlan, error)
}

func orderIDProperty(what string) map[string]any {
	return agenttoolschema.RecordID(permission.ResourceOrder, what,
		"list_orders, get_order or the orderId get_shipment shows")
}

func targetOrder(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramOrderID, permission.ResourceOrder)
}

func orderRecord(entity *order.Order) toolpreview.Record {
	label := "Order " + entity.OrderNumber
	if strings.TrimSpace(entity.OrderNumber) == "" {
		label = "New order"
	}

	return toolpreview.Record{
		Resource: permission.ResourceOrder,
		ID:       entity.ID,
		Label:    label,
		Version:  pinnedVersion(entity.Version),
	}
}

func orderResult(action string, entity *order.Order) *agent.ToolExecutionResult {
	result := &agent.ToolExecutionResult{Action: action, Kind: kindOrder}
	if entity != nil {
		result.Name = entity.OrderNumber
		result.IDs = map[string]string{paramOrderID: entity.ID.String()}
		result.Record = recordOf(orderRecordEntity, entity.ID)
	}

	return result
}

func orderMoneyProperty(what string) map[string]any {
	return amountProperty(what + ", as a decimal such as 1250.00. An empty string clears it.")
}

func orderProperties(customerRequired bool) map[string]any {
	customer := "The customer the order is for, from list_customers."
	if !customerRequired {
		customer += " Only an order with no shipments can change customer."
	}

	return map[string]any{
		fieldCustomerID: agenttoolschema.RecordIDText(
			permission.ResourceCustomer,
			customer+" Never guess one.",
		),
		paramOwnerID: agenttoolschema.IDText("The user who owns the order: the person who asked, " +
			"or an ownerId get_order shows. Never guess one."),
		paramPONumber: stringProperty("The customer's purchase order number.",
			maxOrderReference),
		fieldBol: stringProperty("The bill of lading number.", maxOrderReference),
		paramCurrencyCode: stringProperty("The ISO 4217 currency, such as USD. Defaults to "+
			"USD.", currencyCodeLength),
		paramQuotedAmount: orderMoneyProperty("What the customer was quoted"),
		paramBaseAmount:   orderMoneyProperty("The base amount before accessorials"),
	}
}

// applyOrderArgs lays what the call sends over an order, the way the order
// form does: an absent argument leaves the value alone.
func applyOrderArgs(entity *order.Order, params map[string]any) error {
	if _, given := params[fieldCustomerID]; given {
		customerID, err := requirePulid(params, fieldCustomerID)
		if err != nil {
			return err
		}
		entity.CustomerID = customerID
	}
	if _, given := params[paramOwnerID]; given {
		ownerID, err := requirePulid(params, paramOwnerID)
		if err != nil {
			return err
		}
		entity.OwnerID = ownerID
	}
	for key, target := range map[string]*string{
		paramPONumber: &entity.PONumber,
		fieldBol:      &entity.BOL,
	} {
		value, err := optionalBoundedText(params, key, maxOrderReference)
		if err != nil {
			return err
		}
		if value != nil {
			*target = *value
		}
	}
	currency, err := optionalBoundedText(params, paramCurrencyCode, currencyCodeLength)
	if err != nil {
		return err
	}
	if currency != nil {
		entity.CurrencyCode = strings.ToUpper(*currency)
	}
	for key, target := range map[string]*decimal.NullDecimal{
		paramQuotedAmount: &entity.QuotedAmount,
		paramBaseAmount:   &entity.BaseAmount,
	} {
		if err = applyOptionalAmount(params, key, target); err != nil {
			return err
		}
	}

	return nil
}

func applyOptionalAmount(params map[string]any, key string, target *decimal.NullDecimal) error {
	raw, given := params[key]
	if !given {
		return nil
	}
	if text, isText := raw.(string); isText && strings.TrimSpace(text) == "" {
		*target = decimal.NullDecimal{}

		return nil
	}
	value, present, err := optionalDecimal(params, key)
	if err != nil {
		return err
	}
	if present {
		if value.IsNegative() {
			return fmt.Errorf("parameter %q must not be negative", key)
		}
		*target = decimal.NewNullDecimal(value)
	}

	return nil
}

func renderOrderChange(verb string) func(
	*order.Order,
	*orderservice.OrderChange,
) (*agent.ToolPreview, error) {
	return func(_ *order.Order, plan *orderservice.OrderChange) (*agent.ToolPreview, error) {
		if plan.Before == nil {
			change, err := toolpreview.Create(
				orderRecord(plan.After),
				plan.After,
				toolpreview.Only(orderFields...),
				toolpreview.WithRefs(orderRefs),
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build("Would open a new Draft order; its number is issued "+
				"when it is saved.", change), nil
		}

		change, err := toolpreview.Changed(
			orderRecord(plan.Before),
			plan.Before,
			plan.After,
			toolpreview.Only(orderFields...),
			toolpreview.WithRefs(orderRefs),
		)
		if err != nil {
			return nil, err
		}

		return toolpreview.Build(fmt.Sprintf("Would %s order %s.", verb,
			plan.Before.OrderNumber), change), nil
	}
}

func newCreateOrderTool(orders orderKeeper) serviceports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "create_order",
		artifact: orderRecordEntity,
		description: "Open a customer order: the commercial record that groups a " +
			"customer's shipments and order-level charges under their PO or BOL. It starts " +
			"as a Draft with its own number; attach shipments with attach_order_shipments " +
			"and add charges with add_order_charge. Only from what the person or the " +
			"customer asked for.",
		resource:    permission.ResourceOrder,
		operation:   permission.OpCreate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Opens an empty order inside Trenova; nothing is billed or sent, and an " +
			"order opened in error is canceled.",
		properties: orderProperties(true),
		required:   []string{fieldCustomerID},
	}, receivablePlan[*order.Order, *orderservice.OrderChange]{
		request: func(params *serviceports.ToolExecuteParams) (*order.Order, error) {
			if _, err := requirePulid(params.Params, fieldCustomerID); err != nil {
				return nil, err
			}
			entity := &order.Order{
				OrganizationID: params.OrganizationID,
				BusinessUnitID: params.BusinessUnitID,
				EnteredByID:    params.Actor.UserID,
				Status:         order.StatusDraft,
				CurrencyCode:   "USD",
			}
			if err := applyOrderArgs(entity, params.Params); err != nil {
				return nil, err
			}

			return entity, nil
		},
		plan: func(
			ctx context.Context,
			entity *order.Order,
			_ *serviceports.ToolExecuteParams,
		) (*orderservice.OrderChange, error) {
			probe := *entity

			return orders.PlanCreate(ctx, &probe)
		},
		refused: func(*order.Order) string { return "Would open a new order." },
		render:  renderOrderChange("open"),
		run: func(
			ctx context.Context,
			entity *order.Order,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			created, err := orders.Create(ctx, entity, params.Actor)
			if err != nil {
				return nil, err
			}

			return orderResult("created", created), nil
		},
	})
}

type orderEdit struct {
	get    repositories.GetOrderByIDRequest
	params map[string]any
}

func (e *orderEdit) entity(ctx context.Context, orders orderKeeper) (*order.Order, error) {
	current, err := orders.Get(ctx, e.get)
	if err != nil {
		return nil, err
	}
	if err = applyOrderArgs(current, e.params); err != nil {
		return nil, err
	}

	return current, nil
}

func newUpdateOrderTool(orders orderKeeper) serviceports.AgentTool {
	base := newReportingReceivableTool(&receivableSpec{
		name:        "update_order",
		searchTerms: []string{"purchase order", "po", "fix bol", "edit order"},
		artifact:    orderRecordEntity,
		description: "Change an order's customer, owner, PO number, BOL, currency, quoted " +
			"amount or base amount. Send only what changes. Its status and total follow its " +
			"shipments and charges and cannot be set here.",
		resource:    permission.ResourceOrder,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Edits an order's own fields inside Trenova; nothing is sent, and the old " +
			"values are set back the same way. A changed amount is a money change.",
		properties: withOrderID(orderProperties(false), "The order to change"),
		required:   []string{paramOrderID},
		target:     targetOrder,
	}, receivablePlan[*orderEdit, *orderservice.OrderChange]{
		request: func(params *serviceports.ToolExecuteParams) (*orderEdit, error) {
			orderID, err := requirePulid(params.Params, paramOrderID)
			if err != nil {
				return nil, err
			}
			if !sendsAny(params.Params, orderFields...) {
				return nil, errNothingToChange
			}
			probe := &order.Order{}
			if err = applyOrderArgs(probe, params.Params); err != nil {
				return nil, err
			}

			return &orderEdit{
				get: repositories.GetOrderByIDRequest{
					ID:         orderID,
					TenantInfo: tenantFrom(*params),
				},
				params: params.Params,
			}, nil
		},
		plan: func(
			ctx context.Context,
			edit *orderEdit,
			_ *serviceports.ToolExecuteParams,
		) (*orderservice.OrderChange, error) {
			entity, err := edit.entity(ctx, orders)
			if err != nil {
				return nil, err
			}

			return orders.PlanUpdate(ctx, entity)
		},
		refused: func(*orderEdit) string { return "Would change the order." },
		render: func(_ *orderEdit, plan *orderservice.OrderChange) (*agent.ToolPreview, error) {
			return renderOrderChange("change")(nil, plan)
		},
		run: func(
			ctx context.Context,
			edit *orderEdit,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			entity, err := edit.entity(ctx, orders)
			if err != nil {
				return nil, err
			}
			updated, err := orders.Update(ctx, entity, params.Actor)
			if err != nil {
				return nil, err
			}

			return orderResult("updated", updated), nil
		},
	})

	return classifiedTool[*orderEdit, *orderservice.OrderChange]{
		reportingReceivableTool: base,
		egress:                  []agent.EgressClass{agent.EgressInternal, agent.EgressMoney},
		classify:                classifyOrderAmounts,
	}
}

func classifyOrderAmounts(
	params serviceports.ToolExecuteParams, //nolint:gocritic // ToolPolicy.Classify passes params by value
) serviceports.CallPolicy {
	if sendsAny(params.Params, paramQuotedAmount, paramBaseAmount) {
		return serviceports.CallPolicy{Egress: agent.EgressMoney}
	}

	return serviceports.CallPolicy{Egress: agent.EgressInternal}
}

func sendsAny(params map[string]any, keys ...string) bool {
	for _, key := range keys {
		if _, given := params[key]; given {
			return true
		}
	}

	return false
}

func withOrderID(properties map[string]any, what string) map[string]any {
	properties[paramOrderID] = orderIDProperty(what)

	return properties
}

type orderMembership struct {
	tenant      pagination.TenantInfo
	orderID     pulid.ID
	shipmentIDs []pulid.ID
}

func newAttachOrderShipmentsTool(orders orderKeeper) serviceports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "attach_order_shipments",
		artifact: orderRecordEntity,
		description: "Put shipments on an order so they are billed and tracked together. " +
			"Each must be the order's customer's and neither canceled nor invoiced; a " +
			"shipment on another open order moves off it, and an order it leaves empty is " +
			"removed. A Billed or Closed order takes no more shipments.",
		resource:    permission.ResourceOrder,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Regroups shipments under an order inside Trenova; nothing is sent, and " +
			"detach_order_shipment moves one back out.",
		properties: map[string]any{
			paramOrderID: orderIDProperty("The order to add them to"),
			paramShipmentIDs: toolschema.RecordSubset(permission.ResourceShipment.String(),
				agenttoolschema.RecordIDs(
					permission.ResourceShipment,
					"The shipments to attach, from search_shipments or "+
						"get_shipment.",
					maxOrderAttachments,
				)),
		},
		required: []string{paramOrderID, paramShipmentIDs},
		target:   targetOrder,
	}, receivablePlan[*orderMembership, *orderservice.MembershipPlan]{
		request: func(params *serviceports.ToolExecuteParams) (*orderMembership, error) {
			orderID, err := requirePulid(params.Params, paramOrderID)
			if err != nil {
				return nil, err
			}
			shipmentIDs, err := requirePulidSlice(params.Params, paramShipmentIDs,
				maxOrderAttachments)
			if err != nil {
				return nil, err
			}

			return &orderMembership{
				tenant:      tenantFrom(*params),
				orderID:     orderID,
				shipmentIDs: shipmentIDs,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *orderMembership,
			_ *serviceports.ToolExecuteParams,
		) (*orderservice.MembershipPlan, error) {
			return orders.PlanAttachShipments(ctx, req.tenant, req.orderID, req.shipmentIDs)
		},
		refused: func(*orderMembership) string {
			return "Would attach shipments to the order."
		},
		render: renderAttachShipments,
		run: func(
			ctx context.Context,
			req *orderMembership,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			updated, err := orders.AttachShipments(ctx, req.tenant, req.orderID,
				req.shipmentIDs, params.Actor)
			if err != nil {
				return nil, err
			}

			return orderResult("shipments attached", updated), nil
		},
	})
}

type shipmentOrderView struct {
	OrderID pulid.ID `json:"orderId"`
}

func renderAttachShipments(
	req *orderMembership,
	plan *orderservice.MembershipPlan,
) (*agent.ToolPreview, error) {
	if len(plan.ShipmentIDs) == 0 {
		return toolpreview.Build(fmt.Sprintf(
			"Would leave order %s as it is: every shipment named is already on it.",
			plan.Order.OrderNumber,
		)), nil
	}

	changes := make([]*agent.RecordChange, 0, len(plan.ShipmentIDs))
	for _, shipmentID := range plan.ShipmentIDs {
		change, err := toolpreview.Changed(
			toolpreview.Record{Resource: permission.ResourceShipment, ID: shipmentID},
			&shipmentOrderView{},
			&shipmentOrderView{OrderID: plan.Order.ID},
			toolpreview.WithRefs(map[string]permission.Resource{
				paramOrderID: permission.ResourceOrder,
			}),
		)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}

	summary := fmt.Sprintf("Would attach %s to order %s.",
		countOf(len(plan.ShipmentIDs), "shipment"), plan.Order.OrderNumber)
	if skipped := len(req.shipmentIDs) - len(plan.ShipmentIDs); skipped > 0 {
		summary += fmt.Sprintf(" %d named are already on it.", skipped)
	}
	if len(plan.SourceOrderIDs) > 0 {
		summary += fmt.Sprintf(" They leave %s, which is removed if left empty.",
			countOf(len(plan.SourceOrderIDs), "other order"))
	}

	return toolpreview.Build(summary, changes...), nil
}

type orderLeg struct {
	tenant     pagination.TenantInfo
	orderID    pulid.ID
	shipmentID pulid.ID
}

func newDetachOrderShipmentTool(orders orderKeeper) serviceports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "detach_order_shipment",
		artifact: orderRecordEntity,
		description: "Take one shipment off an order onto an order of its own, so it is " +
			"billed on its own. The order's only shipment and an invoiced one stay where " +
			"they are.",
		resource:    permission.ResourceOrder,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		rationale: "Moves a shipment onto a new order inside Trenova; nothing is sent, and " +
			"attach_order_shipments puts it back.",
		properties: map[string]any{
			paramOrderID:    orderIDProperty("The order the shipment is on"),
			paramShipmentID: shipmentIDProperty("The shipment to take off it"),
		},
		required: []string{paramOrderID, paramShipmentID},
		target:   targetOrder,
	}, receivablePlan[*orderLeg, *orderservice.DetachPlan]{
		request: func(params *serviceports.ToolExecuteParams) (*orderLeg, error) {
			orderID, err := requirePulid(params.Params, paramOrderID)
			if err != nil {
				return nil, err
			}
			shipmentID, err := requirePulid(params.Params, paramShipmentID)
			if err != nil {
				return nil, err
			}

			return &orderLeg{tenant: tenantFrom(*params), orderID: orderID,
				shipmentID: shipmentID}, nil
		},
		plan: func(
			ctx context.Context,
			req *orderLeg,
			_ *serviceports.ToolExecuteParams,
		) (*orderservice.DetachPlan, error) {
			return orders.PlanDetachShipment(ctx, req.tenant, req.orderID, req.shipmentID)
		},
		refused: func(*orderLeg) string { return "Would detach the shipment from the order." },
		render: func(_ *orderLeg, plan *orderservice.DetachPlan) (*agent.ToolPreview, error) {
			created, err := toolpreview.Create(
				orderRecord(plan.Replacement),
				plan.Replacement,
				toolpreview.Only(fieldCustomerID, fieldStatus, paramCurrencyCode, "totalAmount"),
				toolpreview.WithRefs(orderRefs),
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would take shipment %s off order %s onto a new order of its own.",
				plan.Leg.ProNumber, plan.Order.OrderNumber,
			), created), nil
		},
		run: func(
			ctx context.Context,
			req *orderLeg,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			updated, err := orders.DetachShipment(ctx, req.tenant, req.orderID, req.shipmentID,
				params.Actor)
			if err != nil {
				return nil, err
			}

			return orderResult("shipment detached", updated), nil
		},
	})
}

type orderOnly struct {
	tenant  pagination.TenantInfo
	orderID pulid.ID
	reason  string
}

func readOrderOnly(
	params *serviceports.ToolExecuteParams,
	reasonRequired bool,
) (*orderOnly, error) {
	orderID, err := requirePulid(params.Params, paramOrderID)
	if err != nil {
		return nil, err
	}
	req := &orderOnly{tenant: tenantFrom(*params), orderID: orderID}
	if reasonRequired {
		if req.reason, err = requireBoundedText(params.Params, paramCancelReason,
			maxOperationNoteChars); err != nil {
			return nil, err
		}
	}

	return req, nil
}

func newCloseOrderTool(orders orderKeeper) serviceports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "close_order",
		artifact: orderRecordEntity,
		description: "Close a Billed order once nothing more will be billed on it. Closing " +
			"is final: a Closed order takes no more shipments or charges.",
		resource:    permission.ResourceOrder,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		personOnly:  true,
		rationale: "Settles an order for good; nothing reopens it, so only a person " +
			"closes one.",
		properties: map[string]any{paramOrderID: orderIDProperty("The Billed order")},
		required:   []string{paramOrderID},
		target:     targetOrder,
	}, receivablePlan[*orderOnly, *orderservice.OrderChange]{
		request: func(params *serviceports.ToolExecuteParams) (*orderOnly, error) {
			return readOrderOnly(params, false)
		},
		plan: func(
			ctx context.Context,
			req *orderOnly,
			_ *serviceports.ToolExecuteParams,
		) (*orderservice.OrderChange, error) {
			return orders.PlanClose(ctx, req.tenant, req.orderID)
		},
		refused: func(*orderOnly) string { return "Would close the order." },
		render: func(_ *orderOnly, plan *orderservice.OrderChange) (*agent.ToolPreview, error) {
			return renderOrderChange("close")(nil, plan)
		},
		run: func(
			ctx context.Context,
			req *orderOnly,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			closed, err := orders.Close(ctx, req.tenant, req.orderID, params.Actor)
			if err != nil {
				return nil, err
			}

			return orderResult("closed", closed), nil
		},
	})
}

func newCancelOrderTool(orders orderKeeper) serviceports.AgentTool {
	return newReportingReceivableTool(&receivableSpec{
		name:     "cancel_order",
		artifact: orderRecordEntity,
		description: "Cancel an order and every shipment on it that is not already " +
			"canceled, with the reason the customer gave. An order with an invoiced " +
			"shipment cannot be canceled; credit the invoice first. Canceling a shipment " +
			"tells its carrier and trading partner as cancel_shipment would.",
		resource:    permission.ResourceOrder,
		operation:   permission.OpUpdate,
		egress:      agent.EgressExternalRecipient,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		personOnly:  true,
		rationale: "Cancels every live shipment on the order, which reaches carriers and " +
			"trading partners; only a person cancels an order.",
		properties: map[string]any{
			paramOrderID: orderIDProperty("The order to cancel"),
			paramCancelReason: stringProperty("Why, as the customer said it.",
				maxOperationNoteChars),
		},
		required: []string{paramOrderID, paramCancelReason},
		target:   targetOrder,
	}, receivablePlan[*orderOnly, *orderservice.CancelPlan]{
		request: func(params *serviceports.ToolExecuteParams) (*orderOnly, error) {
			return readOrderOnly(params, true)
		},
		plan: func(
			ctx context.Context,
			req *orderOnly,
			_ *serviceports.ToolExecuteParams,
		) (*orderservice.CancelPlan, error) {
			return orders.PlanCancel(ctx, req.tenant, req.orderID)
		},
		refused: func(*orderOnly) string { return "Would cancel the order." },
		render:  renderCancelOrder,
		run: func(
			ctx context.Context,
			req *orderOnly,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			canceled, err := orders.Cancel(ctx, req.tenant, req.orderID, req.reason, params.Actor)
			if err != nil {
				return nil, err
			}

			return orderResult("canceled", canceled), nil
		},
	})
}

type legStatusView struct {
	Status shipment.Status `json:"status"`
}

func renderCancelOrder(req *orderOnly, plan *orderservice.CancelPlan) (*agent.ToolPreview, error) {
	changes := make([]*agent.RecordChange, 0, len(plan.Legs))
	for _, leg := range plan.Legs {
		change, err := toolpreview.Changed(
			shipmentRecord(leg),
			&legStatusView{Status: leg.Status},
			&legStatusView{Status: shipment.StatusCanceled},
		)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would cancel order %s and %s on it: %s",
		plan.Order.OrderNumber,
		countOf(len(plan.Legs), "live shipment"),
		req.reason,
	), changes...), nil
}
