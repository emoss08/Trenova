package common

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvTest        Environment = "test"
	EnvStaging     Environment = "staging"
	EnvProduction  Environment = "production"
)

type OperationType string

const (
	OpMigrate     OperationType = "migrate"
	OpRollback    OperationType = "rollback"
	OpSeed        OperationType = "seed"
	OpReset       OperationType = "reset"
	OpBackup      OperationType = "backup"
	OpRestore     OperationType = "restore"
	OpHealthCheck OperationType = "health_check"
)

type OperationResult struct {
	Type      OperationType
	Success   bool
	Message   string
	Details   map[string]any
	StartTime time.Time
	EndTime   time.Time
	Error     error
}

type OperationOptions struct {
	DryRun      bool
	Force       bool
	Backup      bool
	Interactive bool
	Verbose     bool
	Target      string // specific migration or seed target
	Environment Environment
}

type DatabaseManager interface {
	Migrate(ctx context.Context, opts OperationOptions) (*OperationResult, error)
	Rollback(ctx context.Context, opts OperationOptions) (*OperationResult, error)
	MigrationStatus(ctx context.Context) ([]*MigrationStatus, error)
	Seed(ctx context.Context, opts OperationOptions) (*OperationResult, error)
	SeedStatus(ctx context.Context) ([]*SeedStatus, error)
	Reset(ctx context.Context, opts OperationOptions) (*OperationResult, error)
	HealthCheck(ctx context.Context) (*HealthStatus, error)
	Backup(ctx context.Context, opts OperationOptions) (*OperationResult, error)
	Restore(ctx context.Context, backupFile string, opts OperationOptions) (*OperationResult, error)
}

type MigrationStatus struct {
	ID          int64
	Name        string
	Group       int64
	MigratedAt  time.Time
	Checksum    string
	Applied     bool
	Description string
}

type SeedStatus struct {
	Name        string
	Version     string
	AppliedAt   time.Time
	Checksum    string
	Environment Environment
	Status      string
}

type HealthStatus struct {
	Connected         bool
	Version           string
	MigrationsCurrent bool
	PendingMigrations int
	LastMigration     *time.Time
	LastSeed          *time.Time
	DatabaseSize      string
	TableCount        int
	ConnectionPool    ConnectionPoolStatus
}

type ConnectionPoolStatus struct {
	MaxConnections  int
	OpenConnections int
	InUse           int
	Idle            int
}

type ProgressReporter interface {
	Start(total int, message string)
	Update(current int, message string)
	Complete(message string)
	Error(err error)
}

type ConsoleProgressReporter struct {
	total   int
	current int
}

func NewConsoleProgressReporter() *ConsoleProgressReporter {
	return &ConsoleProgressReporter{}
}

func (r *ConsoleProgressReporter) Start(total int, message string) {
	r.total = total
	r.current = 0
}

func (r *ConsoleProgressReporter) Update(current int, message string) {
	r.current = current
}

func (r *ConsoleProgressReporter) Complete(message string) {
	r.current = r.total
}

func (r *ConsoleProgressReporter) Error(err error) {}

type DatabaseConfig struct {
	DB               *bun.DB
	Environment      Environment
	MigrationsPath   string
	MigrationsTable  string
	SeedsPath        string
	SeedsTable       string
	FixturesPath     string
	BackupPath       string
	BackupRetention  time.Duration
	RequireBackup    bool
	AllowDestructive bool
	MaxRollback      int
}
