package documentextractionservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/documentconversionconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/monitoringconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/searchconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentextractionservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentshipmentdraftportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSearchDocuments_KeepsAQuotedResourceInsideItsOwnFilterCondition(t *testing.T) {
	t.Parallel()

	metricRegistry, err := metrics.NewRegistry(&monitoringconfig.Config{}, zap.NewNop())
	require.NoError(t, err)

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	documentID := pulid.MustNew("doc_")
	resourceID := `shp_1" OR organization_id EXISTS OR resource_id = "x`

	searchRepo := customerportsmock.NewMockSearchRepository(t)
	searchRepo.EXPECT().Enabled().Return(true)
	var request customerports.SearchRequest
	searchRepo.EXPECT().
		Search(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req customerports.SearchRequest) ([]map[string]any, error) {
			request = req
			return []map[string]any{{"id": documentID.String()}}, nil
		})

	documentRepo := documentportsmock.NewMockDocumentRepository(t)
	documentRepo.EXPECT().
		GetByIDs(mock.Anything, documentports.BulkDeleteDocumentRequest{
			IDs:        []pulid.ID{documentID},
			TenantInfo: tenantInfo,
		}).
		Return([]*document.Document{{ID: documentID}}, nil)

	service := documentextractionservice.New(documentextractionservice.Params{
		Logger:                   zap.NewNop(),
		DocumentConversionConfig: &documentconversionconfig.Config{},
		SearchConfig: &searchconfig.Config{
			Meilisearch: searchconfig.MeilisearchConfig{
				Indexes: searchconfig.MeilisearchIndexConfig{Documents: "documents"},
			},
		},
		Metrics:      metricRegistry,
		DocumentRepo: documentRepo,
		SearchRepo:   searchRepo,
		DraftRepo:    documentshipmentdraftportsmock.NewMockDocumentShipmentDraftRepository(t),
	})

	docs, err := service.SearchDocuments(t.Context(), tenantInfo, "shipment", resourceID, "invoice")
	require.NoError(t, err)
	require.Len(t, docs, 1)

	assert.Equal(t, []string{"id"}, request.Attributes)
	assert.Equal(
		t,
		`(organization_id = "`+tenantInfo.OrgID.String()+`") AND `+
			`(business_unit_id = "`+tenantInfo.BuID.String()+`") AND `+
			`(resource_type = "shipment") AND `+
			`(resource_id = "shp_1\" OR organization_id EXISTS OR resource_id = \"x")`,
		request.Filter,
	)
}
