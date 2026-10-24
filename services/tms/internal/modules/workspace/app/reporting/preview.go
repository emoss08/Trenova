package reporting

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

type previewSlot struct {
	generation uint64
	cancel     context.CancelFunc
}

// previewLimiter enforces single-flight previews per user: starting a new
// preview cancels the user's previous in-flight one. Only callers that opt in
// participate — a surface that loads several previews at once (a dashboard)
// would otherwise cancel all but its last one.
type previewLimiter struct {
	mu         sync.Mutex
	generation uint64
	inflight   map[string]previewSlot
}

func newPreviewLimiter() *previewLimiter {
	return &previewLimiter{inflight: make(map[string]previewSlot)}
}

func (p *previewLimiter) acquire(userID string, cancel context.CancelFunc) uint64 {
	p.mu.Lock()
	defer p.mu.Unlock()

	if previous, ok := p.inflight[userID]; ok {
		previous.cancel()
	}
	p.generation++
	p.inflight[userID] = previewSlot{generation: p.generation, cancel: cancel}
	return p.generation
}

func (p *previewLimiter) release(userID string, generation uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if current, ok := p.inflight[userID]; ok && current.generation == generation {
		delete(p.inflight, userID)
	}
}

type PreviewRequest struct {
	Request

	Definition *report.Definition
	Params     map[string]any
	// Supersede marks this preview as replacing the caller's previous one, so
	// the builder's in-flight query is dropped on the next keystroke. Surfaces
	// that issue independent previews in parallel must leave it unset.
	Supersede bool

	// export marks a preview whose rows leave the product as a file, which
	// needs the export operation like a report run.
	export bool
}

type PreviewResult struct {
	Columns   []reportports.ReportResultColumn
	Rows      []reportports.ReportRow
	Totals    reportports.ReportRow
	Truncated bool
	Timezone  string
}

func (s *Service) Preview(ctx context.Context, req *PreviewRequest) (*PreviewResult, error) {
	started := time.Now()

	previewCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	if req.Supersede {
		userKey := req.TenantInfo.UserID.String()
		generation := s.previews.acquire(userKey, cancel)
		defer s.previews.release(userKey, generation)
	}

	result, err := s.runPreview(previewCtx, req)

	if s.metrics != nil {
		outcome := "success"
		switch {
		case errors.Is(previewCtx.Err(), context.Canceled):
			outcome = "superseded"
		case err != nil:
			outcome = "error"
		}
		s.metrics.RecordPreview(outcome, time.Since(started))
	}

	return result, err
}

func (s *Service) runPreview(
	ctx context.Context,
	req *PreviewRequest,
) (*PreviewResult, error) {
	compile := s.compiler.CompileForPreview
	if req.export {
		compile = s.compiler.Compile
	}
	timezone := s.orgTimezone(ctx, req.TenantInfo)
	rowLimit := int64(s.cfg.GetPreviewRowLimit())
	compiled, err := compile(ctx, &reportports.ReportCompileRequest{
		Definition:  req.Definition,
		Tenant:      req.TenantInfo,
		Params:      req.Params,
		OrgTimezone: timezone,
		NowUnix:     timeutils.NowUnix(),
		MaxRows:     rowLimit,
	})
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordCompileError("preview")
		}
		return nil, err
	}

	reader, err := s.executor.Open(ctx, &reportports.OpenReportDatasetRequest{
		Compiled: compiled,
		MaxRows:  rowLimit,
		Timeout:  s.cfg.GetPreviewStatementTimeout(),
	})
	if err != nil {
		s.l.Error("failed to open preview dataset", zap.Error(err))
		return nil, err
	}
	defer reader.Close()

	rows := make([]reportports.ReportRow, 0, rowLimit)
	for {
		row, nextErr := reader.Next(ctx)
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return nil, nextErr
		}
		rows = append(rows, row)
	}

	totals, err := reader.Totals(ctx)
	if err != nil {
		return nil, err
	}

	return &PreviewResult{
		Columns:   reader.Schema(),
		Rows:      rows,
		Totals:    totals,
		Truncated: reader.Truncated(),
		Timezone:  timezone,
	}, nil
}
