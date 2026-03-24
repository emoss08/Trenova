package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/system"
)

type DatabaseSessionRepository interface {
	ListBlocked(ctx context.Context) ([]*system.DatabaseSessionChain, error)
	Terminate(ctx context.Context, pid int64) (*system.TerminateDatabaseSessionResult, error)
}
