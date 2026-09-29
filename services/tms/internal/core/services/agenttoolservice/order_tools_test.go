package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/orderservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeOrders struct {
	guard   *writeGuard
	refusal error
	current *order.Order
	charge  *order.OrderCharge
	legs    []*shipment.Shipment

	created    *order.Order
	updated    *order.Order
	attached   []pulid.ID
	detached   pulid.ID
	closed     pulid.ID
	canceled   string
	added      *orderservice.AddChargeRequest
	edited     *orderservice.UpdateChargeRequest
	allocated  *orderservice.SetChargeAllocationsRequest
	removed    pulid.ID
	lastTenant pagination.TenantInfo
}

func newFakeOrders() *fakeOrders {
	orderID := pulid.MustNew("ord_")
	payer := pulid.MustNew("cus_")

	return &fakeOrders{
		guard: &writeGuard{},
		current: &order.Order{
			ID:           orderID,
			OrderNumber:  "ORD-1001",
			CustomerID:   pulid.MustNew("cus_"),
			Status:       order.StatusConfirmed,
			CurrencyCode: "USD",
			PONumber:     "PO-1",
			Version:      6,
		},
		charge: &order.OrderCharge{
			ID:          pulid.MustNew("ordchg_"),
			OrderID:     orderID,
			Description: "Consolidation fee",
			Amount:      decimal.RequireFromString("150"),
			Version:     2,
			Allocations: []*shipment.ChargeAllocation{{
				ID:               pulid.MustNew("chal_"),
				BillToCustomerID: payer,
				Method:           shipment.ChargeAllocationMethodPercent,
				Percent:          decimal.NewNullDecimal(decimal.RequireFromString("100")),
				Version:          3,
			}},
		},
	}
}

func (f *fakeOrders) write() error { return f.guard.write() }

func (f *fakeOrders) Get(context.Context, repositories.GetOrderByIDRequest) (*order.Order, error) {
	copied := *f.current

	return &copied, nil
}

func (f *fakeOrders) Create(
	_ context.Context,
	entity *order.Order,
	_ *serviceports.RequestActor,
) (*order.Order, error) {
	if err := f.write(); err != nil {
		return nil, err
	}
	f.created = entity

	return entity, nil
}

func (f *fakeOrders) PlanCreate(
	_ context.Context,
	entity *order.Order,
) (*orderservice.OrderChange, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}

	return &orderservice.OrderChange{After: entity}, nil
}

func (f *fakeOrders) Update(
	_ context.Context,
	entity *order.Order,
	_ *serviceports.RequestActor,
) (*order.Order, error) {
	if err := f.write(); err != nil {
		return nil, err
	}
	f.updated = entity

	return entity, nil
}

func (f *fakeOrders) PlanUpdate(
	_ context.Context,
	entity *order.Order,
) (*orderservice.OrderChange, error) {
	before := *f.current

	return &orderservice.OrderChange{Before: &before, After: entity}, nil
}

func (f *fakeOrders) AttachShipments(
	_ context.Context,
	tenant pagination.TenantInfo,
	_ pulid.ID,
	ids []pulid.ID,
	_ *serviceports.RequestActor,
) (*order.Order, error) {
	if err := f.write(); err != nil {
		return nil, err
	}
	f.attached = ids
	f.lastTenant = tenant

	return f.current, nil
}

func (f *fakeOrders) PlanAttachShipments(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	ids []pulid.ID,
) (*orderservice.MembershipPlan, error) {
	return &orderservice.MembershipPlan{
		Order:          f.current,
		ShipmentIDs:    ids[1:],
		SourceOrderIDs: []pulid.ID{pulid.MustNew("ord_")},
	}, nil
}

func (f *fakeOrders) DetachShipment(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	shipmentID pulid.ID,
	_ *serviceports.RequestActor,
) (*order.Order, error) {
	if err := f.write(); err != nil {
		return nil, err
	}
	f.detached = shipmentID

	return f.current, nil
}

