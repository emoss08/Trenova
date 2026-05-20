package controlplane

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type heartbeatClient struct {
	called bool
	err    error
	req    *services.InstanceHeartbeatRequest
}

func (c *heartbeatClient) CheckFeature(
	context.Context,
	*services.FeatureCheckRequest,
) (*services.FeatureCheckResult, error) {
	return nil, nil
}

func (c *heartbeatClient) AuthorizeAccess(
	context.Context,
	*services.AccessAuthorizeRequest,
) (*services.AccessAuthorizeResult, error) {
	return nil, nil
}

func (c *heartbeatClient) CheckLimit(
	context.Context,
	*services.UsageLimitCheckRequest,
) (*services.UsageLimitCheckResult, error) {
	return nil, nil
}

func (c *heartbeatClient) RecordUsage(
	context.Context,
	*services.UsageRecordRequest,
) (*services.UsageRecordResult, error) {
	return nil, nil
}

func (c *heartbeatClient) Heartbeat(
	_ context.Context,
	req *services.InstanceHeartbeatRequest,
) (*services.InstanceHeartbeatResult, error) {
	c.called = true
	c.req = req
	if c.err != nil {
		return nil, c.err
	}
	return &services.InstanceHeartbeatResult{Accepted: true}, nil
}

func (c *heartbeatClient) SyncTenants(
	context.Context,
	*services.TenantSyncRequest,
) (*services.TenantSyncResult, error) {
	return nil, nil
}

func (c *heartbeatClient) GetBillingSummary(
	context.Context,
	*services.BillingSummaryRequest,
) (*services.BillingSummaryResult, error) {
	return nil, nil
}

func TestHeartbeatReporter_StartOnlyWhenControlPlaneEnabled(t *testing.T) {
	t.Parallel()

	registry := testRegistry(t)
	client := &heartbeatClient{}
	reporter := &HeartbeatReporter{
		cfg: &config.Config{
			App: config.AppConfig{Name: "trenova", Env: config.EnvTest, Version: "1.0.0"},
			Platform: config.PlatformConfig{
				InstanceID: "inst_01",
			},
		},
		client:   client,
		registry: registry,
		logger:   zap.NewNop(),
		now:      func() time.Time { return time.Unix(100, 0) },
	}

	require.NoError(t, reporter.start(t.Context()))
	require.False(t, client.called)
}

func TestHeartbeatReporter_FailsClosedOutsideDevelopment(t *testing.T) {
	t.Parallel()

	reporter := &HeartbeatReporter{
		cfg: &config.Config{
			App: config.AppConfig{Name: "trenova", Env: config.EnvProduction, Version: "1.0.0"},
			Platform: config.PlatformConfig{
				InstanceID: "inst_01",
				ControlPlane: config.PlatformControlPlaneConfig{
					Enabled:           true,
					HeartbeatInterval: time.Hour,
				},
			},
		},
		client:   &heartbeatClient{err: errors.New("down")},
		registry: testRegistry(t),
		logger:   zap.NewNop(),
		now:      func() time.Time { return time.Unix(100, 0) },
	}

	require.ErrorContains(t, reporter.start(t.Context()), "send control plane heartbeat")
}

func TestHeartbeatReporter_BuildsCatalogHeartbeat(t *testing.T) {
	t.Parallel()

	registry := testRegistry(t)
	client := &heartbeatClient{}
	reporter := &HeartbeatReporter{
		cfg: &config.Config{
			App: config.AppConfig{Name: "trenova", Env: config.EnvDevelopment, Version: "1.2.3"},
			Platform: config.PlatformConfig{
				Mode:       config.PlatformModeDevelopment,
				InstanceID: "inst_01",
				ControlPlane: config.PlatformControlPlaneConfig{
					Enabled:           true,
					HeartbeatInterval: time.Hour,
				},
			},
		},
		client:   client,
		registry: registry,
		logger:   zap.NewNop(),
		now:      func() time.Time { return time.Unix(100, 0) },
	}

	require.NoError(t, reporter.start(t.Context()))
	defer func() { require.NoError(t, reporter.stop(t.Context())) }()

	require.True(t, client.called)
	require.Equal(t, "inst_01", client.req.InstanceID)
	require.Equal(t, "1.2.3", client.req.AppVersion)
	require.NotEmpty(t, client.req.CatalogHash)
	require.NotEmpty(t, client.req.Products)
	require.NotEmpty(t, client.req.Features)
	require.NotEmpty(t, client.req.Meters)
}

func testRegistry(t *testing.T) *platformcatalog.Registry {
	t.Helper()

	registry, err := platformcatalog.NewRegistry(platformcatalog.RegistryParams{
		Providers: []platformcatalog.CatalogProvider{platformcatalog.NewStaticProvider()},
	})
	require.NoError(t, err)
	return registry
}
