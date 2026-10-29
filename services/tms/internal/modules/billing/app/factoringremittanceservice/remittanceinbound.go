package factoringremittanceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/app/factoringaction"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringremittanceports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
)

type RemittanceIntake struct {
	remittances factoringremittanceports.RemittanceService
}

var _ inboundmessageports.RemittanceIntake = (*RemittanceIntake)(nil)

func NewRemittanceIntake(
	remittances factoringremittanceports.RemittanceService,
) inboundmessageports.RemittanceIntake {
	return &RemittanceIntake{remittances: remittances}
}

func (i *RemittanceIntake) IngestRemittances(
	ctx context.Context,
	req *inboundmessageports.RemittanceIntakeRequest,
) (*inboundmessageports.RemittanceIntakeResult, error) {
	if err := factoringaction.RequireRequest(req != nil); err != nil {
		return nil, err
	}
	attachments := make(
		[]factoringremittanceports.RemittanceEmailAttachment,
		0,
		len(req.Attachments),
	)
	for _, attachment := range req.Attachments {
		attachments = append(attachments, factoringremittanceports.RemittanceEmailAttachment{
			DocumentID:  attachment.DocumentID,
			FileName:    attachment.FileName,
			ContentType: attachment.ContentType,
		})
	}
	result, err := i.remittances.IngestEmail(
		ctx,
		&factoringremittanceports.IngestRemittanceEmailRequest{
			TenantInfo:       req.TenantInfo,
			InboundMessageID: req.MessageID,
			FromAddress:      req.FromAddress,
			ReceivedAt:       req.ReceivedAt,
			Attachments:      attachments,
		},
	)
	if err != nil {
		return nil, err
	}
	return &inboundmessageports.RemittanceIntakeResult{
		FactoringCompanyID: result.FactoringCompanyID,
		FactorName:         result.FactorName,
		RemittanceIDs:      result.RemittanceIDs,
		Problem:            result.Problem,
	}, nil
}
