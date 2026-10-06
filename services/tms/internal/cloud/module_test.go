package cloud_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/routegroup"
	"github.com/emoss08/trenova/internal/bootstrap"
	"github.com/emoss08/trenova/internal/cloud"
	"github.com/emoss08/trenova/internal/cloud/aitraining/aicli"
	"github.com/emoss08/trenova/internal/cloud/cloudconfig"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestEditionDescribesTheCloudBuild(t *testing.T) {
	t.Parallel()

	e := cloud.Edition()
	assert.Equal(t, cloud.EditionName, e.Name)
	assert.False(t, e.IsCommunity())
	require.Len(t, e.ConfigSections, 1)
	assert.Equal(t, cloudconfig.SectionName, e.ConfigSections[0].Name())
	require.Len(t, e.Commands, 2)
	assert.Equal(t, "ai", e.Commands[0].Name())
	assert.Equal(t, "cloud", e.Commands[1].Name())
	assert.NotEmpty(t, e.Options)
	assert.NotEmpty(t, e.APIOptions)
}

func TestCloudAPIGraphResolves(t *testing.T) {
	t.Parallel()

	e := cloud.Edition()
	require.NoError(t, fx.ValidateApp(
		bootstrap.OptionsFor(&e),
		bootstrap.APIOptionsFor(&e),
		fx.Invoke(func(
			services.PlanService,
			services.QuotaGuard,
			services.EntitlementProvider,
			services.BillingProvider,
			services.UsageProvider,
			services.AccessAuthorizer,
			services.EditionInfo,
			services.PlatformEmailService,
			services.TenantProvisioningService,
			services.AITrainingExportStarter,
			services.AITrainingExportOperator,
			services.AITrainingExportRunner,
			services.AITrainingHistoryService,
			services.AITrainingDatasetRenderer,
			routegroup.Registrars,
		) {
		}),
	))
}

func TestCloudWorkerGraphResolves(t *testing.T) {
	t.Parallel()

	e := cloud.Edition()
	require.NoError(t, fx.ValidateApp(
		bootstrap.OptionsFor(&e),
		bootstrap.WorkerOptionsFor(&e),
		fx.Invoke(func(
			services.PlanService,
			services.QuotaGuard,
			services.UsageProvider,
			services.AITrainingExportStarter,
			services.AITrainingHistoryService,
		) {
		}),
	))
}

func TestTrainingExportCommandResolvesWithoutAProcess(t *testing.T) {
	t.Parallel()

	require.NoError(t, fx.ValidateApp(
		aicli.CommandOptions(),
		fx.Invoke(func(
			services.AITrainingExportOperator,
			services.AITrainingDatasetRenderer,
			services.AIRetrainingService,
		) {
		}),
	))
}
