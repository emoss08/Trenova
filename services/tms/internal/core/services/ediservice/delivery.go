package ediservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/editransport"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
)

var deliverableMethods = []edi.ConnectionMethod{
	edi.ConnectionMethodSFTP,
	edi.ConnectionMethodVAN,
	edi.ConnectionMethodAS2,
}

func (s *Service) DeliverMessage(
	ctx context.Context,
	payload *DeliverEDIMessageWorkflowPayload,
) (*DeliverEDIMessageWorkflowResult, error) {
	message, earlyResult, err := s.loadDeliverableMessage(ctx, payload)
	if err != nil || earlyResult != nil {
		return earlyResult, err
	}

	profile, err := s.deliveryProfileForMessage(ctx, message)
	if err != nil {
		return nil, err
	}
	secrets, err := s.ProfileTransportSecrets(profile)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	message, err = s.messageRepo.UpdateMessageDelivery(
		ctx,
		&repositories.UpdateEDIMessageDeliveryRequest{
			ID:                    message.ID,
			TenantInfo:            messageTenantInfo(message),
			DeliveryStatus:        edi.MessageDeliveryStatusSending,
			DeliveryRemotePath:    message.DeliveryRemotePath,
			IncrementAttempts:     true,
			DeliveryLastAttemptAt: &now,
			DeliveryLastError:     message.DeliveryLastError,
		},
	)
	if err != nil {
		return nil, err
	}

	transportResult, deliveryErr := s.transport.Deliver(
		ctx,
		profile.Method,
		&services.EDITransportRequest{
			Profile:  profile,
			Secrets:  secrets,
			FileName: editransport.OutboundFileName(profile, message),
			Contents: message.RawX12,
		},
	)
	remotePath := ""
	as2MessageID := ""
	as2MIC := ""
	pending := false
	if transportResult != nil {
		remotePath = transportResult.RemotePath
		as2MessageID = transportResult.MessageID
		as2MIC = transportResult.MIC
		pending = transportResult.Pending
	}
	if deliveryErr != nil {
		s.recordDeliveryFailure(ctx, message, remotePath, &now, deliveryErr)
		return nil, deliveryErr
	}

	if pending {
		message, err = s.messageRepo.UpdateMessageDelivery(
			ctx,
			&repositories.UpdateEDIMessageDeliveryRequest{
				ID:                    message.ID,
				TenantInfo:            messageTenantInfo(message),
				DeliveryStatus:        edi.MessageDeliveryStatusSending,
				DeliveryRemotePath:    remotePath,
				AS2MessageID:          as2MessageID,
				AS2MIC:                as2MIC,
				DeliveryLastAttemptAt: &now,
			},
		)
		if err != nil {
			return nil, err
		}
		return &DeliverEDIMessageWorkflowResult{
			MessageID:      message.ID,
			DeliveryStatus: message.DeliveryStatus,
			RemotePath:     remotePath,
		}, nil
	}

	message, err = s.messageRepo.UpdateMessageDelivery(
		ctx,
		&repositories.UpdateEDIMessageDeliveryRequest{
			ID:                    message.ID,
			TenantInfo:            messageTenantInfo(message),
			DeliveryStatus:        edi.MessageDeliveryStatusSent,
			DeliveryRemotePath:    remotePath,
			AS2MessageID:          as2MessageID,
			AS2MIC:                as2MIC,
			DeliveryLastAttemptAt: &now,
			DeliverySentAt:        &now,
		},
	)
	if err != nil {
		return nil, err
	}
	if err = s.completeTenderChangeDelivery(ctx, message); err != nil {
		return nil, err
	}
	return &DeliverEDIMessageWorkflowResult{
		MessageID:      message.ID,
		DeliveryStatus: message.DeliveryStatus,
		RemotePath:     remotePath,
	}, nil
}

