package analysisrunservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/ports/analysisports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	runListLimit = 200
	runNotFound  = "That analysis was not found"
)

var (
	_ analysisports.RunRecorder = (*Service)(nil)
	_ analysisports.Runs        = (*Service)(nil)
)

type Params struct {
	fx.In

	Repo   analysisports.RunRepository
	Logger *zap.Logger
}

type Service struct {
	repo analysisports.RunRepository
	l    *zap.Logger
}

func New(p Params) *Service {
	return &Service{repo: p.Repo, l: p.Logger.Named("service.agent-analysis")}
}

func (s *Service) Record(ctx context.Context, run *analysis.Run) (*analysis.Run, error) {
	run.Bound()

	return s.repo.Create(ctx, run)
}

func (s *Service) ListRuns(
	ctx context.Context,
	req *analysisports.ListRunsRequest,
) (*analysisports.RunListResponse, error) {
	if req.Limit <= 0 || req.Limit > runListLimit {
		req.Limit = runListLimit
	}

	runs, err := s.repo.List(ctx, req)
	if err != nil {
		return nil, errortypes.NewBusinessError("failed to list analyses").WithInternal(err)
	}

	response := &analysisports.RunListResponse{Items: runs}
	if len(runs) == req.Limit {
		last := runs[len(runs)-1]
		response.NextBefore = last.StartedAt
		response.NextBeforeID = last.ID
	}

	return response, nil
}

// Run is one analysis whole. Someone who does not oversee agents reads only
// the analyses run for them, and is told any other does not exist.
func (s *Service) Run(
	ctx context.Context,
	req *analysisports.RunRequest,
) (*analysis.Run, error) {
	run, err := s.repo.Get(ctx, req.TenantInfo, req.RunID)
	if errortypes.IsNotFoundError(err) {
		return nil, errortypes.NewNotFoundError(runNotFound)
	}
	if err != nil {
		return nil, err
	}
	if !req.Oversees && run.RequestedByID != req.TenantInfo.UserID {
		return nil, errortypes.NewNotFoundError(runNotFound)
	}

	return run, nil
}
