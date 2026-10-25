package reporting

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/keel/analytics/table"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/pkg/reportvalue"
)

const previewSupersedeKey = "preview"

type PreviewRequest struct {
	Request

	Definition *ir.Definition
	Params     map[string]any
	Supersede  bool
	Cursor     string
	PageSize   int

	export bool
}

type PreviewResult struct {
	Columns   []table.Column
	Rows      [][]any
	Levels    []table.Level
	Totals    []any
	Truncated bool
	Cursor    string
	Timezone  string
	Rollup    string
	Cached    bool
}

func (s *Service) Preview(ctx context.Context, req *PreviewRequest) (*PreviewResult, error) {
	started := time.Now()
	result, err := s.runPreview(ctx, req)

	if s.metrics != nil {
		outcome := "success"
		switch {
		case errors.Is(err, context.Canceled):
			outcome = "superseded"
		case err != nil:
			outcome = "error"
		}
		s.metrics.RecordPreview(outcome, time.Since(started))
	}

	return result, err
}

func (s *Service) runPreview(ctx context.Context, req *PreviewRequest) (*PreviewResult, error) {
	mode := reportaccess.ModeRead
	if req.export {
		mode = reportaccess.ModeExport
	}
	key := ""
	if req.Supersede {
		key = previewSupersedeKey
	}
	prepared, err := s.Prepare(ctx, &PrepareParams{
		Tenant:     req.TenantInfo,
		Definition: req.Definition,
		Params:     req.Params,
		Mode:       mode,
		Key:        key,
	})
	if err != nil {
		return nil, err
	}

	var result *analytics.Result
	if req.PageSize > 0 {
		size := min(req.PageSize, s.cfg.GetPreviewRowLimit())
		result, err = s.engine.Page(prepared.Context, &prepared.Request, req.Cursor, size)
	} else {
		result, err = s.engine.Preview(prepared.Context, &prepared.Request)
	}
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordCompileError("preview")
		}
		return nil, EngineError(prepared.Context, prepared.Access, err)
	}

	out, err := collectPreview(prepared.Context, result)
	if err != nil {
		return nil, err
	}
	out.Timezone = prepared.Location.String()
	return out, nil
}

func collectPreview(ctx context.Context, result *analytics.Result) (*PreviewResult, error) {
	columns := result.Columns()
	out := &PreviewResult{
		Columns: columns,
		Rows:    make([][]any, 0, result.Len()),
		Rollup:  result.Rollup(),
		Cached:  result.Cached(),
	}
	for {
		batch, err := result.Next(ctx)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		for i := range batch.Len {
			row := reportvalue.Row(columns, batch, i)
			level := batch.Level(i)
			if level == table.Total {
				out.Totals = row
				continue
			}
			out.Rows = append(out.Rows, row)
			out.Levels = append(out.Levels, level)
		}
	}
	out.Truncated = result.Truncated()
	out.Cursor = result.Cursor()
	return out, nil
}
