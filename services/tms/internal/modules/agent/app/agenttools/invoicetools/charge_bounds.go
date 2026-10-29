package invoicetools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/operations/domain/chargeallocation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const correctChargeBoundText = "Runs on its own only while nothing has been invoiced from the " +
	"billing queue item and every charge it adds, removes or changes, counted in full, is " +
	"within the organization's money limits for agents; any other call waits for a person."

func (t *CorrectChargeCodeTool) withinBounds(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the TierCondition bounds pass params by value
) agenttoolports.MoneyBound {
	if t.billing == nil || t.profiles == nil {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
	}
	request, err := t.request(&params)
	if err != nil {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
	}
	plan, err := t.billing.PreviewUpdateCharges(ctx, request, params.Actor)
	if err != nil || plan == nil || plan.Item == nil || plan.Before == nil || plan.After == nil {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
	}
	if !uninvoiced(plan.Item) {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyRecordState)
	}
	if plan.Item.BillToCustomerID.IsNil() {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
	}
	profile, err := t.profiles.GetBillingProfile(
		ctx,
		customerports.GetCustomerBillingProfileRequest{
			CustomerID: plan.Item.BillToCustomerID,
			TenantInfo: pagination.TenantInfo{
				OrgID: params.OrganizationID,
				BuID:  params.BusinessUnitID,
			},
		},
	)
	if err != nil || profile == nil {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
	}
	payer := customer.Customer{BillingProfile: profile}

	gross, err := grossChargeChange(ctx, plan.Before, plan.After)
	if err != nil {
		return agenttoolports.OutsideMoneyBound(agenttooltrust.HeldByMoneyUnproven)
	}

	return agenttoolports.WithinMoneyBound(money.MinorUnits(gross), payer.BillingCurrencyCode())
}

func grossChargeChange(
	ctx context.Context,
	before, after *shipment.Shipment,
) (decimal.Decimal, error) {
	beforeFreight := before.FreightChargeAmount.Decimal
	afterFreight := after.FreightChargeAmount.Decimal
	gross := afterFreight.Sub(beforeFreight).Abs()
	totalsBefore, err := chargeallocation.ChargeTotalsByCharge(ctx, before)
	if err != nil {
		return decimal.Zero, err
	}
	totalsAfter, err := chargeallocation.ChargeTotalsByCharge(ctx, after)
	if err != nil {
		return decimal.Zero, err
	}

	kept := make(map[pulid.ID]*shipment.AdditionalCharge, len(before.AdditionalCharges))
	for _, charge := range before.AdditionalCharges {
		if charge != nil && charge.ID.IsNotNil() {
			kept[charge.ID] = charge
		}
	}
	for _, charge := range after.AdditionalCharges {
		if charge == nil {
			continue
		}
		total := totalsAfter[charge]
		previous, found := kept[charge.ID]
		if !found || charge.ID.IsNil() {
			gross = gross.Add(total.Abs())
			continue
		}
		delete(kept, charge.ID)
		gross = gross.Add(total.Sub(totalsBefore[previous]).Abs())
	}
	for _, charge := range before.AdditionalCharges {
		if charge == nil {
			continue
		}
		if _, removed := kept[charge.ID]; removed || charge.ID.IsNil() {
			gross = gross.Add(totalsBefore[charge].Abs())
		}
	}

	return gross, nil
}

func uninvoiced(item *billingqueue.BillingQueueItem) bool {
	return item.Status == billingqueue.StatusInReview &&
		item.InvoiceID.IsNil() &&
		(item.SourceInvoiceID == nil || item.SourceInvoiceID.IsNil()) &&
		(item.SourceInvoiceAdjustmentID == nil || item.SourceInvoiceAdjustmentID.IsNil()) &&
		(item.SourceCreditMemoInvoiceID == nil || item.SourceCreditMemoInvoiceID.IsNil())
}
