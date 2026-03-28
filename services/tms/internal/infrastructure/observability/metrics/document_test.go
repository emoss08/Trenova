package metrics

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewDocument_Disabled(t *testing.T) {
	t.Parallel()

	m := NewDocument(nil, zap.NewNop(), false)

	require.NotNil(t, m)
	assert.False(t, m.IsEnabled())
}

func TestDocumentMetrics_Recorders(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	m := NewDocument(registry, zap.NewNop(), true)

	m.RecordExtraction("succeeded", documentcontent.SourceKindNative, "none")
	m.RecordExtraction("failed", "", "EXTRACTION_FAILED")
	m.RecordShipmentDraft("ready", "shipment", "RateConfirmation")
	m.RecordTypeAssociation("created", "RateConfirmation")
	m.RecordSearchProjectionSync(true)
	m.RecordSearchProjectionSync(false)
	m.RecordSearchQuery("meilisearch", "success")
	m.RecordReconciliationQueue(true)

	families, err := registry.Gather()
	require.NoError(t, err)

	names := make([]string, 0, len(families))
	for _, family := range families {
		names = append(names, family.GetName())
	}

	assert.Contains(t, names, "trenova_document_intelligence_extraction_total")
	assert.Contains(t, names, "trenova_document_intelligence_shipment_draft_total")
	assert.Contains(t, names, "trenova_document_intelligence_type_association_total")
	assert.Contains(t, names, "trenova_document_intelligence_search_projection_sync_total")
	assert.Contains(t, names, "trenova_document_intelligence_search_query_total")
	assert.Contains(t, names, "trenova_document_intelligence_reconciliation_queue_total")
}
