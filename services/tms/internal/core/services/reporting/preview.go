package reporting

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

type previewSlot struct {
	generation uint64
	cancel     context.CancelFunc
}

// previewLimiter enforces single-flight previews per user: starting a new
// preview cancels the user's previous in-flight one.
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
}

type PreviewResult struct {
	Columns   []services.ReportResultColumn
	Rows      []services.ReportRow
	Truncated bool
}

func (s *Service) Preview(ctx context.Context, req *PreviewRequest) (*PreviewResult, error) {
	started := time.Now()

	previewCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	userKey := req.TenantInfo.UserID.String()
	generation := s.previews.acquire(userKey, cancel)
	defer s.previews.release(userKey, generation)

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
	compiled, err := s.compiler.CompileForPreview(ctx, &services.ReportCompileRequest{
		Definition:  req.Definition,
		Tenant:      req.TenantInfo,
		Params:      req.Params,
		OrgTimezone: s.orgTimezone(ctx, req.TenantInfo),
		NowUnix:     timeutils.NowUnix(),
	})
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordCompileError("preview")
		}
		return nil, err
	}

	rowLimit := int64(s.cfg.GetPreviewRowLimit())
	reader, err := s.executor.Open(ctx, &services.OpenReportDatasetRequest{
		Compiled: compiled,
		MaxRows:  rowLimit,
		Timeout:  s.cfg.GetPreviewStatementTimeout(),
	})
	if err != nil {
		s.l.Error("failed to open preview dataset", zap.Error(err))
		return nil, err
	}
	defer reader.Close()

	rows := make([]services.ReportRow, 0, rowLimit)
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

	return &PreviewResult{
		Columns:   reader.Schema(),
		Rows:      rows,
		Truncated: reader.Truncated(),
	}, nil
}
