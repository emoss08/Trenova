package canned_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/reporting/canned"
	"github.com/emoss08/trenova/internal/core/services/reporting/compiler"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type allowAllEngine struct{}

func (allowAllEngine) GetResourcePermissions(
	_ context.Context,
	_, _ pulid.ID,
	resource string,
) (*services.ResourcePermissionDetail, error) {
	return &services.ResourcePermissionDetail{
		Resource:       resource,
		Operations:     []permission.Operation{permission.OpRead, permission.OpExport},
		DataScope:      permission.DataScopeOrganization,
		MaxSensitivity: permission.SensitivityConfidential,
	}, nil
}

func (allowAllEngine) Check(
	context.Context, *services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	panic("not used")
}

func (allowAllEngine) CheckBatch(
	context.Context, *services.BatchPermissionCheckRequest,
) (*services.BatchPermissionCheckResult, error) {
	panic("not used")
}

func (allowAllEngine) GetLightManifest(
	context.Context, pulid.ID, pulid.ID,
) (*services.LightPermissionManifest, error) {
	panic("not used")
}

func (allowAllEngine) InvalidateUser(context.Context, pulid.ID, pulid.ID) error {
	panic("not used")
}

func (allowAllEngine) GetEffectivePermissions(
	context.Context, pulid.ID, pulid.ID,
) (*services.EffectivePermissions, error) {
	panic("not used")
}

func (allowAllEngine) SimulatePermissions(
	context.Context, *services.SimulatePermissionsRequest,
) (*services.EffectivePermissions, error) {
	panic("not used")
}

// TestEveryCannedReportCompilesAgainstCurrentCatalog is the CI gate for canned
// reports: schema or catalog drift that breaks a shipped report fails the
// build, not the customer.
func TestEveryCannedReportCompilesAgainstCurrentCatalog(t *testing.T) {
	c := compiler.NewWithCatalog(
		&reportcatalog.Default,
		allowAllEngine{},
		permission.NewRegistry(),
		&config.ReportingConfig{},
		zap.NewNop(),
	)

	tenant := pagination.TenantInfo{
		OrgID:  pulid.ID("org_test"),
		BuID:   pulid.ID("bu_test"),
		UserID: pulid.ID("usr_test"),
	}

	entries := canned.Default().All()
	require.NotEmpty(t, entries)

	for _, entry := range entries {
		t.Run(entry.Key, func(t *testing.T) {
			require.NotEmpty(t, entry.Version)
			require.NotEmpty(t, entry.Name)
			require.True(t, entry.DefaultFormat.IsValid())
			require.NotNil(t, entry.Definition)
			assert.Equal(t, report.CurrentIRVersion, entry.Definition.IRVersion)

			compiled, err := c.Compile(context.Background(), &services.ReportCompileRequest{
				Definition:  entry.Definition,
				Tenant:      tenant,
				OrgTimezone: "America/Chicago",
				NowUnix:     1784131200,
			})
			require.NoError(t, err, "canned report %q no longer compiles", entry.Key)
			assert.NotEmpty(t, compiled.SQL)
			assert.NotEmpty(t, compiled.Columns)
		})
	}
}

func TestRegistryLookup(t *testing.T) {
	registry := canned.Default()

	entry, ok := registry.Get("revenue-by-customer")
	require.True(t, ok)
	assert.Equal(t, "Revenue by Customer", entry.Name)

	_, ok = registry.Get("nonexistent")
	assert.False(t, ok)

	keys := make(map[string]bool)
	for _, e := range registry.All() {
		assert.False(t, keys[e.Key], "duplicate canned key %q", e.Key)
		keys[e.Key] = true
	}
}
