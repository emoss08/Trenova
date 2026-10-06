package shipmentboardservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"golang.org/x/sync/errgroup"
)

var boardAITasks = [...]aiprovider.Task{
	aiprovider.TaskDailyBriefing,
	aiprovider.TaskOperationalInsights,
}

func (s *Service) Capabilities(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*services.ShipmentBoardCapabilities, error) {
	out := new(services.ShipmentBoardCapabilities)
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		available, err := s.anyAITask(gctx, tenantInfo, boardAITasks[:])
		out.AI = available
		return err
	})
	g.Go(func() error {
		caps, err := s.organizations.GetCapabilities(
			gctx,
			repositories.GetOrganizationCapabilitiesRequest{TenantInfo: tenantInfo},
		)
		if err != nil {
			return err
		}
		out.OperationType = tenant.OperationTypeOf(
			caps.BrokerageEnabled,
			caps.AssetOperationsEnabled,
		)
		return nil
	})
	g.Go(func() error {
		records, err := s.integrations.ListByTenant(gctx, tenantInfo)
		if err != nil {
			return err
		}
		out.HOS = integration.HasEnabledTelematics(records)
		return nil
	})
	g.Go(func() error {
		cfg, err := s.maps.GetClientRuntimeConfig(gctx, tenantInfo, integration.TypeGoogleMaps)
		if err != nil {
			return err
		}
		out.Maps = cfg != nil && cfg.Ready
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *Service) AIAvailable(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	task aiprovider.Task,
) (bool, error) {
	return s.anyAITask(ctx, tenantInfo, []aiprovider.Task{task})
}

func (s *Service) anyAITask(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	tasks []aiprovider.Task,
) (bool, error) {
	for _, task := range tasks {
		providers, err := s.aiProviders.ListForTask(ctx, repositories.ListAIProvidersForTaskRequest{
			Task:       task,
			TenantInfo: tenantInfo,
		})
		if err != nil {
			return false, err
		}
		if len(providers) > 0 {
			return true, nil
		}
	}
	return false, nil
}
