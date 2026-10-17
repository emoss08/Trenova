package documentextractionservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentextractionservice"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentaiextractionportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentcontentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentshipmentdraftportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestExtractionStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name              string
		engine            tenant.ExtractionEngine
		providerAvailable bool
		want              documentports.ExtractionStatus
	}{
		{
			name:              "docling without a provider cannot extract",
			engine:            tenant.ExtractionEngineDocling,
			providerAvailable: false,
			want: documentports.ExtractionStatus{
				Engine:                 "docling",
				ModelProviderAvailable: false,
				ExtractionAvailable:    false,
			},
		},
		{
			name:              "docling with a provider extracts",
			engine:            tenant.ExtractionEngineDocling,
			providerAvailable: true,
			want: documentports.ExtractionStatus{
				Engine:                 "docling",
				ModelProviderAvailable: true,
				ExtractionAvailable:    true,
			},
		},
		{
			name:              "legacy reads documents without a provider",
			engine:            tenant.ExtractionEngineLegacy,
			providerAvailable: false,
			want: documentports.ExtractionStatus{
				Engine:                 "legacy",
				ModelProviderAvailable: false,
				ExtractionAvailable:    true,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			metricRegistry, err := metrics.NewRegistry(&config.Config{}, zap.NewNop())
			require.NoError(t, err)

			orgID := pulid.MustNew("org_")
			buID := pulid.MustNew("bu_")
			tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: pulid.MustNew("usr_")}

			control := tenant.NewDefaultDocumentControl(orgID, buID)
			control.ExtractionEngine = tc.engine
			controlRepo := tenantportsmock.NewMockDocumentControlRepository(t)
			controlRepo.EXPECT().GetOrCreate(mock.Anything, orgID, buID).Return(control, nil)

			aiDocuments := documentaiextractionportsmock.NewMockAIDocumentService(t)
			aiDocuments.EXPECT().
				ExtractionAvailable(mock.Anything, tenantInfo).
				Return(tc.providerAvailable, nil)

			service := documentextractionservice.New(documentextractionservice.Params{
				Logger:              zap.NewNop(),
				Config:              &config.Config{},
				Metrics:             metricRegistry,
				DocumentControlRepo: controlRepo,
				DocumentRepo:        documentportsmock.NewMockDocumentRepository(t),
				ContentRepo:         documentcontentportsmock.NewMockDocumentContentRepository(t),
				DraftRepo:           documentshipmentdraftportsmock.NewMockDocumentShipmentDraftRepository(t),
				WorkflowStarter:     kernelportsmock.NewMockWorkflowStarter(t),
				AIDocuments:         aiDocuments,
			})

			status, err := service.ExtractionStatus(t.Context(), tenantInfo)
			require.NoError(t, err)
			assert.Equal(t, tc.want, *status)
		})
	}
}
