package inboundreads

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) Thread(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	threadKey string,
) ([]*inboundmessage.InboundMessage, error) {
	threadKey = strings.TrimSpace(threadKey)
	if threadKey == "" || len(threadKey) > inboundmessage.MaxThreadKeyLength {
		return nil, errortypes.NewValidationError("threadKey", errortypes.ErrInvalid,
			"Say which thread to read")
	}

	return s.queueRepo.ListThread(ctx, inboundmessageports.ListThreadRequest{
		TenantInfo: tenantInfo,
		ThreadKey:  threadKey,
		Limit:      inboundmessageports.MaxThreadMessages,
	})
}
