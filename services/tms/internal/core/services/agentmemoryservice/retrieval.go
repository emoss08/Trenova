package agentmemoryservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

func (s *Service) queueForRetrieval(ctx context.Context, memory *agent.Memory) {
	if s.indexer == nil || memory == nil {
		return
	}

	tenant := pagination.TenantInfo{OrgID: memory.OrganizationID, BuID: memory.BusinessUnitID}
	if err := s.indexer.MarkStale(
		ctx,
		tenant,
		airetrieval.SourceTypeMemory,
		memory.ID,
	); err != nil {
		s.l.Warn("agent memory: could not be queued for semantic indexing",
			zap.String("memory", memory.ID.String()),
			zap.Error(err),
		)
	}
}
