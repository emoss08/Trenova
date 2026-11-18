package aianalyticsservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
)

func TestProviderRef_NamesTheBuiltInSearchService(t *testing.T) {
	t.Parallel()

	ref := providerIndex(i18n.EN, providers()).ref(aiprovider.BuiltInEmbeddingID.String())

	assert.Equal(t, "Trenova search service", ref.Name)
	assert.Equal(t, aiprovider.BuiltInEmbeddingID, ref.ID)
	assert.True(t, ref.Removed, "there is no provider of the organization's to open for it")
}
