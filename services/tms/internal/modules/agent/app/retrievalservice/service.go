package retrievalservice

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxStaleIDsPerCall = 1000

var (
	_ retrievalports.RetrievalIndexer       = (*Service)(nil)
	_ retrievalports.RetrievalIndexPipeline = (*Service)(nil)
)

type Params struct {
	fx.In

	Logger     *zap.Logger
	Repo       retrievalports.AIRetrievalRepository
	Sources    retrievalports.RetrievalSourceRepository
	Usage      aiusageports.AIUsageRepository
	Registry   *permissioncatalog.Registry
	Embeddings aiusageports.EmbeddingService     `optional:"true"`
	Signals    kernelports.WorkflowSignalStarter `optional:"true"`
	Workflows  kernelports.WorkflowStarter       `optional:"true"`
}

type Service struct {
	l          *zap.Logger
	repo       retrievalports.AIRetrievalRepository
	sources    retrievalports.RetrievalSourceRepository
	usage      aiusageports.AIUsageRepository
	registry   *permissioncatalog.Registry
	embeddings aiusageports.EmbeddingService
	signals    kernelports.WorkflowSignalStarter
	workflows  kernelports.WorkflowStarter
	now        func() int64
}

func New(p Params) *Service { //nolint:gocritic // fx param structs are passed by value
	registry := p.Registry
	if registry == nil {
		registry = permissioncatalog.NewRegistry()
	}

	return &Service{
		l:          p.Logger.Named("service.retrieval-indexer"),
		repo:       p.Repo,
		sources:    p.Sources,
		usage:      p.Usage,
		registry:   registry,
		embeddings: p.Embeddings,
		signals:    p.Signals,
		workflows:  p.Workflows,
		now:        timeutils.NowUnix,
	}
}

func AsIndexer(s *Service) retrievalports.RetrievalIndexer { return s }

func AsPipeline(s *Service) retrievalports.RetrievalIndexPipeline { return s }

func validateSourceType(sourceType airetrieval.SourceType) error {
	if sourceType.IsValid() {
		return nil
	}

	return errortypes.NewValidationError(
		"sourceType",
		errortypes.ErrInvalid,
		"Source type must be Memory, Document or InboundMessage",
	)
}

func (s *Service) MarkStale(
	ctx context.Context,
	tenant pagination.TenantInfo,
	sourceType airetrieval.SourceType,
	ids ...pulid.ID,
) error {
	if err := validateSourceType(sourceType); err != nil {
		return err
	}
	ids = slices.DeleteFunc(slices.Clone(ids), pulid.ID.IsNil)
	if len(ids) == 0 {
		return nil
	}

	settings, err := s.repo.GetSettings(ctx, tenant)
	if err != nil {
		return fmt.Errorf("read retrieval settings: %w", err)
	}
	if !settings.SourceEnabled(sourceType) || !settings.HasActiveModel() {
		return nil
	}

	marked, err := s.markStale(ctx, tenant, sourceType, settings.IndexedModelKeys(), ids)
	if err != nil {
		return err
	}
	if marked > 0 && !settings.Paused {
		s.wake(ctx, tenant, IndexSignal{SourceType: sourceType, Count: marked})
	}

	return nil
}

func (s *Service) markStale(
	ctx context.Context,
	tenant pagination.TenantInfo,
	sourceType airetrieval.SourceType,
	modelKeys []string,
	ids []pulid.ID,
) (int, error) {
	marked := 0
	for batch := range slices.Chunk(ids, maxStaleIDsPerCall) {
		count, err := s.repo.MarkStale(ctx, &retrievalports.MarkAIRetrievalStaleRequest{
			TenantInfo: tenant,
			SourceType: sourceType,
			SourceIDs:  batch,
			ModelKeys:  modelKeys,
			Now:        s.now(),
		})
		if err != nil {
			return marked, fmt.Errorf("mark %s sources stale: %w", sourceType, err)
		}
		marked += count
	}

	return marked, nil
}

func (s *Service) wake(ctx context.Context, tenant pagination.TenantInfo, signal IndexSignal) {
	if s.signals == nil {
		return
	}

	if _, err := s.signals.SignalWithStartWorkflow(
		ctx,
		IndexWorkflowID(tenant.OrgID),
		IndexSignalName,
		signal,
		indexStartOptions(tenant),
		IndexOrganizationWorkflowName,
		IndexOrganizationInput{OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID},
	); err != nil {
		s.l.Warn("retrieval indexer could not be woken; the hourly sweep will catch up",
			zap.String("organizationId", tenant.OrgID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) Wake(ctx context.Context, tenant pagination.TenantInfo) {
	s.wake(ctx, tenant, IndexSignal{})
}

func (s *Service) DeleteSource(
	ctx context.Context,
	tenant pagination.TenantInfo,
	sourceType airetrieval.SourceType,
	id pulid.ID,
) error {
	if err := validateSourceType(sourceType); err != nil {
		return err
	}
	if id.IsNil() {
		return nil
	}

	if _, err := s.repo.DeleteSource(ctx, &retrievalports.AIRetrievalSourceRef{
		TenantInfo: tenant,
		SourceType: sourceType,
		SourceID:   id,
	}); err != nil {
		return fmt.Errorf("delete %s %s from the retrieval index: %w", sourceType, id, err)
	}

	return nil
}

func (s *Service) Reindex(
	ctx context.Context,
	tenant pagination.TenantInfo,
	sourceType airetrieval.SourceType,
) error {
	if err := validateSourceType(sourceType); err != nil {
		return err
	}
	if s.workflows == nil {
		return errortypes.NewBusinessError(
			"Background work is not available, so the source cannot be re-indexed.",
		).WithInternal(kernelports.ErrWorkflowStarterDisabled)
	}

	if _, err := s.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                       ReindexWorkflowID(tenant.OrgID, sourceType),
		TaskQueue:                temporaltype.TaskQueueSystem.String(),
		Priority:                 WorkflowPriorityFor(tenant.OrgID),
		WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		StaticSummary:            "Re-index " + sourceType.String() + " for retrieval",
	}, ReindexSourceWorkflowName, ReindexSourceInput{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		SourceType:     sourceType,
	}); err != nil {
		return fmt.Errorf("start re-indexing %s: %w", sourceType, err)
	}

	return nil
}

func isNoProvider(err error) bool {
	return errors.Is(err, llmports.ErrNoProviderConfigured) || errortypes.IsBusinessError(err)
}
