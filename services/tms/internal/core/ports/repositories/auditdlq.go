package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/shared/pulid"
)

type AuditDLQRepository interface {
	Insert(ctx context.Context, entry *audit.DLQEntry) error
	InsertBatch(ctx context.Context, entries []*audit.DLQEntry) error
	GetPendingEntries(ctx context.Context, limit int) ([]*audit.DLQEntry, error)
	GetByID(ctx context.Context, id pulid.ID) (*audit.DLQEntry, error)
	Update(ctx context.Context, entry *audit.DLQEntry) error
	MarkAsRecovered(ctx context.Context, ids []pulid.ID) error
	MarkAsFailed(ctx context.Context, id pulid.ID, errMsg string) error
	DeleteRecovered(ctx context.Context, olderThan int64) (int64, error)
	Count(ctx context.Context) (int64, error)
	CountByStatus(ctx context.Context, status audit.DLQStatus) (int64, error)
}
