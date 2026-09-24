package realtimebroker

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var ErrConnectionNotFound = errortypes.NewNotFoundError(
	"This live connection has ended. Reconnect and try again.",
)

// presenceData is what a member carries besides its connection id.
type presenceData struct {
	UserID string `json:"u"`
	Name   string `json:"n"`
}

// sweepScript removes members whose connection stopped refreshing them and
// returns them, so the caller can announce each departure exactly once even
// when several instances sweep the same key.
//
// KEYS[1] expiry zset, KEYS[2] data hash, KEYS[3] index set.
// ARGV[1] now in ms, ARGV[2] batch size, ARGV[3] index member.
var sweepScript = redis.NewScript(`
local expired = redis.call('ZRANGEBYSCORE', KEYS[1], '-inf', ARGV[1], 'LIMIT', 0, tonumber(ARGV[2]))
local out = {}
if #expired > 0 then
  local data = redis.call('HMGET', KEYS[2], unpack(expired))
  redis.call('ZREM', KEYS[1], unpack(expired))
  redis.call('HDEL', KEYS[2], unpack(expired))
  for i = 1, #expired do
    out[#out + 1] = expired[i]
    out[#out + 1] = data[i] or ''
  end
end
if redis.call('ZCARD', KEYS[1]) == 0 then
  redis.call('DEL', KEYS[1], KEYS[2])
  redis.call('SREM', KEYS[3], ARGV[3])
end
return out
`)

func (b *Broker) presenceTTL() time.Duration {
	return b.cfg.GetPresenceTTL()
}

func (b *Broker) expiresAt() float64 {
	return float64(time.Now().Add(b.presenceTTL()).UnixMilli())
}

func (b *Broker) registerConnection(ctx context.Context, conn *services.RealtimeConnection) error {
	portal := "0"
	if conn.Portal {
		portal = "1"
	}

	key := connectionKey(conn.ConnectionID)
	pipe := b.client.TxPipeline()
	pipe.HSet(ctx, key,
		connFieldUser, conn.UserID.String(),
		connFieldOrg, conn.OrganizationID.String(),
		connFieldBU, conn.BusinessUnitID.String(),
		connFieldName, conn.Name,
		connFieldPortal, portal,
	)
	pipe.Expire(ctx, key, b.presenceTTL())
	_, err := pipe.Exec(ctx)
	return err
}

func (b *Broker) Connection(
	ctx context.Context,
	connectionID string,
) (*services.RealtimeConnection, error) {
	values, err := b.client.HGetAll(ctx, connectionKey(connectionID)).Result()
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, ErrConnectionNotFound
	}

	userID, err := pulid.Parse(values[connFieldUser])
	if err != nil {
		return nil, ErrConnectionNotFound
	}
	orgID, err := pulid.Parse(values[connFieldOrg])
	if err != nil {
		return nil, ErrConnectionNotFound
	}
	buID, err := pulid.Parse(values[connFieldBU])
	if err != nil {
		return nil, ErrConnectionNotFound
	}

	return &services.RealtimeConnection{
		ConnectionID:   connectionID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		UserID:         userID,
		Name:           values[connFieldName],
		Portal:         values[connFieldPortal] == "1",
	}, nil
}

// JoinPresence adds the connection to scope and returns everyone in it.
//
// The membership is written and announced before the snapshot is read. Anyone
// whose own announcement came before ours is therefore already in the snapshot,
// and anyone whose came after reaches us live, so the reader never misses a
// member between the two.
func (b *Broker) JoinPresence(
	ctx context.Context,
	conn *services.RealtimeConnection,
	scope string,
) (*services.RealtimePresenceSnapshot, error) {
	tenant := tenantKey(conn.OrganizationID, conn.BusinessUnitID)
	expKey := presenceExpiryKey(tenant, scope)
	dataKey := presenceDataKey(tenant, scope)

	data, err := sonic.Marshal(presenceData{UserID: conn.UserID.String(), Name: conn.Name})
	if err != nil {
		return nil, err
	}

	pipe := b.client.TxPipeline()
	pipe.ZAdd(ctx, expKey, redis.Z{Score: b.expiresAt(), Member: conn.ConnectionID})
	pipe.HSet(ctx, dataKey, conn.ConnectionID, data)
	pipe.SAdd(ctx, presenceIndexKey, presenceIndexMember(tenant, scope))
	pipe.SAdd(ctx, connectionScopesKey(conn.ConnectionID), scope)
	pipe.Expire(ctx, connectionScopesKey(conn.ConnectionID), b.presenceTTL())
	if _, err = pipe.Exec(ctx); err != nil {
		return nil, err
	}

	if err = b.announce(ctx, conn, scope, services.RealtimePresenceEnter); err != nil {
		return nil, err
	}

	return b.members(ctx, tenant, scope)
}