func (s *Service) loadDeliverableMessage(
	ctx context.Context,
	payload *DeliverEDIMessageWorkflowPayload,
) (*edi.EDIMessage, *DeliverEDIMessageWorkflowResult, error) {
	if payload == nil || payload.MessageID.IsNil() {
		return nil, nil, temporal.NewNonRetryableApplicationError(
			"EDI message ID is required for delivery",
			"InvalidDeliveryPayload",
			nil,
		)
	}
	message, err := s.messageRepo.GetMessageByID(ctx, repositories.GetEDIMessageByIDRequest{
		ID:         payload.MessageID,
		TenantInfo: payload.TenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}
	if message.Direction != edi.DocumentDirectionOutbound {
		return nil, nil, temporal.NewNonRetryableApplicationError(
			"only outbound EDI messages can be delivered",
			"MessageNotOutbound",
			nil,
		)
	}
	if message.DeliveryStatus == edi.MessageDeliveryStatusSent {
		return nil, &DeliverEDIMessageWorkflowResult{
			MessageID:      message.ID,
			DeliveryStatus: message.DeliveryStatus,
			RemotePath:     message.DeliveryRemotePath,
		}, nil
	}
	if !message.DeliveryStatus.IsDeliverable() {
		return nil, nil, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf(
				"EDI message delivery status %s is not deliverable",
				message.DeliveryStatus,
			),
			"MessageNotDeliverable",
			nil,
		)
	}
	return message, nil, nil
}

func (s *Service) recordDeliveryFailure(
	ctx context.Context,
	message *edi.EDIMessage,
	remotePath string,
	attemptedAt *int64,
	deliveryErr error,
) {
	if _, updateErr := s.messageRepo.UpdateMessageDelivery(
		ctx,
		&repositories.UpdateEDIMessageDeliveryRequest{
			ID:                    message.ID,
			TenantInfo:            messageTenantInfo(message),
			DeliveryStatus:        edi.MessageDeliveryStatusFailed,
			DeliveryRemotePath:    remotePath,
			DeliveryLastAttemptAt: attemptedAt,
			DeliveryLastError:     deliveryErr.Error(),
		},
	); updateErr != nil {
		s.l.Warn(
			"failed to record EDI message delivery failure",
			zap.String("messageId", message.ID.String()),
			zap.Error(updateErr),
		)
	}
}

func (s *Service) MarkMessageDeadLettered(
	ctx context.Context,
	payload *MarkEDIMessageDeadLetteredPayload,
) error {
	if payload == nil || payload.MessageID.IsNil() {
		return errors.New("EDI message ID is required for dead-letter handling")
	}
	message, err := s.messageRepo.GetMessageByID(ctx, repositories.GetEDIMessageByIDRequest{
		ID:         payload.MessageID,
		TenantInfo: payload.TenantInfo,
	})
	if err != nil {
		return err
	}
	if message.DeliveryStatus == edi.MessageDeliveryStatusSent {
		return nil
	}
	now := timeutils.NowUnix()
	message, err = s.messageRepo.UpdateMessageDelivery(
		ctx,
		&repositories.UpdateEDIMessageDeliveryRequest{
			ID:                    message.ID,
			TenantInfo:            messageTenantInfo(message),
			DeliveryStatus:        edi.MessageDeliveryStatusDeadLettered,
			DeliveryRemotePath:    message.DeliveryRemotePath,
			DeliveryLastAttemptAt: &now,
			DeliveryLastError:     payload.Reason,
		},
	)
	if err != nil {
		return err
	}
	s.NotifyOperationalFailure(ctx, &EDIOperationalAlert{
		OrganizationID: message.OrganizationID,
		BusinessUnitID: message.BusinessUnitID,
		EventType:      EDIAlertEventMessageDeadLettered,
		PartnerID:      message.EDIPartnerID,
		Title:          "EDI message dead-lettered",
		Message: fmt.Sprintf(
			"Outbound %s message %s exhausted its delivery retries: %s",
			message.TransactionSet,
			message.ID,
			payload.Reason,
		),
		RelatedEntities: map[string]any{
			"messageId": message.ID,
			"partnerId": message.EDIPartnerID,
		},
		Data: map[string]any{
			"transactionSet": message.TransactionSet,
			"error":          payload.Reason,
			"link":           "/edi/messages?panelType=edit&panelEntityId=" + message.ID.String(),
		},
	})
	return s.failTenderChangeDelivery(ctx, message, payload.Reason)
}

