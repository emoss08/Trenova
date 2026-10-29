package edidelivery

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/app/edibulk"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type BulkRetryMessageDeliveryRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	MessageIDs []pulid.ID            `json:"messageIds"`
}

func (s *Service) BulkRetryMessageDelivery(
	ctx context.Context,
	req *BulkRetryMessageDeliveryRequest,
) (*edibulk.Result, error) {
	if err := edibulk.ValidateIDs("messageIds", req.MessageIDs); err != nil {
		return nil, err
	}
	result := edibulk.NewResult(len(req.MessageIDs))
	for _, messageID := range req.MessageIDs {
		if _, err := s.RetryMessageDelivery(ctx, &RetryMessageDeliveryRequest{
			MessageID:  messageID,
			TenantInfo: req.TenantInfo,
		}); err != nil {
			result.Failed = append(result.Failed, edibulk.Failure{
				ID:    messageID,
				Error: err.Error(),
			})
			continue
		}
		result.Succeeded = append(result.Succeeded, messageID)
	}
	return result, nil
}
