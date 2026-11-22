package redis

import (
	"errors"
	"strings"

	"github.com/emoss08/trenova/shared/redis/internal/pool"
	"github.com/emoss08/trenova/shared/redis/internal/resp"
)

const (
	Nil         = resp.Nil         //nolint:errname // go-redis names this sentinel; callers match it
	TxFailedErr = resp.TxFailedErr //nolint:errname // go-redis names this sentinel; callers match it
)

var (
	ErrClosed      = pool.ErrClosed
	ErrPoolTimeout = pool.ErrPoolTimeout
)

func hasErrorPrefix(err error, prefix string) bool {
	var redisErr resp.Error
	if !errors.As(err, &redisErr) {
		return false
	}
	return strings.HasPrefix(strings.TrimPrefix(string(redisErr), "ERR "), prefix)
}
