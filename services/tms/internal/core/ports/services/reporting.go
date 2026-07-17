package services

import (
	"context"
	"io"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/shared/pulid"
)

type ReportCompileRequest struct {
	Definition  *report.Definition
	Tenant      pagination.TenantInfo
	Principal   PrincipalInfo
	Params      map[string]any
	OrgTimezone string
	NowUnix     int64
}

type ReportResultColumn struct {
	ID          string
	Label       string
	Type        reportcatalog.FieldType
	Format      reportcatalog.FormatHint
	Sensitivity permission.FieldSensitivity
}

type ReportComplexity struct {
	ToOneJoins       int
	ToManySubqueries int
	Dimensions       int
	Measures         int
	PivotColumns     int
	Score            int
}

type CompiledReportQuery struct {
	SQL                string
	Args               []any
	Columns            []ReportResultColumn
	Complexity         ReportComplexity
	ReferencedEntities []string
	ReferencedTables   []string
	Limit              int
}

type ReportValidationResult struct {
	ReferencedEntities []string
	Columns            []ReportResultColumn
	Complexity         ReportComplexity
}

type ReportCompiler interface {
	ValidateAndAuthorize(
		ctx context.Context,
		req *ReportCompileRequest,
	) (*ReportValidationResult, error)
	Compile(ctx context.Context, req *ReportCompileRequest) (*CompiledReportQuery, error)
	CompileForPreview(
		ctx context.Context,
		req *ReportCompileRequest,
	) (*CompiledReportQuery, error)
}

type ReportRow []any

// ReportDatasetReader is the seam between execution and rendering: a
// single-pass, already-authorized row stream. Next returns io.EOF after the
// final row; Truncated reports whether the row cap was hit and is only
// meaningful after EOF.
type ReportDatasetReader interface {
	Schema() []ReportResultColumn
	Next(ctx context.Context) (ReportRow, error)
	RowCount() int64
	Truncated() bool
	Close() error
}

type ReportRunMeta struct {
	Title           string
	Description     string
	GeneratedAtUnix int64
	Timezone        string
	RequestedBy     string
	Params          map[string]any
}

type ReportRenderRequest struct {
	Dataset ReportDatasetReader
	Sink    io.Writer
	Meta    ReportRunMeta
}

type ReportRenderStats struct {
	Rows      int64
	Truncated bool
}

type ReportRenderer interface {
	Format() report.Format
	Render(ctx context.Context, req *ReportRenderRequest) (*ReportRenderStats, error)
}

type OpenReportDatasetRequest struct {
	Compiled *CompiledReportQuery
	MaxRows  int64
	Timeout  time.Duration
}

type ReportDatasetExecutor interface {
	Open(ctx context.Context, req *OpenReportDatasetRequest) (ReportDatasetReader, error)
}

type ReportRendererRegistry interface {
	For(format report.Format) (ReportRenderer, error)
}

type ReportCacheEntry struct {
	ArtifactKey       string `json:"artifactKey"`
	RowCount          int64  `json:"rowCount"`
	ByteSize          int64  `json:"byteSize"`
	Truncated         bool   `json:"truncated"`
	ArtifactExpiresAt int64  `json:"artifactExpiresAt"`
}

// ReportResultCache caches finished artifacts keyed by the compiled query, the
// runner's authorization envelope (already baked into the compiled SQL), and a
// per-table data-version vector maintained from CDC events — so a cached
// result is reused only while the underlying tables are unchanged.
type ReportResultCache interface {
	Key(
		ctx context.Context,
		compiled *CompiledReportQuery,
		format report.Format,
		orgID pulid.ID,
	) (string, error)
	Lookup(ctx context.Context, key string) (*ReportCacheEntry, bool, error)
	Store(ctx context.Context, key string, entry *ReportCacheEntry) error
}
