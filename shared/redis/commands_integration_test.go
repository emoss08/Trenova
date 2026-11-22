//go:build integration

package redis_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/redis"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newClient(t *testing.T, tune func(*redis.Options)) *redis.Client {
	t.Helper()
	base := testutil.SetupTestRedis(t)
	if tune == nil {
		return base
	}
	opt := *base.Options()
	tune(&opt)
	client := redis.NewClient(&opt)
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestStringCommands(t *testing.T) {
	client := newClient(t, nil)
	ctx := t.Context()

	_, err := client.Get(ctx, "missing").Result()
	require.ErrorIs(t, err, redis.Nil)
	_, err = client.Get(ctx, "missing").Bytes()
	require.ErrorIs(t, err, redis.Nil)

	require.NoError(t, client.Set(ctx, "k", []byte("v"), 0).Err())
	got, err := client.Get(ctx, "k").Result()
	require.NoError(t, err)
	assert.Equal(t, "v", got)
	assert.Equal(t, "v", client.Get(ctx, "k").Val())

	require.NoError(t, client.Set(ctx, "short", "x", 1500*time.Millisecond).Err())
	pttl, err := client.PTTL(ctx, "short").Result()
	require.NoError(t, err)
	assert.Greater(t, pttl, time.Second)
	assert.LessOrEqual(t, pttl, 1500*time.Millisecond)

	require.NoError(t, client.Set(ctx, "long", "x", time.Minute).Err())
	ttl, err := client.TTL(ctx, "long").Result()
	require.NoError(t, err)
	assert.Equal(t, time.Minute, ttl)
	missingTTL, err := client.PTTL(ctx, "missing").Result()
	require.NoError(t, err)
	assert.Equal(t, time.Duration(-2), missingTTL)

	n, err := client.Incr(ctx, "counter").Result()
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)
	asInt, err := client.Get(ctx, "counter").Int64()
	require.NoError(t, err)
	assert.EqualValues(t, 1, asInt)

	set, err := client.SetNX(ctx, "nx", "a", 0).Result()
	require.NoError(t, err)
	assert.True(t, set)
	set, err = client.SetNX(ctx, "nx", "b", 0).Result()
	require.NoError(t, err)
	assert.False(t, set)
	set, err = client.SetNX(ctx, "nx-ttl", "a", time.Minute).Result()
	require.NoError(t, err)
	assert.True(t, set)
	set, err = client.SetNX(ctx, "nx-ttl", "b", time.Minute).Result()
	require.NoError(t, err)
	assert.False(t, set)

	_, err = client.SetArgs(ctx, "lease", "1", redis.SetArgs{Mode: "NX", TTL: time.Minute}).Result()
	require.NoError(t, err)
	_, err = client.SetArgs(ctx, "lease", "2", redis.SetArgs{Mode: "NX", TTL: time.Minute}).Result()
	require.ErrorIs(t, err, redis.Nil)

	taken, err := client.GetDel(ctx, "lease").Result()
	require.NoError(t, err)
	assert.Equal(t, "1", taken)
	_, err = client.GetDel(ctx, "lease").Result()
	require.ErrorIs(t, err, redis.Nil)

	exists, err := client.Exists(ctx, "k", "counter", "missing").Result()
	require.NoError(t, err)
	assert.EqualValues(t, 2, exists)

	keys, err := client.Keys(ctx, "nx*").Result()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"nx", "nx-ttl"}, keys)

	removed, err := client.Del(ctx, "k", "missing").Result()
	require.NoError(t, err)
	assert.EqualValues(t, 1, removed)
	unlinked, err := client.Unlink(ctx, "counter").Result()
	require.NoError(t, err)
	assert.EqualValues(t, 1, unlinked)

	require.NoError(t, client.Ping(ctx).Err())
}

