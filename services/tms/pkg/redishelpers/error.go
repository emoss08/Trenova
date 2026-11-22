package redishelpers

import (
	"errors"

	"github.com/emoss08/trenova/shared/redis"
)

func IsRedisNil(err error) bool {
	return errors.Is(err, redis.Nil)
}
