package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/audit"
)

type AuditRepository interface {
	InsertAuditEntries(ctx context.Context, entries []*audit.Entry) error
}
