package shipmentinputmapping

import (
	"fmt"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/shipmentmapping"
	shipmentdomain "github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/pkg/authctx"
)

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
	freight, err := shipmentmapping.ChargeAllocationsFromInput(
		input.FreightAllocations,
		shipmentvocab.ChargeAllocationKindFreight,
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
		accessorial, chargeErr := shipmentmapping.ChargeAllocationsFromInput(
			charge.Allocations,
			shipmentvocab.ChargeAllocationKindAccessorial,
			fmt.Sprintf("additionalCharges[%d].allocations", idx),
			authCtx,
			&index,
		)
		if chargeErr != nil {
			return nil, chargeErr
		}
		for _, row := range accessorial {
			if charge.ID != nil {
				chargeID, idErr := gqlid.OptionalScopedID(
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
