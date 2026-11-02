package shipmentboardservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentboardports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/pagination"
	"golang.org/x/sync/errgroup"
)

func (s *Service) Capabilities(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*shipmentboardports.ShipmentBoardCapabilities, error) {
	out := new(shipmentboardports.ShipmentBoardCapabilities)
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		available, err := s.novaReady(gctx, tenantInfo)
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

// novaReady reports whether Nova, which writes the board's briefings and
// insights, runs for the tenant.
func (s *Service) novaReady(ctx context.Context, tenantInfo pagination.TenantInfo) (bool, error) {
	return s.nova.Available(ctx, tenantInfo)
}
