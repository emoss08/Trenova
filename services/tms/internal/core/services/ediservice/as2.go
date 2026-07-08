package ediservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/editransport"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/as2"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

type ApplyAS2MDNRequest struct {
	ContentType string
	Body        []byte
}

func (s *Service) ApplyAS2MDN(ctx context.Context, req *ApplyAS2MDNRequest) error {
	if req == nil || len(req.Body) == 0 {
		return errortypes.NewValidationError(
			"body",
			errortypes.ErrRequired,
			"AS2 MDN body is required",
		)
	}
	mdn, err := as2.ParseMDN(req.ContentType, req.Body, nil)
	if err != nil {
		return err
	}
	if mdn.OriginalMessageID == "" {
		return errortypes.NewValidationError(
			"originalMessageId",
			errortypes.ErrRequired,
			"AS2 MDN does not reference an original message",
		)
	}
	message, err := s.messageRepo.GetOutboundMessageByAS2MessageID(ctx, mdn.OriginalMessageID)
	if err != nil {
		return err
	}

	if mdn.Signed {
		if err = s.verifyAS2MDNSignature(ctx, message, req); err != nil {
			return err
		}
	}

	if message.DeliveryStatus == edi.MessageDeliveryStatusSent {
		return nil
	}
	now := timeutils.NowUnix()
	if !mdn.Processed() {
		s.recordDeliveryFailure(ctx, message, message.DeliveryRemotePath, &now, fmt.Errorf(
			"AS2 partner reported a processing failure: %s",
			mdn.Disposition,
		))
		return nil
	}
	if message.AS2MIC != "" && mdn.ReceivedContentMIC != "" &&
		!as2.MICMatches(message.AS2MIC, mdn.ReceivedContentMIC) {
		s.recordDeliveryFailure(
			ctx,
			message,
			message.DeliveryRemotePath,
			&now,
			errAS2MICMismatch,
		)
		return nil
	}

	sentAttemptAt := message.DeliveryLastAttemptAt
	message, err = s.messageRepo.UpdateMessageDelivery(
		ctx,
		&repositories.UpdateEDIMessageDeliveryRequest{
			ID:                    message.ID,
			TenantInfo:            messageTenantInfo(message),
			DeliveryStatus:        edi.MessageDeliveryStatusSent,
			DeliveryRemotePath:    message.DeliveryRemotePath,
			DeliveryLastAttemptAt: &now,
			DeliverySentAt:        &now,
		},
	)
	if err != nil {
		return err
	}
	if sentAttemptAt != nil && now >= *sentAttemptAt {
		s.metrics.RecordMDNRoundTrip("async", float64(now-*sentAttemptAt))
	}
	s.l.Info(
		"AS2 async MDN resolved outbound message delivery",
		zap.String("messageId", message.ID.String()),
	)
	return s.completeTenderChangeDelivery(ctx, message)
}

var errAS2MICMismatch = fmt.Errorf("AS2 MDN MIC does not match the transmitted content")

func (s *Service) verifyAS2MDNSignature(
	ctx context.Context,
	message *edi.EDIMessage,
	req *ApplyAS2MDNRequest,
) error {
	profile, err := s.deliveryProfileForMessage(ctx, message)
	if err != nil {
		return err
	}
	secrets, err := s.ProfileTransportSecrets(profile)
	if err != nil {
		return err
	}
	cfg, err := editransport.AS2ConfigFromProfile(profile, secrets)
	if err != nil {
		return err
	}
	if cfg.PartnerSigningCertificate == nil {
		return nil
	}
	if _, err = as2.ParseMDN(
		req.ContentType,
		req.Body,
		cfg.PartnerSigningCertificate,
	); err != nil {
		return fmt.Errorf("AS2 MDN signature verification failed: %w", err)
	}
	return nil
}
