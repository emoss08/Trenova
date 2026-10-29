package shipmentboardservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentboardports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/pagination"
	"golang.org/x/sync/errgroup"
)

var boardAITasks = [...]aitask.Task{
	aitask.TaskDailyBriefing,
	aitask.TaskOperationalInsights,
}

func (s *Service) Capabilities(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*shipmentboardports.ShipmentBoardCapabilities, error) {
	out := new(shipmentboardports.ShipmentBoardCapabilities)
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		available, err := s.anyAITask(gctx, tenantInfo, boardAITasks[:])
		out.AI = available
		return err
	})
	g.Go(func() error {
		caps, err := s.organizations.GetCapabilities(
			gctx,
			tenantports.GetOrganizationCapabilitiesRequest{TenantInfo: tenantInfo},
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
	task aitask.Task,
) (bool, error) {
	return s.anyAITask(ctx, tenantInfo, []aitask.Task{task})
}

func (s *Service) anyAITask(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	tasks []aitask.Task,
) (bool, error) {
	for _, task := range tasks {
		providers, err := s.aiProviders.ListForTask(
			ctx,
			aiproviderports.ListAIProvidersForTaskRequest{
				Task:       task,
				TenantInfo: tenantInfo,
			},
		)
		if err != nil {
			return false, err
		}
		if len(providers) > 0 {
			return true, nil
		}
	}
	return false, nil
}