func (s *Service) RetryMessageDelivery(
	ctx context.Context,
	req *RetryMessageDeliveryRequest,
) (*edi.EDIMessage, error) {
	if req == nil || req.MessageID.IsNil() {
		return nil, errortypes.NewValidationError(
			"messageId",
			errortypes.ErrRequired,
			"EDI message ID is required",
		)
	}
	message, err := s.messageRepo.GetMessageByID(ctx, repositories.GetEDIMessageByIDRequest{
		ID:         req.MessageID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if message.Direction != edi.DocumentDirectionOutbound {
		return nil, errortypes.NewValidationError(
			"messageId",
			errortypes.ErrInvalidOperation,
			"Only outbound EDI messages can be delivered",
		)
	}
	if message.DeliveryStatus != edi.MessageDeliveryStatusQueued &&
		!message.DeliveryStatus.IsRetryable() {
		return nil, errortypes.NewValidationError(
			"deliveryStatus",
			errortypes.ErrInvalidOperation,
			"Only queued, failed, or dead-lettered EDI messages can be retried",
		)
	}
	if _, err = s.deliveryProfileForMessage(ctx, message); err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewValidationError(
				"messageId",
				errortypes.ErrInvalidOperation,
				"EDI partner has no active SFTP or VAN communication profile for delivery",
			)
		}
		return nil, err
	}
	now := timeutils.NowUnix()
	message, err = s.messageRepo.UpdateMessageDelivery(
		ctx,
		&repositories.UpdateEDIMessageDeliveryRequest{
			ID:                    message.ID,
			TenantInfo:            messageTenantInfo(message),
			DeliveryStatus:        edi.MessageDeliveryStatusQueued,
			DeliveryRemotePath:    message.DeliveryRemotePath,
			DeliveryLastAttemptAt: &now,
			DeliveryLastError:     message.DeliveryLastError,
		},
	)
	if err != nil {
		return nil, err
	}
	if err = s.startDeliveryWorkflow(ctx, message); err != nil {
		return nil, err
	}
	return message, nil
}

func (s *Service) queueMessageForDelivery(
	ctx context.Context,
	message *edi.EDIMessage,
) error {
	if message == nil ||
		message.Direction != edi.DocumentDirectionOutbound ||
		message.Status != edi.MessageStatusGenerated {
		return nil
	}
	partner, err := s.partnerRepo.GetByID(ctx, repositories.GetEDIPartnerByIDRequest{
		ID:         message.EDIPartnerID,
		TenantInfo: messageTenantInfo(message),
	})
	if err != nil {
		return err
	}
	if partner.Kind != edi.PartnerKindExternal {
		return nil
	}
	if _, err = s.deliveryProfileForMessage(ctx, message); err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil
		}
		return err
	}
	if message.DeliveryStatus != edi.MessageDeliveryStatusQueued {
		now := timeutils.NowUnix()
		updated, updateErr := s.messageRepo.UpdateMessageDelivery(
			ctx,
			&repositories.UpdateEDIMessageDeliveryRequest{
				ID:                    message.ID,
				TenantInfo:            messageTenantInfo(message),
				DeliveryStatus:        edi.MessageDeliveryStatusQueued,
				DeliveryLastAttemptAt: &now,
			},
		)
		if updateErr != nil {
			return updateErr
		}
		*message = *updated
	}
	return s.startDeliveryWorkflow(ctx, message)
}

func (s *Service) startDeliveryWorkflow(ctx context.Context, message *edi.EDIMessage) error {
	if s.workflowStarter == nil || !s.workflowStarter.Enabled() {
		return errortypes.NewBusinessError("EDI delivery workflow is not configured")
	}
	payload := &DeliverEDIMessageWorkflowPayload{
		MessageID:  message.ID,
		TenantInfo: messageTenantInfo(message),
	}
	_, err := s.workflowStarter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       buildDeliverMessageWorkflowID(message.ID),
			TaskQueue:                                temporaltype.EDITaskQueue,
			WorkflowExecutionErrorWhenAlreadyStarted: true,
			StaticSummary: fmt.Sprintf(
				"Deliver EDI message %s",
				message.ID.String(),
			),
		},
		temporaltype.DeliverEDIMessageWorkflowName,
		payload,
	)
	if err != nil {
		var alreadyStartedErr *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &alreadyStartedErr) {
			return nil
		}
		return errortypes.NewBusinessError("failed to start EDI delivery workflow").
			WithInternal(err)
	}
	return nil
}

