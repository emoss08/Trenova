package retrievaljobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Pipeline retrievalports.RetrievalIndexPipeline
	Tenants  tenantports.TenantSyncRepository
	Logger   *zap.Logger
}

type Activities struct {
	pipeline retrievalports.RetrievalIndexPipeline
	tenants  temporaljobs.TenantPager
	l        *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		pipeline: p.Pipeline,
		tenants:  p.Tenants,
		l:        p.Logger.Named("job.retrieval-index"),
	}
}

func (a *Activities) PlanRetrievalIndexActivity(
	ctx context.Context,
	input *PlanInput,
) (*retrievalports.RetrievalIndexPlan, error) {
	plan, err := a.pipeline.Plan(ctx, input.TenantInfo())
	if err != nil {
		return nil, fmt.Errorf("plan retrieval indexing: %w", err)
	}

	return &plan, nil
}

func (a *Activities) IndexRetrievalBatchActivity(
	ctx context.Context,
	input *retrievalports.RetrievalIndexBatchRequest,
) (*retrievalports.RetrievalIndexBatchResult, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	result, err := a.pipeline.IndexBatch(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("index a retrieval batch: %w", err)
	}

	a.l.Debug("retrieval batch indexed",
		zap.String("organizationId", input.TenantInfo.OrgID.String()),
		zap.String("modelKey", input.ModelKey),
		zap.Int("claimed", result.Claimed),
		zap.Int("indexed", result.Indexed),
		zap.Int("failed", result.Failed),
		zap.Int("chunksEmbedded", result.ChunksEmbedded),
	)

	return &result, nil
}

func (a *Activities) CompleteRetrievalModelChangeActivity(
	ctx context.Context,
	input *ModelChangeInput,
) (*retrievalports.RetrievalModelChangeResult, error) {
	result, err := a.pipeline.CompleteModelChange(ctx, input.TenantInfo(), input.PendingModelKey)
	if err != nil {
		return nil, fmt.Errorf("complete the embedding model change: %w", err)
	}

	return &result, nil
}

func (a *Activities) PurgeRetrievalModelActivity(
	ctx context.Context,
	input *PurgeInput,
) (*retrievalports.RetrievalPurgeResult, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	result, err := a.pipeline.PurgeRetiredModel(ctx, input.TenantInfo(), input.ModelKey)
	if err != nil {
		return nil, fmt.Errorf("purge a retired embedding model: %w", err)
	}

	return &result, nil
}

func (a *Activities) SweepRetrievalSourcesActivity(
	ctx context.Context,
	input *OrganizationSweepInput,
) (*retrievalports.RetrievalSweepResult, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	tenant := input.TenantInfo()
	result, err := a.pipeline.Sweep(ctx, tenant)
	if err != nil {
		return nil, fmt.Errorf("sweep retrieval sources: %w", err)
	}
	if input.Wake && result.HasWork() {
		a.pipeline.Wake(ctx, tenant)
	}

	return &result, nil
}

func (a *Activities) ListRetrievalOrganizationsActivity(
	ctx context.Context,
	input *ListOrganizationsInput,
) (*temporaljobs.TenantPage, error) {
	page, err := temporaljobs.OrganizationPage(ctx, a.tenants, input.Need, input.After, input.Limit)
	if err != nil {
		return nil, fmt.Errorf("list organizations to sweep for retrieval: %w", err)
	}

	return page, nil
}

func (a *Activities) ReindexRetrievalPageActivity(
	ctx context.Context,
	input *retrievalports.RetrievalReindexPageRequest,
) (*retrievalports.RetrievalReindexPage, error) {
	page, err := a.pipeline.ReindexPage(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("re-index a page of %s sources: %w", input.SourceType, err)
	}

	return &page, nil
}
