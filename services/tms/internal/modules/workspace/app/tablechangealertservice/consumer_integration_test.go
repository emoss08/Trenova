//go:build integration

package tablechangealertservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/testutil"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// A flush (or a Redis restart without persistence) drops the tca:events key and
// its consumer group after Start. The consumer must recreate the group and keep
// reading what GTC writes next, instead of failing every read with NOGROUP.
func TestConsumerResumesReadingAfterStreamKeyIsDropped(t *testing.T) {
	ctx := context.Background()
	client := testutil.SetupTestRedis(t)

	consumer := NewConsumer(ConsumerParams{Logger: zap.NewNop(), Redis: client})
	consumer.Start(ctx)
	t.Cleanup(consumer.Stop)

	groups, err := client.XInfoGroups(ctx, tcaStreamName).Result()
	require.NoError(t, err)
	require.Len(t, groups, 1)

	require.NoError(t, client.Del(ctx, tcaStreamName).Err())

	// The group comes back on the consumer's next failed read.
	require.Eventually(t, func() bool {
		groups, err := client.XInfoGroups(ctx, tcaStreamName).Result()
		return err == nil && len(groups) == 1 && groups[0].Name == tcaConsumerGroup
	}, 15*time.Second, 100*time.Millisecond)

	// An entry written after the flush is delivered to the group. Its payload
	// does not decode, so the consumer leaves it pending instead of acking it,
	// which is how this test observes the delivery.
	id, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: tcaStreamName,
		Values: map[string]any{"payload": "not json"},
	}).Result()
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		pending, err := client.XPendingExt(ctx, &redis.XPendingExtArgs{
			Stream: tcaStreamName,
			Group:  tcaConsumerGroup,
			Start:  "-",
			End:    "+",
			Count:  10,
		}).Result()
		return err == nil && len(pending) == 1 &&
			pending[0].ID == id && pending[0].Consumer == tcaConsumerName
	}, 15*time.Second, 100*time.Millisecond)
}