func (f *fakeOrders) PlanDetachShipment(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	shipmentID pulid.ID,
) (*orderservice.DetachPlan, error) {
	leg := &shipment.Shipment{ID: shipmentID, ProNumber: "PRO-9", CustomerID: f.current.CustomerID}

	return &orderservice.DetachPlan{
		Order: f.current,
		Leg:   leg,
		Replacement: &order.Order{
			CustomerID:   leg.CustomerID,
			Status:       order.StatusConfirmed,
			CurrencyCode: "USD",
		},
	}, nil
}

func (f *fakeOrders) Close(
	_ context.Context,
	_ pagination.TenantInfo,
	orderID pulid.ID,
	_ *serviceports.RequestActor,
) (*order.Order, error) {
	if err := f.write(); err != nil {
		return nil, err
	}
	f.closed = orderID

	return f.current, nil
}

func (f *fakeOrders) PlanClose(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*orderservice.OrderChange, error) {
	if f.refusal != nil {
		return nil, f.refusal
	}
	before := *f.current
	after := before
	after.Status = order.StatusClosed

	return &orderservice.OrderChange{Before: &before, After: &after}, nil
}

func (f *fakeOrders) Cancel(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	reason string,
	_ *serviceports.RequestActor,
) (*order.Order, error) {
	if err := f.write(); err != nil {
		return nil, err
	}
	f.canceled = reason

	return f.current, nil
}

func (f *fakeOrders) PlanCancel(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*orderservice.CancelPlan, error) {
	return &orderservice.CancelPlan{Order: f.current, Legs: f.legs}, nil
}

func (f *fakeOrders) ListCharges(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) ([]*order.OrderCharge, error) {
	return []*order.OrderCharge{f.charge}, nil
}

func (f *fakeOrders) AddChargeWithAllocations(
	_ context.Context,
	req *orderservice.AddChargeRequest,
	_ *serviceports.RequestActor,
) (*order.Order, error) {
	if err := f.write(); err != nil {
		return nil, err
	}
	f.added = req

	return f.current, nil
}

func (f *fakeOrders) PlanAddCharge(
	_ context.Context,
	req *orderservice.AddChargeRequest,
) (*orderservice.ChargePlan, error) {
	return &orderservice.ChargePlan{Order: f.current, After: &order.OrderCharge{
		Description: req.Description,
		Amount:      req.Amount,
		Allocations: req.Allocations,
	}}, nil
}

func (f *fakeOrders) UpdateCharge(
	_ context.Context,
	req *orderservice.UpdateChargeRequest,
	_ *serviceports.RequestActor,
) (*order.Order, error) {
	if err := f.write(); err != nil {
		return nil, err
	}
	f.edited = req

	return f.current, nil
}

func (f *fakeOrders) PlanUpdateCharge(
	_ context.Context,
	req *orderservice.UpdateChargeRequest,
) (*orderservice.ChargePlan, error) {
	after := *f.charge
	after.Description = req.Description
	after.Amount = req.Amount
	if req.Allocations != nil {
		after.Allocations = req.Allocations
	}

	return &orderservice.ChargePlan{Order: f.current, Before: f.charge, After: &after}, nil
}

func (f *fakeOrders) SetChargeAllocations(
	_ context.Context,
	req *orderservice.SetChargeAllocationsRequest,
	_ *serviceports.RequestActor,
) (*order.Order, error) {
	if err := f.write(); err != nil {
		return nil, err
	}
	f.allocated = req

	return f.current, nil
}

func (f *fakeOrders) PlanSetChargeAllocations(
	_ context.Context,
	req *orderservice.SetChargeAllocationsRequest,
) (*orderservice.ChargePlan, error) {
	after := *f.charge
	after.Allocations = req.Allocations

	return &orderservice.ChargePlan{Order: f.current, Before: f.charge, After: &after}, nil
}

func (f *fakeOrders) RemoveCharge(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
	chargeID pulid.ID,
	_ *serviceports.RequestActor,
) (*order.Order, error) {
	if err := f.write(); err != nil {
		return nil, err
	}
	f.removed = chargeID

	return f.current, nil
}

func (f *fakeOrders) PlanRemoveCharge(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
	pulid.ID,
) (*orderservice.ChargePlan, error) {
	return &orderservice.ChargePlan{Order: f.current, Before: f.charge}, nil
}

