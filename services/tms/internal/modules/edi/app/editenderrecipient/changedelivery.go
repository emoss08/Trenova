package editenderrecipient

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) ChangeForOutboundMessage(
	ctx context.Context,
	message *edimessage.EDIMessage,
) (*editenderchange.TenderChange, bool, error) {
	change, err := s.tenderChangeRepo.GetTenderChangeByOutboundMessageID(
		ctx,
		editenderports.GetEDITenderChangeByOutboundMessageIDRequest{
			OutboundMessageID: message.ID,
			TenantInfo:        MessageTenantInfo(message),
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

func (s *Service) CompleteChangeDelivery(
	ctx context.Context,
	message *edimessage.EDIMessage,
) error {
	change, found, err := s.ChangeForOutboundMessage(ctx, message)
	if err != nil || !found {
		return err
	}
	if change.Status != editenderchange.TenderChangeStatusQueued &&
		change.Status != editenderchange.TenderChangeStatusFailed {
		return nil
	}
	change.Status = editenderchange.TenderChangeStatusSent
	change.FailureReason = ""
	if _, err = s.tenderChangeRepo.UpdateTenderChange(ctx, change); err != nil {
		return err
	}
	if change.Recipient == nil {
		return nil
	}
	return s.AdvanceBaseline(
		ctx,
		change.Recipient,
		&change.NewTenderPayload,
		editenderchange.TenderRecipientBaselineStatusSent,
	)
}

func (s *Service) FailChangeDelivery(
	ctx context.Context,
	message *edimessage.EDIMessage,
	reason string,
) error {
	change, found, err := s.ChangeForOutboundMessage(ctx, message)
	if err != nil || !found {
		return err
	}
	if change.Status != editenderchange.TenderChangeStatusQueued {
		return nil
	}
	change.Status = editenderchange.TenderChangeStatusFailed
	change.FailureReason = reason
	_, err = s.tenderChangeRepo.UpdateTenderChange(ctx, change)
	return err
}

func MessageTenantInfo(message *edimessage.EDIMessage) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: message.OrganizationID,
		BuID:  message.BusinessUnitID,
	}
}
