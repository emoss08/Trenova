package meilisearch

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/datausageports"
)

const indexSizeTimeout = 5 * time.Second

type indexSize struct {
	c *Client
}

func NewIndexSize(c *Client) datausageports.Source {
	return &indexSize{c: c}
}

func (s *indexSize) Figure() datausageports.Figure {
	return datausageports.FigureSearch
}

func (s *indexSize) Timeout() time.Duration {
	return indexSizeTimeout
}

func (s *indexSize) Measure(ctx context.Context) (int64, error) {
	if !s.c.Enabled() {
		return 0, datausageports.ErrNotMeasured
	}
	stats, err := s.c.client.GetStatsWithContext(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("read the search index stats: %w", err)
	}
	return databaseBytes(stats.DatabaseSize)
}

func databaseBytes(raw any) (int64, error) {
	switch v := raw.(type) {
	case float64:
		if v < 0 || v >= math.MaxInt64 || v != math.Trunc(v) {
			return 0, fmt.Errorf("the search index reported a database size of %v", v)
		}
		return int64(v), nil
	case int64:
		return v, nil
	case interface{ Int64() (int64, error) }:
		return v.Int64()
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("the search index reported a database size of %q", v)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("the search index reported no database size (%T)", raw)
	}
}