func (b *Broker) LeavePresence(
	ctx context.Context,
	conn *services.RealtimeConnection,
	scope string,
) error {
	tenant := tenantKey(conn.OrganizationID, conn.BusinessUnitID)
	return b.withdraw(ctx, conn, tenant, scope)
}

func (b *Broker) withdraw(
	ctx context.Context,
	conn *services.RealtimeConnection,
	tenant, scope string,
) error {
	pipe := b.client.TxPipeline()
	removed := pipe.ZRem(ctx, presenceExpiryKey(tenant, scope), conn.ConnectionID)
	pipe.HDel(ctx, presenceDataKey(tenant, scope), conn.ConnectionID)
	pipe.SRem(ctx, connectionScopesKey(conn.ConnectionID), scope)
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}
	if removed.Val() == 0 {
		return nil
	}

	return b.announce(ctx, conn, scope, services.RealtimePresenceLeave)
}

// dropConnection withdraws every membership a closed connection held. The
// scopes the connection learned locally are joined with the ones recorded in
// Redis, since a join served by another instance may not have reached this
// one before the stream closed.
func (b *Broker) dropConnection(
	ctx context.Context,
	conn *services.RealtimeConnection,
	tenant string,
	local []string,
) {
	scopes := make(map[string]struct{}, len(local)+2)
	for _, scope := range local {
		scopes[scope] = struct{}{}
	}
	recorded, err := b.client.SMembers(ctx, connectionScopesKey(conn.ConnectionID)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		b.l.Warn("could not read a closed connection's presence",
			zap.String("connectionId", conn.ConnectionID), zap.Error(err))
	}
	for _, scope := range recorded {
		scopes[scope] = struct{}{}
	}

	for scope := range scopes {
		if withdrawErr := b.withdraw(ctx, conn, tenant, scope); withdrawErr != nil {
			b.l.Warn("could not withdraw presence for a closed connection",
				zap.String("connectionId", conn.ConnectionID),
				zap.String("scope", scope),
				zap.Error(withdrawErr),
			)
		}
	}

	if err = b.client.Del(
		ctx,
		connectionKey(conn.ConnectionID),
		connectionScopesKey(conn.ConnectionID),
	).Err(); err != nil {
		b.l.Warn("could not remove a closed connection",
			zap.String("connectionId", conn.ConnectionID), zap.Error(err))
	}
}

func (b *Broker) members(
	ctx context.Context,
	tenant, scope string,
) (*services.RealtimePresenceSnapshot, error) {
	now := strconv.FormatInt(time.Now().UnixMilli(), 10)
	ids, err := b.client.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:     presenceExpiryKey(tenant, scope),
		Start:   "(" + now,
		Stop:    "+inf",
		ByScore: true,
	}).Result()
	if err != nil {
		return nil, err
	}

	snapshot := &services.RealtimePresenceSnapshot{
		Scope:   scope,
		Members: make([]services.RealtimePresenceMember, 0, len(ids)),
	}
	if len(ids) == 0 {
		return snapshot, nil
	}

	values, err := b.client.HMGet(ctx, presenceDataKey(tenant, scope), ids...).Result()
	if err != nil {
		return nil, err
	}

	for i, id := range ids {
		raw, ok := values[i].(string)
		if !ok {
			continue
		}
		var data presenceData
		if sonic.UnmarshalString(raw, &data) != nil || data.UserID == "" {
			continue
		}
		snapshot.Members = append(snapshot.Members, services.RealtimePresenceMember{
			UserID:       data.UserID,
			ConnectionID: id,
			Name:         data.Name,
		})
	}

	return snapshot, nil
}

func (b *Broker) announce(
	ctx context.Context,
	conn *services.RealtimeConnection,
	scope, action string,
) error {
	return b.announceMember(ctx, &announcement{
		orgID:        conn.OrganizationID,
		buID:         conn.BusinessUnitID,
		scope:        scope,
		action:       action,
		userID:       conn.UserID.String(),
		connectionID: conn.ConnectionID,
		name:         conn.Name,
	})
}

type announcement struct {
	orgID        pulid.ID
	buID         pulid.ID
	scope        string
	action       string
	userID       string
	connectionID string
	name         string
}

