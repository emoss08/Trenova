package redis

import "github.com/redis/go-redis/v9"

var client *redis.Client

func GetClient() *redis.Client {
	return client
}

func SetClient(newClient *redis.Client) {
	client = newClient
}

func NewRedisClient(addr string) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	return client
}
