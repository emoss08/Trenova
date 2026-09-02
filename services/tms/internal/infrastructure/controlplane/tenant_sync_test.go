package controlplane

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type tenantSyncClient struct {
	heartbeatClient
	syncCalled bool
	syncErr    error
	syncReq    *services.TenantSyncRequest
}

func (c *tenantSyncClient) SyncTenants(
	_ context.Context,
	req *services.TenantSyncRequest,
) (*services.TenantSyncResult, error) {
	c.syncCalled = true
	c.syncReq = req
	if c.syncErr != nil {
		return nil, c.syncErr
	}
	return &services.TenantSyncResult{Accepted: true}, nil
}

type tenantSyncRepo struct {
	businessUnits []tenant.SyncBusinessUnit
	organizations []tenant.SyncOrganization
	byIDCalled    bool
}

func (r *tenantSyncRepo) ListBusinessUnits(context.Context) ([]tenant.SyncBusinessUnit, error) {
	return r.businessUnits, nil
}

func (r *tenantSyncRepo) ListOrganizations(context.Context) ([]tenant.SyncOrganization, error) {
	return r.organizations, nil
}

func (r *tenantSyncRepo) ListBusinessUnitsByID(
	context.Context,
	[]pulid.ID,
) ([]tenant.SyncBusinessUnit, error) {
	r.byIDCalled = true
	return r.businessUnits, nil
}

func (r *tenantSyncRepo) ListOrganizationsByID(
	context.Context,
	[]pulid.ID,
) ([]tenant.SyncOrganization, error) {
	r.byIDCalled = true
	return r.organizations, nil
}

func tenantSyncConfig(enabled bool, env string, failOpen bool) *config.Config {
	return &config.Config{
		App: config.AppConfig{Name: "trenova", Env: env, Version: "1.0.0"},
		Platform: config.PlatformConfig{
			InstanceID: "inst_01",
			ControlPlane: config.PlatformControlPlaneConfig{
				Enabled:            enabled,
				TenantSyncInterval: time.Hour,
				FailOpenOnError:    failOpen,
			},
		},
	}
}

func newTestTenantSyncer(cfg *config.Config, client Client, repo *tenantSyncRepo) *TenantSyncer {
	return &TenantSyncer{
		cfg:    cfg,
		client: client,
		repo:   repo,
		now:    func() time.Time { return time.Unix(100, 0) },
		logger: zap.NewNop(),
	}
}

func TestTenantSyncer_StartOnlyWhenControlPlaneEnabled(t *testing.T) {
	t.Parallel()

	client := &tenantSyncClient{}
	syncer := newTestTenantSyncer(
		tenantSyncConfig(false, config.EnvTest, false),
		client,
		&tenantSyncRepo{},
	)

	require.NoError(t, syncer.start(t.Context()))
	require.False(t, client.syncCalled)
}

func TestTenantSyncer_StartupFullSyncSendsTenants(t *testing.T) {
	t.Parallel()

	client := &tenantSyncClient{}
	repo := &tenantSyncRepo{
		businessUnits: []tenant.SyncBusinessUnit{{ID: pulid.MustNew("bu_")}},
		organizations: []tenant.SyncOrganization{{ID: pulid.MustNew("org_")}},
	}
	syncer := newTestTenantSyncer(
		tenantSyncConfig(true, config.EnvDevelopment, false),
		client,
		repo,
	)

	require.NoError(t, syncer.start(t.Context()))
	defer func() { require.NoError(t, syncer.stop(t.Context())) }()

	require.True(t, client.syncCalled)
	require.Equal(t, services.TenantSyncModeFull, client.syncReq.Mode)
	require.Len(t, client.syncReq.BusinessUnits, 1)
	require.Len(t, client.syncReq.Organizations, 1)
	require.Equal(t, int64(100), client.syncReq.SentAt)
}

func TestTenantSyncer_FailsClosedOutsideDevelopment(t *testing.T) {
	t.Parallel()

	client := &tenantSyncClient{syncErr: errors.New("down")}
	syncer := newTestTenantSyncer(
		tenantSyncConfig(true, config.EnvProduction, true),
		client,
		&tenantSyncRepo{},
	)

	require.ErrorContains(t, syncer.start(t.Context()), "sync tenants with control plane")
}

func TestTenantSyncer_FailsOpenInDevelopment(t *testing.T) {
	t.Parallel()

	client := &tenantSyncClient{syncErr: errors.New("down")}
	syncer := newTestTenantSyncer(
		tenantSyncConfig(true, config.EnvDevelopment, true),
		client,
		&tenantSyncRepo{},
	)

	require.NoError(t, syncer.start(t.Context()))
	require.NoError(t, syncer.stop(t.Context()))
	require.True(t, client.syncCalled)
}

func TestTenantSyncer_SyncDeltaSkipsEmptyDelta(t *testing.T) {
	t.Parallel()

	client := &tenantSyncClient{}
	syncer := newTestTenantSyncer(
		tenantSyncConfig(true, config.EnvDevelopment, false),
		client,
		&tenantSyncRepo{},
	)

	require.NoError(t, syncer.SyncDelta(t.Context(), services.TenantSyncDelta{
		BusinessUnitIDs: []pulid.ID{pulid.MustNew("bu_")},
	}))
	require.False(t, client.syncCalled)
}

func TestTenantSyncer_SyncDeltaSendsChangedTenants(t *testing.T) {
	t.Parallel()

	client := &tenantSyncClient{}
	repo := &tenantSyncRepo{
		organizations: []tenant.SyncOrganization{{ID: pulid.MustNew("org_")}},
	}
	syncer := newTestTenantSyncer(
		tenantSyncConfig(true, config.EnvDevelopment, false),
		client,
		repo,
	)

	require.NoError(t, syncer.SyncDelta(t.Context(), services.TenantSyncDelta{
		OrganizationIDs: []pulid.ID{repo.organizations[0].ID},
	}))
	require.True(t, client.syncCalled)
	require.Equal(t, services.TenantSyncModeDelta, client.syncReq.Mode)
	require.True(t, repo.byIDCalled)
}
