//go:build integration

package redis_test

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamConsumerGroupLifecycle(t *testing.T) {
	client := newClient(t, nil)
	ctx := t.Context()
	const stream, group = "events", "workers"

	require.NoError(t, client.XGroupCreateMkStream(ctx, stream, group, "0").Err())
	err := client.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "BUSYGROUP"))

	groups, err := client.XInfoGroups(ctx, stream).Result()
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, group, groups[0].Name)

	first, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]any{"payload": "one"},
	}).Result()
	require.NoError(t, err)
	second, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		MaxLen: 1000,
		Approx: true,
		ID:     "*",
		Values: []any{"payload", "two", "n", 2},
	}).Result()
	require.NoError(t, err)

	read, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: "alice",
		Streams:  []string{stream, ">"},
		Count:    10,
		Block:    time.Second,
	}).Result()
	require.NoError(t, err)
	require.Len(t, read, 1)
	assert.Equal(t, stream, read[0].Stream)
	require.Len(t, read[0].Messages, 2)
	assert.Equal(t, redis.XMessage{ID: first, Values: map[string]any{"payload": "one"}},
		read[0].Messages[0])
	assert.Equal(t, map[string]any{"payload": "two", "n": "2"}, read[0].Messages[1].Values)

	_, err = client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: "alice",
		Streams:  []string{stream, ">"},
		Block:    50 * time.Millisecond,
	}).Result()
	require.ErrorIs(t, err, redis.Nil, "a blocking read that times out with nothing new is Nil")

	pending, err := client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: stream, Group: group, Start: "-", End: "+", Count: 10,
	}).Result()
	require.NoError(t, err)
	require.Len(t, pending, 2)
	assert.Equal(t, first, pending[0].ID)
	assert.Equal(t, "alice", pending[0].Consumer)
	assert.EqualValues(t, 1, pending[0].RetryCount)

	time.Sleep(20 * time.Millisecond)
	idle, err := client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: stream, Group: group, Idle: 10 * time.Millisecond,
		Start: "-", End: "+", Count: 10, Consumer: "alice",
	}).Result()
	require.NoError(t, err)
	assert.Len(t, idle, 2)
	assert.GreaterOrEqual(t, idle[0].Idle, 10*time.Millisecond)

	claimed, err := client.XClaim(ctx, &redis.XClaimArgs{
		Stream: stream, Group: group, Consumer: "bob",
		MinIdle: 0, Messages: []string{first},
	}).Result()
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	assert.Equal(t, first, claimed[0].ID)

	consumers, err := client.XInfoConsumers(ctx, stream, group).Result()
	require.NoError(t, err)
	names := make([]string, 0, len(consumers))
	for _, c := range consumers {
		names = append(names, c.Name)
		assert.GreaterOrEqual(t, c.Idle, time.Duration(0))
	}
	assert.ElementsMatch(t, []string{"alice", "bob"}, names)

	acked, err := client.XAck(ctx, stream, group, first, second).Result()
	require.NoError(t, err)
	assert.EqualValues(t, 2, acked)

	dropped, err := client.XGroupDelConsumer(ctx, stream, group, "bob").Result()
	require.NoError(t, err)
	assert.Zero(t, dropped, "bob held nothing pending after the ack")
}

func TestStreamRangesReadsAndTrims(t *testing.T) {
	client := newClient(t, nil)
	ctx := t.Context()
	const stream = "log"

	ids := make([]string, 0, 5)
	for idx := range 5 {
		id, err := client.XAdd(ctx, &redis.XAddArgs{
			Stream: stream,
			Values: []string{"n", string(rune('a' + idx))},
		}).Result()
		require.NoError(t, err)
		ids = append(ids, id)
	}

	newest, err := client.XRevRangeN(ctx, stream, "+", "-", 1).Result()
	require.NoError(t, err)
	require.Len(t, newest, 1)
	assert.Equal(t, ids[4], newest[0].ID)

	after, err := client.XRangeN(ctx, stream, "("+ids[1], "+", 2).Result()
	require.NoError(t, err)
	require.Len(t, after, 2)
	assert.Equal(t, []string{ids[2], ids[3]}, []string{after[0].ID, after[1].ID})

	read, err := client.XRead(ctx, &redis.XReadArgs{
		Streams: []string{stream, ids[3]},
		Count:   10,
		Block:   -1,
	}).Result()
	require.NoError(t, err)
	require.Len(t, read, 1)
	require.Len(t, read[0].Messages, 1)
	assert.Equal(t, ids[4], read[0].Messages[0].ID)

	_, err = client.XRead(ctx, &redis.XReadArgs{
		Streams: []string{stream, "$"},
		Block:   100 * time.Millisecond,
	}).Result()
	require.ErrorIs(t, err, redis.Nil)

	trimmed, err := client.XTrimMaxLenApprox(ctx, stream, 0, 0).Result()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, trimmed, int64(0))
	size, err := client.XLen(ctx, stream).Result()
	require.NoError(t, err)
	assert.LessOrEqual(t, size, int64(5))

	exact, err := client.Do(ctx, "xtrim", stream, "maxlen", 0).Int()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, exact, 0)
	size, err = client.XLen(ctx, stream).Result()
	require.NoError(t, err)
	assert.Zero(t, size)

	empty, err := client.XRangeN(ctx, "missing", "-", "+", 1).Result()
	require.NoError(t, err)
	assert.Empty(t, empty)
}
