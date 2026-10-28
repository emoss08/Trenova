package checkcallresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func parseMoveID(value string) (pulid.ID, error) {
	moveID, err := pulid.MustParse(value)
	if err != nil || moveID.IsNil() {
		return pulid.Nil, errortypes.NewValidationError(
			"shipmentMoveId",
			errortypes.ErrInvalid,
			"Invalid move",
		)
	}

	return moveID, nil
}

func checkCallPayload(result *checkcallports.CheckCallResult) *gqlmodel.LogCheckCallPayload {
	payload := &gqlmodel.LogCheckCallPayload{
		CheckCall:     result.CheckCall,
		OutcomeReason: gqlscalar.NonEmptyPtr(result.Reason),
	}
	if result.Outcome != "" {
		outcome := result.Outcome
		payload.Outcome = &outcome
	}

	return payload
}
