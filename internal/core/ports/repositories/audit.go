package repositories

import (
	"context"

	"github.com/trenova-app/transport/internal/core/domain/audit"
)

type AuditRepository interface {
	InsertAuditEntries(ctx context.Context, entries []*audit.Entry) error
}
