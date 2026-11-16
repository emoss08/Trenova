package meilisearch

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/datausageports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type number string

func (n number) Int64() (int64, error) { return 42, nil }

func TestTheIndexSizeReadsTheDatabaseSizeMeilisearchReports(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		raw  any
		want int64
	}{
		"a JSON number": {raw: float64(1 << 30), want: 1 << 30},
		"an integer":    {raw: int64(7), want: 7},
		"a JSON Number": {raw: number("42"), want: 42},
		"a string":      {raw: "9000", want: 9000},
		"zero":          {raw: float64(0), want: 0},
	} {
		got, err := databaseBytes(tc.raw)
		require.NoError(t, err, name)
		assert.Equal(t, tc.want, got, name)
	}

	for name, raw := range map[string]any{
		"nothing":    nil,
		"a negative": float64(-1),
		"a fraction": 1.5,
		"too large":  float64(1 << 63),
		"words":      "a lot",
	} {
		_, err := databaseBytes(raw)
		assert.Error(t, err, name)
	}
}

func TestTheIndexSizeIsNotMeasuredWithoutSearch(t *testing.T) {
	t.Parallel()

	source := NewIndexSize(&Client{enabled: false, logger: zap.NewNop()})

	assert.Equal(t, datausageports.FigureSearch, source.Figure())
	_, err := source.Measure(t.Context())
	require.ErrorIs(t, err, datausageports.ErrNotMeasured)
}
