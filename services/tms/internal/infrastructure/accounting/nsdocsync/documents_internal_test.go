package nsdocsync

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPurchaseIsCreditReadsEitherRef(t *testing.T) {
	t.Parallel()

	credit, err := PurchaseIsCredit(
		map[string]string{accountingsync.ExternalRefDocumentType: RecordVendorCredit},
	)
	require.NoError(t, err)
	assert.True(t, credit)
	credit, err = PurchaseIsCredit(nil)
	require.NoError(t, err)
	assert.False(t, credit)
	_, err = PurchaseIsCredit(map[string]string{accountingsync.ExternalRefDocumentType: "journal"})
	require.ErrorIs(t, err, nsbinding.ErrPurchaseDocumentType)
}