func TestCreateOrder_PreviewsTheDraftAndCreates(t *testing.T) {
	t.Parallel()

	orders := newFakeOrders()
	tool := newCreateOrderTool(orders)
	customerID := pulid.MustNew("cus_")
	params := executeParams(map[string]any{
		fieldCustomerID:   customerID.String(),
		paramPONumber:     "PO-77",
		paramQuotedAmount: "2400.00",
		paramCurrencyCode: "cad",
	})

	preview := previewWithoutWrites(t, orders.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	change := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, change.Operation)
	assert.Equal(t, "PO-77", fieldByPath(t, change, paramPONumber).After)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, orders.created)
	assert.Equal(t, customerID, orders.created.CustomerID)
	assert.Equal(t, "CAD", orders.created.CurrencyCode)
	assert.Equal(t, order.StatusDraft, orders.created.Status)
	assert.Equal(t, params.Actor.UserID, orders.created.EnteredByID)
	assert.True(t, orders.created.QuotedAmount.Decimal.Equal(decimal.RequireFromString("2400")))

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceOrder, policy.Resource)
	assert.Equal(t, permission.OpCreate, policy.Operation)
	assert.Equal(t, agent.TierActWithApproval, policy.MaxTier)
}

func TestCreateOrder_RefusesBadArguments(t *testing.T) {
	t.Parallel()

	tool := newCreateOrderTool(newFakeOrders())
	for name, raw := range map[string]map[string]any{
		"no customer":        {paramPONumber: "PO-1"},
		"a bad customer":     {fieldCustomerID: "acme"},
		"a negative quote":   {fieldCustomerID: pulid.MustNew("cus_").String(), paramQuotedAmount: "-5"},
		"a long currency":    {fieldCustomerID: pulid.MustNew("cus_").String(), paramCurrencyCode: "DOLLARS"},
		"a non-number quote": {fieldCustomerID: pulid.MustNew("cus_").String(), paramQuotedAmount: "lots"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
				executeParams(raw)))
		})
	}
}

func TestUpdateOrder_ChangesOnlyWhatIsSentAndIsMoneyOnlyForAmounts(t *testing.T) {
	t.Parallel()

	orders := newFakeOrders()
	tool := newUpdateOrderTool(orders)
	params := executeParams(map[string]any{
		paramOrderID: orders.current.ID.String(),
		fieldBol:     "BOL-9",
	})

	preview := previewWithoutWrites(t, orders.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	change := previewChange(t, preview, 0)
	assert.Equal(t, "BOL-9", fieldByPath(t, change, fieldBol).After)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, orders.updated)
	assert.Equal(t, "PO-1", orders.updated.PONumber, "an unsent field keeps its value")
	assert.Equal(t, int64(6), orders.updated.Version)

	policy := tool.Policy()
	assert.Equal(t, agent.EgressInternal, policy.Classify(params).Egress)
	assert.Equal(t, agent.EgressMoney, policy.Classify(executeParams(map[string]any{
		paramBaseAmount: "10",
	})).Egress)
	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceOrder, target.Resource)

	require.ErrorIs(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{paramOrderID: orders.current.ID.String()})),
		errNothingToChange)
}

