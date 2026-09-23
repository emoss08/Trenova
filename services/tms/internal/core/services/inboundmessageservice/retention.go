package inboundmessageservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// RetentionDays is how long a settled message is kept. Six months covers a
// detention dispute or a claim raised a season later; the documents a message
// brought in outlive it on the records they were attached to.
const RetentionDays = 180

// PurgeSettledRequest is one bounded pass of retention over one tenant.
type PurgeSettledRequest struct {
	TenantInfo pagination.TenantInfo
	Before     int64
	Limit      int
}

// PurgeSettled removes the oldest messages somebody finished with, and reports
// how many went.
//
// The stored copy of each message goes first and the row second. A copy that
// cannot be deleted keeps its row, so the next pass finds it again; the other
// order would leave an object in storage that nothing points at any more.
// Messages still waiting on a person — held for review or quarantined — are
// never old enough to go.
func (s *Service) PurgeSettled(ctx context.Context, req PurgeSettledRequest) (int, error) {
	settled, err := s.messageRepo.ListSettledBefore(
		ctx,
		repositories.ListSettledInboundMessagesRequest{
			TenantInfo: req.TenantInfo,
			Before:     req.Before,
			Limit:      req.Limit,
		},
	)
	if err != nil {
		return 0, fmt.Errorf("list settled inbound messages: %w", err)
	}
	if len(settled) == 0 {
		return 0, nil
	}

	ids := make([]pulid.ID, 0, len(settled))
	for _, message := range settled {
		if message.HTMLKey != "" && s.storage != nil {
			if dErr := s.storage.Delete(ctx, message.HTMLKey); dErr != nil {
				s.l.Warn("kept a settled message whose stored copy could not be removed",
					zap.String("messageId", message.ID.String()), zap.Error(dErr))

				continue
			}
		}
		ids = append(ids, message.ID)
	}

	deleted, err := s.messageRepo.DeleteByIDs(ctx, repositories.DeleteInboundMessagesRequest{
		TenantInfo: req.TenantInfo,
		IDs:        ids,
	})
	if err != nil {
		return 0, fmt.Errorf("delete settled inbound messages: %w", err)
	}

	return deleted, nil
}
