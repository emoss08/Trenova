package documentextractionservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/documentconversionconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentextractionservice"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
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
		name               string
		availability       documentaiextractionports.ExtractionAvailability
		converterAvailable bool
		want               documentports.ExtractionStatus
	}{
		{
			name:               "without a provider nothing is extracted",
			availability:       documentaiextractionports.ExtractionNoProviderAssigned,
			converterAvailable: true,
			want: documentports.ExtractionStatus{
				ModelProviderAvailable: false,
				ModelProviderReason:    "no_model_provider",
				ConverterAvailable:     true,
				ExtractionAvailable:    false,
			},
		},
		{
			name:               "with a provider and a converter documents are extracted",
			availability:       documentaiextractionports.ExtractionAvailable,
			converterAvailable: true,
			want: documentports.ExtractionStatus{
				ModelProviderAvailable: true,
				ModelProviderReason:    "available",
				ConverterAvailable:     true,
				ExtractionAvailable:    true,
			},
		},
		{
			name:               "without a converter nothing is extracted, provider or not",
			availability:       documentaiextractionports.ExtractionAvailable,
			converterAvailable: false,
			want: documentports.ExtractionStatus{
				ModelProviderAvailable: true,
				ModelProviderReason:    "available",
				ConverterAvailable:     false,
				ExtractionAvailable:    false,
			},
		},
		{
			name:               "a server with document extraction off names the switch",
			availability:       documentaiextractionports.ExtractionOffOnServer,
			converterAvailable: true,
			want: documentports.ExtractionStatus{
				ModelProviderAvailable: false,
				ModelProviderReason:    "off_on_server",
				ConverterAvailable:     true,
				ExtractionAvailable:    false,
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
			tenantInfo := pagination.TenantInfo{
				OrgID:  orgID,
				BuID:   buID,
				UserID: pulid.MustNew("usr_"),
			}

			aiDocuments := documentaiextractionportsmock.NewMockAIDocumentService(t)
			aiDocuments.EXPECT().
				ExtractionAvailability(mock.Anything, tenantInfo).
				Return(tc.availability, nil)

			cfg := &config.Config{}
			if tc.converterAvailable {
				cfg.DocumentConversion = documentconversionconfig.Config{
					DoclingURL: "http://docling:5001",
					APIKey:     "key",
				}
			}
			service := documentextractionservice.New(documentextractionservice.Params{
				Logger:                   zap.NewNop(),
				DocumentConversionConfig: &cfg.DocumentConversion,
				SearchConfig:             &cfg.Search,
				Metrics:                  metricRegistry,
				DocumentControlRepo:      tenantportsmock.NewMockDocumentControlRepository(t),
				DocumentRepo:             documentportsmock.NewMockDocumentRepository(t),
				ContentRepo: documentcontentportsmock.NewMockDocumentContentRepository(
					t,
				),
				DraftRepo: documentshipmentdraftportsmock.NewMockDocumentShipmentDraftRepository(
					t,
				),
				WorkflowStarter: kernelportsmock.NewMockWorkflowStarter(t),
				AIDocuments:     aiDocuments,
			})

			status, err := service.ExtractionStatus(t.Context(), tenantInfo)
			require.NoError(t, err)
			assert.Equal(t, tc.want, *status)
		})
	}
}
