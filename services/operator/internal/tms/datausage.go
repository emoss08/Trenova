package tms

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
)

var ErrNoDataUsage = errors.New("this Trenova release does not measure its data")

type DataUsage struct {
	MeasuredAt     time.Time `json:"measuredAt"`
	DatabaseBytes  *int64    `json:"databaseBytes,omitempty"`
	DocumentsBytes *int64    `json:"documentsBytes,omitempty"`
	SearchBytes    *int64    `json:"searchBytes,omitempty"`
}

func (c *Client) DataUsage(ctx context.Context) (DataUsage, error) {
	u := c.base.JoinPath("managed", "v1", "data-usage")
	out, err := c.do(ctx, http.MethodGet, u, nil, http.StatusOK)
	if err != nil {
		var status *StatusError
		if errors.As(err, &status) && status.Status == http.StatusNotFound {
			return DataUsage{}, ErrNoDataUsage
		}
		return DataUsage{}, fmt.Errorf("read what the instance's data takes: %w", err)
	}
	var got DataUsage
	if err = sonic.Unmarshal(out, &got); err != nil {
		return DataUsage{}, fmt.Errorf("read what the instance's data takes: %w", err)
	}
	got.MeasuredAt = got.MeasuredAt.UTC()
	got.DatabaseBytes = nonNegative(got.DatabaseBytes)
	got.DocumentsBytes = nonNegative(got.DocumentsBytes)
	got.SearchBytes = nonNegative(got.SearchBytes)
	return got, nil
}

func nonNegative(n *int64) *int64 {
	if n == nil || *n < 0 {
		return nil
	}
	return n
}
