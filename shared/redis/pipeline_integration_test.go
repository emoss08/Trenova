//go:build integration

package redis_test

import (
	"crypto/sha1" //nolint:gosec // the digest EVALSHA names a script by
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineReturnsEveryReplyAndTheFirstError(t *testing.T) {
	client := newClient(t, nil)
	ctx := t.Context()
	require.NoError(t, client.SAdd(ctx, "set", "a").Err())

	pipe := client.Pipeline()
	set := pipe.Set(ctx, "k", "v", 0)
	missing := pipe.Get(ctx, "missing")
	wrong := pipe.Incr(ctx, "set")
	got := pipe.Get(ctx, "k")
	assert.NoError(t, set.Err(), "nothing runs before Exec")

	cmds, err := pipe.Exec(ctx)
	require.ErrorIs(t, err, redis.Nil, "the first failed command's error")
	require.Len(t, cmds, 4)
	require.NoError(t, set.Err())
	require.ErrorIs(t, missing.Err(), redis.Nil)
	require.Error(t, wrong.Err())
	assert.Contains(t, wrong.Err().Error(), "WRONGTYPE")
	assert.Equal(t, "v", got.Val())

	none, err := pipe.Exec(ctx)
	require.NoError(t, err)
	assert.Empty(t, none, "Exec drains the pipeline")

	pipe.Incr(ctx, "counter")
	again, err := pipe.Exec(ctx)
	require.NoError(t, err)
	assert.Len(t, again, 1, "a drained pipeline can be filled again")
}

func TestPipelinedRunsTheQueueBuiltByFn(t *testing.T) {
	client := newClient(t, nil)
	ctx := t.Context()

	var ttl *redis.DurationCmd
	cmds, err := client.Pipelined(ctx, func(pipe *redis.Pipeline) error {
		pipe.Set(ctx, "a", "1", 0)
		pipe.Expire(ctx, "a", time.Minute)
		ttl = pipe.PTTL(ctx, "a")
		return nil
	})
	require.NoError(t, err)
	assert.Len(t, cmds, 3)
	assert.Positive(t, ttl.Val())

	stop := errors.New("stop")
	cmds, err = client.Pipelined(ctx, func(pipe *redis.Pipeline) error {
		pipe.Set(ctx, "b", "1", 0)
		return stop
	})
	require.ErrorIs(t, err, stop)
	assert.Nil(t, cmds)
	require.ErrorIs(t, client.Get(ctx, "b").Err(), redis.Nil, "an aborted queue is never sent")
}

func TestTxPipelineAppliesAtomically(t *testing.T) {
	client := newClient(t, nil)
	ctx := t.Context()

	pipe := client.TxPipeline()
	incr := pipe.Incr(ctx, "n")
	pipe.Expire(ctx, "n", time.Minute)
	held := pipe.HGet(ctx, "h", "missing")
	_, err := pipe.Exec(ctx)
	require.ErrorIs(t, err, redis.Nil)
	assert.EqualValues(t, 1, incr.Val())
	require.ErrorIs(t, held.Err(), redis.Nil)

	_, err = client.TxPipelined(ctx, func(pipe *redis.Pipeline) error {
		pipe.Incr(ctx, "n")
		pipe.Do(ctx, "set", "only-a-key")
		return nil
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "EXECABORT")
	value, err := client.Get(ctx, "n").Int64()
	require.NoError(t, err)
	assert.EqualValues(t, 1, value, "a rejected command aborts the whole transaction")
}

func TestWatchFailsTheTransactionWhenAWatchedKeyChanges(t *testing.T) {
	client := newClient(t, nil)
	other := newClient(t, func(*redis.Options) {})
	ctx := t.Context()
	require.NoError(t, client.Set(ctx, "balance", "10", 0).Err())

	err := client.Watch(ctx, func(tx *redis.Tx) error {
		current, err := tx.Get(ctx, "balance").Int64()
		if err != nil {
			return err
		}
		require.NoError(t, other.Set(ctx, "balance", "99", 0).Err())
		_, err = tx.TxPipelined(ctx, func(pipe *redis.Pipeline) error {
			pipe.Set(ctx, "balance", current+1, 0)
			return nil
		})
		return err
	}, "balance")
	require.ErrorIs(t, err, redis.TxFailedErr)
	assert.Equal(t, "99", client.Get(ctx, "balance").Val())

	err = client.Watch(ctx, func(tx *redis.Tx) error {
		current, err := tx.Get(ctx, "balance").Int64()
		if err != nil {
			return err
		}
		_, err = tx.TxPipelined(ctx, func(pipe *redis.Pipeline) error {
			pipe.Set(ctx, "balance", current+1, 0)
			return nil
		})
		return err
	}, "balance")
	require.NoError(t, err)
	assert.Equal(t, "100", client.Get(ctx, "balance").Val())
}

func TestWatchReleasesItsConnectionWithoutAWatchLeft(t *testing.T) {
	client := newClient(t, func(o *redis.Options) { o.PoolSize = 1 })
	other := newClient(t, func(*redis.Options) {})
	ctx := t.Context()

	stop := errors.New("stop")
	err := client.Watch(ctx, func(*redis.Tx) error { return stop }, "w")
	require.ErrorIs(t, err, stop)

	require.NoError(t, other.Set(ctx, "w", "changed", 0).Err())
	_, err = client.TxPipelined(ctx, func(pipe *redis.Pipeline) error {
		pipe.Set(ctx, "after", "1", 0)
		return nil
	})
	require.NoError(t, err, "the single pooled connection carries no stale WATCH")
}

const counterSource = `
local n = redis.call("INCRBY", KEYS[1], ARGV[1])
return {n, ARGV[2]}
`

var counterScript = redis.NewScript(counterSource)

func TestScriptFallsBackToEvalWhenTheServerDoesNotKnowIt(t *testing.T) {
	client := newClient(t, nil)
	ctx := t.Context()
	require.NoError(t, client.Do(ctx, "script", "flush").Err())

	reply, err := counterScript.Run(ctx, client, []string{"script:n"}, 5, "tag").Slice()
	require.NoError(t, err)
	assert.Equal(t, []any{int64(5), "tag"}, reply)

	digest := sha1.Sum([]byte(counterSource))
	known, err := client.Do(ctx, "script", "exists", hex.EncodeToString(digest[:])).Slice()
	require.NoError(t, err)
	assert.Equal(t, []any{int64(1)}, known, "EVAL cached the script for the next EVALSHA")

	reply, err = counterScript.Run(ctx, client, []string{"script:n"}, 2, "again").Slice()
	require.NoError(t, err)
	assert.Equal(t, []any{int64(7), "again"}, reply)

	ints, err := redis.NewScript(`return {1, "2", 3}`).Run(ctx, client, nil).Int64Slice()
	require.NoError(t, err)
	assert.Equal(t, []int64{1, 2, 3}, ints)

	strs, err := redis.NewScript(`return {"a", "b"}`).Run(ctx, client, nil).StringSlice()
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, strs)

	_, err = redis.NewScript(`return false`).Run(ctx, client, nil).Int()
	require.ErrorIs(t, err, redis.Nil)

	err = redis.NewScript(`return redis.error_reply("BOOM failed")`).Run(ctx, client, nil).Err()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BOOM failed")
}
