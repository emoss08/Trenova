package inboundmessageservice

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func ApplyLink(message *inboundmessage.InboundMessage, req *LinkRequest) {
	message.MatchedShipmentID = req.ShipmentID
	message.MatchedCustomerID = req.CustomerID
	message.MatchedCarrierID = req.CarrierID
	message.MatchReason = strings.TrimSpace(req.Reason)
	message.ReviewedBy = req.ReviewerID
}

func ApplyReview(message *inboundmessage.InboundMessage, req *ReviewRequest, now int64) error {
	note, err := reviewNote(req)
	if err != nil {
		return err
	}

	message.Status = req.Status
	message.ReviewedBy = req.ReviewerID
	message.ReviewedAt = now
	if note != "" {
		message.ReviewNote = note
	}

	return nil
}

func reviewNote(req *ReviewRequest) (string, error) {
	if req.Status != inboundmessage.StatusActioned &&
		req.Status != inboundmessage.StatusIgnored {
		return "", errortypes.NewValidationError("status", errortypes.ErrInvalid,
			"A message can be marked handled or ignored, nothing else")
	}

	note := strings.TrimSpace(req.Note)
	if len(note) > maxReviewNoteLength {
		return "", errortypes.NewValidationError("note", errortypes.ErrInvalid,
			fmt.Sprintf("A note may be at most %d characters", maxReviewNoteLength))
	}

	return note, nil
}
