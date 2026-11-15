package documentextractionservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/documentconversionconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/monitoringconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/searchconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentextractionservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestReextractRefusesWhenDocumentIntelligenceIsOff(t *testing.T) {
	t.Parallel()

	metricRegistry, err := metrics.NewRegistry(&monitoringconfig.Config{}, zap.NewNop())
	require.NoError(t, err)
	tenant := plantest.Tenant()
	documentID := pulid.MustNew("doc_")

	documentRepo := documentportsmock.NewMockDocumentRepository(t)
	documentRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&document.Document{
		ID:                documentID,
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		ProcessingProfile: document.IntelligenceProcessingProfiles()[0],
	}, nil)

	service := documentextractionservice.New(documentextractionservice.Params{
		Logger:                   zap.NewNop(),
		DocumentConversionConfig: &documentconversionconfig.Config{},
		SearchConfig:             &searchconfig.Config{},
		Metrics:                  metricRegistry,
		DocumentRepo:             documentRepo,
		Features:                 plantest.Without(platformcatalog.FeatureDocumentIntelligence),
	})

	err = service.Reextract(t.Context(), documentID, tenant)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}

func TestUploadingADocumentMakesNoExtractionCallWithoutDocumentIntelligence(t *testing.T) {
	t.Parallel()

	metricRegistry, err := metrics.NewRegistry(&monitoringconfig.Config{}, zap.NewNop())
	require.NoError(t, err)
	tenant := plantest.Tenant()

	controls := tenantportsmock.NewMockDocumentControlRepository(t)
	workflows := kernelportsmock.NewMockWorkflowStarter(t)

	service := documentextractionservice.New(documentextractionservice.Params{
		Logger:                   zap.NewNop(),
		DocumentConversionConfig: &documentconversionconfig.Config{},
		SearchConfig:             &searchconfig.Config{},
		Metrics:                  metricRegistry,
		DocumentControlRepo:      controls,
		WorkflowStarter:          workflows,
		Features:                 plantest.BaseOnly(),
	})

	err = service.EnqueueExtraction(t.Context(), &document.Document{
		ID:                pulid.MustNew("doc_"),
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		ProcessingProfile: document.IntelligenceProcessingProfiles()[0],
	}, tenant.UserID)

	require.NoError(t, err, "the control is not read and no workflow starts: either call fails the mock")
}
