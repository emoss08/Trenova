package shipmentcommentservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/operations/domain/commentbody"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func (s *service) applyBody(entity *shipment.ShipmentComment) *errortypes.MultiError {
	if entity.Body == nil {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	commentbody.ValidateCommentBody(entity.Body, multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	entity.Comment = strings.TrimSpace(commentbody.RenderCommentBodyText(entity.Body))
	entity.MentionedUserIDs = append(
		entity.MentionedUserIDs,
		commentbody.ExtractMentionUserIDs(entity.Body)...,
	)

	return nil
}
