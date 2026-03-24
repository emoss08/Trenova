package ports

import (
	"context"

	"github.com/emoss08/gtc/internal/core/domain"
)

type RecordHandler func(ctx context.Context, record domain.SourceRecord) error
type TransactionHandler func(ctx context.Context, tx domain.TransactionRecords) error

type TailReader interface {
	Start(ctx context.Context, startLSN string, handler TransactionHandler) error
	Stop(ctx context.Context) error
	AdvanceLSN(lsn string) error
	CurrentLSN() string
}

type SnapshotReader interface {
	CurrentLSN(ctx context.Context) (string, error)
	Run(ctx context.Context, bindings []domain.SnapshotBinding, handler RecordHandler) error
	Backfill(ctx context.Context, bindings []domain.SnapshotBinding, handler RecordHandler) error
	HealthCheck(ctx context.Context) error
}

type SnapshotProgress struct {
	TableName string
	Cursor    string
	Completed bool
}

type MetadataStore interface {
	LoadTableMetadata(ctx context.Context, schema string, table string) (domain.TableMetadata, error)
}

type CheckpointStore interface {
	Ensure(ctx context.Context) error
	HealthCheck(ctx context.Context) error
	LoadBootstrapLSN(ctx context.Context) (string, error)
	SaveBootstrapLSN(ctx context.Context, lsn string) error
	LoadWALLSN(ctx context.Context) (string, error)
	SaveWALLSN(ctx context.Context, lsn string) error
	LoadSnapshotProgress(ctx context.Context, tableName string) (SnapshotProgress, error)
	SaveSnapshotProgress(ctx context.Context, progress SnapshotProgress) error
}

type Sink interface {
	Kind() domain.DestinationKind
	Name() string
	Initialize(ctx context.Context) error
	Write(ctx context.Context, projection domain.Projection, record domain.SourceRecord) error
	HealthCheck(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

type DeadLetterWriter interface {
	Write(ctx context.Context, entry domain.DeadLetterRecord) error
}
