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
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const paramChargeID = "chargeId"

const paramPONumber = "poNumber"

const paramCurrencyCode = "currencyCode"

const paramQuotedAmount = "quotedAmount"

const paramBaseAmount = "baseAmount"

const kindOrder = "order"

const maxOrderReference = 100

const maxOrderChargeText = 255

const maxOrderAttachments = 50

var orderFields = []string{
	common.FieldCustomerID, common.ParamOwnerID, paramPONumber, common.FieldBol, paramCurrencyCode,
	paramQuotedAmount, paramBaseAmount, common.FieldStatus,
}

var orderRefs = map[string]permission.Resource{
	common.FieldCustomerID: permission.ResourceCustomer,
	common.ParamOwnerID:    permission.ResourceUser,
}

type orderKeeper interface {
	Get(ctx context.Context, req orderports.GetOrderByIDRequest) (*order.Order, error)
	Create(
		ctx context.Context,
		entity *order.Order,
		actor *permissionports.RequestActor,
	) (*order.Order, error)
	PlanCreate(ctx context.Context, entity *order.Order) (*orderservice.OrderChange, error)
	Update(
		ctx context.Context,
		entity *order.Order,
		actor *permissionports.RequestActor,
	) (*order.Order, error)
	PlanUpdate(ctx context.Context, entity *order.Order) (*orderservice.OrderChange, error)
	AttachShipments(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
		shipmentIDs []pulid.ID,
		actor *permissionports.RequestActor,
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
		actor *permissionports.RequestActor,
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
		actor *permissionports.RequestActor,
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
		actor *permissionports.RequestActor,
	) (*order.Order, error)
	PlanCancel(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		orderID pulid.ID,
	) (*orderservice.CancelPlan, error)
}

func orderIDProperty(what string) map[string]any {
	return agenttoolschema.RecordID(permission.ResourceOrder, what,
		"order_list, order_get or the orderId shipment_get shows")
}

func targetOrder(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, common.ParamOrderID, permission.ResourceOrder)
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
		Version:  common.PinnedVersion(entity.Version),
	}
}

func orderResult(action string, entity *order.Order) *agent.ToolExecutionResult {
	result := &agent.ToolExecutionResult{Action: action, Kind: kindOrder}
	if entity != nil {
		result.Name = entity.OrderNumber
		result.IDs = map[string]string{common.ParamOrderID: entity.ID.String()}
		result.Record = common.RecordOf(common.OrderRecordEntity, entity.ID)
	}

	return result
}

func orderMoneyProperty(what string) map[string]any {
	return toolschema.KeepEmpty(
		common.AmountProperty(what + ", as a decimal such as 1250.00. An empty string clears it."),
	)
}

func orderProperties(customerRequired bool) map[string]any {
	customer := "The customer the order is for, from list_customers."
	if !customerRequired {
		customer += " Only an order with no shipments can change customer."
	}

	return map[string]any{
		common.FieldCustomerID: agenttoolschema.RecordIDText(
			permission.ResourceCustomer,
			customer+" Never guess one.",
		),
		common.ParamOwnerID: agenttoolschema.RecordIDText(permission.ResourceUser,
			"The user who owns the order: the person who asked, "+
				"or an ownerId order_get shows. Never guess one."),
		paramPONumber: common.StringProperty("The customer's purchase order number.",
			maxOrderReference),
		common.FieldBol: common.StringProperty("The bill of lading number.", maxOrderReference),
		paramCurrencyCode: common.StringProperty("The ISO 4217 currency, such as USD. Defaults to "+
			"USD.", common.CurrencyCodeLength),
		paramQuotedAmount: orderMoneyProperty("What the customer was quoted"),
		paramBaseAmount:   orderMoneyProperty("The base amount before accessorials"),
	}
}