func (b *Broker) announceMember(ctx context.Context, a *announcement) error {
	payload, err := sonic.Marshal(services.RealtimePresenceEvent{
		Scope:        a.scope,
		Action:       a.action,
		UserID:       a.userID,
		ConnectionID: a.connectionID,
		Name:         a.name,
	})
	if err != nil {
		return err
	}

	return b.Publish(ctx, &services.RealtimeEnvelope{
		OrganizationID:       a.orgID,
		BusinessUnitID:       a.buID,
		Event:                services.RealtimeEventPresence,
		Scope:                a.scope,
		Payload:              payload,
		PresenceConnectionID: a.connectionID,
		PresenceAction:       a.action,
		Ephemeral:            true,
	})
}

// refresh extends every membership held by a connection open here. It runs on
// the heartbeat, so a live connection's entries never reach their expiry.
func (b *Broker) refresh(subs []*subscriber) {
	if len(subs) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(b.ctx, maintenanceTimeout)
	defer cancel()

	ttl := b.presenceTTL()
	score := b.expiresAt()
	pipe := b.client.Pipeline()
	queued := 0

	flush := func() {
		if queued == 0 {
			return
		}
		if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
			b.l.Warn("could not refresh realtime presence", zap.Error(err))
		}
		queued = 0
	}

	for _, sub := range subs {
		id := sub.conn.ConnectionID
		pipe.Expire(ctx, connectionKey(id), ttl)
		pipe.Expire(ctx, connectionScopesKey(id), ttl)
		queued += 2
		for _, scope := range sub.joinedScopes() {
			pipe.ZAddXX(
				ctx,
				presenceExpiryKey(sub.tenant, scope),
				redis.Z{Score: score, Member: id},
			)
			queued++
		}
		if queued >= refreshChunk {
			flush()
		}
	}
	flush()
}

// sweep removes members whose connection died without closing, such as an
// instance that was killed, and announces each departure. One instance sweeps
// per interval; the script keeps two that overlap from announcing twice.
func (b *Broker) sweep() {
	ctx, cancel := context.WithTimeout(b.ctx, maintenanceTimeout)
	defer cancel()

	interval := b.cfg.GetHeartbeatInterval()
	locked, err := b.client.SetNX(
		ctx, sweepLockKey, "1", max(interval-time.Second, interval/2),
	).Result()
	if err != nil || !locked {
		return
	}

	index, err := b.client.SMembers(ctx, presenceIndexKey).Result()
	if err != nil {
		b.l.Warn("could not read the presence index", zap.Error(err))
		return
	}

	now := strconv.FormatInt(time.Now().UnixMilli(), 10)
	swept := 0
	for _, member := range index {
		tenant, scope, ok := splitPresenceIndexMember(member)
		if !ok {
			continue
		}
		swept += b.sweepScope(ctx, tenant, scope, member, now)
	}

	if swept > 0 {
		b.metrics.RecordPresenceSwept(swept)
	}
}

func (b *Broker) sweepScope(ctx context.Context, tenant, scope, member, now string) int {
	orgText, buText, ok := strings.Cut(tenant, ":")
	if !ok {
		return 0
	}
	orgID, orgErr := pulid.Parse(orgText)
	buID, buErr := pulid.Parse(buText)
	if orgErr != nil || buErr != nil {
		return 0
	}

	result, err := sweepScript.Run(
		ctx,
		b.client,
		[]string{
			presenceExpiryKey(tenant, scope),
			presenceDataKey(tenant, scope),
			presenceIndexKey,
		},
		now,
		sweepBatch,
		member,
	).StringSlice()
	if err != nil && !errors.Is(err, redis.Nil) {
		b.l.Warn("could not sweep presence",
			zap.String("scope", scope), zap.Error(err))
		return 0
	}

	for i := 0; i+1 < len(result); i += 2 {
		var data presenceData
		_ = sonic.UnmarshalString(result[i+1], &data)
		if announceErr := b.announceMember(ctx, &announcement{
			orgID:        orgID,
			buID:         buID,
			scope:        scope,
			action:       services.RealtimePresenceLeave,
			userID:       data.UserID,
			connectionID: result[i],
			name:         data.Name,
		}); announceErr != nil {
			b.l.Warn("could not announce a swept presence member", zap.Error(announceErr))
		}
	}

	return len(result) / 2
}

func (b *Broker) Throttle(ctx context.Context, key string, window time.Duration) (bool, error) {
	return b.client.SetNX(ctx, throttleKey(key), "1", window).Result()
}

func (b *Broker) Release(ctx context.Context, key string) error {
	return b.client.Del(ctx, throttleKey(key)).Err()
}
