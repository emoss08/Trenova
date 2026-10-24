package seeder

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
)

type Seeder interface {
	Execute(ctx context.Context, opts ExecuteOptions) (*ExecutionReport, error)
	Reconcile(ctx context.Context, env common.Environment) (*ExecutionReport, error)
	Status(ctx context.Context) ([]*common.SeedStatus, error)
	Registry() *Registry
}

var (
	_ Seeder = (*Engine)(nil)
)
