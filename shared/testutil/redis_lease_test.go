package testutil

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const leasableRedisDatabases = 15

func leaseTestRedisAddr(t *testing.T) string {
	t.Helper()

	if addr := strings.TrimSpace(os.Getenv(RedisAddrEnv)); addr != "" {
		return addr
	}
	if testing.Short() {
		t.Skip("needs a Redis server: set " + RedisAddrEnv + " or run without -short")
	}
	return SetupRedis(t).Address()
}

func leaseTestPrefix() string {
	return "trenova:test:lease-test:" + ulid.Make().String() + ":"
}

func mustLease(t *testing.T, addr, prefix string) *redisDBLease {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	lease, err := leaseRedisDB(ctx, addr, prefix)
	require.NoError(t, err)
	t.Cleanup(func() { _ = lease.release() })
	return lease
}

func TestLeaseRedisDBGivesEveryHolderItsOwnDatabase(t *testing.T) {
	addr := leaseTestRedisAddr(t)
	prefix := leaseTestPrefix()

	seen := make(map[int]struct{}, leasableRedisDatabases)
	for range leasableRedisDatabases {
		lease := mustLease(t, addr, prefix)
		assert.GreaterOrEqual(t, lease.db, 1, "database 0 holds the leases and is never handed out")
		assert.Less(t, lease.db, 16)
		_, taken := seen[lease.db]
		require.Falsef(t, taken, "database %d was leased twice", lease.db)
		seen[lease.db] = struct{}{}
	}
}

func TestLeaseRedisDBWaitsWhileEveryDatabaseIsHeld(t *testing.T) {
	addr := leaseTestRedisAddr(t)
	prefix := leaseTestPrefix()

	for range leasableRedisDatabases {
		mustLease(t, addr, prefix)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 600*time.Millisecond)
	defer cancel()

	lease, err := leaseRedisDB(ctx, addr, prefix)
	if lease != nil {
		_ = lease.release()
	}
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestLeaseRedisDBReclaimsTheDatabaseOfAHolderThatHasGone(t *testing.T) {
	addr := leaseTestRedisAddr(t)
	prefix := leaseTestPrefix()

	leases := make([]*redisDBLease, 0, leasableRedisDatabases)
	for range leasableRedisDatabases {
		leases = append(leases, mustLease(t, addr, prefix))
	}

	gone := leases[len(leases)/2]
	require.NoError(t, gone.release())

	next := mustLease(t, addr, prefix)
	assert.Equal(t, gone.db, next.db)
}
