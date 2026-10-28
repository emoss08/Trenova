package boardcardservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) readEvents(
	ctx context.Context,
	req *agentports.BoardFillRequest,
	reads *board.Reads,
) error {
	audits := false
	checked := false
	for idx := range req.Board.Blocks {
		block := &req.Board.Blocks[idx]
		if block.Type != board.BlockTimeline || block.Records.Empty() {
			continue
		}
		set := block.Records
		if _, read := reads.Summaries[set.Resource][set.IDs[0]]; !read ||
			!reads.Readable[set.Resource] {
			continue
		}
		if !checked {
			allowed, err := s.allowed(
				ctx, req.Actor, permission.ResourceAuditLog, permission.OpRead,
			)
			if err != nil {
				return err
			}
			audits, checked = allowed, true
		}
		if !audits {
			return nil
		}
		events, err := s.events(ctx, req, set.IDs[0], block.Limit)
		if err != nil {
			return err
		}
		reads.Events[set.IDs[0]] = events
	}

	return nil
}

func (s *Service) events(
	ctx context.Context,
	req *agentports.BoardFillRequest,
	resourceID pulid.ID,
	limit int,
) ([]board.Event, error) {
	tenant := req.Actor.TenantInfo()
	result, err := s.activity.ListByResourceID(ctx, &auditports.ListByResourceIDRequest{
		ResourceID: resourceID,
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: limit},
			FieldFilters: []domaintypes.FieldFilter{{
				Field:    "operation",
				Operator: dbtype.OpNotEqual,
				Value:    "read",
			}},
			Sort: []domaintypes.SortField{{
				Field:     "timestamp",
				Direction: dbtype.SortDirectionDesc,
			}},
		},
	})
	if err != nil {
		return nil, err
	}

	events := make([]board.Event, 0, len(result.Items))
	for _, entry := range result.Items {
		event := board.Event{
			At:        entry.Timestamp,
			Operation: string(entry.Operation),
			Principal: entry.PrincipalType,
			Comment:   entry.Comment,
			Changes:   entry.Changes,
		}
		if entry.User != nil {
			event.Actor = entry.User.Name
		}
		events = append(events, event)
	}
	slices.Reverse(events)

	return events, nil
}
