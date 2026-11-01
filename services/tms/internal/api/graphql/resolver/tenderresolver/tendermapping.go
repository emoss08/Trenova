package tenderresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendercreate"
)

func spotTenderRequestFromInput(
	input *gqlmodel.CreateSpotTenderInput,
) (*tendercreate.CreateSpotTenderRequest, error) {
	moveID, err := gqlid.ParseIDField("shipmentMoveId", input.ShipmentMoveID)
	if err != nil {
		return nil, err
	}

	lines := make([]tendercreate.SpotTenderLine, 0, len(input.Lines))
	for _, line := range input.Lines {
		if line == nil {
			continue
		}
		carrierID, lineErr := gqlid.ParseIDField("lines.carrierId", line.CarrierID)
		if lineErr != nil {
			return nil, lineErr
		}
		rate, lineErr := gqldecimal.ParseDecimalField("lines.rate", line.Rate, true)
		if lineErr != nil {
			return nil, lineErr
		}
		lines = append(lines, tendercreate.SpotTenderLine{
			CarrierID:       carrierID,
			RateMethod:      line.RateMethod,
			Rate:            rate,
			OfferTTLSeconds: clampTTLSeconds(line.OfferTTLSeconds),
			Channel:         line.Channel,
			Email:           gqlscalar.StringValue(line.Email),
		})
	}

	return &tendercreate.CreateSpotTenderRequest{
		ShipmentMoveID:            moveID,
		Mode:                      input.Mode,
		Lines:                     lines,
		OverrideInsuranceWarnings: gqlscalar.BoolValue(input.OverrideInsuranceWarnings),
	}, nil
}

func clampTTLSeconds(seconds int) int32 {
	const maxTTL = int(^uint32(0) >> 1)
	switch {
	case seconds < 0:
		return 0
	case seconds > maxTTL:
		return int32(maxTTL)
	default:
		return int32(seconds)
	}
}
