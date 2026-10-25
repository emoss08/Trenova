package reportports

import (
	"context"
	"io"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/keel/analytics/table"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
)

type ReportEngine interface {
	Catalog() *analytics.Catalog
	Preview(ctx context.Context, req *analytics.Request) (*analytics.Result, error)
	Page(
		ctx context.Context,
		req *analytics.Request,
		cursor string,
		size int,
	) (*analytics.Result, error)
	Count(ctx context.Context, req *analytics.Request) (int64, error)
	Drill(
		ctx context.Context,
		req *analytics.Request,
		row []ir.DrillValue,
		column string,
		opts *analytics.DrillOptions,
	) (*analytics.Result, error)
	Compile(ctx context.Context, req *analytics.Request) (*analytics.Compiled, error)
	Run(ctx context.Context, req *analytics.Request) (*analytics.Stream, error)
}

type ReportParam struct {
	Name  string
	Value any
}

type ReportRunMeta struct {
	Title           string
	Description     string
	GeneratedAtUnix int64
	Timezone        string
	Locale          string
	RequestedBy     string
	Params          []ReportParam
}

type ReportRenderRequest struct {
	Dataset table.Reader
	Sink    io.Writer
	Meta    ReportRunMeta
}

type ReportRenderStats struct {
	Rows      int64
	Bytes     int64
	Truncated bool
}

type ReportRenderer interface {
	Format() report.Format
	Render(ctx context.Context, req *ReportRenderRequest) (*ReportRenderStats, error)
}

type ReportRendererRegistry interface {
	For(format report.Format) (ReportRenderer, error)
}

const MaxDigestRows = 20

type ReportDigest struct {
	Envelope  []byte `json:"envelope"`
	Truncated bool   `json:"truncated,omitempty"`
}

func (d *ReportDigest) IsEmpty() bool {
	return d == nil || len(d.Envelope) == 0
}
