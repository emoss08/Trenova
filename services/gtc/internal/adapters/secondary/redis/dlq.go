package redis

import (
	"context"
	"fmt"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/emoss08/gtc/internal/core/domain"
	"github.com/emoss08/gtc/internal/core/ports"
	"go.uber.org/zap"
)

type DeadLetterWriter struct {
	*baseSink
	stream string
}

var _ ports.DeadLetterWriter = (*DeadLetterWriter)(nil)

func NewDeadLetterWriter(redisURL string, stream string, logger *zap.Logger) (*DeadLetterWriter, error) {
	base, err := newBaseSink(redisURL, logger.With(zap.String("mode", "dlq")))
	if err != nil {
		return nil, err
	}

	return &DeadLetterWriter{
		baseSink: base,
		stream:   stream,
	}, nil
}

func (w *DeadLetterWriter) Write(ctx context.Context, entry domain.DeadLetterRecord) error {
	payload, err := sonic.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal dlq payload: %w", err)
	}

	return w.client.XAdd(ctx, streamArgs(w.stream, string(payload))).Err()
}

func (w *DeadLetterWriter) Close() error {
	return w.client.Close()
}

type DeadLetterEntry struct {
	ID     string
	Record domain.DeadLetterRecord
}

func (w *DeadLetterWriter) Read(ctx context.Context, limit int64) ([]DeadLetterEntry, error) {
	if limit <= 0 {
		limit = 100
	}

	entries, err := w.client.XRangeN(ctx, w.stream, "-", "+", limit).Result()
	if err != nil {
		return nil, fmt.Errorf("read dlq stream: %w", err)
	}

	results := make([]DeadLetterEntry, 0, len(entries))
	for _, entry := range entries {
		raw, ok := entry.Values["payload"]
		if !ok {
			continue
		}

		payload, err := valueToString(raw)
		if err != nil {
			return nil, err
		}

		var record domain.DeadLetterRecord
		if err := sonic.Unmarshal([]byte(payload), &record); err != nil {
			return nil, fmt.Errorf("decode dlq payload %s: %w", entry.ID, err)
		}

		results = append(results, DeadLetterEntry{
			ID:     entry.ID,
			Record: record,
		})
	}

	return results, nil
}

func (w *DeadLetterWriter) Delete(ctx context.Context, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}

	if err := w.client.XDel(ctx, w.stream, ids...).Err(); err != nil {
		return fmt.Errorf("delete dlq entries: %w", err)
	}

	return nil
}

func valueToString(value any) (string, error) {
	switch typed := value.(type) {
	case string:
		return typed, nil
	case []byte:
		return string(typed), nil
	case fmt.Stringer:
		return typed.String(), nil
	case int64:
		return strconv.FormatInt(typed, 10), nil
	default:
		return "", fmt.Errorf("unsupported redis payload type %T", value)
	}
}
