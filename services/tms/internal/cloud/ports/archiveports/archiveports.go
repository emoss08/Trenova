package archiveports

import (
	"context"
	"errors"
	"io"

	"github.com/emoss08/trenova/internal/cloud/archiveplan"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/pkg/dbscope"
)

var ErrMissing = errors.New("the archive location holds no object at that key")

type Bucket interface {
	Put(ctx context.Context, key string, body io.Reader) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	List(ctx context.Context, prefix string, fn func(key string, size int64) error) error
	Children(ctx context.Context, prefix string) ([]string, error)
	DeletePrefix(ctx context.Context, prefix string) (int64, error)
}

type Buckets interface {
	Open(location *tenantarchive.Location, credentials *tenantarchive.Credentials) (Bucket, error)
}

type Dangling struct {
	Constraint string `json:"constraint" db:"constraint_name"`
	Table      string `json:"table"      db:"table_name"`
	Columns    string `json:"columns"    db:"column_names"`
	Rows       int64  `json:"rows"       db:"row_count"`
}

type TableCount struct {
	Table string `json:"table" db:"table_name"`
	Rows  int64  `json:"rows"  db:"row_count"`
}

type Reference struct {
	Table string
	Key   []string
	Rows  map[string][]string
}

type ObjectColumn struct {
	Table  string
	Column string
}

type Snapshot interface {
	Plan() *archiveplan.Plan
	Tenant() dbscope.Tenant
	OrganizationName() string
	CopyOut(ctx context.Context, table *archiveplan.PlannedTable, dst io.Writer) error
	References(ctx context.Context) ([]Reference, error)
	Outside(ctx context.Context) ([]Dangling, error)
	Close(ctx context.Context)
}

type SnapshotRequest struct {
	Tenant   dbscope.Tenant
	Purpose  tenantarchive.Purpose
	Envelope map[string][]string
}

type TrailRequest struct {
	Restoring      bool
	ManifestSHA256 string
	Manifest       *tenantarchive.Manifest
	At             int64
}

type ChainOutcome string

const (
	ChainEmpty    = ChainOutcome("empty")
	ChainKept     = ChainOutcome("kept")
	ChainResealed = ChainOutcome("resealed")
	ChainBroken   = ChainOutcome("broken")
)

type ChainResult struct {
	Outcome  ChainOutcome `json:"outcome"`
	Links    int64        `json:"links"`
	KeyID    string       `json:"keyId,omitempty"`
	BrokenAt int64        `json:"brokenAt,omitempty"`
}

type TrailResult struct {
	AuditSeals ChainResult `json:"auditSeals"`
	AIAudit    ChainResult `json:"aiAudit"`
}

type LoadRequest struct {
	Tenant   dbscope.Tenant
	Manifest *tenantarchive.Manifest
	Envelope map[string][]string
	Beat     func()
}

type Load interface {
	Plan() *archiveplan.Plan
	CopyIn(ctx context.Context, table *archiveplan.PlannedTable, src io.Reader) (int64, error)
	Finish(ctx context.Context) error
	Counts(ctx context.Context) ([]TableCount, error)
	Dangling(ctx context.Context) ([]Dangling, error)
	MarkForeignRecoveryCodes(ctx context.Context) (int64, error)
	Trails(ctx context.Context, req *TrailRequest) (*TrailResult, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context)
}

type Rows interface {
	SchemaVersion() string
	OpenSnapshot(ctx context.Context, req *SnapshotRequest) (Snapshot, error)
	PlanFor(
		ctx context.Context,
		purpose tenantarchive.Purpose,
		envelope map[string][]string,
	) (*archiveplan.Plan, error)
	TenantExists(ctx context.Context, tenant dbscope.Tenant) (bool, error)
	Holding(ctx context.Context, plan *archiveplan.Plan, tenant dbscope.Tenant) ([]string, error)
	ExistingKeys(
		ctx context.Context,
		table *archiveplan.PlannedTable,
		keys [][]string,
	) (int64, error)
	Resolve(
		ctx context.Context,
		table string,
		key []string,
		values [][]string,
	) (map[string]string, error)
	TakenIdentities(ctx context.Context, emails, usernames []string) (bool, error)
	TakenSlugs(ctx context.Context, slugs []string) (bool, error)
	EnsureAuditPartitions(ctx context.Context, from int64) error
	OpenLoad(ctx context.Context, req *LoadRequest) (Load, error)
}