func (s *Service) deliveryProfileForMessage(
	ctx context.Context,
	message *edi.EDIMessage,
) (*edi.EDICommunicationProfile, error) {
	change, found, err := s.tenderChangeForOutboundMessage(ctx, message)
	if err != nil {
		return nil, err
	}
	if found &&
		change.Recipient != nil &&
		change.Recipient.CommunicationProfileID.IsNotNil() {
		return s.profileRepo.GetProfileByID(
			ctx,
			repositories.GetEDICommunicationProfileByIDRequest{
				ID:         change.Recipient.CommunicationProfileID,
				TenantInfo: messageTenantInfo(message),
			},
		)
	}
	if message.EDIPartnerID.IsNil() {
		return nil, errors.New("EDI partner is required for message delivery")
	}
	return s.profileRepo.GetActiveProfileByPartner(
		ctx,
		repositories.GetActiveEDICommunicationProfileByPartnerRequest{
			PartnerID:  message.EDIPartnerID,
			TenantInfo: messageTenantInfo(message),
			Methods:    deliverableMethods,
		},
	)
}

func (s *Service) ProfileTransportSecrets(
	profile *edi.EDICommunicationProfile,
) (map[string]string, error) {
	secrets := make(map[string]string, 3)
	for _, key := range []string{"password", "privateKey", "basicAuthPassword"} {
		value, err := s.decryptProfileSecret(profile, key)
		if err != nil {
			return nil, err
		}
		if value != "" {
			secrets[key] = value
		}
	}
	return secrets, nil
}

func (s *Service) tenderChangeForOutboundMessage(
	ctx context.Context,
	message *edi.EDIMessage,
) (*edi.TenderChange, bool, error) {
	change, err := s.tenderChangeRepo.GetTenderChangeByOutboundMessageID(
		ctx,
		repositories.GetEDITenderChangeByOutboundMessageIDRequest{
			OutboundMessageID: message.ID,
			TenantInfo:        messageTenantInfo(message),
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return change, true, nil
}

func (s *Service) completeTenderChangeDelivery(
	ctx context.Context,
	message *edi.EDIMessage,
) error {
	change, found, err := s.tenderChangeForOutboundMessage(ctx, message)
	if err != nil || !found {
		return err
	}
	if change.Status != edi.TenderChangeStatusQueued &&
		change.Status != edi.TenderChangeStatusFailed {
		return nil
	}
	change.Status = edi.TenderChangeStatusSent
	change.FailureReason = ""
	if _, err = s.tenderChangeRepo.UpdateTenderChange(ctx, change); err != nil {
		return err
	}
	if change.Recipient == nil {
		return nil
	}
	return s.advanceTenderRecipientBaseline(
		ctx,
		change.Recipient,
		&change.NewTenderPayload,
		edi.TenderRecipientBaselineStatusSent,
	)
}

func (s *Service) failTenderChangeDelivery(
	ctx context.Context,
	message *edi.EDIMessage,
	reason string,
) error {
	change, found, err := s.tenderChangeForOutboundMessage(ctx, message)
	if err != nil || !found {
		return err
	}
	if change.Status != edi.TenderChangeStatusQueued {
		return nil
	}
	change.Status = edi.TenderChangeStatusFailed
	change.FailureReason = reason
	_, err = s.tenderChangeRepo.UpdateTenderChange(ctx, change)
	return err
}

func messageTenantInfo(message *edi.EDIMessage) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: message.OrganizationID,
		BuID:  message.BusinessUnitID,
	}
}

func buildDeliverMessageWorkflowID(messageID pulid.ID) string {
	return "edi-deliver-message-" + messageID.String()
}
