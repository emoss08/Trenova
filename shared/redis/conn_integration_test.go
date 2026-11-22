//go:build integration

package redis_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/redis"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/embedded"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestCancellingTheContextAbortsABlockingRead(t *testing.T) {
	client := newClient(t, func(o *redis.Options) { o.PoolSize = 1 })
	ctx, cancel := context.WithCancel(t.Context())
	time.AfterFunc(100*time.Millisecond, cancel)

	start := time.Now()
	err := client.XRead(ctx, &redis.XReadArgs{Streams: []string{"quiet", "$"}, Block: 0}).Err()
	require.ErrorIs(t, err, context.Canceled)
	assert.Less(t, time.Since(start), 2*time.Second, "a read that blocks forever still ends")

	require.NoError(t, client.Ping(t.Context()).Err(), "the interrupted connection was replaced")
}

func TestAContextDeadlineEndsABlockingReadEarly(t *testing.T) {
	client := newClient(t, nil)
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := client.XRead(ctx, &redis.XReadArgs{
		Streams: []string{"quiet", "$"},
		Block:   10 * time.Second,
	}).Err()
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 2*time.Second)
}

func TestABlockingReadOutlivesTheReadTimeout(t *testing.T) {
	client := newClient(t, func(o *redis.Options) { o.ReadTimeout = 100 * time.Millisecond })

	start := time.Now()
	err := client.XRead(t.Context(), &redis.XReadArgs{
		Streams: []string{"quiet", "$"},
		Block:   300 * time.Millisecond,
	}).Err()
	require.ErrorIs(t, err, redis.Nil, "the server's block ends the read, not the read timeout")
	assert.GreaterOrEqual(t, time.Since(start), 300*time.Millisecond)
}

func TestACancelledContextFailsBeforeSending(t *testing.T) {
	client := newClient(t, nil)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	require.ErrorIs(t, client.Set(ctx, "k", "v", 0).Err(), context.Canceled)
	require.ErrorIs(t, client.Get(t.Context(), "k").Err(), redis.Nil)
}

func killClient(t *testing.T, id int64) {
	t.Helper()
	killer := newClient(t, func(*redis.Options) {})
	require.NoError(t, killer.Do(t.Context(), "client", "kill", "id", id).Err())
}

func TestABrokenConnectionIsDiscardedAndRetried(t *testing.T) {
	client := newClient(t, func(o *redis.Options) { o.PoolSize = 1 })
	ctx := t.Context()

	before, err := client.ClientID(ctx).Result()
	require.NoError(t, err)
	killClient(t, before)

	require.NoError(t, client.Ping(ctx).Err(), "the dead connection is retried on a new one")
	after, err := client.ClientID(ctx).Result()
	require.NoError(t, err)
	assert.NotEqual(t, before, after)
}

func TestABrokenConnectionIsNeverReturnedToThePool(t *testing.T) {
	client := newClient(t, func(o *redis.Options) {
		o.PoolSize = 1
		o.MaxRetries = -1
	})
	ctx := t.Context()

	before, err := client.ClientID(ctx).Result()
	require.NoError(t, err)
	killClient(t, before)

	require.Error(t, client.Ping(ctx).Err(), "without retries the failure reaches the caller")
	after, err := client.ClientID(ctx).Result()
	require.NoError(t, err)
	assert.NotEqual(t, before, after, "the next command runs on a fresh connection")
}

func TestAnExhaustedPoolTimesOut(t *testing.T) {
	client := newClient(t, func(o *redis.Options) {
		o.PoolSize = 1
		o.PoolTimeout = 100 * time.Millisecond
		o.MaxRetries = -1
	})
	ctx := t.Context()

	held := client.Conn()
	require.NoError(t, held.Ping(ctx).Err())

	start := time.Now()
	require.ErrorIs(t, client.Ping(ctx).Err(), redis.ErrPoolTimeout)
	assert.GreaterOrEqual(t, time.Since(start), 100*time.Millisecond)

	require.NoError(t, held.Close())
	require.NoError(t, client.Ping(ctx).Err(), "closing the Conn returns its connection")
	require.ErrorIs(t, held.Ping(ctx).Err(), redis.ErrClosed)
}

func TestAConnKeepsOneServerConnection(t *testing.T) {
	client := newClient(t, nil)
	ctx := t.Context()

	conn := client.Conn()
	t.Cleanup(func() { _ = conn.Close() })
	first, err := conn.ClientID(ctx).Result()
	require.NoError(t, err)
	for range 3 {
		id, err := conn.ClientID(ctx).Result()
		require.NoError(t, err)
		assert.Equal(t, first, id)
	}

	killClient(t, first)
	require.Error(t, conn.Ping(ctx).Err())
	require.Error(t, conn.Ping(ctx).Err(), "a Conn whose connection died never switches silently")
}

