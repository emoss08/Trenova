package controlplane

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/require"
)

type failingClient struct{}

func (c failingClient) CheckFeature(
	context.Context,
	*services.FeatureCheckRequest,
) (*services.FeatureCheckResult, error) {
	return nil, errors.New("unavailable")
}

func (c failingClient) AuthorizeAccess(
	context.Context,
	*services.AccessAuthorizeRequest,
) (*services.AccessAuthorizeResult, error) {
	return nil, errors.New("unavailable")
}

func (c failingClient) CheckLimit(
	context.Context,
	*services.UsageLimitCheckRequest,
) (*services.UsageLimitCheckResult, error) {
	return nil, errors.New("unavailable")
}

func (c failingClient) RecordUsage(
	context.Context,
	*services.UsageRecordRequest,
) (*services.UsageRecordResult, error) {
	return nil, errors.New("unavailable")
}

func (c failingClient) Heartbeat(
	context.Context,
	*services.InstanceHeartbeatRequest,
) (*services.InstanceHeartbeatResult, error) {
	return nil, errors.New("unavailable")
}

func (c failingClient) SyncTenants(
	context.Context,
	*services.TenantSyncRequest,
) (*services.TenantSyncResult, error) {
	return nil, errors.New("unavailable")
}

func (c failingClient) GetBillingSummary(
	context.Context,
	*services.BillingSummaryRequest,
) (*services.BillingSummaryResult, error) {
	return nil, errors.New("unavailable")
}

func TestCloudUsageProvider_RequiresIdempotencyKey(t *testing.T) {
	t.Parallel()

	provider := NewCloudUsageProvider(CloudUsageProviderParams{
		Config: &config.Config{},
		Client: failingClient{},
	})

	result, err := provider.RecordUsage(context.Background(), &services.UsageRecordRequest{
		MeterKey: platformcatalog.MeterAPIRequests,
		Quantity: 1,
	})

	require.Nil(t, result)
	require.ErrorContains(t, err, "idempotency key is required")
}

func TestCloudUsageProvider_FailOpenOnlyInDevelopment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		cfg         *config.Config
		wantAllowed bool
		wantErr     bool
	}{
		{
			name: "development fail open",
			cfg: &config.Config{
				App: config.AppConfig{Env: config.EnvDevelopment},
				Platform: config.PlatformConfig{
					ControlPlane: config.PlatformControlPlaneConfig{FailOpenOnError: true},
				},
			},
			wantAllowed: true,
		},
		{
			name: "test fails closed",
			cfg: &config.Config{
				App: config.AppConfig{Env: config.EnvTest},
				Platform: config.PlatformConfig{
					ControlPlane: config.PlatformControlPlaneConfig{FailOpenOnError: true},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewCloudUsageProvider(CloudUsageProviderParams{
				Config: tt.cfg,
				Client: failingClient{},
			})

			result, err := provider.CheckLimit(context.Background(), &services.UsageLimitCheckRequest{
				MeterKey:       platformcatalog.MeterAPIRequests,
				Quantity:       1,
				IdempotencyKey: "request-1",
			})

			if tt.wantErr {
				require.Nil(t, result)
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantAllowed, result.Allowed)
			require.True(t, result.FailOpen)
		})
	}
}
