package realtimebroker

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

const (
	changeFeedPage     = 256
	changeFeedMaxPages = 8
	actionDeleted      = "deleted"
)

type ChangeFeedParams struct {
	fx.In

	Client *redis.Client
	Config *config.Config
}

type ChangeFeed struct {
	client *redis.Client
	shards int
}

func NewChangeFeed(p ChangeFeedParams) services.RecordChangeFeed {
	return &ChangeFeed{
		client: p.Client,
		shards: p.Config.GetRealtimeConfig().GetShardCount(),
	}
}

type changedRecord struct {
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	EntityID    string    `json:"entityId"`
	RecordID    string    `json:"recordId"`
	Fields      []string  `json:"fields"`
	ActorType   string    `json:"actorType"`
	ActorUserID string    `json:"actorUserId"`
	OccurredAt  time.Time `json:"occurredAt"`
}

func (r *changedRecord) id() string {
	if r.RecordID != "" {
		return r.RecordID
	}

	return r.EntityID
}

func (f *ChangeFeed) Head(ctx context.Context, orgID, buID pulid.ID) (string, error) {
	shard := shardFor(tenantKey(orgID, buID), f.shards)

	return f.head(ctx, shard)
}

func (f *ChangeFeed) head(ctx context.Context, shard int) (string, error) {
	msgs, err := f.client.XRevRangeN(ctx, streamKey(shard), "+", "-", 1).Result()
	if err != nil {
		return "", fmt.Errorf("read the newest realtime entry: %w", err)
	}
	if len(msgs) == 0 {
		return formatCursor(shard, "0-0"), nil
	}

	return formatCursor(shard, msgs[0].ID), nil
}

func (f *ChangeFeed) Since(
	ctx context.Context,
	req *services.RecordChangesRequest,
) (*services.RecordChanges, error) {
	tenant := tenantKey(req.OrganizationID, req.BusinessUnitID)
	shard := shardFor(tenant, f.shards)

	pos, ok := parseCursor(req.Cursor)
	if !ok || pos.shard != shard || len(req.Records) == 0 {
		head, err := f.head(ctx, shard)
		if err != nil {
			return nil, err
		}
		return &services.RecordChanges{Cursor: head}, nil
	}

	watched := make(map[string]struct{}, len(req.Records))
	needles := make([][]byte, 0, len(req.Records))
	for _, id := range req.Records {
		if _, seen := watched[id]; seen || id == "" {
			continue
		}
		watched[id] = struct{}{}
		needles = append(needles, []byte(id))
	}

	merged := newChangeSet()
	from := pos.id
	key := streamKey(shard)
	for range changeFeedMaxPages {
		msgs, err := f.client.XRangeN(ctx, key, "("+from.String(), "+", changeFeedPage).Result()
		if err != nil {
			return nil, fmt.Errorf("read realtime entries: %w", err)
		}
		for i := range msgs {
			e, decoded := decodeEntry(shard, &msgs[i])
			if !decoded {
				continue
			}
			from = e.id
			if !relevant(e, tenant, needles) {
				continue
			}
			var change changedRecord
			if sonic.Unmarshal(e.payload, &change) != nil {
				continue
			}
			if _, watching := watched[change.id()]; watching {
				merged.add(&change)
			}
		}
		if len(msgs) < changeFeedPage {
			break
		}
	}

	return &services.RecordChanges{
		Cursor:  formatCursor(shard, from.String()),
		Changes: merged.changes(),
	}, nil
}

func relevant(e *entry, tenant string, needles [][]byte) bool {
	if e.tenant != tenant || e.event != services.RealtimeEventInvalidation ||
		e.ephemeral || e.audience != "" || e.scope != "" {
		return false
	}

	return slices.ContainsFunc(needles, func(needle []byte) bool {
		return bytes.Contains(e.payload, needle)
	})
}

type changeSet struct {
	order []string
	byID  map[string]*services.WatchedRecordChange
}

func newChangeSet() *changeSet {
	return &changeSet{byID: make(map[string]*services.WatchedRecordChange)}
}

func (s *changeSet) add(change *changedRecord) {
	id := change.id()
	at := change.OccurredAt.Unix()
	current, seen := s.byID[id]
	if !seen {
		s.order = append(s.order, id)
		s.byID[id] = &services.WatchedRecordChange{
			RecordID:    id,
			Resource:    change.Resource,
			Action:      change.Action,
			Fields:      slices.Clone(change.Fields),
			ActorType:   change.ActorType,
			ActorUserID: change.ActorUserID,
			At:          at,
		}
		return
	}

	for _, field := range change.Fields {
		if !slices.Contains(current.Fields, field) {
			current.Fields = append(current.Fields, field)
		}
	}
	if current.Action != actionDeleted {
		current.Action = change.Action
	}
	if at >= current.At {
		current.At = at
		current.ActorType = change.ActorType
		current.ActorUserID = change.ActorUserID
	}
}

func (s *changeSet) changes() []services.WatchedRecordChange {
	if len(s.order) == 0 {
		return nil
	}
	out := make([]services.WatchedRecordChange, 0, len(s.order))
	for _, id := range s.order {
		out = append(out, *s.byID[id])
	}

	return out
}
