package netsuite

import (
	"context"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/netsuiteerrors"
)

const (
	MaxQueryLimit   = 1000
	QueryTimeLayout = "2006-01-02 15:04:05"
	QueryDateLayout = "2006-01-02"
	limitParam      = "limit"
)

type Query struct {
	SQL    string
	Params []any
	Limit  int
}

type QueryPage struct {
	HasMore      bool
	Count        int
	TotalResults int
}

type queryRequest struct {
	Q      string `json:"q"`
	Params []any  `json:"params,omitempty"`
}

type queryResponse struct {
	Items        any  `json:"items"`
	HasMore      bool `json:"hasMore"`
	Count        int  `json:"count"`
	TotalResults int  `json:"totalResults"`
}

func (c *Client) Query(ctx context.Context, q Query, rows any) (*QueryPage, error) {
	sql := strings.TrimSpace(q.SQL)
	if sql == "" || strings.ContainsRune(sql, 0) {
		return nil, netsuiteerrors.ErrQueryRequired
	}
	if q.Limit > MaxQueryLimit || q.Limit < 0 {
		return nil, netsuiteerrors.ErrInvalidLimit
	}
	limit := q.Limit
	if limit == 0 {
		limit = MaxQueryLimit
	}
	target := reflect.ValueOf(rows)
	if !target.IsValid() || target.Kind() != reflect.Pointer || target.IsNil() ||
		target.Elem().Kind() != reflect.Slice {
		return nil, netsuiteerrors.ErrInvalidRows
	}
	target.Elem().SetLen(0)

	wire := queryResponse{Items: rows}
	resp, err := c.core.do(ctx, &call{
		endpoint:   "suiteql",
		method:     http.MethodPost,
		path:       suiteQLPath,
		query:      url.Values{limitParam: {strconv.Itoa(limit)}},
		body:       &queryRequest{Q: sql, Params: q.Params},
		out:        &wire,
		prefer:     preferTransient,
		idempotent: true,
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Body) == 0 {
		return nil, netsuiteerrors.ErrUnexpectedPayload
	}
	return &QueryPage{
		HasMore:      wire.HasMore,
		Count:        wire.Count,
		TotalResults: wire.TotalResults,
	}, nil
}

func ParseQueryTime(value string) (time.Time, error) {
	parsed, err := time.ParseInLocation(QueryTimeLayout, strings.TrimSpace(value), time.UTC)
	if err != nil {
		return time.Time{}, netsuiteerrors.ErrInvalidQueryTime
	}
	return parsed, nil
}

func ParseQueryDate(value string) (time.Time, error) {
	parsed, err := time.ParseInLocation(QueryDateLayout, strings.TrimSpace(value), time.UTC)
	if err != nil {
		return time.Time{}, netsuiteerrors.ErrInvalidQueryTime
	}
	return parsed, nil
}

func FormatQueryTime(t time.Time) string {
	return t.UTC().Format(QueryTimeLayout)
}