func TestExpireCommands(t *testing.T) {
	client := newClient(t, nil)
	ctx := t.Context()

	require.NoError(t, client.Set(ctx, "k", "v", 0).Err())

	ok, err := client.ExpireNX(ctx, "k", time.Minute).Result()
	require.NoError(t, err)
	assert.True(t, ok)
	ok, err = client.ExpireNX(ctx, "k", time.Hour).Result()
	require.NoError(t, err)
	assert.False(t, ok, "NX leaves a key that already expires alone")

	ok, err = client.ExpireGT(ctx, "k", time.Second).Result()
	require.NoError(t, err)
	assert.False(t, ok, "GT never shortens")
	ok, err = client.ExpireGT(ctx, "k", time.Hour).Result()
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = client.Expire(ctx, "k", 10*time.Minute).Result()
	require.NoError(t, err)
	assert.True(t, ok)
	ok, err = client.Expire(ctx, "missing", time.Minute).Result()
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, client.ExpireAt(ctx, "k", time.Now().Add(time.Hour)).Err())
	ttl, err := client.TTL(ctx, "k").Result()
	require.NoError(t, err)
	assert.Greater(t, ttl, 59*time.Minute)
}

func TestScanWalksEveryKey(t *testing.T) {
	client := newClient(t, nil)
	ctx := t.Context()

	for _, key := range []string{"scan:a", "scan:b", "scan:c", "other"} {
		require.NoError(t, client.Set(ctx, key, "1", 0).Err())
	}

	var (
		cursor uint64
		found  []string
	)
	for {
		keys, next, err := client.Scan(ctx, cursor, "scan:*", 1).Result()
		require.NoError(t, err)
		found = append(found, keys...)
		if next == 0 {
			break
		}
		cursor = next
	}
	assert.ElementsMatch(t, []string{"scan:a", "scan:b", "scan:c"}, found)
}

func TestHashCommands(t *testing.T) {
	client := newClient(t, nil)
	ctx := t.Context()

	added, err := client.HSet(ctx, "h", "a", "1", "b", 2).Result()
	require.NoError(t, err)
	assert.EqualValues(t, 2, added)
	require.NoError(t, client.HSet(ctx, "h", map[string]any{"c": "3"}).Err())

	a, err := client.HGet(ctx, "h", "a").Bytes()
	require.NoError(t, err)
	assert.Equal(t, []byte("1"), a)
	_, err = client.HGet(ctx, "h", "missing").Result()
	require.ErrorIs(t, err, redis.Nil)

	values, err := client.HMGet(ctx, "h", "a", "missing", "b").Result()
	require.NoError(t, err)
	assert.Equal(t, []any{"1", nil, "2"}, values)

	all, err := client.HGetAll(ctx, "h").Result()
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"a": "1", "b": "2", "c": "3"}, all)

	removed, err := client.HDel(ctx, "h", "a", "missing").Result()
	require.NoError(t, err)
	assert.EqualValues(t, 1, removed)

	empty, err := client.HGetAll(ctx, "missing").Result()
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestSetAndListCommands(t *testing.T) {
	client := newClient(t, nil)
	ctx := t.Context()

	require.NoError(t, client.SAdd(ctx, "s", "a", "b", "c").Err())
	require.NoError(t, client.SRem(ctx, "s", "b").Err())
	members, err := client.SMembers(ctx, "s").Result()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"a", "c"}, members)

	require.NoError(t, client.LPush(ctx, "l", "1", "2", "3").Err())
	size, err := client.LLen(ctx, "l").Result()
	require.NoError(t, err)
	assert.EqualValues(t, 3, size)
	popped, err := client.RPopCount(ctx, "l", 2).Result()
	require.NoError(t, err)
	assert.Equal(t, []string{"1", "2"}, popped)
	_, err = client.RPopCount(ctx, "missing", 2).Result()
	require.ErrorIs(t, err, redis.Nil)

	require.Error(t, client.LLen(ctx, "s").Err(), "a type error reaches the caller")
}

