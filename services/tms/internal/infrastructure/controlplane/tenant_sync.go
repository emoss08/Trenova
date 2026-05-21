package controlplane

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type TenantSyncerParams struct {
	fx.In

	Config *config.Config
	Client Client
	Repo   repositories.TenantSyncRepository
	Logger *zap.Logger
}

type TenantSyncer struct {
	cfg    *config.Config
	client Client
	repo   repositories.TenantSyncRepository
	now    func() time.Time
	logger *zap.Logger
}

func NewTenantSyncer(p TenantSyncerParams) *TenantSyncer {
	syncer := &TenantSyncer{
		cfg:    p.Config,
		client: p.Client,
		repo:   p.Repo,
		now:    time.Now,
		logger: p.Logger.Named("control-plane-tenant-sync"),
	}

	return syncer
}

func (s *TenantSyncer) SyncFull(ctx context.Context) error {
	if !s.cfg.Platform.ControlPlane.Enabled {
		return nil
	}

	businessUnits, err := s.repo.ListBusinessUnits(ctx)
	if err != nil {
		return fmt.Errorf("list business units for tenant sync: %w", err)
	}
	organizations, err := s.repo.ListOrganizations(ctx)
	if err != nil {
		return fmt.Errorf("list organizations for tenant sync: %w", err)
	}

	return s.sync(ctx, &services.TenantSyncRequest{
		Mode:          services.TenantSyncModeFull,
		BusinessUnits: businessUnits,
		Organizations: organizations,
		SentAt:        s.now().Unix(),
	})
}

func (s *TenantSyncer) SyncDelta(ctx context.Context, delta services.TenantSyncDelta) error {
	if !s.cfg.Platform.ControlPlane.Enabled {
		return nil
	}

	businessUnits, err := s.repo.ListBusinessUnitsByID(ctx, delta.BusinessUnitIDs)
	if err != nil {
		return fmt.Errorf("list business units for tenant delta sync: %w", err)
	}
	organizations, err := s.repo.ListOrganizationsByID(ctx, delta.OrganizationIDs)
	if err != nil {
		return fmt.Errorf("list organizations for tenant delta sync: %w", err)
	}

	if len(businessUnits) == 0 && len(organizations) == 0 {
		return nil
	}

	return s.sync(ctx, &services.TenantSyncRequest{
		Mode:          services.TenantSyncModeDelta,
		BusinessUnits: businessUnits,
		Organizations: organizations,
		SentAt:        s.now().Unix(),
	})
}

func (s *TenantSyncer) sync(ctx context.Context, req *services.TenantSyncRequest) error {
	_, err := s.client.SyncTenants(ctx, req)
	if err != nil {
		return fmt.Errorf("sync tenants with control plane: %w", err)
	}
	return nil
}
