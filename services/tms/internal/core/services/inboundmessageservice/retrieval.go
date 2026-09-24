package inboundmessageservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

func (s *Service) queueForRetrieval(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
) {
	if s.indexer == nil || message == nil {
		return
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: message.OrganizationID,
		BuID:  message.BusinessUnitID,
	}
	if err := s.indexer.MarkStale(
		ctx,
		tenantInfo,
		airetrieval.SourceTypeInboundMessage,
		message.ID,
	); err != nil {
		s.l.Warn("inbound message could not be queued for semantic indexing",
			zap.String("messageId", message.ID.String()),
			zap.Error(err),
		)
	}
}