func TestSortedSetCommands(t *testing.T) {
	client := newClient(t, nil)
	ctx := t.Context()

	added, err := client.ZAdd(ctx, "z",
		redis.Z{Score: 1, Member: "a"},
		redis.Z{Score: 2, Member: "b"},
		redis.Z{Score: 3, Member: "c"},
	).Result()
	require.NoError(t, err)
	assert.EqualValues(t, 3, added)

	require.NoError(t, client.ZAddXX(ctx, "z", redis.Z{Score: 10, Member: "a"}).Err())
	require.NoError(t, client.ZAddXX(ctx, "z", redis.Z{Score: 10, Member: "new"}).Err())
	require.NoError(t, client.ZAddGT(ctx, "z", redis.Z{Score: 1, Member: "b"}).Err())
	require.NoError(t, client.ZAddGT(ctx, "z", redis.Z{Score: 5, Member: "c"}).Err())

	score, err := client.ZScore(ctx, "z", "a").Result()
	require.NoError(t, err)
	assert.InDelta(t, 10, score, 0)
	_, err = client.ZScore(ctx, "z", "new").Result()
	require.ErrorIs(t, err, redis.Nil, "XX never adds")

	scores, err := client.ZMScore(ctx, "z", "b", "missing", "c").Result()
	require.NoError(t, err)
	assert.Equal(t, []float64{2, 0, 5}, scores, "GT only raises; a missing member reads as zero")

	members, err := client.ZRange(ctx, "z", 0, -1).Result()
	require.NoError(t, err)
	assert.Equal(t, []string{"b", "c", "a"}, members)

	byScore, err := client.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key: "z", Start: "(2", Stop: "+inf", ByScore: true,
	}).Result()
	require.NoError(t, err)
	assert.Equal(t, []string{"c", "a"}, byScore)

	reversed, err := client.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key: "z", Start: 0, Stop: -1, Rev: true,
	}).Result()
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "c", "b"}, reversed)

	reversedByScore, err := client.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key: "z", Start: "-inf", Stop: "6", ByScore: true, Rev: true,
	}).Result()
	require.NoError(t, err)
	assert.Equal(t, []string{"c", "b"}, reversedByScore)

	card, err := client.ZCard(ctx, "z").Result()
	require.NoError(t, err)
	assert.EqualValues(t, 3, card)

	popped, err := client.ZPopMin(ctx, "z", 1).Result()
	require.NoError(t, err)
	assert.Equal(t, []redis.Z{{Score: 2, Member: "b"}}, popped)

	removed, err := client.ZRemRangeByScore(ctx, "z", "-inf", "(6").Result()
	require.NoError(t, err)
	assert.EqualValues(t, 1, removed)
	require.NoError(t, client.ZRem(ctx, "z", "a").Err())
	card, err = client.ZCard(ctx, "z").Result()
	require.NoError(t, err)
	assert.Zero(t, card)
}

type session struct {
	ID       string `json:"id"`
	Accessed int64  `json:"lastAccessedAt"`
}

func TestJSONCommands(t *testing.T) {
	client := newClient(t, nil)
	ctx := t.Context()

	_, err := client.JSONGet(ctx, "doc", ".").Result()
	require.ErrorIs(t, err, redis.Nil)

	require.NoError(t, client.JSONSet(ctx, "doc", ".", session{ID: "s1", Accessed: 1}).Err())
	raw, err := client.JSONGet(ctx, "doc", ".").Result()
	require.NoError(t, err)
	assert.JSONEq(t, `{"id":"s1","lastAccessedAt":1}`, raw)

	require.NoError(t, client.JSONSet(ctx, "doc", ".lastAccessedAt", int64(7)).Err())
	require.NoError(t, client.JSONSet(ctx, "other", ".", `{"a":1}`).Err())

	multi, err := client.JSONGet(ctx, "doc", ".id", ".lastAccessedAt").Result()
	require.NoError(t, err)
	assert.JSONEq(t, `{".id":"s1",".lastAccessedAt":7}`, multi)

	err = client.JSONSet(ctx, "doc", ".nope.deep", 1).Err()
	require.ErrorIs(t, err, redis.Nil, "a path below a missing parent is not created")
	require.Error(t, client.JSONSet(ctx, "absent", ".lastAccessedAt", 1).Err())

	require.Error(t, client.JSONSet(ctx, "doc", ".", make(chan int)).Err())
}

func TestDoRunsAnyCommand(t *testing.T) {
	client := newClient(t, nil)
	ctx := t.Context()

	text, err := client.Do(ctx, "echo", "hi").Text()
	require.NoError(t, err)
	assert.Equal(t, "hi", text)

	n, err := client.Do(ctx, "strlen", "missing").Int()
	require.NoError(t, err)
	assert.Zero(t, n)

	_, err = client.Do(ctx, "get", "missing").Text()
	require.ErrorIs(t, err, redis.Nil)

	err = client.Do(ctx, "nosuchcommand").Err()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")

	require.Error(
		t,
		client.Do(ctx, "set", "k", struct{}{}).Err(),
		"unsupported arguments fail before sending",
	)
	require.NoError(t, client.Ping(ctx).Err(), "and leave the connection usable")
}
