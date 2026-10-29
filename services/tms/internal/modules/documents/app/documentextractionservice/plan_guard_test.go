package documentextractionservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/documentconversionconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/searchconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentextractionservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestReextractRefusesWhenThePlanRestrictsDocumentIntelligence(t *testing.T) {
	t.Parallel()

	metricRegistry, err := metrics.NewRegistry(&config.Config{}, zap.NewNop())
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
		Plans: plantest.Restricting(
			t,
			platformplan.CapabilityDocumentIntelligence,
		),
	})

	err = service.Reextract(t.Context(), documentID, tenant)
	require.True(t, errortypes.IsPlanRestrictionError(err))
}
