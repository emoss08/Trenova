package resolver

import (
	"fmt"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	shipmentdomain "github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// chargeAllocationsFromInput turns one charge's allocation rows into domain rows
// of the given kind. Accessorial rows point at their charge by position, which the
// repository resolves once the charge has an id.
func chargeAllocationsFromInput(
	inputs []*gqlmodel.ChargeAllocationInput,
	kind shipmentdomain.ChargeAllocationKind,
	path string,
	authCtx *authctx.AuthContext,
	chargeIndex *int,
) ([]*shipmentdomain.ChargeAllocation, error) {
	rows := make([]*shipmentdomain.ChargeAllocation, 0, len(inputs))
	for idx, input := range inputs {
		if input == nil {
			continue
		}
		rowPath := fmt.Sprintf("%s[%d]", path, idx)
		id, err := optionalScopedID(rowPath+".id", input.ID)
		if err != nil {
			return nil, err
		}
		billTo, err := requiredID(rowPath+".billToCustomerId", input.BillToCustomerID)
		if err != nil {
			return nil, err
		}

		row := &shipmentdomain.ChargeAllocation{
			ID:               id,
			OrganizationID:   authCtx.OrganizationID,
			BusinessUnitID:   authCtx.BusinessUnitID,
			ChargeKind:       kind,
			BillToCustomerID: billTo,
			Method:           input.Method,
		}
		if input.Sequence != nil {
			if *input.Sequence < 0 || *input.Sequence > 32767 {
				return nil, errortypes.NewValidationError(
					rowPath+".sequence",
					errortypes.ErrInvalid,
					"Sequence must be between 0 and 32767",
				)
			}
			row.Sequence = int16(*input.Sequence)
		}
		if input.Version != nil {
			row.Version = int64(*input.Version)
		}
		if input.Percent != nil {
			percent, parseErr := decimal.NewFromString(*input.Percent)
			if parseErr != nil {
				return nil, errortypes.NewValidationError(
					rowPath+".percent",
					errortypes.ErrInvalid,
					"Percent must be a valid number",
				)
			}
			row.Percent = decimal.NewNullDecimal(percent)
		}
		if input.Amount != nil {
			amount, parseErr := decimal.NewFromString(*input.Amount)
			if parseErr != nil {
				return nil, errortypes.NewValidationError(
					rowPath+".amount",
					errortypes.ErrInvalid,
					"Amount must be a valid number",
				)
			}
			row.Amount = decimal.NewNullDecimal(amount)
		}
		if kind == shipmentdomain.ChargeAllocationKindAccessorial && chargeIndex != nil {
			index := *chargeIndex
			row.AdditionalChargeIndex = &index
		}
		rows = append(rows, row)
	}

	return rows, nil
}

// shipmentChargeAllocationsFromInput flattens the nested allocation inputs of a
// shipment save into the single list the repository reconciles. It returns nil
// when nothing was sent, which leaves stored allocations untouched.
func shipmentChargeAllocationsFromInput(
	input *gqlmodel.ShipmentInput,
	authCtx *authctx.AuthContext,
) ([]*shipmentdomain.ChargeAllocation, error) {
	provided := input.FreightAllocations != nil
	for _, charge := range input.AdditionalCharges {
		if charge != nil && charge.Allocations != nil {
			provided = true
			break
		}
	}
	if !provided {
		return nil, nil
	}

	rows := make([]*shipmentdomain.ChargeAllocation, 0, len(input.FreightAllocations))
	freight, err := chargeAllocationsFromInput(
		input.FreightAllocations,
		shipmentdomain.ChargeAllocationKindFreight,
		"freightAllocations",
		authCtx,
		nil,
	)
	if err != nil {
		return nil, err
	}
	rows = append(rows, freight...)

	for idx, charge := range input.AdditionalCharges {
		if charge == nil || charge.Allocations == nil {
			continue
		}
		index := idx
		accessorial, chargeErr := chargeAllocationsFromInput(
			charge.Allocations,
			shipmentdomain.ChargeAllocationKindAccessorial,
			fmt.Sprintf("additionalCharges[%d].allocations", idx),
			authCtx,
			&index,
		)
		if chargeErr != nil {
			return nil, chargeErr
		}
		for _, row := range accessorial {
			if charge.ID != nil {
				chargeID, idErr := optionalScopedID(
					fmt.Sprintf("additionalCharges[%d].id", idx),
					charge.ID,
				)
				if idErr != nil {
					return nil, idErr
				}
				if chargeID.IsNotNil() {
					row.AdditionalChargeID = &chargeID
					row.AdditionalChargeIndex = nil
				}
			}
		}
		rows = append(rows, accessorial...)
	}

	return rows, nil
}

// billingSplitSummaryToModel reports what each payer owes on a shipment loaded
// with its charges. A shipment read without them returns nothing rather than a
// freight-only figure that would understate every payer.
func billingSplitSummaryToModel(
	entity *shipmentdomain.Shipment,
) []*gqlmodel.ShipmentBillingSplitSummary {
	if entity == nil || entity.AdditionalCharges == nil {
		return []*gqlmodel.ShipmentBillingSplitSummary{}
	}

	resolution, err := shipmentdomain.ResolveShares(entity, entity.ChargeAllocations)
	if err != nil || resolution == nil {
		return []*gqlmodel.ShipmentBillingSplitSummary{}
	}

	names := payerNames(entity)
	rows := make([]*gqlmodel.ShipmentBillingSplitSummary, 0, len(resolution.Shares))
	for _, share := range resolution.Shares {
		if share == nil {
			continue
		}
		name, code := names[share.PayerID].name, names[share.PayerID].code
		rows = append(rows, &gqlmodel.ShipmentBillingSplitSummary{
			PayerID:           share.PayerID.String(),
			PayerName:         name,
			PayerCode:         code,
			IsPrimary:         share.PayerID == resolution.DefaultPayerID,
			FreightAmount:     share.FreightAmount.StringFixed(shipmentdomain.SharePlaces),
			AccessorialAmount: share.AccessorialAmount.StringFixed(shipmentdomain.SharePlaces),
			TotalAmount:       share.TotalAmount.StringFixed(shipmentdomain.SharePlaces),
			IsSplit:           resolution.IsSplit,
		})
	}

	return rows
}

type payerLabel struct {
	name string
	code string
}

// payerNames collects every customer already loaded alongside the shipment, so
// the summary can label payers without a second read.
func payerNames(entity *shipmentdomain.Shipment) map[pulid.ID]payerLabel {
	labels := make(map[pulid.ID]payerLabel, 2+len(entity.ChargeAllocations))
	add := func(cus *customer.Customer) {
		if cus == nil || cus.ID.IsNil() {
			return
		}
		labels[cus.ID] = payerLabel{name: cus.Name, code: cus.Code}
	}
	add(entity.Customer)
	add(entity.BillToCustomer)
	for _, allocation := range entity.ChargeAllocations {
		if allocation != nil {
			add(allocation.BillToCustomer)
		}
	}

	return labels
}
