package billingqueueresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/billingqueuemapping"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/shipmentinputmapping"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/shipmentmapping"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/tariffmapping"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	defaultActivityLimit = 30
	maxActivityLimit     = 100
)

func activityRequest(
	id string,
	limit *int,
	before *gqlmodel.BillingQueueActivityCursorInput,
	tenantInfo pagination.TenantInfo,
) (*billingqueueports.ListBillingQueueActivityRequest, error) {
	itemID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return nil, err
	}

	req := &billingqueueports.ListBillingQueueActivityRequest{
		ItemID:     itemID,
		TenantInfo: tenantInfo,
		Limit:      defaultActivityLimit,
	}
	if limit != nil && *limit > 0 {
		req.Limit = min(*limit, maxActivityLimit)
	}
	if before != nil {
		beforeID, parseErr := gqlid.ParseIDField("before.id", before.ID)
		if parseErr != nil {
			return nil, parseErr
		}
		req.BeforeAt = int64(before.At)
		req.BeforeID = beforeID
	}

	return req, nil
}

func updateChargesRequest(
	id string,
	input *gqlmodel.BillingQueueUpdateChargesInput,
	authCtx *authctx.AuthContext,
) (*billingqueueports.UpdateChargesRequest, error) {
	itemID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return nil, err
	}
	formulaTemplateID, err := gqlid.OptionalPulid(input.FormulaTemplateID)
	if err != nil {
		return nil, gqlid.InvalidIDError("formulaTemplateId", "Invalid formula template")
	}

	req := &billingqueueports.UpdateChargesRequest{
		ItemID:                       itemID,
		FormulaTemplateID:            formulaTemplateID,
		TenantInfo:                   resolvercore.TenantInfo(authCtx),
		ConvertAmountSplitsToPercent: gqlscalar.BoolValue(input.ConvertAmountSplitsToPercent),
	}
	if input.TariffInputs != nil {
		inputs, inputErr := tariffmapping.TariffInputsFromModel("tariffInputs", input.TariffInputs)
		if inputErr != nil {
			return nil, inputErr
		}
		req.TariffInputs = inputs
	}
	if input.AdditionalCharges != nil {
		charges, chargeErr := shipmentinputmapping.ShipmentAdditionalChargesFromInput(
			input.AdditionalCharges,
			authCtx,
		)
		if chargeErr != nil {
			return nil, chargeErr
		}
		req.AdditionalCharges = charges
	}

	return req, nil
}

func invoiceDetailsRequest(
	id string,
	input *gqlmodel.BillingQueueInvoiceDetailsInput,
	tenantInfo pagination.TenantInfo,
) (*billingqueueports.UpdateInvoiceDetailsRequest, error) {
	itemID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return nil, err
	}
	packetDocumentIDs, err := gqlid.ParseIDListField("packetDocumentIds", input.PacketDocumentIds)
	if err != nil {
		return nil, err
	}

	return &billingqueueports.UpdateInvoiceDetailsRequest{
		ItemID:            itemID,
		TenantInfo:        tenantInfo,
		Version:           int64(input.Version),
		PONumber:          gqlscalar.StringValue(input.PoNumber),
		InvoiceMemo:       gqlscalar.StringValue(input.InvoiceMemo),
		PaymentTerm:       gqlscalar.StringValue(input.PaymentTerm),
		InvoiceDate:       gqlscalar.Int64Ptr(input.InvoiceDate),
		PacketDocumentIDs: packetDocumentIDs,
	}, nil
}

func reassignChargeRequest(
	id string,
	input *gqlmodel.BillingQueueReassignChargeInput,
	authCtx *authctx.AuthContext,
) (*billingqueueports.ReassignChargeRequest, error) {
	itemID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return nil, err
	}
	if !input.ChargeKind.IsValid() {
		return nil, errortypes.NewValidationError(
			"chargeKind",
			errortypes.ErrInvalid,
			"Charge kind is invalid",
		)
	}
	additionalChargeID, err := gqlid.OptionalID(input.AdditionalChargeID)
	if err != nil {
		return nil, gqlid.InvalidIDError("additionalChargeId", "Invalid charge")
	}
	allocations, err := shipmentmapping.ChargeAllocationsFromInput(
		input.Allocations,
		input.ChargeKind,
		"allocations",
		authCtx,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &billingqueueports.ReassignChargeRequest{
		ItemID:             itemID,
		TenantInfo:         resolvercore.TenantInfo(authCtx),
		ChargeKind:         input.ChargeKind,
		AdditionalChargeID: additionalChargeID,
		Allocations:        allocations,
	}, nil
}

func reassignChargeResultToModel(
	ctx context.Context,
	result *billingqueueports.ReassignChargeResult,
) (*gqlmodel.BillingQueueReassignChargeResult, error) {
	item, err := billingqueuemapping.RequiredBillingQueueItemToModel(ctx, result.Item)
	if err != nil {
		return nil, err
	}
	items, err := billingqueuemapping.BillingQueueItemsToModel(ctx, result.Items)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.BillingQueueReassignChargeResult{
		Item:            item,
		Items:           items,
		CreatedItemIds:  gqlid.PulidsToStrings(result.CreatedItemIDs),
		CanceledItemIds: gqlid.PulidsToStrings(result.CanceledItemIDs),
	}, nil
}