// applyOrderArgs lays what the call sends over an order, the way the order
// form does: an absent argument leaves the value alone.
func applyOrderArgs(entity *order.Order, params map[string]any) error {
	if _, given := params[common.FieldCustomerID]; given {
		customerID, err := common.RequirePulid(params, common.FieldCustomerID)
		if err != nil {
			return err
		}
		entity.CustomerID = customerID
	}
	if _, given := params[common.ParamOwnerID]; given {
		ownerID, err := common.RequirePulid(params, common.ParamOwnerID)
		if err != nil {
			return err
		}
		entity.OwnerID = ownerID
	}
	for key, target := range map[string]*string{
		paramPONumber:   &entity.PONumber,
		common.FieldBol: &entity.BOL,
	} {
		value, err := common.OptionalBoundedText(params, key, maxOrderReference)
		if err != nil {
			return err
		}
		if value != nil {
			*target = *value
		}
	}
	currency, err := common.OptionalBoundedText(
		params,
		paramCurrencyCode,
		common.CurrencyCodeLength,
	)
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
	value, present, err := common.OptionalDecimal(params, key)
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

func NewCreateOrderTool(orders orderKeeper) agentports.AgentTool {
	return common.NewReportingReceivableTool(&common.ReceivableSpec{
		Name:     "order_create",
		Artifact: common.OrderRecordEntity,
		Description: "Open a customer order: the commercial record that groups a " +
			"customer's shipments and order-level charges under their PO or BOL. It starts " +
			"as a Draft with its own number; attach shipments with order_attach_shipments " +
			"and add charges with order_add_charge. Only from what the person or the " +
			"customer asked for.",
		SearchTerms: []string{
			"order for several loads",
			"group loads on one order",
			"order covering loads",
		},
		Resource:    permission.ResourceOrder,
		Operation:   permission.OpCreate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Opens an empty order inside Trenova; nothing is billed or sent, and an " +
			"order opened in error is canceled.",
		Properties: orderProperties(true),
		Required:   []string{common.FieldCustomerID},
	}, common.ReceivablePlan[*order.Order, *orderservice.OrderChange]{
		Request: func(params *agentports.ToolExecuteParams) (*order.Order, error) {
			if _, err := common.RequirePulid(params.Params, common.FieldCustomerID); err != nil {
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
		Plan: func(
			ctx context.Context,
			entity *order.Order,
			_ *agentports.ToolExecuteParams,
		) (*orderservice.OrderChange, error) {
			probe := *entity

			return orders.PlanCreate(ctx, &probe)
		},
		Refused: func(*order.Order) string { return "Would open a new order." },
		Render:  renderOrderChange("open"),
		Run: func(
			ctx context.Context,
			entity *order.Order,
			params *agentports.ToolExecuteParams,
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
	get    orderports.GetOrderByIDRequest
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

func NewUpdateOrderTool(orders orderKeeper) agentports.AgentTool {
	base := common.NewReportingReceivableTool(&common.ReceivableSpec{
		Name:        "order_update",
		SearchTerms: []string{"purchase order", "po", "fix bol", "edit order"},
		Artifact:    common.OrderRecordEntity,
		Description: "Change an order's customer, owner, PO number, BOL, currency, quoted " +
			"amount or base amount. Send only what changes. Its status and total follow its " +
			"shipments and charges and cannot be set here.",
		Resource:    permission.ResourceOrder,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Edits an order's own fields inside Trenova; nothing is sent, and the old " +
			"values are set back the same way. A changed amount is a money change.",
		Properties: withOrderID(orderProperties(false), "The order to change"),
		Required:   []string{common.ParamOrderID},
		Target:     targetOrder,
	}, common.ReceivablePlan[*orderEdit, *orderservice.OrderChange]{
		Request: func(params *agentports.ToolExecuteParams) (*orderEdit, error) {
			orderID, err := common.RequirePulid(params.Params, common.ParamOrderID)
			if err != nil {
				return nil, err
			}
			if !common.SendsAny(params.Params, orderFields...) {
				return nil, common.ErrNothingToChange
			}
			probe := &order.Order{}
			if err = applyOrderArgs(probe, params.Params); err != nil {
				return nil, err
			}

			return &orderEdit{
				get: orderports.GetOrderByIDRequest{
					ID:         orderID,
					TenantInfo: common.TenantFrom(*params),
				},
				params: params.Params,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			edit *orderEdit,
			_ *agentports.ToolExecuteParams,
		) (*orderservice.OrderChange, error) {
			entity, err := edit.entity(ctx, orders)
			if err != nil {
				return nil, err
			}

			return orders.PlanUpdate(ctx, entity)
		},
		Refused: func(*orderEdit) string { return "Would change the order." },
		Render: func(_ *orderEdit, plan *orderservice.OrderChange) (*agent.ToolPreview, error) {
			return renderOrderChange("change")(nil, plan)
		},
		Run: func(
			ctx context.Context,
			edit *orderEdit,
			params *agentports.ToolExecuteParams,
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

	return common.ClassifiedTool[*orderEdit, *orderservice.OrderChange]{
		ReportingReceivableTool: base,
		Egress:                  []agent.EgressClass{agent.EgressInternal, agent.EgressMoney},
		Classify:                classifyOrderAmounts,
	}
}

func classifyOrderAmounts(
	params agentports.ToolExecuteParams, //nolint:gocritic // ToolPolicy.Classify passes params by value
) agentports.CallPolicy {
	if common.SendsAny(params.Params, paramQuotedAmount, paramBaseAmount) {
		return agentports.CallPolicy{Egress: agent.EgressMoney}
	}

	return agentports.CallPolicy{Egress: agent.EgressInternal}
}

func withOrderID(properties map[string]any, what string) map[string]any {
	properties[common.ParamOrderID] = orderIDProperty(what)

	return properties
}

type orderMembership struct {
	tenant      pagination.TenantInfo
	orderID     pulid.ID
	shipmentIDs []pulid.ID
}

func NewAttachOrderShipmentsTool(orders orderKeeper) agentports.AgentTool {
	return common.NewReportingReceivableTool(&common.ReceivableSpec{
		Name:     "order_attach_shipments",
		Artifact: common.OrderRecordEntity,
		Description: "Put shipments on an order so they are billed and tracked together. " +
			"Each must be the order's customer's and neither canceled nor invoiced; a " +
			"shipment on another open order moves off it, and an order it leaves empty is " +
			"removed. A Billed or Closed order takes no more shipments.",
		Resource:    permission.ResourceOrder,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Regroups shipments under an order inside Trenova; nothing is sent, and " +
			"order_detach_shipment moves one back out.",
		Properties: map[string]any{
			common.ParamOrderID: orderIDProperty("The order to add them to"),
			common.ParamShipmentIDs: toolschema.RecordSubset(permission.ResourceShipment.String(),
				agenttoolschema.RecordIDs(
					permission.ResourceShipment,
					"The shipments to attach, from shipment_list or "+
						"shipment_get.",
					maxOrderAttachments,
				)),
		},
		Required: []string{common.ParamOrderID, common.ParamShipmentIDs},
		Target:   targetOrder,
	}, common.ReceivablePlan[*orderMembership, *orderservice.MembershipPlan]{
		Request: func(params *agentports.ToolExecuteParams) (*orderMembership, error) {
			orderID, err := common.RequirePulid(params.Params, common.ParamOrderID)
			if err != nil {
				return nil, err
			}
			shipmentIDs, err := common.RequirePulidSlice(params.Params, common.ParamShipmentIDs,
				maxOrderAttachments)
			if err != nil {
				return nil, err
			}

			return &orderMembership{
				tenant:      common.TenantFrom(*params),
				orderID:     orderID,
				shipmentIDs: shipmentIDs,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *orderMembership,
			_ *agentports.ToolExecuteParams,
		) (*orderservice.MembershipPlan, error) {
			return orders.PlanAttachShipments(ctx, req.tenant, req.orderID, req.shipmentIDs)
		},
		Refused: func(*orderMembership) string {
			return "Would attach shipments to the order."
		},
		Render: renderAttachShipments,
		Run: func(
			ctx context.Context,
			req *orderMembership,
			params *agentports.ToolExecuteParams,
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
				common.ParamOrderID: permission.ResourceOrder,
			}),
		)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}

	summary := fmt.Sprintf("Would attach %s to order %s.",
		common.CountOf(len(plan.ShipmentIDs), "shipment"), plan.Order.OrderNumber)
	if skipped := len(req.shipmentIDs) - len(plan.ShipmentIDs); skipped > 0 {
		summary += fmt.Sprintf(" %d named are already on it.", skipped)
	}
	if len(plan.SourceOrderIDs) > 0 {
		summary += fmt.Sprintf(" They leave %s, which is removed if left empty.",
			common.CountOf(len(plan.SourceOrderIDs), "other order"))
	}

	return toolpreview.Build(summary, changes...), nil
}

type orderLeg struct {
	tenant     pagination.TenantInfo
	orderID    pulid.ID
	shipmentID pulid.ID
}

func NewDetachOrderShipmentTool(orders orderKeeper) agentports.AgentTool {
	return common.NewReportingReceivableTool(&common.ReceivableSpec{
		Name:     "order_detach_shipment",
		Artifact: common.OrderRecordEntity,
		Description: "Take one shipment off an order onto an order of its own, so it is " +
			"billed on its own. The order's only shipment and an invoiced one stay where " +
			"they are.",
		Resource:    permission.ResourceOrder,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Moves a shipment onto a new order inside Trenova; nothing is sent, and " +
			"order_attach_shipments puts it back.",
		Properties: map[string]any{
			common.ParamOrderID:    orderIDProperty("The order the shipment is on"),
			common.ParamShipmentID: common.ShipmentIDProperty("The shipment to take off it"),
		},
		Required: []string{common.ParamOrderID, common.ParamShipmentID},
		Target:   targetOrder,
	}, common.ReceivablePlan[*orderLeg, *orderservice.DetachPlan]{
		Request: func(params *agentports.ToolExecuteParams) (*orderLeg, error) {
			orderID, err := common.RequirePulid(params.Params, common.ParamOrderID)
			if err != nil {
				return nil, err
			}
			shipmentID, err := common.RequirePulid(params.Params, common.ParamShipmentID)
			if err != nil {
				return nil, err
			}

			return &orderLeg{tenant: common.TenantFrom(*params), orderID: orderID,
				shipmentID: shipmentID}, nil
		},
		Plan: func(
			ctx context.Context,
			req *orderLeg,
			_ *agentports.ToolExecuteParams,
		) (*orderservice.DetachPlan, error) {
			return orders.PlanDetachShipment(ctx, req.tenant, req.orderID, req.shipmentID)
		},
		Refused: func(*orderLeg) string { return "Would detach the shipment from the order." },
		Render: func(_ *orderLeg, plan *orderservice.DetachPlan) (*agent.ToolPreview, error) {
			created, err := toolpreview.Create(
				orderRecord(plan.Replacement),
				plan.Replacement,
				toolpreview.Only(
					common.FieldCustomerID,
					common.FieldStatus,
					paramCurrencyCode,
					"totalAmount",
				),
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
		Run: func(
			ctx context.Context,
			req *orderLeg,
			params *agentports.ToolExecuteParams,
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
	params *agentports.ToolExecuteParams,
	reasonRequired bool,
) (*orderOnly, error) {
	orderID, err := common.RequirePulid(params.Params, common.ParamOrderID)
	if err != nil {
		return nil, err
	}
	req := &orderOnly{tenant: common.TenantFrom(*params), orderID: orderID}
	if reasonRequired {
		if req.reason, err = common.RequireBoundedText(params.Params, common.ParamReason,
			common.MaxOperationNoteChars); err != nil {
			return nil, err
		}
	}

	return req, nil
}

func NewCloseOrderTool(orders orderKeeper) agentports.AgentTool {
	return common.NewReportingReceivableTool(&common.ReceivableSpec{
		Name:     "order_close",
		Artifact: common.OrderRecordEntity,
		Description: "Close a Billed order once nothing more will be billed on it. Closing " +
			"is final: a Closed order takes no more shipments or charges.",
		Resource:    permission.ResourceOrder,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		PersonOnly:  true,
		Rationale: "Settles an order for good; nothing reopens it, so only a person " +
			"closes one.",
		Properties: map[string]any{common.ParamOrderID: orderIDProperty("The Billed order")},
		Required:   []string{common.ParamOrderID},
		Target:     targetOrder,
	}, common.ReceivablePlan[*orderOnly, *orderservice.OrderChange]{
		Request: func(params *agentports.ToolExecuteParams) (*orderOnly, error) {
			return readOrderOnly(params, false)
		},
		Plan: func(
			ctx context.Context,
			req *orderOnly,
			_ *agentports.ToolExecuteParams,
		) (*orderservice.OrderChange, error) {
			return orders.PlanClose(ctx, req.tenant, req.orderID)
		},
		Refused: func(*orderOnly) string { return "Would close the order." },
		Render: func(_ *orderOnly, plan *orderservice.OrderChange) (*agent.ToolPreview, error) {
			return renderOrderChange("close")(nil, plan)
		},
		Run: func(
			ctx context.Context,
			req *orderOnly,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			closed, err := orders.Close(ctx, req.tenant, req.orderID, params.Actor)
			if err != nil {
				return nil, err
			}

			return orderResult("closed", closed), nil
		},
	})
}

func NewCancelOrderTool(orders orderKeeper) agentports.AgentTool {
	return common.NewReportingReceivableTool(&common.ReceivableSpec{
		Name:     "order_cancel",
		Artifact: common.OrderRecordEntity,
		Description: "Cancel an order and every shipment on it that is not already " +
			"canceled, with the reason the customer gave. An order with an invoiced " +
			"shipment cannot be canceled; credit the invoice first. Canceling a shipment " +
			"tells its carrier and trading partner as shipment_cancel would.",
		Resource:    permission.ResourceOrder,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressExternalRecipient,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		PersonOnly:  true,
		Rationale: "Cancels every live shipment on the order, which reaches carriers and " +
			"trading partners; only a person cancels an order.",
		Properties: map[string]any{
			common.ParamOrderID: orderIDProperty("The order to cancel"),
			common.ParamReason: common.StringProperty("Why, as the customer said it.",
				common.MaxOperationNoteChars),
		},
		Required: []string{common.ParamOrderID, common.ParamReason},
		Target:   targetOrder,
	}, common.ReceivablePlan[*orderOnly, *orderservice.CancelPlan]{
		Request: func(params *agentports.ToolExecuteParams) (*orderOnly, error) {
			return readOrderOnly(params, true)
		},
		Plan: func(
			ctx context.Context,
			req *orderOnly,
			_ *agentports.ToolExecuteParams,
		) (*orderservice.CancelPlan, error) {
			return orders.PlanCancel(ctx, req.tenant, req.orderID)
		},
		Refused: func(*orderOnly) string { return "Would cancel the order." },
		Render:  renderCancelOrder,
		Run: func(
			ctx context.Context,
			req *orderOnly,
			params *agentports.ToolExecuteParams,
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
			common.ShipmentRecord(leg),
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
		common.CountOf(len(plan.Legs), "live shipment"),
		req.reason,
	), changes...), nil
}
