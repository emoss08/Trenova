package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/audit"
)

type AuditBufferRepository interface {
	Push(ctx context.Context, entry *audit.Entry) error
	PushBatch(ctx context.Context, entries []*audit.Entry) error
	Pop(ctx context.Context, count int) ([]*audit.Entry, error)
	Size(ctx context.Context) (int64, error)
}
