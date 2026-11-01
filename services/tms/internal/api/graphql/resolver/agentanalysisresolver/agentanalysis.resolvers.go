package agentanalysisresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/ports/analysisports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const maxRunsPerPage = 200

func (r *QueryResolver) AgentAnalysisRuns(ctx context.Context, before *int, beforeID *string, limit *int) (*analysisports.RunListResponse, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceAgentRun, permission.OpRead)
	if err != nil {
		return nil, err
	}

	req, err := listRunsRequest(before, beforeID, limit)
	if err != nil {
		return nil, err
	}
	req.TenantInfo = resolvercore.TenantInfo(authCtx)

	return r.AnalysisRunService.ListRuns(ctx, req)
}

func (r *QueryResolver) AgentAnalysisRun(ctx context.Context, id string) (*analysis.Run, error) {
	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil, errortypes.NewAuthenticationError("Authentication required")
	}

	oversees := r.HasPermission(ctx, authCtx, permission.ResourceAgentRun, permission.OpRead)
	if !oversees && !r.HasPermission(ctx, authCtx, permission.ResourceCodeMode, permission.OpRead) {
		return nil, errortypes.NewAuthorizationError(
			"You don't have permission to perform this action: {0} {1}",
			permission.ResourceAgentRun, permission.OpRead,
		)
	}

	runID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return nil, err
	}

	return r.AnalysisRunService.Run(ctx, &analysisports.RunRequest{
		TenantInfo: resolvercore.TenantInfo(authCtx),
		RunID:      runID,
		Oversees:   oversees,
	})
}

func (r *AgentAnalysisBlockResolver) Kind(ctx context.Context, obj *analysis.Block) (string, error) {
	return string(obj.Kind), nil
}

func (r *AgentAnalysisChartAxisResolver) Kind(ctx context.Context, obj *analysis.ChartAxis) (string, error) {
	return string(obj.Kind), nil
}

func (r *AgentAnalysisChartResolver) Kind(ctx context.Context, obj *analysis.Chart) (string, error) {
	return string(obj.Kind), nil
}

func (r *AgentAnalysisChartResolver) ValueKind(ctx context.Context, obj *analysis.Chart) (string, error) {
	return string(obj.ValueKind), nil
}

func (r *AgentAnalysisDatasetResolver) Source(ctx context.Context, obj *analysis.Dataset) (string, error) {
	return string(obj.Source), nil
}

func (r *AgentAnalysisFigureResolver) Kind(ctx context.Context, obj *analysis.Figure) (string, error) {
	return string(obj.Kind), nil
}

func (r *AgentAnalysisRunResolver) Status(ctx context.Context, obj *analysis.Run) (string, error) {
	return string(obj.Status), nil
}

func (r *AgentAnalysisTableColumnResolver) Kind(ctx context.Context, obj *analysis.TableColumn) (string, error) {
	return string(obj.Kind), nil
}
