package wire

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"

	"github.com/emoss08/trenova/shared/redis/internal/pool"
	"github.com/emoss08/trenova/shared/redis/internal/resp"
)

var retryablePrefixes = []string{
	"ERR max number of clients reached",
	"LOADING ",
	"READONLY ",
	"CLUSTERDOWN ",
	"TRYAGAIN ",
	"MASTERDOWN ",
	"NOREPLICAS ",
}

func shouldRetry(err error, retryTimeout bool) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		return true
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, pool.ErrPoolTimeout) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return !netErr.Timeout() || retryTimeout
	}
	var redisErr resp.Error
	if !errors.As(err, &redisErr) {
		return false
	}
	for _, prefix := range retryablePrefixes {
		if strings.HasPrefix(string(redisErr), prefix) {
			return true
		}
	}
	return false
}

func isServerError(err error) bool {
	var redisErr resp.Error
	return errors.As(err, &redisErr) && redisErr != resp.Nil && redisErr != resp.TxFailedErr
}
