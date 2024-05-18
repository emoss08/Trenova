// Package redis provides a client for interacting with a Redis server.
package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const Nil = redis.Nil

// Client represents a client to the Redis server.
type Client struct {
	config *redis.Options // config holds the configuration options for the client.
	client *redis.Client  // client represents the Redis client.
}

// NewClient creates a new Redis client using the given options.
//
// It returns an initialized Redis client.
func NewClient(config *redis.Options) *Client {
	rClient := &Client{config: config}
	rClient.initialize()
	return rClient
}

// initialize sets up the Redis client. It is called only by NewClient.
func (c *Client) initialize() {
	c.client = redis.NewClient(c.config)
}

// Close terminates the connection to the Redis server. It does nothing if the client is nil.
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// Ping verifies the connection to the Redis server.
//
// It returns an error if the connection check fails.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx).Result()
	return err
}

// Get retrieves the value associated with the key from the Redis server.
//
// It returns the value or an error if the key does not exist or another error occurs.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	value, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return value, nil
}

// Set assigns the value to the key with an expiration time on the Redis server.
//
// It returns an error if the operation fails.
func (c *Client) Set(ctx context.Context, key string, value any, timeout time.Duration) error {
	_, err := c.client.Set(ctx, key, value, timeout).Result()
	return err
}

func (c *Client) Del(ctx context.Context, keys ...string) error {
	_, err := c.client.Del(ctx, keys...).Result()
	return err
}