func TestAttachOrderShipments_ShowsOnlyTheShipmentsThatMove(t *testing.T) {
	t.Parallel()

	orders := newFakeOrders()
	tool := newAttachOrderShipmentsTool(orders)
	ids := []any{pulid.MustNew("shp_").String(), pulid.MustNew("shp_").String()}
	params := executeParams(map[string]any{
		paramOrderID:     orders.current.ID.String(),
		paramShipmentIDs: ids,
	})

	preview := previewWithoutWrites(t, orders.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	require.Len(t, preview.Changes, 1)
	assert.Contains(t, preview.Summary, "1 named are already on it")
	assert.Contains(t, preview.Summary, "1 other order")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.Len(t, orders.attached, 2)
	assert.Equal(t, params.OrganizationID, orders.lastTenant.OrgID)

	properties := tool.ParamSchema()[toolschema.KeyProperties].(map[string]any)
	assert.Equal(t, permission.ResourceShipment.String(),
		properties[paramShipmentIDs].(map[string]any)[toolschema.KeySubsetOf])
}

func TestDetachOrderShipment_ShowsTheOrderItMovesTo(t *testing.T) {
	t.Parallel()

	orders := newFakeOrders()
	tool := newDetachOrderShipmentTool(orders)
	shipmentID := pulid.MustNew("shp_")
	params := executeParams(map[string]any{
		paramOrderID:    orders.current.ID.String(),
		paramShipmentID: shipmentID.String(),
	})

	preview := previewWithoutWrites(t, orders.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, agent.PreviewOperationCreate, previewChange(t, preview, 0).Operation)
	assert.Contains(t, preview.Summary, "PRO-9")

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, shipmentID, orders.detached)
}

func TestCloseOrder_IsAPersonsAndRefusesAnOrderNotBilled(t *testing.T) {
	t.Parallel()

	orders := newFakeOrders()
	tool := newCloseOrderTool(orders)
	params := executeParams(map[string]any{paramOrderID: orders.current.ID.String()})

	preview := previewWithoutWrites(t, orders.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, string(order.StatusClosed), fieldByPath(t, previewChange(t, preview, 0),
		fieldStatus).After)

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrNeedsAPersonsApproval)
	require.NoError(t, tool.Execute(t.Context(), approvedParams(params.Params)))
	assert.Equal(t, orders.current.ID, orders.closed)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)

	refused := newFakeOrders()
	refused.refusal = errortypes.NewValidationError("orderId", errortypes.ErrInvalidOperation,
		"Only a Billed order can be closed")
	closer := newCloseOrderTool(refused)
	refusedPreview, err := closer.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	requireWarning(t, refusedPreview, agent.PreviewWarningWouldFail)
	require.Error(t, closer.(serviceports.ToolValidator).Validate(t.Context(), params))
}

func TestCancelOrder_NamesEveryLiveShipmentAndNeedsAReason(t *testing.T) {
	t.Parallel()

	orders := newFakeOrders()
	orders.legs = []*shipment.Shipment{
		{ID: pulid.MustNew("shp_"), ProNumber: "PRO-1", Status: shipment.StatusNew},
		{ID: pulid.MustNew("shp_"), ProNumber: "PRO-2", Status: shipment.StatusInTransit},
	}
	tool := newCancelOrderTool(orders)
	params := approvedParams(map[string]any{
		paramOrderID:      orders.current.ID.String(),
		paramCancelReason: "Customer canceled the PO",
	})

	preview := previewWithoutWrites(t, orders.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	require.Len(t, preview.Changes, 2)
	assert.Contains(t, preview.Summary, "2 live shipments")

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, "Customer canceled the PO", orders.canceled)
	assert.Equal(t, []agent.EgressClass{agent.EgressExternalRecipient}, tool.Policy().Egress)

	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{paramOrderID: orders.current.ID.String()})))
}

func TestAddOrderCharge_ShowsTheMoneyAndReadsTheSplit(t *testing.T) {
	t.Parallel()

	orders := newFakeOrders()
	tool := newAddOrderChargeTool(orders)
	payerA, payerB := pulid.MustNew("cus_"), pulid.MustNew("cus_")
	params := executeParams(map[string]any{
		paramOrderID:     orders.current.ID.String(),
		fieldDescription: "Lumper",
		paramAmount:      "200.00",
		paramAllocations: []any{
			map[string]any{paramBillToCustomer: payerA.String(), paramAllocMethod: "Percent",
				paramAllocPercent: "60"},
			map[string]any{paramBillToCustomer: payerB.String(), paramAllocMethod: "Percent",
				paramAllocPercent: "40"},
		},
	})

	preview := previewWithoutWrites(t, orders.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	change := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationCreate, change.Operation)
	require.NotNil(t, change.Money)
	assert.Equal(t, "2 payers: 60% / 40%", fieldByPath(t, change, "split").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, orders.added)
	require.Len(t, orders.added.Allocations, 2)
	assert.Equal(t, payerB, orders.added.Allocations[1].BillToCustomerID)
	assert.Equal(t, int16(1), orders.added.Allocations[1].Sequence)
	assert.Equal(t, shipment.ChargeAllocationKindOrderCharge, orders.added.Allocations[0].ChargeKind)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, tool.Policy().Egress)
}

