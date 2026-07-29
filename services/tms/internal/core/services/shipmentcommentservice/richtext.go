package shipmentcommentservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func (s *service) applyBody(entity *shipment.ShipmentComment) *errortypes.MultiError {
	if entity.Body == nil {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	shipment.ValidateCommentBody(entity.Body, multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	entity.Comment = strings.TrimSpace(shipment.RenderCommentBodyText(entity.Body))
	entity.MentionedUserIDs = append(
		entity.MentionedUserIDs,
		shipment.ExtractMentionUserIDs(entity.Body)...,
	)

	return nil
}
