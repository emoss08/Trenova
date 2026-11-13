package aifeedbackjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentreflectionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aifeedbackports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Maintenance aifeedbackports.AIFeedbackMaintenance
	Memories    agentmemoryports.AgentMemoryService
	Reflections agentreflectionports.AgentReflectionService
	LookBacks   agentreflectionports.AgentReflectionScheduler
	Tenants     tenantports.TenantSyncRepository
	Logger      *zap.Logger
}

type Activities struct {
	maintenance aifeedbackports.AIFeedbackMaintenance
	memories    agentmemoryports.AgentMemoryService
	reflections agentreflectionports.AgentReflectionService
	lookBacks   agentreflectionports.AgentReflectionScheduler
	tenants     tenantports.TenantSyncRepository
	l           *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		maintenance: p.Maintenance,
		memories:    p.Memories,
		reflections: p.Reflections,
		lookBacks:   p.LookBacks,
		tenants:     p.Tenants,
		l:           p.Logger.Named("job.aifeedback-maintenance"),
	}
}

func (a *Activities) ListAIFeedbackOrganizationsActivity(
	ctx context.Context,
	input *ListAIFeedbackOrganizationsInput,
) (*temporaljobs.TenantPage, error) {
	return temporaljobs.OrganizationPage(ctx, a.tenants, input.After, input.Limit)
}

func (a *Activities) MaintainOrganizationAIFeedbackActivity(
	ctx context.Context,
	input *OrganizationAIFeedbackInput,
) (*OrganizationAIFeedbackResult, error) {
	tenant := input.TenantInfo()

	purged, err := a.maintenance.PurgeExpired(ctx, aifeedbackports.PurgeExpiredAIFeedbackRequest{
		TenantInfo: tenant,
		Now:        input.Now,
	})
	if err != nil {
		return nil, fmt.Errorf("purge expired ai feedback: %w", err)
	}

	suggested, err := a.maintenance.SuggestMemories(
		ctx,
		aifeedbackports.SuggestAgentMemoriesRequest{
			TenantInfo: tenant,
			Now:        input.Now,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("suggest agent memories: %w", err)
	}

	lapsed, err := a.memories.LapseOffers(ctx, tenant, input.Now)
	if err != nil {
		return nil, fmt.Errorf("lapse agent memory offers: %w", err)
	}

	corrections, err := a.reflections.PlanCorrections(ctx, tenant, input.Now)
	if err != nil {
		return nil, fmt.Errorf("find what people keep changing of agents' work: %w", err)
	}
	for _, correction := range corrections {
		a.lookBacks.AfterCorrections(ctx, correction)
	}

	a.l.Debug("ai feedback maintained",
		zap.String("organization", tenant.OrgID.String()),
		zap.Int64("purged", purged),
		zap.Int("suggested", suggested.Suggested),
		zap.Int64("lapsed", lapsed),
		zap.Int("corrections", len(corrections)),
	)

	return &OrganizationAIFeedbackResult{
		Purged:      purged,
		Lapsed:      lapsed,
		Corrections: len(corrections),
		Suggested:   suggested.Suggested,
		Covered:     suggested.Covered,
		BelowFloor:  suggested.BelowFloor,
	}, nil
}