func TestAddOrderCharge_RefusesASplitThatDoesNotSayHow(t *testing.T) {
	t.Parallel()

	tool := newAddOrderChargeTool(newFakeOrders())
	payer := pulid.MustNew("cus_").String()
	base := func(row map[string]any) map[string]any {
		return map[string]any{
			paramOrderID:     pulid.MustNew("ord_").String(),
			fieldDescription: "Fee",
			paramAmount:      "10",
			paramAllocations: []any{row},
		}
	}
	for name, raw := range map[string]map[string]any{
		"percent without a percent": base(map[string]any{paramBillToCustomer: payer,
			paramAllocMethod: "Percent", paramAmount: "10"}),
		"an unknown method": base(map[string]any{paramBillToCustomer: payer,
			paramAllocMethod: "Share", paramAllocPercent: "10"}),
		"no payer": base(map[string]any{paramAllocMethod: "Amount", paramAmount: "10"}),
		"no amount": {paramOrderID: pulid.MustNew("ord_").String(),
			fieldDescription: "Fee"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
				executeParams(raw)))
		})
	}
}

func TestUpdateOrderCharge_KeepsWhatIsNotSentAndThePayersRow(t *testing.T) {
	t.Parallel()

	orders := newFakeOrders()
	tool := newUpdateOrderChargeTool(orders)
	payer := orders.charge.Allocations[0].BillToCustomerID
	params := executeParams(map[string]any{
		paramOrderID:  orders.current.ID.String(),
		paramChargeID: orders.charge.ID.String(),
		paramAmount:   "175.00",
		paramAllocations: []any{map[string]any{
			paramBillToCustomer: payer.String(), paramAllocMethod: "Amount", paramAmount: "175.00",
		}},
	})

	preview := previewWithoutWrites(t, orders.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	require.NotNil(t, previewChange(t, preview, 0).Money)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, orders.edited)
	assert.Equal(t, "Consolidation fee", orders.edited.Description)
	assert.True(t, orders.edited.Amount.Equal(decimal.RequireFromString("175")))
	assert.Equal(t, int64(2), orders.edited.Version)
	require.Len(t, orders.edited.Allocations, 1)
	assert.Equal(t, orders.charge.Allocations[0].ID, orders.edited.Allocations[0].ID)
	assert.Equal(t, int64(3), orders.edited.Allocations[0].Version)

	require.ErrorIs(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{
			paramOrderID:  orders.current.ID.String(),
			paramChargeID: orders.charge.ID.String(),
		})), errNothingToChange)
}

func TestSetOrderChargeAllocations_AnEmptySplitBillsTheOrdersCustomer(t *testing.T) {
	t.Parallel()

	orders := newFakeOrders()
	tool := newSetOrderChargeAllocationsTool(orders)
	params := executeParams(map[string]any{
		paramOrderID:     orders.current.ID.String(),
		paramChargeID:    orders.charge.ID.String(),
		paramAllocations: []any{},
	})

	preview := previewWithoutWrites(t, orders.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, "Billed to the order's customer",
		fieldByPath(t, previewChange(t, preview, 0), "split").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, orders.allocated)
	assert.NotNil(t, orders.allocated.Allocations)
	assert.Empty(t, orders.allocated.Allocations)
}

func TestRemoveOrderCharge_PreviewsTheDeletion(t *testing.T) {
	t.Parallel()

	orders := newFakeOrders()
	tool := newRemoveOrderChargeTool(orders)
	params := executeParams(map[string]any{
		paramOrderID:  orders.current.ID.String(),
		paramChargeID: orders.charge.ID.String(),
	})

	preview := previewWithoutWrites(t, orders.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, agent.PreviewOperationDelete, previewChange(t, preview, 0).Operation)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, orders.charge.ID, orders.removed)
}
