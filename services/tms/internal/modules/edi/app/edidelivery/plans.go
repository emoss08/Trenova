package edidelivery

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

type DeliveryPlan struct {
	Message *edimessage.EDIMessage
	Profile *edi.EDICommunicationProfile
}

func (s *Service) PlanRetryMessageDelivery(
	ctx context.Context,
	req *RetryMessageDeliveryRequest,
) (*DeliveryPlan, error) {
	message, err := s.deliverableMessage(ctx, req)
	if err != nil {
		return nil, err
	}
	if message.Direction != edidocument.DocumentDirectionOutbound {
		return nil, errortypes.NewValidationError(
			"messageId",
			errortypes.ErrInvalidOperation,
			"Only outbound EDI messages can be delivered",
		)
	}
	if message.DeliveryStatus != edimessage.MessageDeliveryStatusQueued &&
		!message.DeliveryStatus.IsRetryable() {
		return nil, errortypes.NewValidationError(
			"deliveryStatus",
			errortypes.ErrInvalidOperation,
			"Only queued, failed, or dead-lettered EDI messages can be retried",
		)
	}
	profile, err := s.deliveryProfileForMessage(ctx, message)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewValidationError(
				"messageId",
				errortypes.ErrInvalidOperation,
				"EDI partner has no active SFTP or VAN communication profile for delivery",
			)
		}
		return nil, err
	}

	return &DeliveryPlan{Message: message, Profile: profile}, nil
}

func (s *Service) PlanReplayMessageDelivery(
	ctx context.Context,
	req *RetryMessageDeliveryRequest,
) (*DeliveryPlan, error) {
	message, err := s.deliverableMessage(ctx, req)
	if err != nil {
		return nil, err
	}
	if message.Direction != edidocument.DocumentDirectionOutbound {
		return nil, errortypes.NewValidationError(
			"messageId",
			errortypes.ErrInvalidOperation,
			"Only outbound EDI messages can be replayed",
		)
	}
	if message.DeliveryStatus != edimessage.MessageDeliveryStatusSent {
		return nil, errortypes.NewValidationError(
			"deliveryStatus",
			errortypes.ErrInvalidOperation,
			"Only delivered EDI messages can be replayed; use retry for failed messages",
		)
	}
	if message.RawPurgedAt != nil {
		return nil, errortypes.NewValidationError(
			"messageId",
			errortypes.ErrInvalidOperation,
			"The raw X12 payload was purged by the retention policy and can no longer be replayed",
		)
	}
	profile, err := s.deliveryProfileForMessage(ctx, message)
	if err != nil {
		return nil, err
	}

	return &DeliveryPlan{Message: message, Profile: profile}, nil
}

func (s *Service) deliverableMessage(
	ctx context.Context,
	req *RetryMessageDeliveryRequest,
) (*edimessage.EDIMessage, error) {
	if req == nil || req.MessageID.IsNil() {
		return nil, errortypes.NewValidationError(
			"messageId",
			errortypes.ErrRequired,
			"EDI message ID is required",
		)
	}

	return s.messageRepo.GetMessageByID(ctx, edimessageports.GetEDIMessageByIDRequest{
		ID:         req.MessageID,
		TenantInfo: req.TenantInfo,
	})
}
