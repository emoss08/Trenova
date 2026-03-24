package seeder

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
)

type Seeder interface {
	Execute(ctx context.Context, opts ExecuteOptions) (*ExecutionReport, error)
	Status(ctx context.Context) ([]*common.SeedStatus, error)
	Registry() *Registry
}

type DatabaseMigrator interface {
	Initialize(ctx context.Context) error
	Migrate(ctx context.Context, opts common.OperationOptions) (*common.OperationResult, error)
	Rollback(ctx context.Context, opts common.OperationOptions) (*common.OperationResult, error)
	Status(ctx context.Context) ([]*common.MigrationStatus, error)
	Reset(ctx context.Context, opts common.OperationOptions) (*common.OperationResult, error)
	CreateMigration(ctx context.Context, name string, transactional bool) ([]string, error)
}

var (
	_ Seeder = (*Engine)(nil)
)
