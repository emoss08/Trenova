package testutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	redisDatabaseCount  = 16
	redisLeaseDatabase  = 0
	redisLeaseKeyPrefix = "trenova:test:redis-db-lease:"
	redisLeaseRetry     = 100 * time.Millisecond
)

var takeOverRedisLease = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	redis.call("SET", KEYS[1], ARGV[2])
	return 1
end
return 0
`)

type redisDBLease struct {
	client *redis.Client
	conn   *redis.Conn
	db     int
}

func (l *redisDBLease) release() error {
	return errors.Join(l.conn.Close(), l.client.Close())
}

func leaseRedisDB(ctx context.Context, addr, keyPrefix string) (*redisDBLease, error) {
	client := redis.NewClient(&redis.Options{Addr: addr, DB: redisLeaseDatabase})
	conn := client.Conn()
	lease := &redisDBLease{client: client, conn: conn}

	id, err := conn.ClientID(ctx).Result()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("identify the Redis lease connection: %w", err), lease.release())
	}
	holder := strconv.FormatInt(id, 10)

	leasable := redisDatabaseCount - 1
	start := os.Getpid() % leasable
	for {
		for offset := range leasable {
			db := 1 + (start+offset)%leasable
			claimed, claimErr := claimRedisDB(ctx, conn, keyPrefix+strconv.Itoa(db), holder)
			if claimErr != nil {
				return nil, errors.Join(
					fmt.Errorf("lease Redis database %d: %w", db, claimErr),
					lease.release(),
				)
			}
			if claimed {
				lease.db = db
				return lease, nil
			}
		}

		timer := time.NewTimer(redisLeaseRetry)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, errors.Join(
				fmt.Errorf("every Redis database on %s is leased by a running test process: %w",
					addr, ctx.Err()),
				lease.release(),
			)
		case <-timer.C:
		}
	}
}

func claimRedisDB(ctx context.Context, conn *redis.Conn, key, holder string) (bool, error) {
	claimed, err := conn.SetNX(ctx, key, holder, 0).Result()
	if err != nil || claimed {
		return claimed, err
	}

	current, err := conn.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	connected, err := redisClientConnected(ctx, conn, current)
	if err != nil || connected {
		return false, err
	}

	taken, err := takeOverRedisLease.Run(ctx, conn, []string{key}, current, holder).Int()
	return taken == 1, err
}

func redisClientConnected(ctx context.Context, conn *redis.Conn, id string) (bool, error) {
	if _, err := strconv.ParseUint(id, 10, 64); err != nil {
		return false, nil
	}
	clients, err := conn.Do(ctx, "CLIENT", "LIST", "ID", id).Text()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(clients) != "", nil
}
