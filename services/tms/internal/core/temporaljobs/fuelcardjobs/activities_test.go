package fuelcardjobs_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/temporaljobs/fuelcardjobs"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubIntegrationRepo struct {
	byType map[integration.Type][]*integration.Integration
}

func (s *stubIntegrationRepo) ListByTenant(
	context.Context,
	pagination.TenantInfo,
) ([]*integration.Integration, error) {
	return nil, nil
}

func (s *stubIntegrationRepo) ListEnabledByType(
	_ context.Context,
	typ integration.Type,
) ([]*integration.Integration, error) {
	return s.byType[typ], nil
}

func (s *stubIntegrationRepo) GetByType(
	context.Context,
	pagination.TenantInfo,
	integration.Type,
) (*integration.Integration, error) {
	return nil, nil
}

func (s *stubIntegrationRepo) Upsert(
	_ context.Context,
	entity *integration.Integration,
) (*integration.Integration, error) {
	return entity, nil
}

// A complete Comdata file connection. HasRequiredConfiguration checks every field
// the spec marks required, so anything short of this is treated as not connected.
func comdataConfig() map[string]any {
	return map[string]any{
		integration.ConfigKeyFuelAccountNumber:    "ACC-1",
		integration.ConfigKeyFuelHost:             "sftp.example.com",
		integration.ConfigKeyFuelUsername:         "trenova",
		integration.ConfigKeyFuelAuthMode:         integration.FuelAuthModePassword,
		integration.ConfigKeyFuelKnownHostKey:     "ssh-ed25519 AAAA",
		integration.ConfigKeyFuelRemoteDir:        "/outbound",
		integration.ConfigKeyFuelFileFormat:       integration.FuelFileFormatFixedWidth,
		integration.ConfigKeyFuelFixedWidthLayout: "Date:1-8",
	}
}

func record(orgID, buID pulid.ID, config map[string]any) *integration.Integration {
	return &integration.Integration{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Configuration:  config,
	}
}

func newActivities(repo *stubIntegrationRepo) *fuelcardjobs.Activities {
	return fuelcardjobs.NewActivities(fuelcardjobs.ActivitiesParams{
		IntegrationRepo: repo,
		Logger:          zap.NewNop(),
	})
}

// A carrier can hold cards on more than one network. The tenant has to be listed
// once even so, because the sync reads every one of its feeds in a single pass.
func TestListTenantsReportsATenantOnceAcrossProviders(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	repo := &stubIntegrationRepo{byType: map[integration.Type][]*integration.Integration{
		integration.TypeComdataFuel: {record(orgID, buID, comdataConfig())},
		integration.TypeWEXFuel: {record(orgID, buID, func() map[string]any {
			config := comdataConfig()
			config[integration.ConfigKeyFuelCardProvider] = "WEX"
			config[integration.ConfigKeyFuelFileFormat] = integration.FuelFileFormatDelimited
			return config
		}())},
	}}

	result, err := newActivities(repo).ListFuelCardTenantsActivity(
		t.Context(),
		&fuelcardjobs.ListFuelCardTenantsPayload{},
	)
	require.NoError(t, err)
	require.Len(t, result.Tenants, 1)
	require.Equal(t, orgID, result.Tenants[0].OrganizationID)
	require.Equal(t, buID, result.Tenants[0].BusinessUnitID)
}

// A connection somebody started and never finished must not be polled: dialling
// it would fail every hour and bury the tenants that are actually configured.
func TestListTenantsSkipsAnIncompleteConnection(t *testing.T) {
	t.Parallel()

	incomplete := comdataConfig()
	delete(incomplete, integration.ConfigKeyFuelHost)

	repo := &stubIntegrationRepo{byType: map[integration.Type][]*integration.Integration{
		integration.TypeComdataFuel: {
			record(pulid.MustNew("org_"), pulid.MustNew("bu_"), incomplete),
		},
	}}

	result, err := newActivities(repo).ListFuelCardTenantsActivity(
		t.Context(),
		&fuelcardjobs.ListFuelCardTenantsPayload{},
	)
	require.NoError(t, err)
	require.Empty(t, result.Tenants)
}

func TestListTenantsReportsEveryConnectedTenant(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	first := record(pulid.MustNew("org_"), buID, comdataConfig())
	second := record(pulid.MustNew("org_"), buID, comdataConfig())

	repo := &stubIntegrationRepo{byType: map[integration.Type][]*integration.Integration{
		integration.TypeComdataFuel: {first, second},
	}}

	result, err := newActivities(repo).ListFuelCardTenantsActivity(
		t.Context(),
		&fuelcardjobs.ListFuelCardTenantsPayload{},
	)
	require.NoError(t, err)
	require.Len(t, result.Tenants, 2)
}

func TestListTenantsHonoursTheScanLimit(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	records := make([]*integration.Integration, 0, 3)
	for range 3 {
		records = append(records, record(pulid.MustNew("org_"), buID, comdataConfig()))
	}

	repo := &stubIntegrationRepo{byType: map[integration.Type][]*integration.Integration{
		integration.TypeComdataFuel: records,
	}}

	result, err := newActivities(repo).ListFuelCardTenantsActivity(
		t.Context(),
		&fuelcardjobs.ListFuelCardTenantsPayload{Limit: 2},
	)
	require.NoError(t, err)
	require.Len(t, result.Tenants, 2)
}

func TestListTenantsIsEmptyWhenNobodyIsConnected(t *testing.T) {
	t.Parallel()

	result, err := newActivities(&stubIntegrationRepo{}).ListFuelCardTenantsActivity(
		t.Context(),
		&fuelcardjobs.ListFuelCardTenantsPayload{},
	)
	require.NoError(t, err)
	require.Empty(t, result.Tenants)
}
