package testutil

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	sharedRedisContainer *RedisContainer
	sharedRedisOnce      sync.Once
	sharedRedisErr       error
)

type RedisContainer struct {
	container testcontainers.Container
	address   string
}

func (r *RedisContainer) Address() string {
	return r.address
}

func (r *RedisContainer) Client() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: r.address,
	})
}

func (r *RedisContainer) Terminate(ctx context.Context) error {
	if r.container != nil {
		return r.container.Terminate(ctx)
	}
	return nil
}

type RedisOptions struct {
	Image string
}

func DefaultRedisOptions() RedisOptions {
	return RedisOptions{
		Image: "redis:8",
	}
}

func SetupRedis(t *testing.T, opts ...func(*RedisOptions)) *RedisContainer {
	t.Helper()

	options := DefaultRedisOptions()
	for _, opt := range opts {
		opt(&options)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	req := testcontainers.ContainerRequest{
		Image:        options.Image,
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "failed to start redis container")

	host, err := container.Host(ctx)
	require.NoError(t, err, "failed to get redis host")

	port, err := container.MappedPort(ctx, "6379")
	require.NoError(t, err, "failed to get redis port")

	rc := &RedisContainer{
		container: container,
		address:   fmt.Sprintf("%s:%s", host, port.Port()),
	}

	t.Cleanup(func() {
		rc.Terminate(context.Background()) //nolint:errcheck
	})

	return rc
}

func getSharedRedis() (*RedisContainer, error) {
	sharedRedisOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		req := testcontainers.ContainerRequest{
			Name:         "trenova-test-redis",
			Image:        "redis:8",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForLog("Ready to accept connections"),
		}

		container, err := testcontainers.GenericContainer(
			ctx,
			testcontainers.GenericContainerRequest{
				ContainerRequest: req,
				Started:          true,
				Reuse:            true,
			},
		)
		if err != nil {
			sharedRedisErr = fmt.Errorf("failed to start redis container: %w", err)
			return
		}

		host, err := container.Host(ctx)
		if err != nil {
			sharedRedisErr = fmt.Errorf("failed to get redis host: %w", err)
			return
		}

		port, err := container.MappedPort(ctx, "6379")
		if err != nil {
			sharedRedisErr = fmt.Errorf("failed to get redis port: %w", err)
			return
		}

		sharedRedisContainer = &RedisContainer{
			container: container,
			address:   fmt.Sprintf("%s:%s", host, port.Port()),
		}
	})

	return sharedRedisContainer, sharedRedisErr
}

func SetupTestRedis(t *testing.T) *redis.Client {
	t.Helper()

	rc, err := getSharedRedis()
	require.NoError(t, err, "failed to get shared redis container")

	client := rc.Client()

	client.FlushAll(context.Background())

	t.Cleanup(func() {
		client.Close()
	})

	return client
}
