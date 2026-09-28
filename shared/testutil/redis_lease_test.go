//go:build integration

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

func TestLeaseRedisDB(t *testing.T) {
	addr := strings.TrimSpace(os.Getenv(RedisAddrEnv))
	if addr == "" {
		addr = SetupRedis(t).Address()
	}

	t.Run("gives every holder its own database", func(t *testing.T) {
		leaseGivesEveryHolderItsOwnDatabase(t, addr)
	})
	t.Run("waits while every database is held", func(t *testing.T) {
		leaseWaitsWhileEveryDatabaseIsHeld(t, addr)
	})
	t.Run("reclaims the database of a holder that has gone", func(t *testing.T) {
		leaseReclaimsTheDatabaseOfAHolderThatHasGone(t, addr)
	})
}

func leaseGivesEveryHolderItsOwnDatabase(t *testing.T, addr string) {
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

func leaseWaitsWhileEveryDatabaseIsHeld(t *testing.T, addr string) {
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

func leaseReclaimsTheDatabaseOfAHolderThatHasGone(t *testing.T, addr string) {
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