func TestConcurrentCommandsShareThePool(t *testing.T) {
	client := newClient(t, func(o *redis.Options) { o.PoolSize = 4 })
	ctx := t.Context()

	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			assert.NoError(t, client.Incr(ctx, "hits").Err())
		})
	}
	wg.Wait()
	n, err := client.Get(ctx, "hits").Int64()
	require.NoError(t, err)
	assert.EqualValues(t, 50, n)
}

func TestClosingTheClientRejectsCommands(t *testing.T) {
	client := newClient(t, func(*redis.Options) {})
	require.NoError(t, client.Ping(t.Context()).Err())
	require.NoError(t, client.Close())
	require.ErrorIs(t, client.Ping(t.Context()).Err(), redis.ErrClosed)
}

func TestClosingTheClientEndsAReadThatBlocksForever(t *testing.T) {
	client := newClient(t, nil)
	done := make(chan error, 1)
	go func() {
		done <- client.XRead(
			context.Background(),
			&redis.XReadArgs{Streams: []string{"quiet", "$"}, Block: 0},
		).Err()
	}()
	time.Sleep(100 * time.Millisecond)

	closed := make(chan struct{})
	go func() {
		_ = client.Close()
		close(closed)
	}()

	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("Close waited on a read that blocks forever")
	}
	require.Error(t, <-done)
}

func TestUsernameAndPasswordAuthenticate(t *testing.T) {
	admin := newClient(t, nil)
	ctx := t.Context()
	user := "tester-" + ulid.Make().String()
	require.NoError(t, admin.Do(ctx, "acl", "setuser", user, "on", ">s3cret", "~*", "+@all").Err())
	t.Cleanup(func() {
		_ = admin.Do(context.WithoutCancel(ctx), "acl", "deluser", user).Err()
	})

	authed := newClient(t, func(o *redis.Options) {
		o.Username = user
		o.Password = "s3cret"
	})
	whoami, err := authed.Do(ctx, "acl", "whoami").Text()
	require.NoError(t, err)
	assert.Equal(t, user, whoami)

	rejected := newClient(t, func(o *redis.Options) {
		o.Username = user
		o.Password = "wrong"
		o.MaxRetries = -1
	})
	err = rejected.Ping(ctx).Err()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WRONGPASS")
}

type recordedSpan struct {
	name  string
	attrs map[attribute.Key]attribute.Value
}

type recordingProvider struct {
	embedded.TracerProvider
	mu    sync.Mutex
	spans []recordedSpan
}

func (p *recordingProvider) Tracer(string, ...trace.TracerOption) trace.Tracer {
	return &recordingTracer{p: p}
}

func (p *recordingProvider) named(name string) (recordedSpan, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, span := range p.spans {
		if span.name == name {
			return span, true
		}
	}
	return recordedSpan{}, false
}

type recordingTracer struct {
	embedded.Tracer
	p *recordingProvider
}

func (r *recordingTracer) Start(
	ctx context.Context,
	name string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	cfg := trace.NewSpanStartConfig(opts...)
	attrs := make(map[attribute.Key]attribute.Value, len(cfg.Attributes()))
	for _, kv := range cfg.Attributes() {
		attrs[kv.Key] = kv.Value
	}
	r.p.mu.Lock()
	r.p.spans = append(r.p.spans, recordedSpan{name: name, attrs: attrs})
	r.p.mu.Unlock()
	return noop.NewTracerProvider().Tracer("").Start(ctx, name)
}

func TestTracingNamesSpansLikeRedisotel(t *testing.T) {
	provider := &recordingProvider{}
	client := newClient(t, func(o *redis.Options) { o.TracerProvider = provider })
	ctx := t.Context()

	require.NoError(t, client.Set(ctx, "traced", "v", 0).Err())
	_, err := client.TxPipelined(ctx, func(pipe *redis.Pipeline) error {
		pipe.Incr(ctx, "n")
		pipe.Expire(ctx, "n", time.Minute)
		return nil
	})
	require.NoError(t, err)

	set, ok := provider.named("set")
	require.True(t, ok)
	assert.Equal(t, "redis", set.attrs["db.system"].AsString())
	assert.Equal(t, "set traced ?", set.attrs["db.statement"].AsString())
	assert.Equal(t, "redis://"+client.Options().Addr, set.attrs["db.connection_string"].AsString())
	assert.NotEmpty(t, set.attrs["server.address"].AsString())
	assert.Contains(t, set.attrs["code.function"].AsString(), "TestTracingNamesSpansLikeRedisotel")

	pipeline, ok := provider.named("redis.pipeline multi incr expire exec")
	require.True(t, ok)
	assert.EqualValues(t, 4, pipeline.attrs["db.redis.num_cmd"].AsInt64())

	_, dialed := provider.named("redis.dial")
	assert.True(t, dialed)
}
